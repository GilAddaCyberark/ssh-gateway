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
	buf := make([]byte, 0, 0)
	cwl.logBuffer = bytes.NewBuffer(buf)
	cwl.Logger.SessionStarted("Session Status", "NewCWLRecorder")

	return &cwl, nil
}

func (c *CWLRecorder) Init() error {
	return nil
}
func (c *CWLRecorder) Close() error {
	c.Logger.SessionFinished("Session Status", "ProxySession")
	return nil
}
func (c *CWLRecorder) Write(data []byte, isClientInput bool) error {
	c.logBuffer.Write(data)
	// Flush buffer in case of white spaces found in the end
	// todo : handle empry data
	if data != nil && len(data) > 0 {
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
