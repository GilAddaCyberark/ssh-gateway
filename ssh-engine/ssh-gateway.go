package ssh_engine

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	aws_helpers "ssh-gateway/aws_workers"
	gen "ssh-gateway/ssh-engine/generic-structs"
)

const SERVER_VERSION = "SSH-2.0-EVEREST-SSH-GW"

var context *ServerContext

type ServerContext struct {
	ServerSigner    ssh.Signer
	ServerPublicKey []byte
	TenantId        string
}

type ServerConfig struct {
	ServerAddress string
	ServerKeyPath string
	User          string
}

type SSHGateway struct {
	SshConfig     *ssh.ServerConfig
	RelayInfo     *RelayInfo
	ListeningPort int
	// Set Global Gateway Logger
}

// Create the configuration of a new server and start it
func (s *SSHGateway) NewSSHGateway() error {
	context = &ServerContext{}
	context.TenantId = aws_helpers.TENANT_ID
	serverConfig := &ssh.ServerConfig{
		NoClientAuth:  false,
		MaxAuthTries:  1,
		ServerVersion: SERVER_VERSION,
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// todo : implement password check , user check & other auth methods
			bastionUser, targetUser, err := s.getTargetUser(c.User())
			if err != nil {
				return nil, err
			}

			perms := &ssh.Permissions{
				Extensions: map[string]string{
					"user_id": targetUser,
				},
			}

			log.Printf("Authentiated to IDP with user: %s...\n", bastionUser)
			return perms, nil
		},
		BannerCallback: func(c ssh.ConnMetadata) string {
			return "Welcome to SSH Gateway"
		},
	}

	// Add Server Keys

	signer, err := GetServerSigner()
	if err != nil {
		return fmt.Errorf("Invalid SSH Signer")
	}
	context.ServerSigner = signer
	serverConfig.AddHostKey(context.ServerSigner)
	s.SshConfig = serverConfig

	// Get Public Key
	publicKey, err := GetServerPublicKey()
	if err != nil {
		return fmt.Errorf("Invalid SSH Public Key")
	}
	context.ServerPublicKey = publicKey

	return nil
}

func (s *SSHGateway) getTargetUser(user string) (string, string, error) {

	parts := strings.Split(user, "@")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("Unsupported user format: %s, cannot be parsed into personal user, target, port ", user)
	}
	return parts[0], parts[1], nil
}

func (s *SSHGateway) ListenAndServe() error {

	addr := fmt.Sprintf("0.0.0.0:%d", s.ListeningPort)
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
		log.Printf("Exiting as there is a config problem...%v\n", err)
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
		connStr := srvConn.User()
		ti, err := gen.GetTargetInfoByConnectionString(connStr)
		if err != nil {
			return
		}

		relay, err := NewRelay(ti, s.RelayInfo)
		if err != nil {
			return
		}
		Session_Manager.AddNewSession(&relay)
		log.Printf("After AddSession: %d\n", Session_Manager.GetOpenSessionsCount())

		relay.ProxySession(startTime, srvConn, srvChannel, chans)
		
		Session_Manager.RemoveSession(&relay)
		fmt.Printf("After RemoveSession: %d\n", Session_Manager.GetOpenSessionsCount())
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

func GetServerSigner() (ssh.Signer, error) {
	key, err := ioutil.ReadFile(aws_helpers.PRIVATE_KEY)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
		return nil, err
	}

	return signer, nil
}

func GetServerPublicKey() ([]byte, error) {
	publicKey, err := ioutil.ReadFile(aws_helpers.PUBLIC_KEY)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
		return nil, err
	}
	return publicKey, nil
}
