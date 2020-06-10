package aws_helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"encoding/json"
	"fmt"
	"time"
)

type SSHCertificateSignRequestDto struct {
	UserPublicKey     string `json:"user_public_key"`
	Principal         string `json:"principal"`
	ExpirationSeconds int    `json:"expiration_seconds"`
	KeyId             string `json:"key_id"`
}

type getTargetSertificateRequest struct {
	TenantId              string                       `json:"tenant_id"`
	TargetInstanceId      string                       `json:"instance_id"`
	SshCertificateRequest SSHCertificateSignRequestDto `json:"ssh_certificate_request"`
}

type getTargetSertificateResponse struct {
	Certificate string `json:"body"`
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

	for i := 0; i < 5; i++ {

		result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(physical_lambda_name), Payload: payload})
		if err != nil {

			fmt.Printf("Error calling invokeLambda: %v\n", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return result, nil
	}
	return nil, fmt.Errorf("Error calling invokeLambda")
}

func GetTargetCertificate(tenant_id string, target_instance_id string, token_id string, public_key []byte) (string, error) {

	request := getTargetSertificateRequest{
		tenant_id,
		target_instance_id,
		SSHCertificateSignRequestDto{
			string(public_key),
			USER_NAME,
			EXPIRATION_PERIOD,
			token_id}}

	result, err := invokeLambda(request, PHYSICAL_LAMBDA_NAME)
	if err != nil {
		fmt.Printf("invokeLambda returned error: %v\n", err)
		return "", err
	}
	var resp getTargetSertificateResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		fmt.Printf("Error Unmarshal GetTargetCertificate response: %v\n", err)
		return "", err
	}

	return resp.Certificate, nil
}
