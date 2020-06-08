package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

type RelayInfo struct {
	RecordingsDir   string
	EnableRecording bool
}

type SSHRelay struct {
	RelayTargetInfo *TargetInfo
	RelayInfo       *RelayInfo
}

func NewRelay(TargetInfo *TargetInfo, RelayInfo *RelayInfo) (SSHRelay, error) {
	relay := SSHRelay{}
	relay.RelayTargetInfo = TargetInfo
	relay.RelayInfo = RelayInfo
	return relay, nil
}

func (r *SSHRelay) ProxySession(startTime time.Time, sshConn *ssh.ServerConn, srvNewChannel ssh.NewChannel, chans <-chan ssh.NewChannel) error {

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

	// Handle all incoming channel requests
	go func() {
		for srvNewChannel = range chans {
			if srvNewChannel == nil {
				return
			}

			srvNewChannel.Reject(ssh.Prohibited, "remote server denied channel request")
			continue
		}
	}()

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

	// Set Recorders
	fr := NewFileRecorder(*r.RelayTargetInfo, r.RelayInfo.RecordingsDir)
	cwlRecorder, err := NewCWLRecorder()
	if err != nil {
		return err
	}
	recorders := []Recorder{fr, cwlRecorder}

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
	log.Printf("Starting session proxy...")
	// Start Recordig
	// Log session start
	// relayChannel.Logger.SessionStarted("Session started: "+relayChannel.SessionId, "ProxySession")
	InitRecording(sourceChannel, sourceMaskedReqs, &destChannel, &destRequests, &recorders)
	// Log Session Close
	// Log.Sesso...
	return nil
}
