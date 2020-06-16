package ssh_engine

import (
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/ssh"

	cfg "ssh-gateway/configs"
	aws_helpers "ssh-gateway/internal/pkg/aws_workers"
	generic_structs "ssh-gateway/internal/pkg/ssh-engine/generic-structs"
)

type SSHDialer struct {
	DialerTargetInfo *generic_structs.TargetInfo
}

func NewDialer(target_info *generic_structs.TargetInfo) (SSHDialer, error) {
	dialer := SSHDialer{}
	dialer.DialerTargetInfo = target_info
	return dialer, nil
}

// Dial To target, get target and open session

// get user pass
func (d *SSHDialer) GetSSHUserPassConfig() (*ssh.ClientConfig, error) {
	sshUserPassConfig := &ssh.ClientConfig{
		User: d.DialerTargetInfo.TargetUser,
		// todo : change this implementation to be passed from the idp / secret manage
		Auth:    []ssh.AuthMethod{ssh.Password(cfg.Dialer_Config.Password)},
		Timeout: 15 * time.Second,
	}
	sshUserPassConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	if sshUserPassConfig == nil {
		return nil, fmt.Errorf("No ssh user pass config")
	}
	return sshUserPassConfig, nil
}

func (d *SSHDialer) GetJITSSHClientConfig() (*ssh.ClientConfig, error) {

	const cert_id_template = "cert_for_%s_%s_%s"
	cert_id := fmt.Sprintf(cert_id_template,
		d.DialerTargetInfo.TargetAddress,
		d.DialerTargetInfo.TargetId,
		d.DialerTargetInfo.SessionId)
	certPEM, err := aws_helpers.GetTargetCertificate(
		d.DialerTargetInfo.TargetUser,
		cfg.AWS_Config.TenantId,
		d.DialerTargetInfo.TargetId,
		cert_id,
		context.ServerPublicKey)
	if err != nil {

		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certPEM))
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %v", err)

	}
	_ = pk

	certSigner, err := ssh.NewCertSigner(pk.(*ssh.Certificate), context.ServerSigner)
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User: d.DialerTargetInfo.TargetUser,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(certSigner),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return clientConfig, nil
}

func (d *SSHDialer) connectToTarget(relayChannel io.Writer) (*ssh.Client, error) {
	var clientConfig *ssh.ClientConfig
	var err error

	if len(d.DialerTargetInfo.TargetId) > 0 {
		fmt.Fprintf(relayChannel, "Resolving Instance ID %s to IP Address...", d.DialerTargetInfo.TargetId)
		var publicIP string
		if d.DialerTargetInfo.TargetProvider == "aws" {
			publicIP, err = aws_helpers.GetPuplicIP(d.DialerTargetInfo.TargetId)
			if err != nil {
				fmt.Fprintf(relayChannel, "Failed to Resolve IP for Instance ID: '%s'\r\n", d.DialerTargetInfo.TargetId)
				return nil, err
			}
		} else {
			fmt.Fprintf(relayChannel, "Failed to Resolve provider name: '%s'\r\n", d.DialerTargetInfo.TargetProvider)
			err := fmt.Errorf("Unable to resolve provide")
			return nil, err
		}

		d.DialerTargetInfo.TargetAddress = publicIP
		fmt.Fprintf(relayChannel, "Resolved IP Address:%s\r\n", d.DialerTargetInfo.TargetAddress)

	}

	remoteAddr := fmt.Sprintf("%s:%d", d.DialerTargetInfo.TargetAddress, d.DialerTargetInfo.TargetPort)
	fmt.Fprintf(relayChannel, "Connecting to %s\r\n", remoteAddr)
	log.Printf("Try to connect to target: %s", remoteAddr)

	switch d.DialerTargetInfo.AuthType {
	case "pass":
		{
			clientConfig, err = d.GetSSHUserPassConfig()
			if err != nil {
				return nil, err
			}
		}
	case "cert":
		{
			clientConfig, err = d.GetJITSSHClientConfig()
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("Wrong Auth Type...")
	}

	remoteHost := fmt.Sprintf("%s:%d", d.DialerTargetInfo.TargetAddress, d.DialerTargetInfo.TargetPort)
	client, err := ssh.Dial("tcp", remoteHost, clientConfig)
	if err != nil {
		fmt.Fprintf(relayChannel, "Connection failed to: %v\r\n", err)
		return nil, err
	}

	fmt.Fprintf(relayChannel, "Connection established to : %v\r\n", remoteHost)

	return client, err
}
