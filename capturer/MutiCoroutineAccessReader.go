package capturer

import (
	"data-collector/common"
	"errors"
	"fmt"
	"github.com/google/gopacket/tcpassembly"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
	"sync/atomic"
)

//copied from tcpreader.ReaderStream
type StreamReader struct {
	LossErrors   bool
	reassembled  chan []tcpassembly.Reassembly
	done         chan bool
	current      []tcpassembly.Reassembly
	closed       bool
	lossReported bool
	first        bool
	initiated    bool

	//added
	interrupt chan struct{}
}

func NewReaderStream() StreamReader {
	r := StreamReader{
		//todo 1024 is a temp plan
		reassembled: make(chan []tcpassembly.Reassembly, 1024),
		done:        make(chan bool),
		first:       true,
		initiated:   true,
		interrupt:   make(chan struct{}, 1),
	}
	return r
}

// Reassembled implements capturer.Stream's Reassembled function.
func (r *StreamReader) Reassembled(reassembly []tcpassembly.Reassembly) {
	if !r.initiated {
		panic("ReaderStream not created via NewReaderStream")
	}
	//for _, re := range reassembly {
	//	fmt.Errorf("receive bytes = %s \n", string(re.Bytes))
	//}
	fmt.Printf("send 2 chanel , channel size = %d \n", len(r.reassembled))
	r.reassembled <- reassembly
	fmt.Printf("send 2 chanel successd \n")
	// comment for blocking
	//<-r.done
}

// ReassemblyComplete implements capturer.Stream's ReassemblyComplete function.
func (r *StreamReader) ReassemblyComplete() {
	close(r.reassembled)
	close(r.done)
}

// stripEmpty strips empty reassembly slices off the front of its current set of
// slices.
func (r *StreamReader) stripEmpty() {
	for len(r.current) > 0 && len(r.current[0].Bytes) == 0 {
		r.current = r.current[1:]
		r.lossReported = false
	}
}

// DataLost is returned by the ReaderStream's Read function when it encounters
// a Reassembly with Skip != 0.
var DataLost = errors.New("lost data")

// Read implements io.Reader's Read function.
// Given a byte slice, it will either copy a non-zero number of bytes into
// that slice and return the number of bytes and a nil error, or it will
// leave slice p as is and return 0, io.EOF.
func (r *StreamReader) Read(p []byte) (int, error) {
	if !r.initiated {
		panic("ReaderStream not created via NewReaderStream")
	}
	var ok bool
	r.stripEmpty()
	for !r.closed && len(r.current) == 0 {
		if r.first {
			r.first = false
		} else {
			//r.done <- true
		}
		//if r.current, ok = <-r.reassembled; ok {
		//	r.stripEmpty()
		//} else {
		//	r.closed = true
		//}
		//change to following
		select {
		case r.current, ok = <-r.reassembled:
			if ok {
				r.stripEmpty()
			} else {
				r.closed = true
			}
		case <-r.interrupt:
			log.Debug("interrupted reading operation by interrupt chan")
		}
	}
	if len(r.current) > 0 {
		current := &r.current[0]
		if r.LossErrors && !r.lossReported && current.Skip != 0 {
			r.lossReported = true
			return 0, DataLost
		}
		length := copy(p, current.Bytes)
		current.Bytes = current.Bytes[length:]
		fmt.Printf("read string ------------ \n %s ------------\n", string(p[0:length]))
		return length, nil
	}
	return 0, io.EOF
}

func (r *StreamReader) InterruptBlockingReading() {
	//only one Gorouting will be interrupted
	r.interrupt <- struct{}{}
}

// Close implements io.Closer's Close function, making ReaderStream a
// io.ReadCloser.  It discards all remaining bytes in the reassembly in a
// manner that's safe for the assembler (IE: it doesn't block).
func (r *StreamReader) Close() error {
	r.current = nil
	r.closed = true
	for {
		if _, ok := <-r.reassembled; !ok {
			return nil
		}
		r.done <- true
	}
}

type ConcurrentReader struct {
	Reader          *StreamReader
	ReaderBroadcast chan struct{}
	//todo this splice may increase to very large, how to release it ?
	cache              []byte
	coroutingReadIndex map[uint64]int
	mutex              *sync.Mutex
	//as for the first bit, 0 means Reader is nil and 1 means Reader is not nil
	//as for the second bit, 0 means Concurrent Read is allowed, 1 means Concurrent Read is not allowed
	//as for the third bit, 0 means this stream is not validated, 1 means this stream has been validated (validate means whether user has discard invalid bytes by some way)
	//the following bits is un used for now
	status uint32
}

