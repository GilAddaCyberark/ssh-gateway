package ssh_engine

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
	rec "ssh-gateway/internal/pkg/recorders"
)

type PipedChannel struct {
	parentChannel ssh.Channel
	recorders     *[]rec.Recorder
	isFromClient  bool
}

func NewPipedChannel(isFromClient bool, sourceChannel ssh.Channel, recorders *[]rec.Recorder) PipedChannel {
	p := PipedChannel{}
	p.recorders = recorders
	p.parentChannel = sourceChannel
	p.isFromClient = isFromClient
	return p
}

func (p PipedChannel) Stderr() io.ReadWriter {
	return p.parentChannel.Stderr()
}

func (p PipedChannel) CloseWrite() error {
	return p.parentChannel.CloseWrite()
}
func (p PipedChannel) Read(data []byte) (int, error) {

	n, res := p.parentChannel.Read(data)
	// Write to Recorders
	if len(data) > 0 {
		if p.recorders != nil && p.isFromClient {
			// fmt.Printf("\n--> from read: %s|%v", p.isFromClient, string(data[:n]))
			for _, recorder := range *p.recorders {
				recorder.Write(data[:n], p.isFromClient)
			}
		}
	}
	return n, res
}

func (p PipedChannel) Write(data []byte) (int, error) {
	n, err := p.parentChannel.Write(data)
	return n, err
}

func (p PipedChannel) Close() error {
	return p.parentChannel.Close()
}

func (p PipedChannel) Init() error {
	if p.recorders != nil {
		for _, recorder := range *p.recorders {
			recorder.Init()
		}
	}
	return nil
}

func (p PipedChannel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	if p.parentChannel != nil {
		return p.parentChannel.SendRequest(name, wantReply, payload)
	} else {
		return false, fmt.Errorf("No Channel")
	}
}
