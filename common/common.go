package common

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"io"
	"net"
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

type TCPConn struct {
	clientAddr          *net.TCPAddr
	serverAddr          *net.TCPAddr
	client2ServerStream io.Reader
	server2ClientStream io.Reader
}

func (conn *TCPConn) GetClientAddr() *net.TCPAddr {
	return conn.clientAddr
}

func (conn *TCPConn) GetServerAddr() *net.TCPAddr {
	return conn.serverAddr
}

func (conn *TCPConn) C2SStream() io.Reader {
	return conn.client2ServerStream
}

func (conn *TCPConn) S2CStream() io.Reader {
	return conn.server2ClientStream
}

func (conn *TCPConn) CheckC2SStreamExist() bool {
	return conn.client2ServerStream != nil
}

func (conn *TCPConn) CheckS2CStreamExist() bool {
	return conn.server2ClientStream != nil
}

func (conn *TCPConn) SetC2SStream(stream io.Reader) {
	conn.client2ServerStream = stream
}

func (conn *TCPConn) SetS2CStream(stream io.Reader) {
	conn.server2ClientStream = stream
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

func NewTCPConn(clientAddr, serverAddr *net.TCPAddr, c2sStream, s2cStream io.Reader) *TCPConn {
	return &TCPConn{clientAddr: clientAddr,
		serverAddr:          serverAddr,
		client2ServerStream: c2sStream,
		server2ClientStream: s2cStream}
}
