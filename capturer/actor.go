package capturer

import (
	"data-collector/common"
	"data-collector/msg"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	//"github.com/google/gopacket/tcpassembly"
	log "github.com/sirupsen/logrus"
	"net"
)

type captureManager struct {
	actorTreeCtx *common.ActorTreeContext
	captures     map[string][2]*common.ActorWrapper
	captureConfs map[string]*common.CaptureConf
}

type capture struct {
	name         string
	deviceName   string
	captureType  captureType
	addr         []net.Addr
	bpfStr       string
	status       common.CaptureStatus
	interStatus  captureInternalStatus
	oldStatus    common.CaptureStatus
	handle       *pcap.Handle
	actorTreeCtx *common.ActorTreeContext
}

type captureType int8

type captureInternalStatus int8

const (
	_ captureType = iota
	listenStreamCapture
	connectStreamCapture

	_ captureInternalStatus = iota
	Ready
	Working
	Paused
)

func (c captureType) String() string {
	if c == listenStreamCapture {
		return "listenStreamCapture"
	} else if c == connectStreamCapture {
		return "connectStreamCapture"
	}
	return "unexpected capture type"
}

func (c *captureManager) Receive(context actor.Context) {
	m := context.Message()
	log.WithField("type", fmt.Sprintf("%T", m)).Trace("captureManager receive a msg")
	switch message := m.(type) {
	case *msg.CaptureUpdateMsg:
		c.processCaptureConfUpdateMsg(context, message)
	case msg.CaptureUpdateMsg:
		c.processCaptureConfUpdateMsg(context, &message)
	case *msg.CaptureStatusSetMsg:
		c.processCaptureStatusSetMsg(context, message)
	case msg.CaptureStatusSetMsg:
		c.processCaptureStatusSetMsg(context, &message)
	default:
	}
}

func (c *captureManager) processCaptureConfUpdateMsg(ctx actor.Context, msg *msg.CaptureUpdateMsg) {
	log.WithFields(log.Fields{"device name": msg.DeviceName,
		"listen_ip_and_port":  msg.ListenAddrs,
		"connect_ip_and_port": msg.ConnAddrs}).Trace("start processCaptureConfUpdateMsg")
	device := msg.DeviceName
	if len(device) == 0 {
		log.Warn("empty device name")
		return
	}
	c.captureConfs[device] = &msg.CaptureConf

	captures, exist := c.captures[device]
	if exist {
		listenCapture, connCapture := captures[0], captures[1]
		if listenCapture == nil || connCapture == nil {
			log.WithFields(log.Fields{"deviceName": device}).Warn("why?")
			return
		}
		ctx.Send(listenCapture.Pid, AddrUpdateMsg(msg.ListenAddrs))
		ctx.Send(connCapture.Pid, AddrUpdateMsg(msg.ListenAddrs))
	} else {
		lcn := generateCaptureName(device, listenStreamCapture)
		lc, err := ctx.SpawnNamed(actor.PropsFromFunc(newCapture(lcn, device, msg.ListenAddrs, listenStreamCapture, c.actorTreeCtx).Receive), lcn)
		if err != nil {
			log.WithFields(log.Fields{}).Warn("creat listen stream capture failed")
		}
		ccn := generateCaptureName(device, connectStreamCapture)
		cc, err := ctx.SpawnNamed(actor.PropsFromFunc(newCapture(ccn, device, msg.ConnAddrs, connectStreamCapture, c.actorTreeCtx).Receive), ccn)
		if err != nil {
			log.WithField("err msg", err.Error()).Warn("creat connect stream capture failed")
			return
		}
		a1 := &common.ActorWrapper{lcn, lc, make(map[string]interface{})}
		a2 := &common.ActorWrapper{ccn, cc, make(map[string]interface{})}
		c.captures[device] = [...]*common.ActorWrapper{a1, a2}
	}
}

func (c *captureManager) processCaptureStatusSetMsg(ctx actor.Context, msg *msg.CaptureStatusSetMsg) {
	log.WithFields(log.Fields{"setAll": msg.SetAll, "newStatus": msg.NewStatus, "names": msg.CaptureIds}).Trace("start processCaptureStatusSetMsg")
	if msg.SetAll {
		for _, cs := range c.captures {
			if cs[0] != nil {
				ctx.Send(cs[0].Pid, msg)
			}
			if cs[1] != nil {
				ctx.Send(cs[1].Pid, msg)
			}
		}
		return
	}
	ids := msg.CaptureIds
	for _, id := range ids {
		a := c.findCaptureByName(id)
		if a == nil {
			log.WithField("name", id).Warn("can't find specify capture")
			continue
		}
		ctx.Send(a.Pid, msg)
	}
}
func (c *captureManager) findCaptureByName(name string) *common.ActorWrapper {
	for _, cs := range c.captures {
		if cs[0].ActorName == name {
			return cs[0]
		}
		if cs[1].ActorName == name {
			return cs[1]
		}
	}
	return nil
}

