package aws_helpers

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func GetPuplicIP(target_instance_id string) (ip string, err error) {
	defer func() {
		if r := recover(); r != nil {
			ip = ""
			err = r.(error)
		}
	}()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ec2.New(sess, &aws.Config{Region: aws.String(DEFAULT_REGION)})

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
			&ec2.Filter{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(target_instance_id),
				},
			},
		},
	}

	resp, err := svc.DescribeInstances(params)
	if err != nil {
		fmt.Printf("Error calling DescribeInstances: %v\n", err)
		return "", err
	}

	if resp == nil {
		err := fmt.Errorf("Empty Response from AWS EC2 Metatadata service\n", err)
		return "", err
	}
	public_ip := resp.Reservations[0].Instances[0].PublicIpAddress
	if *public_ip == "" {
		err := fmt.Errorf("Public IP of instance id: %s not found\n", target_instance_id)
		return "", err
	}

	return *public_ip, nil
}
