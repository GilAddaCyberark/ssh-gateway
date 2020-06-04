package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
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
func (d *SSHDialer) GetSSHUserPassConfig() (*ssh.ClientConfig, error) {
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
	return sshUserPassConfig, nil
}

func (d *SSHDialer) GetJITSSHClientConfig() (*ssh.ClientConfig, error) {

	certPEM, err := d.GetTargetCertificate(
		context.TenantId,
		d.DialerTargetInfo.TargetId,
		"Some Token",
		context.ServerPublicKey)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)

	}

	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certPEM))
	if err != nil {
		log.Fatalf("unable to parse public key: %v", err)
	}
	_ = pk

	certSigner, err := ssh.NewCertSigner(pk.(*ssh.Certificate), context.ServerSigner)
	if err != nil {
		log.Fatalf("failed to create cert signer: %v", err)
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

func (d *SSHDialer) connectToTarget(relayChannel *RelayChannel) (*ssh.Client, error) {
	var clientConfig *ssh.ClientConfig
	var err error

	if len(d.DialerTargetInfo.TargetId) > 0 {
		fmt.Fprintf(relayChannel, "Resolving Instance ID %s to IP Address...", d.DialerTargetInfo.TargetId)

		publicIP, err := GetPublicIP(d.DialerTargetInfo.TargetId)
		if err != nil {
			fmt.Fprintf(relayChannel, "Failed to Resolve IP for Instance ID: '%s'\r\n", d.DialerTargetInfo.TargetId)
			return nil, err
		}
		d.DialerTargetInfo.TargetAddress = publicIP
		fmt.Fprintf(relayChannel, "Resolved IP Address:%s\r\n", d.DialerTargetInfo.TargetAddress)

	}

	remoteAddr := fmt.Sprintf("%s:%d", d.DialerTargetInfo.TargetAddress, d.DialerTargetInfo.TargetPort)
	fmt.Fprintf(relayChannel, "Connecting to %s\r\n", remoteAddr)
	log.Printf("Try to connect to target: %s", remoteAddr)

	if d.DialerTargetInfo.AuthType == "pass" {
		clientConfig, err = d.GetSSHUserPassConfig()
		if err != nil {
			return nil, err
		}

	} else if d.DialerTargetInfo.AuthType == "cert" {
		clientConfig, err = d.GetJITSSHClientConfig()
		if err != nil {
			return nil, err
		}

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

type SSHCertificateSignRequestDto struct {
	UserPublicKey     string `json:"user_public_key"`
	Principal         string `json:"principal"`
	ExpirationSeconds int    `json:"expiration_seconds"`
	KeyId             string `json:"key_id"`
}

func (d SSHDialer) GetTargetCertificate(tenant_id string, target_instance_id string, token_id string, public_key []byte) (string, error) {

	request := getTargetCertificateRequest{
		tenant_id,
		target_instance_id,
		SSHCertificateSignRequestDto{
			string(public_key),
			d.DialerTargetInfo.TargetUser,
			EXPIRATION_PERIOD,
			token_id}}

	result, err := invokeLambda(request, PHYSICAL_LAMBDA_NAME)
	if err != nil {
		fmt.Printf("invokeLambda returned error: %v\n", err)
		return "", err
	}
	var resp getTargetCertificateResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		fmt.Printf("Error Unmarshal GetTargetCertificate response: %v\n", err)
		return "", err
	}

	return resp.Certificate, nil
}

func invokeLambda(request interface{}, physical_lambda_name string) (*lambda.InvokeOutput, error) {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := lambda.New(sess, &aws.Config{Region: aws.String(DEFAULT_REGION)})

	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error marshalling invokeLambda request")
		return nil, err
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(physical_lambda_name), Payload: payload})
	if err != nil {
		fmt.Printf("Error calling invokeLambda: %v\n", err)
		return nil, err
	}
	return result, nil
}

type getTargetCertificateRequest struct {
	TenantId              string                       `json:"tenant_id"`
	TargetInstanceId      string                       `json:"instance_id"`
	SshCertificateRequest SSHCertificateSignRequestDto `json:"ssh_certificate_request"`
}

type getTargetCertificateResponse struct {
	Certificate string `json:"body"`
}
