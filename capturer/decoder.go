package capturer

import (
	"context"
	"data-collector/common"
	log "github.com/sirupsen/logrus"
	"io"
)

type DecodeCtx struct {
	Conn         *TCPConn
	SendBackChan chan interface{}
	CtlChan      chan interface{}
}

var factoryFunctions map[string]func() Interface

func init() {
	factoryFunctions = make(map[string]func() Interface, 8)
	//RegisterDecoder(DefaultDecoderName, instanceDefaultDecode)
}

//LocateToValidStartInC2SStream and LocateToValidStartInS2CStream will be used to
type Interface interface {
	Name() string
	//read util find fist valid byte and return the number of invalid bytes
	LocateToValidStartInC2SStream(reader io.Reader) (int, error)
	//read util find fist valid byte and return the number of invalid bytes
	LocateToValidStartInS2CStream(reader io.Reader) (int, error)
	//control which method will be used to make this decoder take effect
	// 0 no method will be used
	// 1 only LocateToValidStartInC2SStream will be used
	// 2 only LocateToValidStartInS2CStream will be used
	// 3 both method will be used
	ValidateType() byte
	//parse stream to protocol data
	Parse(ctx context.Context, conn *TCPConn, actorCtx *common.ActorTreeContext)
}

type defaultDecoder struct {
}

var DefaultDecoder = defaultDecoder{}
var DefaultDecoderName = "defaultDecoder"

func (d defaultDecoder) Name() string {
	return "DefaultEmptyDecoder"
}

func (d defaultDecoder) LocateToValidStartInC2SStream(reader io.Reader) (int, error) {
	return 0, nil
}

func (d defaultDecoder) LocateToValidStartInS2CStream(reader io.Reader) (int, error) {
	return 0, nil
}

func (d defaultDecoder) ValidateType() byte {
	return 1
}

func (d defaultDecoder) Parse(ctx context.Context, conn *TCPConn, actorCtx *common.ActorTreeContext) {
	conn.C2SStream().SetValidStatus()
	conn.S2CStream().SetValidStatus()
	temp := make([]byte, 1024, 1024)

	c2sStreamBroken, s2cStreamBroken := false, false
	if c2sStreamBroken && s2cStreamBroken {
		return
	}

	if !c2sStreamBroken {
		c, err := conn.C2SStream().Read(temp)
		if err != nil {
			log.WithFields(log.Fields{"clientAddr": conn.GetClientAddr().String(), "serverAddr": conn.GetServerAddr().String(), "errMsg": err.Error()}).Debug("default decoder read client to server stream failed")
			c2sStreamBroken = true
		}
		content := temp[0:c]
		log.WithFields(log.Fields{"clientAddr": conn.GetClientAddr().String(), "serverAddr": conn.GetServerAddr().String(), "count": c, "content": content}).Debug("default decoder read  client to server stream success")
	}

	if !s2cStreamBroken {
		c, err := conn.S2CStream().Read(temp)
		if err != nil {
			log.WithFields(log.Fields{"clientAddr": conn.GetClientAddr().String(), "serverAddr": conn.GetServerAddr().String(), "errMsg": err.Error()}).Debug("default decoder read server to client stream failed")
			c2sStreamBroken = true
		}
		content := temp[0:c]
		log.WithFields(log.Fields{"clientAddr": conn.GetClientAddr().String(), "serverAddr": conn.GetServerAddr().String(), "count": c, "content": content}).Debug("default decoder read  server to clientr stream success")
	}
}

func RegisterDecoder(name string, factoryFunc func() Interface) {
	factoryFunctions[name] = factoryFunc
}

func instanceDefaultDecode() Interface {
	return DefaultDecoder
}

func GetDecoders() []Interface {
	ds := make([]Interface, 0, 8)
	for _, fu := range factoryFunctions {
		ds = append(ds, fu())
	}
	return ds
}

func GetDecoder(name string) Interface {
	fu, ok := factoryFunctions[name]
	if !ok {
		return DefaultDecoder
	}
	return fu()
}
