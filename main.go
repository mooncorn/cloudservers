package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/joho/godotenv"
)

var (
	AWS_KEY_PAIR_NAME = "cloudservers"
	AWS_IMAGE_ID      = "ami-0c101f26f147fa7fd"

	AWS_WAIT_MIN_DELAY = 5 * time.Second
	AWS_WAIT_MAX_DELAY = 15 * time.Second
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	CreateMinecraftServer()
}

func CreateMinecraftServer() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Step 1: Create EC2 Instance
	ec2Client := ec2.NewFromConfig(cfg)
	instanceId, err := createEC2Instance(
		ec2Client,
		CreateEC2InstanceConfig{
			InstanceType: types.InstanceTypeT2Micro,
		},
	)
	if err != nil {
		log.Fatal("Failed to create EC2 instance: ", err)
	}

	// Step 2: Install Docker using SSM
	ssmClient := ssm.NewFromConfig(cfg)

	installDocker(instanceId, ssmClient)

	// Step 3: Create Minecraft Server Docker Container
	createMinecraftContainer(instanceId, ssmClient)

	fmt.Println("Minecraft server setup complete!")
}

type CreateEC2InstanceConfig struct {
	InstanceType types.InstanceType
}

func createEC2Instance(client *ec2.Client, config CreateEC2InstanceConfig) (string, error) {
	// Define the instance launch parameters
	runInstancesInput := &ec2.RunInstancesInput{
		ImageId:      &AWS_IMAGE_ID,
		InstanceType: config.InstanceType,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		KeyName:      &AWS_KEY_PAIR_NAME,

		// Add additional configuration such as KeyName, SecurityGroupIds, etc.
	}

	// Launch the instance
	result, err := client.RunInstances(context.TODO(), runInstancesInput)
	if err != nil {
		return "", fmt.Errorf("failed to create EC2 instance: %w", err)
	}

	// Assuming we're only launching one instance and getting the first instance ID
	instanceId := *result.Instances[0].InstanceId
	fmt.Printf("Created EC2 instance with ID: %s\n", instanceId)

	// Wait for the instance to be in the OK state
	err = ec2.NewInstanceStatusOkWaiter(client, func(isowo *ec2.InstanceStatusOkWaiterOptions) {
		isowo.MinDelay = AWS_WAIT_MIN_DELAY
		isowo.MaxDelay = AWS_WAIT_MAX_DELAY
	}).Wait(context.TODO(), &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceId},
	}, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("instance %s did not become running within the timeout period: %v", instanceId, err)
	}

	return instanceId, nil
}

func installDocker(instanceId string, client *ssm.Client) {
	// Define the parameters for the command. Adjust if your SSM document requires different parameters.
	parameters := map[string][]string{"action": {"Install"}}

	// Send the command to install Docker using an SSM document
	sendCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-ConfigureDocker"), // Ensure this is the correct name of the document
		InstanceIds:  []string{instanceId},
		Parameters:   parameters,
	}

	sendCommandOutput, err := client.SendCommand(context.TODO(), sendCommandInput)
	if err != nil {
		log.Fatalf("Error sending command to install Docker: %v", err)
	}

	commandId := sendCommandOutput.Command.CommandId
	fmt.Printf("Command to install Docker sent, Command ID: %s\n", *commandId)

	err = ssm.NewCommandExecutedWaiter(client, func(cewo *ssm.CommandExecutedWaiterOptions) {
		cewo.MinDelay = AWS_WAIT_MIN_DELAY
		cewo.MaxDelay = AWS_WAIT_MAX_DELAY
	}).Wait(context.TODO(),
		&ssm.GetCommandInvocationInput{
			CommandId: commandId,
		},
		5*time.Minute)
	if err != nil {
		log.Fatal("Failed to wait for command output: ", err)
	}

	fmt.Printf("Docker installed successfully.")
}

func createMinecraftContainer(instanceId string, client *ssm.Client) {
	// Define the Docker run command to execute
	dockerRunCmd := "docker run -d -e EULA=TRUE -p 25565:25565 itzg/minecraft-server"

	// Send the command to the instance via SSM
	sendCommandInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"), // Using AWS-RunShellScript document to execute shell commands
		InstanceIds:  []string{instanceId},
		Parameters: map[string][]string{
			"commands": {dockerRunCmd},
		},
	}

	sendCommandOutput, err := client.SendCommand(context.TODO(), sendCommandInput)
	if err != nil {
		log.Fatalf("Error sending command to create Minecraft container: %v", err)
	}

	commandId := sendCommandOutput.Command.CommandId
	fmt.Printf("Command to create Minecraft container sent, Command ID: %s\n", *commandId)

	err = ssm.NewCommandExecutedWaiter(client, func(cewo *ssm.CommandExecutedWaiterOptions) {
		cewo.MinDelay = AWS_WAIT_MIN_DELAY
		cewo.MaxDelay = AWS_WAIT_MAX_DELAY
	}).Wait(context.TODO(),
		&ssm.GetCommandInvocationInput{
			CommandId: commandId,
		},
		5*time.Minute)
	if err != nil {
		log.Fatal("Failed to wait for command output: ", err)
	}

	fmt.Printf("Docker container created successfully.")
}
