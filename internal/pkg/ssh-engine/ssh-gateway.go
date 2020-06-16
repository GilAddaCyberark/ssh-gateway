package ssh_engine

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	cfg "ssh-gateway/configs"
	gen "ssh-gateway/internal/pkg/ssh-engine/generic-structs"
)

const SERVER_VERSION = "SSH-2.0-EVEREST-SSH-GW"
const KEY_BITS = 4096

var context *ServerContext
var Session_Manager SessionManager = *NewSessionManager()

type ServerContext struct {
	ServerSigner    ssh.Signer
	ServerPublicKey []byte
	TenantId        string
}

type SSHGateway struct {
	SshConfig     *ssh.ServerConfig
	RelayInfo     *RelayInfo
	ListeningPort int
	Config        *cfg.ServerConfig
	AWSConfig     *cfg.AWSConfig
	// Set Global Gateway Logger
}

// Create the configuration of a new server and start it
func (s *SSHGateway) NewSSHGateway() error {
	s.Config = cfg.Server_Config
	context = &ServerContext{}
	context.TenantId = cfg.AWS_Config.TenantId
	sshServerConfig := &ssh.ServerConfig{
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

	signer, err := s.GetServerSigner()
	if err != nil {
		return fmt.Errorf("Invalid SSH Signer")
	}
	context.ServerSigner = signer
	sshServerConfig.AddHostKey(context.ServerSigner)
	s.SshConfig = sshServerConfig

	// Get Public Key
	publicKey, err := s.GetServerPublicKey()
	if err != nil {
		return fmt.Errorf("Invalid SSH Public Key")
	}
	context.ServerPublicKey = publicKey

	Session_Manager.StartTerminationThread()

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

	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "connection flow not supported, only interactive sessions are permitted")
			continue
		}
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
		relay.ProxySession(startTime, srvConn, newChannel)
		Session_Manager.RemoveSession(&relay)
		fmt.Printf("After RemoveSession: %d\n", Session_Manager.GetOpenSessionsCount())
	}
}

func (s SSHGateway) GetServerSigner() (ssh.Signer, error) {
	key, err := ioutil.ReadFile(s.Config.PrivateKeyPath)
	if err != nil {
		log.Printf("unable to read private key: %v", err)
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %v", err)
	}

	return signer, nil
}

func (s SSHGateway) GetServerPublicKey() ([]byte, error) {
	publicKey, err := ioutil.ReadFile(s.Config.PublicKeyPath)
	if err != nil {
		log.Printf("unable to read private key: %v", err)
		return nil, err
	}
	return publicKey, nil
}
