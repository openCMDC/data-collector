package capturer

import "testing"

type a struct {
	mm map[string]int
}

func Test1(t *testing.T) {
	b := new(a)
	b.mm["1"] = 1
}
