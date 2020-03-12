package capturer

import (
	"bufio"
	"data-collector/common"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"sync"
)

type StreamType int8

const (
	_ StreamType = iota
	Client2ServerStream
	Server2ClientStream
)

func (t StreamType) String() string {
	if t == Client2ServerStream {
		return "Client2ServerStream"
	} else if t == Server2ClientStream {
		return "Server2ClientStream"
	} else {
		return "unKnown stream type"
	}
}

type DefaultStreamFactory struct {
	actorTreeContext *common.ActorTreeContext
	actor            *capture
	workChan         chan *SingleDirectionStream
	serverEndpoints  map[string]*ServerEndPoint
}

type ServerEndPoint struct {
	treeCtx      *common.ActorTreeContext
	serverAddr   *net.TCPAddr
	decoderReady chan struct{}
	fistConn     bool
	conns        map[string]*TCPConn
	decoderName  string
	mutex        sync.RWMutex
}

type SingleDirectionStream struct {
	reader         *StreamReader
	netLayer       gopacket.Flow
	transportLayer gopacket.Flow
	streamFactory  *DefaultStreamFactory
}

func (sf *DefaultStreamFactory) New(netLayer, transLayer gopacket.Flow) tcpassembly.Stream {
	reader := NewReaderStream()
	s := new(SingleDirectionStream)
	s.reader = &reader
	s.netLayer = netLayer
	s.transportLayer = transLayer
	s.streamFactory = sf

	if netLayer.Src().String() == "192.168.31.101" {
		go func() {
			for {
				_, err := http.ReadRequest(bufio.NewReader(&reader))
				if err != nil {
					fmt.Println(err)
					return
				}
				//fmt.Println("req success")
			}

		}()
	} else {
		go func() {

			for {
				b := make([]byte, 0, 8)
				c, err := reader.Read(b)
				if err != nil {
					fmt.Println(err)
					return
				}
				if c == 0 {
					continue
				}
				fmt.Println(b[0:c])
			}

		}()
	}

	//sf.workChan <- s
	return &reader
	//actorCtx := sf.actorTreeContext.GetActorContext()
	//decoderManager := sf.actorTreeContext.GetDecoderManager()
	//
	//srcAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Src().String(), transLayer.Src().String())
	//if err != nil {
	//	log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
	//	return &Reader
	//}
	//
	//dstAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Dst().String(), transLayer.Dst().String())
	//if err != nil {
	//	log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
	//	return &Reader
	//}
	//clientAddr, serverAddr := distinguishStreamDrc(srcAddr, dstAddr, sf.actor.addr)
	//if clientAddr == nil || serverAddr == nil {
	//	log.WithFields(log.Fields{"srcAddr": srcAddr.String(), "dstAddr": dstAddr.String()}).Warn("fail to recognise client addr and server addr")
	//	return &Reader
	//}
	////
	////   src          dst    src         dst
	////client --------> server1 --------> server2
	////client <-------- server1 <-------- server2
	////   dst          src    dst         src
	////   if src == clientAddr
	//var st msg2.StreamType
	//if clientAddr == srcAddr {
	//	st = msg2.Client2ServerStream
	//} else {
	//	st = msg2.Server2ClientStream
	//}
	//msg := &msg2.StreamCreatedMsg{
	//	Client: clientAddr,
	//	Server: serverAddr,
	//	Stream: &Reader,
	//	Type:   st,
	//	Device: sf.actor.deviceName,
	//}
	//actorCtx.Send(decoderManager, msg)
	//return &Reader
}

func (sf *DefaultStreamFactory) startParseStream() {
	go func() {
		for sds := range sf.workChan {
			ca, sa, st, err := sds.getInfo()
			if err != nil {
				log.WithField("errMsg", err.Error()).Warn("parse stream failed")
				continue
			}
			if ca == nil || sa == nil || st == -1 {
				continue
			}
			endPoint, ok := sf.serverEndpoints[sa.String()]
			if !ok {
				conn := NewTCPConn(ca, sa, nil, nil)
				endPoint = newServerEndpoint(conn)
				sf.serverEndpoints[sa.String()] = endPoint
			}
			endPoint.addStream(ca, sa, sds.reader, st)
		}
		log.Debug("stream factory work fetcher stopped")
	}()
}

func newDSF(actorTreeContext *common.ActorTreeContext, actor *capture) *DefaultStreamFactory {
	f := &DefaultStreamFactory{
		actorTreeContext: actorTreeContext,
		actor:            actor,
		workChan:         make(chan *SingleDirectionStream, 1),
		serverEndpoints:  make(map[string]*ServerEndPoint, 100),
	}
	f.startParseStream()
	return f
}

