package main

import (
	"fmt"
	"io"
	"log"
	"sync"

	"golang.org/x/crypto/ssh"
)

type DialerConfig struct {
	DefaultPort          int
	MinPortToRotate      int
	MaxPortToRotate      int
	AuthType             string
	User                 string
	Password             string
	DefaultTargetAddress string
}

type SSHDialer struct {
	dialerConfig     DialerConfig
	DialerTargetInfo *TargetInfo
}

func NewDialer(TargetInfo *TargetInfo) (SSHDialer, error) {
	dialer := SSHDialer{}
	dialer.DialerTargetInfo = TargetInfo
	return dialer, nil
}

// Dial To target, get target and open session

// get user pass
func GetSSHUserPassConfig() *ssh.ClientConfig {
	sshUserPassConfig := &ssh.ClientConfig{
		User: dialerConfig.User,
		// todo : change this implementation to be passed from the idp / secret manage
		Auth: []ssh.AuthMethod{ssh.Password(dialerConfig.Password)},
	}
	sshUserPassConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	if sshUserPassConfig == nil {
		log.Fatal("No ssh user pass config")
	}
	return sshUserPassConfig
}

func (d *SSHDialer) connectToTarget(remoteAddr string, relayChannel *RelayChannel) (*ssh.Client, error) {
	// WriteAuthLog("Connecting to remote for relay (%s) by %s from %s.", remote.ConnectPath, sshConn.User(), sshConn.RemoteAddr())
	fmt.Fprintf(relayChannel, "Connecting to %s\r\n", remoteAddr)

	var clientConfig *ssh.ClientConfig
	log.Printf("Getting Ready to Dial Remote SSH %s", remoteAddr)

	if dialerConfig.AuthType == "pass" {
		clientConfig = GetSSHUserPassConfig()
	} else {
		log.Fatalf("Wrong Auth Type...")
	}
	// // Change port 22 to a range of 2022 - 2036
	// randomPort := rand.Intn(dialerConfig.MaxPortToRotate-dialerConfig.MinPortToRotate) + dialerConfig.MinPortToRotate
	// remoteHost := fmt.Sprintf("%s:%d", dialerConfig.DefaultTargetAddress, randomPort)
	remoteHost := fmt.Sprintf("%s:%d", d.DialerTargetInfo.TargetAddress, d.DialerTargetInfo.TargetPort)

	client, err := ssh.Dial("tcp", remoteHost, clientConfig)
	fmt.Fprintf(relayChannel, "Connection established to : %v\r\n", remoteHost)
	if err != nil {
		fmt.Fprintf(relayChannel, "Connect failed: %v\r\n", err)
		relayChannel.Close()
		return nil, err
	}

	log.Printf("Starting session proxy...")
	return client, err
}

func proxy(reqs1, reqs2 <-chan *ssh.Request, channel1 *RelayChannel, channel2 ssh.Channel) {
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
