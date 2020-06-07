package main

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	cwlogger "ssh-gateway/cw_logger"

	"golang.org/x/crypto/ssh"
)

type RelayChannel struct {
	StartTime       time.Time
	UserName        string
	SourceChannel   ssh.Channel
	fd              *os.File
	initialBuffer   *bytes.Buffer
	logMutex        *sync.Mutex
	enableRecording bool
	Logger          *cwlogger.Logger
	enableAwsLogs   bool
	SessionId       string
}

func (l *RelayChannel) SyncToFile(remote_name string, recordingDir string) error {
	var err error

	if !l.enableRecording {
		return nil
	}

	filepath := fmt.Sprintf("%s/%d/%d", recordingDir, l.StartTime.Year(), l.StartTime.Month())
	err = os.MkdirAll(filepath, 0750)
	if err != nil {
		return fmt.Errorf("Unable to create required log directory (%s): %s", filepath, err)
	}
	filename := filepath + "/" + fmt.Sprintf("ssh_log_%s_%s_%s", l.StartTime.Format(time.RFC3339), l.UserName, remote_name)

	l.logMutex.Lock()
	l.fd, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	_, err = l.initialBuffer.WriteTo(l.fd)
	if err != nil {
		return err
	}
	l.initialBuffer.Reset()
	l.initialBuffer = nil
	l.logMutex.Unlock()

	return nil
}

func (l *RelayChannel) Read(data []byte) (int, error) {
	return l.SourceChannel.Read(data)
}

func (l *RelayChannel) Write(data []byte) (int, error) {
	l.logMutex.Lock()
	if len(data) > 0 {
		if l.fd != nil && l.enableRecording {
			l.fd.Write(data)
		} else {
			l.initialBuffer.Write(data)
		}

		if l.enableAwsLogs && l.Logger != nil {
			l.Logger.Log(time.Now(), string(data))
		}
	}
	l.logMutex.Unlock()

	return l.SourceChannel.Write(data)

}

func (l *RelayChannel) Close() error {

	if l.enableAwsLogs {
		if l.Logger != nil {
			l.Logger.SessionFinished("Session terminated: "+l.SessionId, "Close")
			l.Logger.Close()
		}
	}

	if !l.enableRecording {
		return nil
	}

	if l.fd != nil {
		l.fd.Close()
	}

	return l.SourceChannel.Close()
}

func (l *RelayChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return l.SourceChannel.SendRequest(name, wantReply, payload)
}
