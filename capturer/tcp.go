package capturer

import "net"
import log "github.com/sirupsen/logrus"

type TCPConn struct {
	clientAddr          *net.TCPAddr
	serverAddr          *net.TCPAddr
	client2ServerStream *ConcurrentReader
	server2ClientStream *ConcurrentReader
	started             bool
}

func (conn *TCPConn) GetClientAddr() *net.TCPAddr {
	return conn.clientAddr
}

func (conn *TCPConn) GetServerAddr() *net.TCPAddr {
	return conn.serverAddr
}

func (conn *TCPConn) C2SStream() *ConcurrentReader {
	return conn.client2ServerStream
}

func (conn *TCPConn) S2CStream() *ConcurrentReader {
	return conn.server2ClientStream
}

func (conn *TCPConn) CheckC2SStreamExist() bool {
	return conn.client2ServerStream != nil
}

func (conn *TCPConn) CheckS2CStreamExist() bool {
	return conn.server2ClientStream != nil
}

func (conn *TCPConn) SetC2SStream(stream *StreamReader) {
	if conn.client2ServerStream.Reader != nil {
		log.Warn("replace client to server stream which already existed is not allowed")
		return
	}
	conn.client2ServerStream.Reader = stream
	conn.client2ServerStream.SetReaderReady()

}

func (conn *TCPConn) SetS2CStream(stream *StreamReader) {
	if conn.server2ClientStream.Reader != nil {
		log.Warn("replace server to clietn stream which already existed is not allowed")
		return
	}
	conn.server2ClientStream.Reader = stream
	conn.server2ClientStream.SetReaderReady()
}

func NewTCPConn(clientAddr, serverAddr *net.TCPAddr, c2sStream, s2cStream *StreamReader) *TCPConn {
	return &TCPConn{clientAddr: clientAddr,
		serverAddr:          serverAddr,
		client2ServerStream: NewMutiCoroutineAccessReader(c2sStream),
		server2ClientStream: NewMutiCoroutineAccessReader(s2cStream),
		started:             false}
}
