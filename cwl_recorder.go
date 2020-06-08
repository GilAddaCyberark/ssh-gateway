package main

import (
	"bytes"
	cwlogger "ssh-gateway/cw_logger"
	"time"
)

type CWLRecorder struct {
	Logger    *cwlogger.Logger
	logBuffer *bytes.Buffer
}

func NewCWLRecorder() (*CWLRecorder, error) {
	cwl := CWLRecorder{}
	cwl.Logger = &cwlogger.Logger{}

	return &cwl, nil
}
func (c *CWLRecorder) Init() error {
	return nil
}
func (c *CWLRecorder) Close() error {
	return nil

}
func (c *CWLRecorder) Write(data []byte) error {
	if c.logBuffer == nil {
		c.logBuffer = bytes.NewBuffer(data)
	} else {
		c.logBuffer.Write(data)
	}
	//c.Logger.Log(time.Now(), strings.data)

	spliter := []byte("]0;") // Commnad splitter instead of cariage return
	for index := (bytes.Index(c.logBuffer.Bytes(), spliter)); index > 0; index = (bytes.Index(c.logBuffer.Bytes(), spliter)) {
		c.Logger.Log(time.Now(), c.logBuffer.String())
		c.logBuffer.Truncate(0)
	}

	return nil

}
