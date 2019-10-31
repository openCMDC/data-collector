package decoder

import (
	"errors"
	"fmt"
	"io"
)

type StreamWrapper struct {
	internalReader   io.Reader
	validStartBytes  []byte
	ready            bool
	streamStartJuger func(reader io.Reader) ([]byte, error)
}

var InternalReaderNil = errors.New("InternalReaderNil")

func (sw *StreamWrapper) Read(p []byte) (n int, err error) {
	//todo
	if sw.internalReader == nil {
		return 0, InternalReaderNil
	}
	if !sw.ready {
		if sw.streamStartJuger == nil {
			return 0, fmt.Errorf("StreamWrapper streamStartJuger func is nil")
		}
		validBytes, err := sw.streamStartJuger(sw.internalReader)
		if err != nil {
			return 0, err
		}
		sw.validStartBytes = validBytes
		sw.ready = true
	}
	if len(sw.validStartBytes) > 0 {
		n := copy(p, sw.validStartBytes)
		if n < len(sw.validStartBytes) {
			sw.validStartBytes = sw.validStartBytes[n:]
		}
		return n, nil
	} else {
		return sw.internalReader.Read(p)
	}
}

func newStreamWarpper(reader io.Reader, validFunc func(reader io.Reader) ([]byte, error)) *StreamWrapper {
	res := new(StreamWrapper)
	res.internalReader = reader
	res.streamStartJuger = validFunc
	return res
}
