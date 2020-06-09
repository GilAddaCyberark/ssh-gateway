package recorders

import (
	"bytes"
	aws_helpers "ssh-gateway/aws_workers"
	cwlogger "ssh-gateway/cw_logger"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type CWLRecorder struct {
	Logger    *cwlogger.Logger
	logBuffer *bytes.Buffer
}

func NewCWLRecorder() (*CWLRecorder, error) {
	cwl := CWLRecorder{}
	// cwl.Logger = &cwlogger.Logger{}
	logger, err := cwlogger.New(&cwlogger.Config{
		LogGroupName: "SSH_Gateway_Logs",
		Client:       cloudwatchlogs.New(session.New(), &aws.Config{Region: aws.String(aws_helpers.DEFAULT_REGION)}),
	})
	if err != nil {
		return nil, err
	}
	cwl.Logger = logger
	buf := make([]byte, 0, 0)
	cwl.logBuffer = bytes.NewBuffer(buf)

	return &cwl, nil
}
func (c *CWLRecorder) Init() error {
	return nil
}
func (c *CWLRecorder) Close() error {
	return nil

}
func (c *CWLRecorder) Write(data []byte, isClientInput bool) error {
	c.logBuffer.Write(data)
	// Flush buffer in case of white spaces found in the end
	if bytes.ContainsAny(data[len(data)-1:], "\r\n ") {
		c.Logger.Log(time.Now(), c.logBuffer.String())
		// fmt.Printf("\n<-> from rec: \n%s", c.logBuffer.String())
		c.logBuffer.Truncate(0)
	}

	return nil

}
