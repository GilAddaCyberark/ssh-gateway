package recorders

import (
	"bytes"
	cwlogger "ssh-gateway/cw_logger"
	gen "ssh-gateway/ssh-engine/generic-structs"
)

type CWLRecorder struct {
	Logger    *cwlogger.Logger
	logBuffer *bytes.Buffer
}

func NewCWLRecorder(targetInfo *gen.TargetInfo) (*CWLRecorder, error) {
	cwl := CWLRecorder{}
	logger, err := cwlogger.NewLoggerByTargetInfo(targetInfo)

	if err != nil {
		return nil, err
	}
	cwl.Logger = logger
	cwl.Logger.SessionStarted("Session started", "NewCWLRecorder")

	return &cwl, nil
}

func (c *CWLRecorder) Init() error {
	return nil
}
func (c *CWLRecorder) Close() error {
	c.Logger.SessionFinished("Session finished", "ProxySession")
	return nil
}

func (c *CWLRecorder) Write(data []byte) error {
	// if c.logBuffer == nil {
	// 	c.logBuffer = bytes.NewBuffer(data)
	// } else {
	// 	c.logBuffer.Write(data)
	// }
	// s := c.logBuffer.String()
	// fmt.Printf(string(data))
	c.Logger.LogInfo(string(data), "CWLRecorder.Write")

	// c.logBuffer.Truncate(0)

	return nil

}
