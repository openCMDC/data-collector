package common

import "fmt"

type CaptureStatus int8

const (
	_ CaptureStatus = iota
	NewCreated
	Ready
	Staring
	Running
	Stoping
	Stoppted
)

func (c CaptureStatus) String() string {
	if c == 1 {
		return "NewCreated"
	} else if c == 2 {
		return "Ready"
	} else if c == 3 {
		return "Staring"
	} else if c == 4 {
		return "Running"
	} else if c == 5 {
		return "Stoping"
	} else if c == 6 {
		return "Stoppted"
	} else {
		return fmt.Sprintf("unknown status with id = %d", int8(c))
	}
}
