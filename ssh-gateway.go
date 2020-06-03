package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
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
	SshConfig     *ssh.ServerConfig
	PersonalUser  string
	PersonalPass  []byte
	TargetInfo    *TargetInfo
	RelayInfo     *RelayInfo
	listeningPort int
}

// Create the configuration of a new server and start it
func (s *SSHGateway) NewSSHGateway() error {
	serverConfig := &ssh.ServerConfig{
		NoClientAuth:  false,
		MaxAuthTries:  1,
		ServerVersion: "SSH-2.0-EVEREST-SSH-GW", //todo: move to configuration + extract version
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// todo : implement password check , user check & other auth methods
			err := s.ParsePSMSyntaxUser(c.User())
			if err != nil {
				return nil, err
			}

			if pass != nil {
				s.PersonalPass = pass
			} else {
				log.Println("Np Password Received")
			}
			perms := &ssh.Permissions{
				Extensions: map[string]string{
					"user_id": s.TargetInfo.TargetUser,
				},
			}
			return perms, nil
		},
		BannerCallback: func(c ssh.ConnMetadata) string {
			return "Welcome to SSH Gateway"
		},
	}
	s.TargetInfo = &TargetInfo{}

	// Add Server Keys

	hostKey, PublicKey, err := generateKeyPair()
	_ = PublicKey // todo : use or delete

	// hostKey, err := ioutil.ReadFile(serverConfig.ServerKeyPath)
	if err != nil {
		return fmt.Errorf("Unable to rgeberate kp")
	}
	hostKeyPem := encodePrivateKeyToPEM(hostKey)
	signer, err := ssh.ParsePrivateKey(hostKeyPem)
	if err != nil {
		return fmt.Errorf("Invalid SSH Signer")
	}

	serverConfig.AddHostKey(signer)

	s.SshConfig = serverConfig

	return nil
}

func (s *SSHGateway) ParsePSMSyntaxUser(user string) error {
	// Parsing the target info from the ssh user format
	//ssh personal_user@target_user@target_address@ssh_gw_address
	var err error = nil
	s.TargetInfo.TargetPort = 22
	parts := strings.Split(user, "@")
	if len(parts) < 3 {
		return fmt.Errorf("Unsupported user format: %s, cannot be parsed into personal user, target, port ")
	}
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
			return fmt.Errorf("Part of port exists, cannot be splited to port ")
		}
		// todo: Check that port and ip address are valid
	}
	return nil
}

func (s *SSHGateway) ListenAndServe() error {

	addr := fmt.Sprintf("0.0.0.0:%d", s.listeningPort)
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
		log.Printf("Exiting as there is a config problem...")
		if srvConn != nil {
			srvConn.Close()
		}
		if c != nil {
			c.Close()
		}
		return
	}
	// Fetch Channels
	srvChannel := <-chans
	if srvChannel == nil {
		log.Printf("Exit Connection: Could Get Channels...")
		srvConn.Close()
	}
	go ssh.DiscardRequests(reqs)

	// See chanel types in:
	// https://net-ssh.github.io/ssh/v1/chapter-3.html#:~:text=Channel%20Types,support%20for%20%E2%80%9Cx11%E2%80%9D%20channels.

	switch srvChannel.ChannelType() {
	case "session":
		relay, err := NewRelay(s.TargetInfo, s.RelayInfo)
		if err != nil {
			return
		}
		relay.ProxySession(startTime, srvConn, srvChannel, chans)
	default:
		log.Printf("Chqnnel Type Unsupported: %s Connection Rejected", srvChannel.ChannelType())
		srvChannel.Reject(ssh.UnknownChannelType, "connection flow not supported, only interactive sessions are permitted.")
	}

	//log.Printf("ALL OK, closing as nothing left to do...")
	srvConn.Close()
}

func generateKeyPair() (*rsa.PrivateKey, []byte, error) {
	const bitSize = 4096
	var err error

	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		log.Fatal(err.Error())
	}

	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal(err.Error())
	}

	return privateKey, publicKeyBytes, err

}

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated")
	return privateKey, nil
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key generated")
	return pubKeyBytes, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}