func newServerEndpoint(conn *TCPConn) *ServerEndPoint {
	se := new(ServerEndPoint)
	se.serverAddr = conn.GetServerAddr()
	se.decoderReady = make(chan struct{})
	se.conns = make(map[string]*TCPConn)
	se.conns[conn.GetClientAddr().String()] = conn
	se.fistConn = true
	se.decoderName = DefaultDecoderName
	se.startParseTcpConn(conn)
	//the first conn is used to chose proper decoder
	go func() {
		ds := GetDecoders()
		wg := &sync.WaitGroup{}
		for _, de := range ds {
			wg.Add(1)
			d := de
			go func() {
				defer wg.Done()
				vt := d.ValidateType()
				validateSuccess := false
				if vt == 1 || vt == 2 {
					var count int
					var err error
					var stream *ConcurrentReader
					if vt == 1 {
						stream = conn.C2SStream()
						count, err = d.LocateToValidStartInC2SStream(newXReader(stream))
					} else {
						stream = conn.S2CStream()
						count, err = d.LocateToValidStartInS2CStream(newXReader(stream))
					}

					if err != nil {
						log.WithField("decoderName", d.Name()).Debug("validate decoder failed")
						return
					}
					log.WithField("decoderName", d.Name()).Debug("validate decoder success")
					stream.SetValidStatus()
					stream.SeekTo(count)

					log.Trace("start forbidden later concurrent read and interrupt concurrent reading")
					stream.ForbiddenConcurrentRead()
					stream.InterruptReading()
					log.Trace("success")
					validateSuccess = true
				} else if vt == 3 {
					c1, err := d.LocateToValidStartInC2SStream(newXReader(conn.C2SStream()))
					if err != nil {
						log.WithField("decoderName", d.Name()).Warn("validate decoder failed")
						return
					}
					c2, err := d.LocateToValidStartInS2CStream(newXReader(conn.S2CStream()))
					if err != nil {
						log.WithField("decoderName", d.Name()).Warn("validate decoder failed")
						return
					}
					conn.C2SStream().SetValidStatus()
					conn.C2SStream().SeekTo(c1)
					conn.C2SStream().ForbiddenConcurrentRead()
					conn.C2SStream().InterruptReading()

					conn.S2CStream().SetValidStatus()
					conn.S2CStream().SeekTo(c2)
					conn.S2CStream().ForbiddenConcurrentRead()
					conn.S2CStream().InterruptReading()
					validateSuccess = true
				} else {
					log.Warn("Unsupported validate type")
					return
				}
				se.mutex.Lock()
				defer se.mutex.Unlock()
				if validateSuccess {
					se.decoderName = d.Name()
				}
			}()
		}
		//todo may wait forever
		wg.Wait()
		close(se.decoderReady)
	}()
	return se
}

func (s *ServerEndPoint) addStream(ca *net.TCPAddr, sa *net.TCPAddr, reader *StreamReader, st StreamType) {
	if sa.String() != s.serverAddr.String() {
		log.WithFields(log.Fields{"addr1": s.serverAddr, "addr2": sa}).Warn("can not add a tcp connection to server endpoint with different server address")
		return
	}
	c, ok := s.conns[ca.String()]
	if !ok {
		s.conns[ca.String()] = NewTCPConn(ca, sa, nil, nil)
		c = s.conns[ca.String()]
		s.startParseTcpConn(c)
	}

	if st == Client2ServerStream {
		c.SetC2SStream(reader)
	} else {
		c.SetS2CStream(reader)
	}
}

func (s *ServerEndPoint) startParseTcpConn(conn *TCPConn) {
	//wait until Reader ready
	go func() {
		<-s.decoderReady
		s.mutex.RLock()
		GetDecoder(s.decoderName).Parse(nil, conn, s.treeCtx)
		s.mutex.RUnlock()
	}()
}

func (sds *SingleDirectionStream) getInfo() (cAddr, sAddr *net.TCPAddr, st StreamType, err error) {
	netLayer, transLayer := sds.netLayer, sds.transportLayer
	srcAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Src().String(), transLayer.Src().String())
	if err != nil {
		log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
		return nil, nil, -1, err
	}

	dstAddr, err := common.ParseIpAndPort2TCPAddr(netLayer.Dst().String(), transLayer.Dst().String())
	if err != nil {
		log.WithField("reason", err.Error()).Warn("parser to tcp addr failed")
		return nil, nil, -1, err
	}
	clientAddr, serverAddr := distinguishStreamDrc(srcAddr, dstAddr, sds.streamFactory.actor.addr)
	if clientAddr == nil || serverAddr == nil {
		log.WithFields(log.Fields{"srcAddr": srcAddr.String(), "dstAddr": dstAddr.String(), "addrs": sds.streamFactory.actor.addr}).Warn("can't determine client server addr")
		return nil, nil, -1, err
	}
	if clientAddr == srcAddr {
		st = Client2ServerStream
	} else {
		st = Server2ClientStream
	}
	return clientAddr, serverAddr, st, nil
}

func distinguishStreamDrc(src, dst *net.TCPAddr, addrs []net.Addr) (clientAddr, serverAddr *net.TCPAddr) {
	for _, add := range addrs {
		if strings.Contains(add.String(), src.String()) {
			return dst, src
		}
		if strings.Contains(add.String(), dst.String()) {
			return src, dst
		}
	}
	return nil, nil
}

//can not come up with a proper name
type XReader struct {
	reader *ConcurrentReader
}

func (s *XReader) Read(p []byte) (n int, err error) {
	return s.reader.ConcurrentRead(p)
}

func newXReader(r *ConcurrentReader) *XReader {
	return &XReader{
		reader: r,
	}
}
