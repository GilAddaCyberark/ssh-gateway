package ssh_engine

import (
	"fmt"
	"log"
	"time"

	gen "ssh-gateway/internal/pkg/ssh-engine/generic-structs"

	"golang.org/x/crypto/ssh"
)

type RelayInfo struct {
	RecordingsDir   string
	EnableRecording bool
}

type SSHRelay struct {
	RelayTargetInfo *gen.TargetInfo
	RelayInfo       *RelayInfo
	Controller      *DataStreamController
}

func NewRelay(targetInfo *gen.TargetInfo, relayInfo *RelayInfo) (SSHRelay, error) {
	relay := SSHRelay{}
	relay.RelayTargetInfo = targetInfo
	relay.RelayInfo = relayInfo
	return relay, nil
}

func (r *SSHRelay) ProxySession(startTime time.Time, sshConn *ssh.ServerConn, srvNewChannel ssh.NewChannel) error {

	// Set Session ID + New Logger per session
	// sessionId :=  guuid.New().String()
	// Create Logger - dima

	// Accept Connection
	sourceChannel, sourceRequests, err := srvNewChannel.Accept()
	if err != nil {
		log.Printf("Session is not accepted, abort...")
		// todo : find more resources needed to close
		sshConn.Close()
		return err
	}
	defer sshConn.Close()

	// todo : replace with audit server

	// Proxy the channel and its requests
	//var agentForwarding bool = false
	sourceMaskedReqs := make(chan *ssh.Request, 5)
	go func() {
		// todo : Check why do we need those requests masking
		for req := range sourceRequests {
			switch req.Type {
			// todo - replace that logging with audit server..
			case "auth-agent-req@openssh.com":
				{
					// agentForwarding = true
					if req.WantReply {
						req.Reply(true, []byte{})
					}
					continue
				}
			case "pty-req", "shell":
				{
					if req.WantReply {
						req.Reply(true, []byte{})
						req.WantReply = false
					}
				}
			}
			sourceMaskedReqs <- req
		}
	}()

	// Dial to Target
	dialer, err := NewDialer(r.RelayTargetInfo)
	if err != nil {
		fmt.Fprintf(sourceChannel, "Failed to Initialize Session.\r\n")
		return err
	}
	client, err := dialer.connectToTarget(sourceChannel)
	if err != nil {
		if client != nil {
			client.Close()
		} else {
			log.Printf("Connection failed to target:")
		}
		return err

	} else {
		defer client.Close()

	}

	destChannel, destRequests, err := client.OpenChannel("session", []byte{})
	if err != nil {
		fmt.Fprintf(sourceChannel, "Remote session setup failed: %v\r\n", err)
		sourceChannel.Close()
		return err
	}

	// Set Data Stream Controller
	r.Controller, err = NewDataStreamController(r)
	if err != nil {
		fmt.Fprintf(sourceChannel, "Data stream controller setup failed: %v\r\n", err)
		sourceChannel.Close()
		return err
	}

	err = r.Controller.Run(sourceChannel, sourceMaskedReqs, &destChannel, &destRequests)
	if err != nil {
		fmt.Fprintf(sourceChannel, "Data stream controller Run failed: %v\r\n", err)
		sourceChannel.Close()
		return err
	}
	return nil
}
