package recorders

import (
	"bytes"
	cwlogger "ssh-gateway/cw_logger"
	gen "ssh-gateway/ssh-engine/generic-structs"
	"time"
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
func (c *CWLRecorder) Write(data []byte, isClientInput bool) error {
	c.logBuffer.Write(data)
	// Flush buffer in case of white spaces found in the end
	if bytes.ContainsAny(data[len(data)-1:], "\r\n ") {
		c.Logger.Log(time.Now(), c.logBuffer.String())
		// fmt.Printf("\n<-> from rec: \n%s", c.logBuffer.String())
		c.logBuffer.Truncate(0)
		// c.Logger.LogInfo(string(data), "CWLRecorder.Write") // Dima - is it needed?
	}

	return nil

}
