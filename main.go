package main

import (
	"data-collector/capturer"
	"data-collector/common"
	_ "data-collector/decoder"
	"data-collector/msg"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"time"
)

func init() {
	// 设置日志格式为json格式
	log.SetFormatter(&log.JSONFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr，标准错误）
	// 日志消息输出可以是任意的io.writer类型
	log.SetOutput(os.Stdout)

	// 设置日志级别为warn以上
	log.SetLevel(log.WarnLevel)
}

func main() {
	ctx := actor.EmptyRootContext
	actorTreeCtx := new(common.ActorTreeContext)
	actorTreeCtx.ActorContext = ctx
	cm, err := ctx.SpawnNamed(actor.PropsFromFunc(capturer.NewCapatureManager(actorTreeCtx).Receive), "CapatureManager")
	if err != nil {
		log.Warnf("instantiate CapatureManager failed of %s", err.Error())
		return
	}
	actorTreeCtx.CaptureManager = cm
	//em, err := ctx.SpawnNamed(actor.PropsFromFunc(conns.NewEndPointManager(actorTreeCtx).Receive), "EndPointManager")
	//if err != nil {
	//	log.Warnf("instantiate CapatureManager failed of %s", err.Error())
	//	return
	//}
	//actorTreeCtx.EndpointsManager = em

	addr, err := common.ParseIpAndPort2TCPAddr("120.55.51.202", "8768")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	c := common.CaptureConf{DeviceName: `\Device\NPF_{7D0089A9-CD14-4C31-926D-B5D61B002A60}`, ListenAddrs: []net.Addr{}, ConnAddrs: []net.Addr{addr}}
	msg1 := msg.CaptureUpdateMsg{c}
	ctx.Send(cm, msg1)
	time.Sleep(3 * time.Second)
	msg2 := msg.CaptureStatusSetMsg{NewStatus: common.Running, SetAll: true}
	ctx.Send(cm, msg2)
	time.Sleep(3 * time.Hour)
}
