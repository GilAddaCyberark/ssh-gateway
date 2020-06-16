package ssh_engine

import (
	"fmt"
	"io"
	config "ssh-gateway/configs"
	rec "ssh-gateway/internal/pkg/recorders"

	"time"

	"golang.org/x/crypto/ssh"
)

type DataStreamController struct {
	relay     *SSHRelay
	recorders []rec.Recorder
	destPC    PipedChannel
	sourcePC  PipedChannel

	terminationChannel chan bool
	LastUserInputTime  time.Time
	SessionStartTime   time.Time
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

	dsc.sourcePC = NewPipedChannel(false, sourceChannel, &dsc.recorders) // note that write direction is inverse
	dsc.destPC = NewPipedChannel(true, *destChannel, &dsc.recorders)

	// defer closer.Do(closeFunc)
	defer func() {
		// Gracefull shutdown
		dsc.destPC.Close()
		dsc.sourcePC.Close()
		if dsc.recorders != nil {
			for _, recorder := range dsc.recorders {
				recorder.Close()
			}
		}
	}()

	dsc.LastUserInputTime = time.Now()
	dsc.SessionStartTime = dsc.LastUserInputTime
	dsc.terminationChannel = make(chan bool, 1)

	// Copy from Source <--> Destination
	go func() {
		io.Copy(dsc.sourcePC, dsc.destPC)
		dsc.terminationChannel <- true
	}()

	go func() {
		io.Copy(dsc.destPC, dsc.sourcePC)
		dsc.terminationChannel <- true
	}()

	// Handle Requets and Pass them to the other chanel
	for {
		select {
		case req := <-sourceRequests:
			if req == nil {
				return nil
			}
			b, err := dsc.destPC.parentChannel.SendRequest(req.Type, req.WantReply, req.Payload)
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

func (dsc *DataStreamController) TerminateSession() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	dsc.terminationChannel <- true
	return nil
}

func (dsc *DataStreamController) SendMessageToUser(msg string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	_, err = dsc.sourcePC.Write([]byte(msg))
	return err
}
