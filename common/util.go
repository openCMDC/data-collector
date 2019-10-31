package common

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func GenerateBPFStr(addrs []net.Addr, isListenAddr bool) string {
	//todo
	return generateBPFV2(addrs)
}

func generateBPFV2(addrs []net.Addr) string {
	inBpf := make([]string, 0)
	outBpf := make([]string, 0)

	inbpf := "(dst host %s and dst port %s)"
	outbpf := "(src host %s and src port %s)"

	for _, a := range addrs {
		p := a.String()
		i := strings.LastIndex(p, ":")
		inBpf = append(inBpf, fmt.Sprintf(inbpf, p[0:i], p[i+1:]))
		outBpf = append(outBpf, fmt.Sprintf(outbpf, p[0:i], p[i+1:]))
	}

	inRes := strings.Join(inBpf, " or ")
	outRes := strings.Join(outBpf, " or ")
	if len(inRes) == 0 {
		return outRes
	}

	if len(outRes) == 0 {
		return inRes
	}
	return inRes + " or " + outRes
}

func generateBPF(ipPort map[string]bool) string {
	inBpf := make([]string, 0)
	outBpf := make([]string, 0)

	inbpf := "(dst host %s and dst port %s)"
	outbpf := "(src host %s and src port %s)"

	for p, _ := range ipPort {
		i := strings.LastIndex(p, ":")
		inBpf = append(inBpf, fmt.Sprintf(inbpf, p[0:i], p[i+1:]))
		outBpf = append(outBpf, fmt.Sprintf(outbpf, p[0:i], p[i+1:]))
	}

	inRes := strings.Join(inBpf, " or ")
	outRes := strings.Join(outBpf, " or ")
	if len(inRes) == 0 {
		return outRes
	}

	if len(outRes) == 0 {
		return inRes
	}
	return inRes + " or " + outRes
}

func ParseIpAndPort2TCPAddr(ip string, port string) (*net.TCPAddr, error) {
	ip1 := net.ParseIP(ip)
	port1, err := strconv.Atoi(port)
	if ip1 == nil || err != nil {
		return nil, fmt.Errorf("parser %s to ip failed or parser %s to port failed", ip, port)
	}
	return &net.TCPAddr{IP: ip1, Port: port1}, nil
}
