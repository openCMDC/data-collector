package conns

import (
	"data-collector/common"
	"data-collector/decoder"
	"data-collector/msg"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	log "github.com/sirupsen/logrus"
	"net"
	"sort"
	"strings"
)

type endPointManager struct {
	decoders     map[string]*common.ActorWrapper
	actorTreeCtx *common.ActorTreeContext
}

type endPoint struct {
	parser     decoder.Interface
	msgChan    chan interface{}
	serverAddr *net.TCPAddr
	conns      map[string]*common.TCPConn
	ctlChans   map[string]chan interface{}
	works      chan *common.TCPConn
}

func (d *endPointManager) Receive(ctx actor.Context) {
	m := ctx.Message()
	log.WithField("type", fmt.Sprintf("%T", m)).Trace("endPointManager receive a msg")
	switch message := m.(type) {
	case msg.StreamCreatedMsg:
		d.processStreamCreatedMsg(ctx, &message)
	case *msg.StreamCreatedMsg:
		d.processStreamCreatedMsg(ctx, message)
	}
}

func (d *endPoint) Receive(ctx actor.Context) {
	m := ctx.Message()
	log.WithField("type", fmt.Sprintf("%T", m)).Trace("endPoint receive a msg")
	switch message := m.(type) {
	case msg.StreamCreatedMsg:
		d.processStreamCreatedMsg(ctx, &message)
	case *msg.StreamCreatedMsg:
		d.processStreamCreatedMsg(ctx, message)
	}
}

func (d *endPoint) init() {
	log.WithFields(log.Fields{"serverAddr": d.serverAddr.String()}).Trace("start init endPoint")
	go func() {
		for conn := range d.works {
			log.WithFields(log.Fields{"serverAddr": conn.GetServerAddr().String(), "clientAddr": conn.GetClientAddr().String()}).Trace("catch a new connection")
			ctlChan := make(chan interface{})
			d.ctlChans[conn.GetClientAddr().String()] = ctlChan
			ctx := &decoder.ParseCtx{
				Conn:         conn,
				SendBackChan: d.msgChan,
				CtlChan:      ctlChan,
			}
			go d.parser.Parse(ctx)
		}
	}()

	go func() {
		for msg := range d.msgChan {
			fmt.Println(msg)
			//todo 处理msg
		}
	}()
}

func (d *endPointManager) processStreamCreatedMsg(ctx actor.Context, message *msg.StreamCreatedMsg) {
	log.WithFields(log.Fields{"clientAddr": message.Client.String(),
		"serverAddr": message.Server.String(),
		"streamType": message.Type.String(),
		"device":     message.Device}).
		Trace("start processStreamCreatedMsg")
	key := message.Server.String()
	decoder, exist := d.decoders[key]
	if !exist {
		prop := actor.PropsFromFunc(newEndPoint(message.Server).Receive)
		pid, err := ctx.SpawnNamed(prop, key)
		if err != nil {
			log.WithFields(log.Fields{"actorName": key}).Warn("err creating decode actor")
			return
		}
		d.decoders[key] = &common.ActorWrapper{
			ActorName: key,
			Pid:       pid,
			Meta:      make(map[string]interface{}),
		}
	}
	decoder = d.decoders[key]
	ctx.Send(decoder.Pid, message)
}

func (d *endPoint) processStreamCreatedMsg(ctx actor.Context, message *msg.StreamCreatedMsg) {
	log.WithFields(log.Fields{"clientAddr": message.Client.String(),
		"serverAddr": message.Server.String(),
		"streamType": message.Type.String(),
		"device":     message.Device}).
		Trace("start processStreamCreatedMsg")
	t := message.Type
	key := message.Client.String()
	conn, exist := d.conns[key]
	if !exist {
		var conn *common.TCPConn
		if t == msg.Client2ServerStream {
			conn = common.NewTCPConn(message.Client, message.Server, nil, nil)
		} else if t == msg.Server2ClientStream {
			conn = common.NewTCPConn(message.Client, message.Server, nil, nil)
		} else {
			log.WithField("type", t).Warn("unsuporeted stream type, why?")
			return
		}
		d.conns[key] = conn
		d.addWork(conn)
	}
	conn = d.conns[key]
	if t == msg.Client2ServerStream {
		if conn.CheckC2SStreamExist() {
			log.Warn("unusual situation: duplicated stream set for client to server stream")
		}
		conn.SetC2SStream(message.Stream)
	} else if t == msg.Server2ClientStream {
		if conn.CheckS2CStreamExist() {
			log.Warn("unusual situation: duplicated stream set for server to client stream")
		}
		conn.SetS2CStream(message.Stream)
	} else {
		log.WithField("streamtype", t).Warn("unsupported stream type")
	}
	//todo notify decoding gorouting switch stream?
}

func (d *endPoint) addWork(conn *common.TCPConn) {
	d.works <- conn
}

func generateTCPConnName(addr1, addr2 *net.TCPAddr) string {
	s1, s2 := addr1.String(), addr2.String()
	slice := []string{s1, s2}
	sort.Strings(slice)
	return strings.Join(slice, "->")
}

func newEndPoint(serverAddr *net.TCPAddr) actor.Actor {
	d := &endPoint{
		parser:     decoder.GetProperParser(serverAddr),
		msgChan:    make(chan interface{}, 500),
		serverAddr: serverAddr,
		conns:      make(map[string]*common.TCPConn),
		ctlChans:   make(map[string]chan interface{}, 1),
		works:      make(chan *common.TCPConn),
	}
	d.init()
	return d
}

func NewEndPointManager(ctx *common.ActorTreeContext) *endPointManager {
	em := new(endPointManager)
	em.actorTreeCtx = ctx
	em.decoders = make(map[string]*common.ActorWrapper)
	return em
}
