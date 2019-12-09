package msg

import (
	"data-collector/common"
)

type CaptureUpdateMsg struct {
	common.CaptureConf
}

type CaptureStatusSetMsg struct {
	CaptureIds []string
	NewStatus  common.CaptureStatus
	SetAll     bool
}
