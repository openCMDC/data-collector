package common

import (
	"bytes"
	"github.com/AsynkronIT/protoactor-go/actor"

	"runtime"
	"strconv"
)

type ActorWrapper struct {
	ActorName string
	Pid       *actor.PID
	Meta      map[string]interface{}
}

type ActorTreeContext struct {
	ActorContext     *actor.RootContext
	CaptureManager   *actor.PID
	EndpointsManager *actor.PID
}

func GoID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func (ctx *ActorTreeContext) GetActorContext() *actor.RootContext {
	return ctx.ActorContext
}

func (ctx *ActorTreeContext) GetCaptureManager() *actor.PID {
	return ctx.CaptureManager
}

func (ctx *ActorTreeContext) GetDecoderManager() *actor.PID {
	return ctx.EndpointsManager
}
