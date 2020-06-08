package recorders

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

type Recorder interface {
	Init() error
	Close() error
	Write(data []byte) error
}

func InitRecording(
	sourceChannel ssh.Channel,
	sourceRequests chan *ssh.Request,
	destChannel *ssh.Channel,
	destReqsChannel *<-chan *ssh.Request,
	recorders *[]Recorder) {

	// Init Recorders
	if recorders != nil {
		for _, recorder := range *recorders {
			recorder.Init()
		}
	}

	destPC := NewPipedChannel(*destChannel, recorders)
	sourcePC := NewPipedChannel(sourceChannel, recorders)

	// defer closer.Do(closeFunc)
	defer func() {
		destPC.Close()
		sourcePC.Close()
	}()

	stopSignalChannel := make(chan bool, 1)

	// Copy from Source <--> Destination
	go func() {
		io.Copy(sourcePC, destPC)
		stopSignalChannel <- true
	}()

	go func() {
		io.Copy(destPC, sourcePC)
		stopSignalChannel <- true
	}()

	// Handle Requets and Pass them to the other chanel
	for {
		select {
		case req := <-sourceRequests:
			if req == nil {
				return
			}
			b, err := destPC.parentChannel.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case req := <-*destReqsChannel:
			if req == nil {
				return
			}

			// todo - check why works with source chanel and not piped chanel
			b, err := sourceChannel.SendRequest(req.Type, req.WantReply, req.Payload)
			fmt.Printf("%v", req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case <-stopSignalChannel:
			return
		}
	}
}

type PipedChannel struct {
	parentChannel ssh.Channel
	recorders     *[]Recorder
}

func NewPipedChannel(sourceChannel ssh.Channel, recorders *[]Recorder) PipedChannel {
	p := PipedChannel{}
	p.recorders = recorders
	p.parentChannel = sourceChannel
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
	// fmt.Printf("<%c", res)
	return n, res
}

func (p PipedChannel) Write(data []byte) (int, error) {
	// fmt.Printf(">%c", data)
	if p.recorders != nil {
		for _, recorder := range *p.recorders {
			recorder.Write(data)
		}
	}
	n, err := p.parentChannel.Write(data)
	return n, err
}

func (p PipedChannel) Close() error {

	if p.recorders != nil {
		for _, recorder := range *p.recorders {
			recorder.Close()
		}
	}
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
