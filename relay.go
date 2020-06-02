package main

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type RelayConfig struct {
	LogsDir string
}

type SSHRelay struct {
	RelayTargetInfo *TargetInfo
}

func NewRelay(TargetInfo *TargetInfo) (SSHRelay, error) {
	relay := SSHRelay{}
	relay.RelayTargetInfo = TargetInfo
	return relay, nil
}

// NewRelay Channel...This is a CTOR of a new relay channel
func (r *SSHRelay) NewRelayChannel(startTime time.Time, channel ssh.Channel, username string) *RelayChannel {
	return &RelayChannel{
		StartTime:     startTime,
		UserName:      username,
		SourceChannel: channel,
		initialBuffer: bytes.NewBuffer([]byte{}),
		logMutex:      &sync.Mutex{},
	}
}

func (r *SSHRelay) ProxySession(startTime time.Time, sshConn *ssh.ServerConn, srvNewChannel ssh.NewChannel, chans <-chan ssh.NewChannel) {

	srvChannel, srvReqs, err := srvNewChannel.Accept()
	if err != nil {
		log.Printf("Session is not accepted, abort...")
		sshConn.Close()
		return
	}
	defer sshConn.Close()

	// todo : replace with audit server

	relayChannel := r.NewRelayChannel(startTime, srvChannel, sshConn.User())

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
	var agentForwarding bool = false
	maskedReqs := make(chan *ssh.Request, 5)
	go func() {
		// For the pty-req and shell request types, we have to reply to those right away.
		// This is for PuTTy compatibility - if we don't, it won't allow any input.
		// We also have to change them to WantReply = false,
		// or a double reply will cause a fatal error client side.
		for req := range srvReqs {

			// todo - replace that logging with audit server..

			// relayChannel.LogRequest(req)
			if req.Type == "auth-agent-req@openssh.com" {
				agentForwarding = true
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
			maskedReqs <- req
		}
	}()

	// Set the window header to SSH Relay login.
	fmt.Fprintf(relayChannel, "%s]0;SSH Bastion Relay Login%s", []byte{27}, []byte{7})

	err = relayChannel.SyncToFile(dialerConfig.DefaultTargetAddress)
	if err != nil {
		fmt.Fprintf(relayChannel, "Failed to Initialize Session.\r\n")
		relayChannel.Close()
		return
	}

	// Dial To target, get target and open session
	dialer, err := NewDialer(r.RelayTargetInfo)
	if err != nil {
		fmt.Fprintf(relayChannel, "Failed to Initialize Session.\r\n")
		return
	}
	client, err := dialer.connectToTarget(dialerConfig.DefaultTargetAddress, relayChannel)
	defer client.Close()
	channel2, reqs2, err := client.OpenChannel("session", []byte{})
	if err != nil {
		fmt.Fprintf(relayChannel, "Remote session setup failed: %v\r\n", err)
		relayChannel.Close()
		return
	}
	log.Printf("Starting session proxy...")
	proxy(maskedReqs, reqs2, relayChannel, channel2)
}
