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
	return &cwl, nil
}
func (c *CWLRecorder) Init() error {
	return nil
}
func (c *CWLRecorder) Close() error {
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
	c.Logger.Log(time.Now(), string(data))
	// c.logBuffer.Truncate(0)

	return nil

}
