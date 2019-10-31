package decoder

import (
	"testing"
)

func TestHttpReqReg(t *testing.T) {
	str := `GET /MFEwTzBNMEswSTAJBgUrDgMCGgUABBTrJdiQ%2Ficg9B19asFe73bPYs%2BreAQUdXGnGUgZvJ2d6kFH35TESHeZ03kCEFslzmkHxCZVZtM5DJmpVK0%3D HTTP/1.1`
	res := checkIsHttpRequestStart(str)
	if res != true {
		t.Errorf("unexpected")
	}
}

func TestHttpRepReg(t *testing.T) {
	str := `HTTP/1.1 200 OK`
	res := checkIsHttpResponceStart(str)
	if res != true {
		t.Errorf("unexpected")
	}
}
