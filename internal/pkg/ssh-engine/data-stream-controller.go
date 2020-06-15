package ssh_engine

import (
	"fmt"
	"io"
	config "ssh-gateway/configs"
	rec "ssh-gateway/internal/pkg/recorders"

	"golang.org/x/crypto/ssh"
)

type DataStreamController struct {
	relay              *SSHRelay
	recorders          []rec.Recorder
	terminationChannel chan bool
}

func NewDataStreamController(relay *SSHRelay) (*DataStreamController, error) {
	retV := DataStreamController{relay: relay}

	return &retV, nil
}

func (dsc *DataStreamController) Run(
	sourceChannel ssh.Channel,
	sourceRequests chan *ssh.Request,
	destChannel *ssh.Channel,
	destReqsChannel *<-chan *ssh.Request) error {

	err := dsc.initRecorders()
	if err != nil {
		return err
	}

	sourcePC := NewPipedChannel(false, sourceChannel, &dsc.recorders) // note that write direction is inverse
	destPC := NewPipedChannel(true, *destChannel, &dsc.recorders)

	// defer closer.Do(closeFunc)
	defer func() {
		// Gracefull shutdown
		destPC.Close()
		sourcePC.Close()
		if dsc.recorders != nil {
			for _, recorder := range dsc.recorders {
				recorder.Close()
			}
		}
	}()

	dsc.terminationChannel = make(chan bool, 1)

	// Copy from Source <--> Destination
	go func() {
		io.Copy(sourcePC, destPC)
		dsc.terminationChannel <- true
	}()

	go func() {
		io.Copy(destPC, sourcePC)
		dsc.terminationChannel <- true
	}()

	// Handle Requets and Pass them to the other chanel
	for {
		select {
		case req := <-sourceRequests:
			if req == nil {
				return nil
			}
			b, err := destPC.parentChannel.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return nil
			}
			req.Reply(b, nil)
		case req := <-*destReqsChannel:
			if req == nil {
				return nil
			}

			// todo - check why works with source chanel and not piped chanel
			b, err := sourceChannel.SendRequest(req.Type, req.WantReply, req.Payload)
			fmt.Printf("%v", req.Payload)
			if err != nil {
				return nil
			}
			req.Reply(b, nil)
		case <-dsc.terminationChannel:
			return nil
		}
	}
	return nil
}

func (dsc *DataStreamController) initRecorders() error {
	if config.Server_Config.EnableFileRecorder {
		var fileRecorder rec.Recorder = rec.NewFileRecorder(*dsc.relay.RelayTargetInfo, dsc.relay.RelayInfo.RecordingsDir)
		err := fileRecorder.Init()
		if err != nil {
			return err
		}
		dsc.recorders = append(dsc.recorders, fileRecorder)
	}

	if config.Server_Config.EnableCWLRecorder {
		cwlRecorder := rec.NewCWLRecorder(dsc.relay.RelayTargetInfo)
		err := cwlRecorder.Init()
		if err != nil {
			return err
		}
		var cwlRecorderIface rec.Recorder = cwlRecorder
		dsc.recorders = append(dsc.recorders, cwlRecorderIface)
	}
	return nil
}
