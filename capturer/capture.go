package capturer

import (
	"data-collector/common"
	msg2 "data-collector/msg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

type DefaultStreamFactory struct {
	actorTreeContext *common.ActorTreeContext
	actor            *capture
}

func (sf *DefaultStreamFactory) New(netLayer, transLayer gopacket.Flow) tcpassembly.Stream {
	reader := tcpreader.NewReaderStream()
	actorCtx := sf.actorTreeContext.GetActorContext()
	decoderManager := sf.actorTreeContext.GetDecoderManager()

	srcAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Src().String(), transLayer.Src().String())
	if err != nil {
		log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
		return &reader
	}

	dstAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Dst().String(), transLayer.Dst().String())
	if err != nil {
		log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
		return &reader
	}
	clientAddr, serverAddr := getClientServerAddr(srcAddr, dstAddr, sf.actor.addr)
	if clientAddr == nil || serverAddr == nil {
		log.WithFields(log.Fields{"srcAddr": srcAddr.String(), "dstAddr": dstAddr.String()}).Warn("fail to recognise client addr and server addr")
		return &reader
	}
	//
	//   src          dst    src         dst
	//client --------> server1 --------> server2
	//client <-------- server1 <-------- server2
	//   dst          src    dst         src
	//   if src == clientAddr
	var st msg2.StreamType
	if clientAddr == srcAddr {
		st = msg2.Client2ServerStream
	} else {
		st = msg2.Server2ClientStream
	}
	msg := &msg2.StreamCreatedMsg{
		Client: clientAddr,
		Server: serverAddr,
		Stream: &reader,
		Type:   st,
		Device: sf.actor.deviceName,
	}
	actorCtx.Send(decoderManager, msg)
	return &reader
}

func newDSF(actorTreeContext *common.ActorTreeContext, actor *capture) *DefaultStreamFactory {
	return &DefaultStreamFactory{
		actorTreeContext: actorTreeContext,
		actor:            actor,
	}
}

func getClientServerAddr(addr1, addr2 *net.TCPAddr, addrs []net.Addr) (clientAddr, serverAddr *net.TCPAddr) {
	for _, add := range addrs {
		if strings.Contains(add.String(), addr1.String()) {
			return addr2, addr1
		}
		if strings.Contains(add.String(), addr2.String()) {
			return addr1, addr2
		}
	}
	return nil, nil
}
