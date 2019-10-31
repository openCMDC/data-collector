package common

import "net"

type CaptureConf struct {
	DeviceName  string
	ListenAddrs []net.Addr
	ConnAddrs   []net.Addr
}