var StreamNotValidateErr = errors.New("make sure the stream beginning is valid and then call method SetValidStatus(true) on ConcurrentReader")
var ConcurrentReadNotAllowedErr = errors.New("Concurrent read is not allowed now on ConcurrentReader ")

func (m *ConcurrentReader) Read(p []byte) (n int, err error) {
	if !m.isValidate() {
		return 0, StreamNotValidateErr
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.cache) > 0 {
		n := copy(p, m.cache)
		if n < len(m.cache) {
			m.cache = m.cache[n:]
		} else {
			m.cache = m.cache[0:0]
		}
		return n, nil
	} else {
		m.waitReaderReady()
		return m.Reader.Read(p)
	}
}

func (m *ConcurrentReader) ConcurrentRead(p []byte) (n int, err error) {
	if !m.allowConcurrentRead() {
		return 0, ConcurrentReadNotAllowedErr
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	cId := common.GoID()
	i, exist := m.coroutingReadIndex[cId]
	if !exist {
		m.coroutingReadIndex[cId] = 0
		i = 0
	}

	m.waitReaderReady()

	if i < len(m.cache) {
		c := copy(p, m.cache[i:])
		m.coroutingReadIndex[cId] += c
		return c, nil
	}
	c, err := m.Reader.Read(p)
	if err != nil {
		if c > 0 {
			m.cache = append(m.cache, p[0:c]...)
		}
		return c, err
	}
	m.cache = append(m.cache, p[0:c]...)
	m.coroutingReadIndex[cId] += c
	return c, nil
}

func (m *ConcurrentReader) InterruptReading() {
	m.Reader.InterruptBlockingReading()
}

//seek start index to index
func (m *ConcurrentReader) SeekTo(index int) {
	m.cache = m.cache[index:]
}

const concurrentReadAllowedChecker = 0x20000000

func (m *ConcurrentReader) ForbiddenConcurrentRead() {
	for {
		oldVal := atomic.LoadUint32(&m.status)
		i := oldVal & concurrentReadAllowedChecker
		if i != 0 {
			return
		}

		newVal := oldVal | concurrentReadAllowedChecker
		if success := atomic.CompareAndSwapUint32(&m.status, oldVal, newVal); success {
			return
		}
	}
}

func (m *ConcurrentReader) allowConcurrentRead() bool {
	val := atomic.LoadUint32(&m.status)
	i := val & concurrentReadAllowedChecker
	if i == 0 {
		return true
	} else {
		return false
	}
}

const validateChecker = 0x40000000

func (m *ConcurrentReader) SetValidStatus() {
	for {
		oldVal := atomic.LoadUint32(&m.status)
		newVal := oldVal | validateChecker
		if newVal == oldVal {
			return
		}
		if success := atomic.CompareAndSwapUint32(&m.status, oldVal, newVal); success {
			return
		}
	}
}

func (m *ConcurrentReader) isValidate() bool {
	val := atomic.LoadUint32(&m.status)
	i := val & validateChecker
	if i == 0 {
		return false
	} else {
		return true
	}
}

const readerReadyChecker = 0x10000000

func (m *ConcurrentReader) waitReaderReady() {
	//for {
	//	val := atomic.LoadUint32(&m.status)
	//	if val&readerReadyChecker != 0 {
	//		return
	//	}
	//	time.Sleep(500 * time.Microsecond)
	//}
	<-m.ReaderBroadcast
}

func (m *ConcurrentReader) SetReaderReady() {
	for {
		oldVal := atomic.LoadUint32(&m.status)
		newVal := oldVal | readerReadyChecker
		if newVal == oldVal {
			return
		}
		if success := atomic.CompareAndSwapUint32(&m.status, oldVal, newVal); success {
			return
		}
	}
	close(m.ReaderBroadcast)
}

func NewMutiCoroutineAccessReader(reader *StreamReader) *ConcurrentReader {
	r := new(ConcurrentReader)
	r.Reader = reader
	r.ReaderBroadcast = make(chan struct{})
	r.cache = make([]byte, 0, 1024)
	r.coroutingReadIndex = make(map[uint64]int, 8)
	r.mutex = new(sync.Mutex)
	r.status = 0x0000
	return r
}
