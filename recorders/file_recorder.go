package recorders

import (
	"bytes"
	"fmt"
	"os"
	gen "ssh-gateway/ssh-engine/generic-structs"
	"sync"
	"time"
)

type FileRecorder struct {
	targetInfo    gen.TargetInfo
	recordingDir  string
	fd            *os.File
	initialBuffer *bytes.Buffer
	fileMutex     *sync.Mutex
}

func NewFileRecorder(targetInfo gen.TargetInfo, recordingDir string) *FileRecorder {
	fr := FileRecorder{}
	fr.recordingDir = recordingDir
	fr.targetInfo = targetInfo
	fr.fileMutex = &sync.Mutex{}

	return &fr
}
func (fr *FileRecorder) Init() error {
	var err error
	var startTime time.Time

	startTime = time.Now()
	filepath := fmt.Sprintf("%s/%d/%d", fr.recordingDir, startTime.Year(), startTime.Month())
	err = os.MkdirAll(filepath, 0750)
	if err != nil {
		return fmt.Errorf("Unable to create required log directory (%s): %s", filepath, err)
	}
	filename := filepath + "/" + fmt.Sprintf("ssh_log_%s_%s_%s", startTime.Format(time.RFC3339),
		fr.targetInfo.TargetUser, fr.targetInfo.TargetAddress)

	fr.fileMutex.Lock()
	fr.fd, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}

	if fr.initialBuffer != nil {
		_, err = fr.initialBuffer.WriteTo(fr.fd)
		if err != nil {
			return err
		}
		fr.initialBuffer.Reset()
	}

	fr.initialBuffer = nil
	fr.fileMutex.Unlock()
	return nil
}

func (fr *FileRecorder) Close() error {
	if fr.fd != nil {
		fr.fd.Close()
		return nil
	}
	return fmt.Errorf("File Close Error")
}

// Write ...Write data to recording file
func (fr *FileRecorder) Write(data []byte, isClientInput bool) error {
	fr.fileMutex.Lock()
	if len(data) > 0 {
		if fr.fd != nil {
			fr.fd.Write(data)
		} else {
			fr.initialBuffer.Write(data)
		}
	}
	fr.fileMutex.Unlock()
	return nil
}
