package conns

import "io"

type StreamUpdateMsg struct {
	newStream io.Reader
}
