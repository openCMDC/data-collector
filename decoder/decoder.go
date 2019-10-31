package decoder

import (
	"data-collector/common"
	"net"
)

type ParseCtx struct {
	Conn         *common.TCPConn
	SendBackChan chan interface{}
	CtlChan      chan interface{}
}

type Interface interface {
	Parse(ctx *ParseCtx)
	//SearchValidBytesInClient2ServerStream(reader io.Reader) (int, error)
	//SearchValidBytesInServer2ClientStream(reader io.Reader) (int, error)
}

func GetProperParser(addr *net.TCPAddr) Interface {
	//todo
	return new(httpDecoder)
	return nil
}
