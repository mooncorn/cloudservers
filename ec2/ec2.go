package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Service struct {
	client *ec2.EC2
}

func NewEC2Service() *EC2Service {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return &EC2Service{
		client: ec2.New(sess),
	}
}

func (ec2s *EC2Service) GetInstance(id string) (*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	}

	result, err := ec2s.client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance with ID %s not found", id)
	}

	return result.Reservations[0].Instances[0], nil
}

func (ec2s *EC2Service) GetInstances() ([]*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{}

	result, err := ec2s.client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	instances := make([]*ec2.Instance, 0)
	for _, reservation := range result.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances, nil
}

func (ec2s *EC2Service) CreateInstance(input *ec2.RunInstancesInput) (*ec2.Instance, error) {
	result, err := ec2s.client.RunInstances(input)
	if err != nil {
		return nil, err
	}

	// Define the input parameters for DescribeInstances API call
	describe := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(*result.Instances[0].InstanceId),
		},
	}

	// Wait for the instance to reach the "running" state
	err = ec2s.client.WaitUntilInstanceRunning(describe)
	if err != nil {
		return nil, err
	}

	instance, err := ec2s.GetInstance(*result.Instances[0].InstanceId)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (ec2s *EC2Service) DeleteInstance(id string) error {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	}

	_, err := ec2s.client.TerminateInstances(input)
	return err
}
