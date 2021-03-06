package recorders

import (
	"bytes"
	cwlogger "ssh-gateway/internal/pkg/cw_logger"
	gen "ssh-gateway/internal/pkg/ssh-engine/generic-structs"
)

type CWLRecorder struct {
	Recorder
	targetInfo *gen.TargetInfo
	Logger     *cwlogger.Logger
	logBuffer  *bytes.Buffer
}

func NewCWLRecorder(targetInfo *gen.TargetInfo) *CWLRecorder {
	cwl := CWLRecorder{targetInfo: targetInfo}
	return &cwl
}

func (c *CWLRecorder) Init() error {
	logger, err := cwlogger.NewLoggerByTargetInfo(c.targetInfo)

	if err != nil {
		return err
	}
	c.Logger = logger
	buf := make([]byte, 0)
	c.logBuffer = bytes.NewBuffer(buf)
	c.Logger.SessionStarted("Session Status", "NewCWLRecorder")
	return nil
}

func (c *CWLRecorder) Close() error {
	c.Logger.SessionFinished("Session Status", "ProxySession")
	c.Logger.Close()
	return nil
}
func (c *CWLRecorder) Write(data []byte, isClientInput bool) error {
	c.logBuffer.Write(data)
	// Flush buffer in case of white spaces found in the end
	// todo : handle empry data
	if len(data) > 0 {
		if bytes.ContainsAny(data[len(data)-1:], "\r\n ") {
			c.Logger.LogInfo(c.logBuffer.String(), "CWLRecorder.Write") // Dima - is it needed?
			// fmt.Printf("\n<-> from rec: \n%s", c.logBuffer.String())
			c.logBuffer.Truncate(0)
		}
	} else {
		if c.logBuffer.Len() > 0 {
			c.Logger.LogInfo(c.logBuffer.String(), "CWLRecorder.Write") // Dima - is it needed?
			c.logBuffer.Truncate(0)
		}
	}

	return nil

}
