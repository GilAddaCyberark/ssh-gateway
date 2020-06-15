package ssh_engine

import (
	"fmt"
	"log"
	config "ssh-gateway/configs"
	"time"

	rec "ssh-gateway/internal/pkg/recorders"
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
			// todo - replace that logging with audit server..
			if req.Type == "auth-agent-req@openssh.com" {
				// agentForwarding = true
				if req.WantReply {
					req.Reply(true, []byte{})
				}
				continue
			} else if (req.Type == "pty-req") && (req.WantReply) {
				req.Reply(true, []byte{})
				req.WantReply = false
			} else if (req.Type == "shell") && (req.WantReply) {
				req.Reply(true, []byte{})
				req.WantReply = false
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

	// Set Recorders
	var recorders []rec.Recorder
	if config.Server_Config.EnableFileRecorder {
		var fileRecorder rec.Recorder = rec.NewFileRecorder(*r.RelayTargetInfo, r.RelayInfo.RecordingsDir)
		recorders = append(recorders, fileRecorder)
	}

	if config.Server_Config.EnableCWLRecorder {
		cwlRecorder, err := rec.NewCWLRecorder(r.RelayTargetInfo)
		if err != nil {
			return err
		}
		var cwlRecorderIface rec.Recorder = cwlRecorder
		recorders = append(recorders, cwlRecorderIface)

	}
	rec.InitRecording(sourceChannel, sourceMaskedReqs, &destChannel, &destRequests, &recorders)
	return nil
}
