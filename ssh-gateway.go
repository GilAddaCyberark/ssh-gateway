package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type TargetInfo struct {
	TargetUser    string
	TargetPass    string
	TargetAddress string
	TargetPort    int
}

type ServerConfig struct {
	ServerAddress string
	ServerKeyPath string
	User          string
}

type SSHGateway struct {
	SshConfig    *ssh.ServerConfig
	PersonalUser string
	PersonalPass []byte
	TargetInfo   *TargetInfo
}

// Create the configuration of a new server and start it
func (s *SSHGateway) NewSSHGateway() error {
	serverConfig := &ssh.ServerConfig{
		NoClientAuth:  false,
		ServerVersion: "SSH-2.0-BASTION", //todo: move to configuration + extract version
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// todo : implement password check , user check & other auth methods
			s.ParsePSMSyntaxUser(c.User())
			s.PersonalPass = pass
			perms := &ssh.Permissions{
				Extensions: map[string]string{
					"user_id":               bastionConfig.User,
					"permit-X11-forwarding": "true", // Set allowed capabilities from
				},
			}
			return perms, nil
		},
	}
	s.TargetInfo = &TargetInfo{}

	// Add Server Keys

	hostKey, err := ioutil.ReadFile(bastionConfig.ServerKeyPath)
	if err != nil {
		return fmt.Errorf("Unable to read host key file (%s): %s", bastionConfig.ServerKeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(hostKey)
	if err != nil {
		return fmt.Errorf("Invalid SSH Host Key (%s)", bastionConfig.ServerKeyPath)
	}

	serverConfig.AddHostKey(signer)

	s.SshConfig = serverConfig
	// s := &SSHGateway{
	// 	SshConfig: serverConfig,
	// }

	return nil
}

func (s *SSHGateway) ParsePSMSyntaxUser(user string) error {
	// Parsing the target info from the ssh user format
	//ssh personal_user@target_user@target_address@ssh_gw_address
	var err error = nil
	s.TargetInfo.TargetPort = 22
	parts := strings.Split(user, "@")
	s.PersonalUser = parts[0]
	s.TargetInfo.TargetUser = parts[1]
	s.TargetInfo.TargetAddress = parts[2]
	if strings.Contains(s.TargetInfo.TargetAddress, ":") {
		addressParts := strings.Split(s.TargetInfo.TargetAddress, ":")
		s.TargetInfo.TargetAddress = addressParts[0]
		s.TargetInfo.TargetPort, err = strconv.Atoi(addressParts[1])
		if err != nil {
			return fmt.Errorf("Error")
		}
	}
	if len(parts) > 3 {
		s.TargetInfo.TargetPort, err = strconv.Atoi(parts[3])
		if err != nil {
			return fmt.Errorf("Part of port exists, cannot be converted to port ")
		}
		// todo: Check that port and ip address are valid
	}
	return nil
}

func (s *SSHGateway) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go s.HandleConn(conn)
	}
}

func (s *SSHGateway) HandleConn(c net.Conn) {
	//log.Printf("Starting Accept SSH Connection...")
	startTime := time.Now()

	srvConn, chans, reqs, err := ssh.NewServerConn(c, s.SshConfig)
	if err != nil {
		//log.Printf("Exiting as there is a config problem...")
		c.Close()
		return
	}

	go ssh.DiscardRequests(reqs)
	srvChannel := <-chans
	if srvChannel == nil {
		//log.Printf("Exiting as couldn't fetch the channel...")
		srvConn.Close()
	}

	switch srvChannel.ChannelType() {
	case "session":
		relay, err := NewRelay(s.TargetInfo)
		if err != nil {
			return
		}
		relay.ProxySession(startTime, srvConn, srvChannel, chans)
	default:
		srvChannel.Reject(ssh.UnknownChannelType, "connection flow not supported, only interactive sessions are permitted.")
	}

	//log.Printf("ALL OK, closing as nothing left to do...")
	srvConn.Close()
}
