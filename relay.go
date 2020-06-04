package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/aws/session"

	"ssh-gateway/cw_logger"
	"ssh-gateway/aws_workers"
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

// NewRelay Channel...This is a CTOR of a new relay channel
func (r *SSHRelay) NewRelayChannel(startTime time.Time, channel ssh.Channel, username string) *RelayChannel {

	enableAws := true
	logger, err := cwlogger.New(&cwlogger.Config{
	    LogGroupName: "GolangGroupName",
	    Client: cloudwatchlogs.New(session.New(), &aws.Config{Region: aws.String(aws_helpers.DEFAULT_REGION)}),
	  })
	
	  if err != nil {
		fmt.Printf("Cannot open cloud watch log group: %v\n", err)
		enableAws = false
	}

	return &RelayChannel{
		StartTime:     startTime,
		UserName:      username,
		SourceChannel: channel,
		initialBuffer: bytes.NewBuffer([]byte{}),
		logMutex:      &sync.Mutex{},
		enableAwsLogs: enableAws,
		logger: logger,
	}
}

func (r *SSHRelay) ProxySession(startTime time.Time, sshConn *ssh.ServerConn, srvNewChannel ssh.NewChannel, chans <-chan ssh.NewChannel) error {

	srvChannel, srvReqs, err := srvNewChannel.Accept()
	if err != nil {
		log.Printf("Session is not accepted, abort...")
		// todo : find more resources needed to close
		sshConn.Close()
		return err
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

	err = relayChannel.SyncToFile(dialerConfig.DefaultTargetAddress, r.RelayInfo.RecordingsDir)
	if err != nil {
		fmt.Fprintf(relayChannel, "Failed to Initialize Session.\r\n")
		relayChannel.Close()
		return err
	}

	// Dial To target, get target and open session
	dialer, err := NewDialer(r.RelayTargetInfo)
	if err != nil {
		fmt.Fprintf(relayChannel, "Failed to Initialize Session.\r\n")
		return err
	}
	client, err := dialer.connectToTarget(relayChannel)
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
	channel2, reqs2, err := client.OpenChannel("session", []byte{})
	if err != nil {
		fmt.Fprintf(relayChannel, "Remote session setup failed: %v\r\n", err)
		relayChannel.Close()
		return err
	}
	log.Printf("Starting session proxy...")
	r.proxy(maskedReqs, reqs2, relayChannel, channel2)

	return nil

}

func (r *SSHRelay) proxy(reqs1, reqs2 <-chan *ssh.Request, channel1 *RelayChannel, channel2 ssh.Channel) {
	var closer sync.Once
	closeFunc := func() {
		channel1.Close()
		channel2.Close()
	}

	defer closer.Do(closeFunc)

	closerChan := make(chan bool, 1)

	// From remote, to client.
	go func() {
		io.Copy(channel1, channel2)
		closerChan <- true
	}()

	go func() {
		io.Copy(channel2, channel1)
		closerChan <- true
	}()

	for {
		select {
		case req := <-reqs1:
			if req == nil {
				return
			}
			b, err := channel2.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case req := <-reqs2:
			if req == nil {
				return
			}
			b, err := channel1.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case <-closerChan:
			return
		}
	}
}