func (c *capture) Receive(context actor.Context) {
	m := context.Message()
	log.WithField("type", fmt.Sprintf("%T", m)).Trace("capture receive a msg")
	switch message := m.(type) {
	case msg.CaptureStatusSetMsg:
		c.processCaptureStatusSetMsg(context, &message)
	case *msg.CaptureStatusSetMsg:
		c.processCaptureStatusSetMsg(context, message)
	case AddrUpdateMsg:
		c.processAddrUpdateMsg(context, message)
	case *AddrUpdateMsg:
		c.processAddrUpdateMsg(context, *message)
	}
}

func (c *capture) processCaptureStatusSetMsg(context actor.Context, msg *msg.CaptureStatusSetMsg) {
	c.oldStatus, c.status = c.status, msg.NewStatus
	switch c.status {
	//case common.NewCreated:
	//	//why? do nothing
	//case common.Ready:
	//	//why? do nothing
	//case common.Staring:
	//	//why? do nothing
	case common.Running:
		//
		err := c.ensureRunning()
		if err != nil {
			log.WithField("errMsg", err.Error()).Warn("start capture failed")
		}
	//case common.Stoping:
	//	//why? do nothing
	case common.Stoppted:
		c.ensureStopped()
	default:
		log.WithField("status", c.status.String()).Warn("unSupported status set action")
	}
}

func (c *capture) ensureRunning() error {
	if len(c.bpfStr) == 0 {
		return nil
	}
	if c.handle == nil {
		handle, err := pcap.OpenLive(c.deviceName, 65535, false, pcap.BlockForever)
		if err != nil {
			return err
		}
		err = handle.SetBPFFilter(c.bpfStr)
		if err != nil {
			return err
		}
		c.handle = handle
		src := gopacket.NewPacketSource(handle, handle.LinkType())
		packets := src.Packets()

		go func() {
			sf := newDSF(c.actorTreeCtx, c)
			assembler := NewAssembler(NewStreamPool(sf))
			for {
				select {
				case packet := <-packets:
					//todo 这一块是不是后面要通过锁的机制来保证可见性？应该是需要的，后面测试一下看下情况
					if c.interStatus == Paused {
						// 通过查看状态的方式来启动或者停止抓包。这里的停止抓包
						// 依然会轮训 packets ，只是在有包的时候忽略这个包。这样设计的主要考虑是
						// 避免底层往packets塞数据被阻塞，packets的默认大小是1000。如果后面有需求不能这么搞
						// 那么在这里直接跳出这个循环就好了。跳出这个循环不能break，这样只能跳出select，需要给
						// 外层for循环一个标记
						// 另外这里对status这个变量的读取应该会有可见性问题，严格来说需要加锁，
						// 但是太损耗性能了，而且暂停抓包这种场景应该用的也不多。而且即便用，出现了
						// 可见行问题，也不会对系统造成不可预估的影响，故而这里就不加锁了
						continue
					}
					if packet.NetworkLayer() == nil ||
						packet.TransportLayer() == nil ||
						packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
						continue
					}
					tcp := packet.TransportLayer().(*layers.TCP)
					if len(tcp.Payload) > 10 {
						fmt.Printf("capture a package %s, payload=%d, content=%d\n", packet.TransportLayer().TransportFlow().String(), len(tcp.LayerPayload()), len(tcp.LayerContents()))
					}
					//fmt.Println(len(tcp.LayerPayload()))
					//fmt.Println(len(tcp.LayerContents()))
					assembler.AssembleWithTimestamp(
						packet.NetworkLayer().NetworkFlow(),
						tcp, packet.Metadata().Timestamp,
					)
					//case <-ticker:
					//	assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
				}
			}
		}()
		c.interStatus = Working
	}
	if c.interStatus != Working {
		c.interStatus = Working
	}
	return nil
}

func (c *capture) ensureStopped() {
	//todo
}

func (c *capture) processAddrUpdateMsg(context actor.Context, msg AddrUpdateMsg) {
	addrs := []net.Addr(msg)
	c.addr = addrs
	newBpf := common.GenerateBPFStr(addrs, c.captureType == listenStreamCapture)
	//todo to be verified later
	c.handle.SetBPFFilter(newBpf)
}

func generateCaptureName(deviceName string, catogry captureType) string {
	return fmt.Sprintf("{%s}-{%s}", deviceName, catogry.String())
}

func newCapture(name, deviceName string, addr []net.Addr, catogry captureType, actorTreeCtx *common.ActorTreeContext) *capture {
	//todo
	isListen := true
	if catogry == connectStreamCapture {
		isListen = false
	}
	return &capture{
		name:         name,
		deviceName:   deviceName,
		addr:         addr,
		captureType:  catogry,
		bpfStr:       common.GenerateBPFStr(addr, isListen),
		status:       common.NewCreated,
		oldStatus:    0,
		interStatus:  Ready,
		actorTreeCtx: actorTreeCtx,
	}
}

func NewCapatureManager(ctx *common.ActorTreeContext) *captureManager {
	cm := new(captureManager)
	cm.actorTreeCtx = ctx
	cm.captures = make(map[string][2]*common.ActorWrapper)
	cm.captureConfs = make(map[string]*common.CaptureConf)
	return cm
}
