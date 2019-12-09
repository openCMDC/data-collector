package common

import (
	"fmt"
	"testing"
)

func TestGetRoutingID(t *testing.T) {
	go func() {
		id := GoID()
		fmt.Println(id)
	}()
	id := GoID()
	fmt.Println(id)
}
