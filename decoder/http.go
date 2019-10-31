package decoder

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
)

var httpRequestFirstLineReg *regexp.Regexp
var httpResponcetFirstLineReg *regexp.Regexp

func init() {
	httpRequestFirstLineReg, _ = regexp.Compile(`((GET)|(HEAD)|(POST)|(PUT)|(DELETE)|(PATCH))\s.*\sHTTP\/\d\.\d`)
	httpResponcetFirstLineReg, _ = regexp.Compile(`HTTP\/\d\.\d\s(\d{0,3})\s\w+`)
}

type httpDecoder struct {
}

func (h *httpDecoder) Parse(ctx *ParseCtx) {
	ctx.Conn.C2SStream()
	r1 := ctx.Conn.C2SStream()
	r2 := ctx.Conn.S2CStream()

	r3 := newStreamWarpper(r1, getValidStartByteFromRequest)
	r4 := newStreamWarpper(r2, getValidStartByteFromResponce)

	r5 := bufio.NewReader(r3)
	r6 := bufio.NewReader(r4)

	reqChan := make(chan *http.Request, 1)
	shutChan := make(chan interface{}, 1)
	repChan := make(chan interface{}, 1)

	go func() {
		for {
			req, err := http.ReadRequest(r5)
			if err != nil {
				if err == InternalReaderNil {
					continue
				} else {
					return
					log.WithField("conn", ctx.Conn).Warnf("parse http request err %s\n", err.Error())
					shutChan <- true
				}
			}
			reqChan <- req
		}
	}()

	go func() {
		select {
		case <-shutChan:
			close(repChan)
			return
		case req := <-reqChan:
			for {
				rep, err := http.ReadResponse(r6, req)
				if err != nil {
					if err == InternalReaderNil {
						continue
					} else {
						log.WithField("conn", ctx.Conn).Warnf("parse http request err %s\n", err.Error())
						close(repChan)
						return
					}
				}
				repChan <- rep
				break
			}
		}
	}()

	for r := range repChan {
		ctx.SendBackChan <- r
	}
}

func getValidStartByteFromRequest(reader io.Reader) ([]byte, error) {
	br := bufio.NewReader(reader)
	for {
		line, prefix, err := br.ReadLine()
		if err != nil {
			return nil, err
		}
		if prefix {
			continue
		}
		l := string(line)
		if !checkIsHttpRequestStart(l) {
			continue
		}
		return line, err
	}
}

func getValidStartByteFromResponce(reader io.Reader) ([]byte, error) {
	br := bufio.NewReader(reader)
	for {
		line, prefix, err := br.ReadLine()
		if err != nil {
			return nil, err
		}
		if prefix {
			continue
		}
		l := string(line)
		if !checkIsHttpResponceStart(l) {
			continue
		}
		return line, err
	}
}

func checkIsHttpRequestStart(line string) bool {
	return httpRequestFirstLineReg.MatchString(line)
}

func checkIsHttpResponceStart(line string) bool {
	return httpResponcetFirstLineReg.MatchString(line)
}
