package main

import (
	"fmt"
	"log"
	"time"

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
func (d *SSHDialer) GetSSHUserPassConfig() *ssh.ClientConfig {
	sshUserPassConfig := &ssh.ClientConfig{
		User: d.DialerTargetInfo.TargetUser,
		// todo : change this implementation to be passed from the idp / secret manage
		Auth:    []ssh.AuthMethod{ssh.Password(dialerConfig.Password)},
		Timeout: 15 * time.Second,
	}
	sshUserPassConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	if sshUserPassConfig == nil {
		log.Fatal("No ssh user pass config")
	}
	return sshUserPassConfig
}

func (d *SSHDialer) GetJITSSHClientConfig() {
	// // Todo : implement ssh.clientconfig - gil
	// 	// SSH Cert Config

	// 	key, err := ioutil.ReadFile("/Users/gadda/.ssh/id_rsa")
	// 	if err != nil {
	// 		log.Fatalf("unable to read private key: %v", err)
	// 	}

	// 	// Create the Signer for this private key.
	// 	signer, err := ssh.ParsePrivateKey(key)
	// 	if err != nil {
	// 		log.Fatalf("unable to parse private key: %v", err)
	// 	}

	// 	// Load the certificate
	// 	cert, err := ioutil.ReadFile("/Users/gadda/.ssh/id_rsa-cert.pub")
	// 	if err != nil {
	// 		log.Fatalf("unable to read certificate file: %v", err)
	// 	}

	// 	pk, _, _, _, err := ssh.ParseAuthorizedKey(cert)
	// 	if err != nil {
	// 		log.Fatalf("unable to parse public key: %v", err)
	// 	}

	// 	certSigner, err := ssh.NewCertSigner(pk.(*ssh.Certificate), signer)
	// 	if err != nil {
	// 		log.Fatalf("failed to create cert signer: %v", err)
	// 	}

	// 	cert_config := &ssh.ClientConfig{
	// 		User: "ec2-user",
	// 		Auth: []ssh.AuthMethod{
	// 			// Use the PublicKeys method for remote authentication.
	// 			ssh.PublicKeys(certSigner),
	// 		},
	// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	// 	}

	// 	return cert_config
}

func (d *SSHDialer) GetJitSSHCeretificate(targetId string) {
	//todo : implememt jit certificate from lambda  ...dima the king
}

func (d *SSHDialer) GetTargetIDAddress(targetId string) {
	//todo : implememt jit certificate from lambda  ...dima the king
}

func (d *SSHDialer) connectToTarget(remoteAddr string, relayChannel *RelayChannel) (*ssh.Client, error) {
	// WriteAuthLog("Connecting to remote for relay (%s) by %s from %s.", remote.ConnectPath, sshConn.User(), sshConn.RemoteAddr())
	fmt.Fprintf(relayChannel, "Connecting to %s\r\n", remoteAddr)

	var clientConfig *ssh.ClientConfig
	log.Printf("Try to connect to target: %s", remoteAddr)

	if dialerConfig.AuthType == "pass" {
		clientConfig = d.GetSSHUserPassConfig()
	} else {
		log.Fatalf("Wrong Auth Type...")
	}
	remoteHost := fmt.Sprintf("%s:%d", d.DialerTargetInfo.TargetAddress, d.DialerTargetInfo.TargetPort)
	client, err := ssh.Dial("tcp", remoteHost, clientConfig)
	if err != nil {
		fmt.Fprintf(relayChannel, "Connection failed to: %v\r\n", err)
		return nil, err
	}

	fmt.Fprintf(relayChannel, "Connection established to : %v\r\n", remoteHost)

	log.Printf("Starting session proxy...")
	return client, err
}
