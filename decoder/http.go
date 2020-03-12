package decoder

import (
	"bufio"
	"context"
	"data-collector/capturer"
	"data-collector/common"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
)

var httpRequestFirstLineReg *regexp.Regexp
var httpResponceFirstLineReg *regexp.Regexp

func init() {
	httpRequestFirstLineReg, _ = regexp.Compile(`((GET)|(HEAD)|(POST)|(PUT)|(DELETE)|(PATCH))\s.*\sHTTP\/\d\.\d`)
	httpResponceFirstLineReg, _ = regexp.Compile(`HTTP\/\d\.\d\s(\d{0,3})\s\w+`)
	capturer.RegisterDecoder("http", NewHttpDecoder)
}

type httpDecoder struct {
}

func (h *httpDecoder) Parse(ctx context.Context, conn *capturer.TCPConn, actorCtx *common.ActorTreeContext) {

	r5 := bufio.NewReader(conn.C2SStream())
	r6 := bufio.NewReader(conn.S2CStream())
	shutChan := make(chan interface{}, 1)
	repChan := make(chan interface{}, 1)
	go func() {

		for range repChan {
			fmt.Println("--------------")
		}
	}()
	//reqChan := make(chan *http.Request, 1)

	for {
		req, err := http.ReadRequest(r5)
		if err != nil {
			if err == InternalReaderNil {
				continue
			} else {
				log.WithFields(log.Fields{"conn": conn, "errMsg": err.Error()}).Warn("parse http request err")
				shutChan <- true
				return
			}
		}
		fmt.Println("success parse req")

		rep, err := http.ReadResponse(r6, nil)
		if err != nil {
			if err == InternalReaderNil {
				continue
			} else {
				log.WithFields(log.Fields{"conn": conn, "errMsg": err.Error()}).Warn("parse http request err")
				shutChan <- true
				return
			}
		}
		fmt.Println("success parse rep")

		rep.Request = req
		repChan <- rep
	}

}

func (h *httpDecoder) Name() string {
	return "http"
}

func (h *httpDecoder) LocateToValidStartInC2SStream(reader io.Reader) (int, error) {
	var totalCount int = 0
	br := bufio.NewReader(reader)
	for {
		line, err := br.ReadSlice('\r')
		if err != nil {
			return -1, err
		}
		if line[len(line)-1] != '\r' {
			totalCount += len(line) + 1
			continue
		}
		l := string(line[:len(line)-1])
		if !checkIsHttpRequestStart(l) {
			totalCount += len(line) + 2
			continue
		}
		return totalCount, nil
	}
}

func (h *httpDecoder) LocateToValidStartInS2CStream(reader io.Reader) (int, error) {
	var totalCount int = 0
	br := bufio.NewReader(reader)
	for {
		line, err := br.ReadSlice('\n')
		if err != nil {
			return -1, err
		}
		if line[len(line)-1] != '\r' {
			totalCount += len(line) + 1
			continue
		}
		l := string(line[:len(line)-1])
		if !checkIsHttpResponceStart(l) {
			totalCount += len(line) + 2
			continue
		}
		return totalCount, nil
	}
}

func (h *httpDecoder) ValidateType() byte {
	return 1
}

func checkIsHttpRequestStart(line string) bool {
	return httpRequestFirstLineReg.MatchString(line)
}

func checkIsHttpResponceStart(line string) bool {
	return httpResponceFirstLineReg.MatchString(line)
}

func NewHttpDecoder() capturer.Interface {
	return new(httpDecoder)
}
