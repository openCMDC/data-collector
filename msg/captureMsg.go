package msg

import (
	"data-collector/common"
	"io"
	"net"
)

type CaptureUpdateMsg struct {
	common.CaptureConf
}

type CaptureStatusSetMsg struct {
	CaptureIds []string
	NewStatus  common.CaptureStatus
	SetAll     bool
}

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

type StreamCreatedMsg struct {
	Client *net.TCPAddr
	Server *net.TCPAddr
	Stream io.Reader
	Type   StreamType
	Device string
}
