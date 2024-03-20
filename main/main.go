package main

import (
	"context"
	mydocker "dasior/cloudservers/docker"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/ssh"
)

type InstanceInfo struct {
	InstanceID    string `json:"instanceID"`
	InstanceType  string `json:"instanceType"`
	PublicIP      string `json:"publicIP"`
	PrivateIP     string `json:"privateIP"`
	LaunchTime    string `json:"launchTime"`
	InstanceState string `json:"instanceState"`
}

// func main() {
// 	// Load environment variables from .env file
// 	if err := godotenv.Load(); err != nil {
// 		fmt.Println("Error loading .env file:", err)
// 		return
// 	}

// 	r := gin.Default()

// 	r.POST("/instance", createEC2InstanceHandler)
// 	r.POST("/ssh", doSomething)
// 	r.GET("/instance/:id", getInstance)

// 	r.Run(":3000")
// }

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// region := "us-east-1"
	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String(region),
	// })
	// if err != nil {
	// 	log.Fatal("could not create session", err)
	// }

	dockerService, err := mydocker.NewDockerService("54.91.26.120")
	if err != nil {
		fmt.Println("Failed to create docker client: ", err)
		return
	}
	defer dockerService.CloseDockerClient()

	container, err := dockerService.CreateContainer(
		&container.Config{
			Image: "itzg/minecraft-server",
			Env: []string{
				"EULA=true",
				"VERSION=latest",
				"TYPE=spigot",
			},
		},
		&container.HostConfig{
			Binds: []string{
				"container-data:/data",
			},
			PortBindings: nat.PortMap{
				"25565/tcp": []nat.PortBinding{{HostPort: "25565"}},
			},
		})
	if err != nil {
		fmt.Println("Failed to create container:", err)
	}

	fmt.Println("created: ", container.ID, container.Name)

	container, err = dockerService.GetContainer()
	if err != nil {
		fmt.Println("Failed to get container:", err)
	}

	fmt.Println("get: ", container.ID, container.Name)

	container, err = dockerService.RemoveContainer()
	if err != nil {
		fmt.Println("Failed to remove container:", err)
	}

	fmt.Println("removed: ", container.ID, container.Name)
}

func executeScriptOnInstance(sess *session.Session, instanceID *string, keyPath *string, scriptPath *string) {
	// Create an EC2 service client
	svc := ec2.New(sess)

	// Get the public IP address of the instance
	describeParams := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(*instanceID)},
	}
	result, err := svc.DescribeInstances(describeParams)
	if err != nil {
		fmt.Println("Error describing instance:", err)
		return
	}
	instanceIP := *result.Reservations[0].Instances[0].PublicIpAddress
	fmt.Println("Public IP address of the instance:", instanceIP)

	cmd := exec.Command("ssh", "-i", *keyPath, fmt.Sprintf("ec2-user@%s", instanceIP), "bash", "-s")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Error obtaining stdin pipe:", err)
		return
	}

	// Open script file
	scriptFile, err := os.Open(*scriptPath)
	if err != nil {
		fmt.Println("Error opening script file:", err)
		return
	}
	defer scriptFile.Close()

	// Write script content to stdin
	_, err = io.Copy(stdin, scriptFile)
	if err != nil {
		fmt.Println("Error writing script to stdin:", err)
		return
	}
	stdin.Close()

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return
	}

	fmt.Println("Output:", string(output))
}

func createSSHConnection(publicIP *string, privateKey *string) *ssh.Client {
	// SSH connection configuration
	config := &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{
			// Add your private key here
			createSSHAuthMethod(*privateKey),
		},
		// Optionally, you can include HostKeyCallback to verify server's host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// SSH connection establishment
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", *publicIP), config)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer conn.Close()

	return conn
}

func getDockerContainers(publicIP *string) {
	// Connect to Docker on the EC2 instance
	cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("tcp://%s:2375", *publicIP)), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// List Docker containers
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}

	// Print container IDs
	for _, container := range containers {
		fmt.Println(container.ID)
	}
}

func getInstance(id *string, region *string) (InstanceInfo, error) {
	// Create a new AWS session using default credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(*region),
	})
	if err != nil {
		return InstanceInfo{}, err
	}

	// Create an EC2 service client
	svc := ec2.New(sess)

	// Specify parameters for the describe instance request
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(*id),
		},
	}

	// Retrieve information about the EC2 instance
	result, err := svc.DescribeInstances(describeInput)
	if err != nil {
		return InstanceInfo{}, err
	}

	// Extract instance info
	var instanceInfo InstanceInfo
	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		instance := result.Reservations[0].Instances[0]
		instanceInfo = InstanceInfo{
			InstanceID:    *instance.InstanceId,
			InstanceType:  *instance.InstanceType,
			PublicIP:      aws.StringValue(instance.PublicIpAddress),
			PrivateIP:     aws.StringValue(instance.PrivateIpAddress),
			LaunchTime:    instance.LaunchTime.String(),
			InstanceState: *instance.State.Name,
		}
	} else {
		return InstanceInfo{}, errors.New("no instance found with the specified id")
	}

	return instanceInfo, nil
}

func createInstance(region *string) (string, error) {
	// Create a new AWS session using default credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(*region),
	},
	)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("failed to create aws session")
	}

	// Create an EC2 service client
	svc := ec2.New(sess)

	keyName := "cloudservers"

	// Specify parameters for the instance
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String("ami-0d7a109bf30624c99"), // AMI ID of the instance
		InstanceType: aws.String("t2.micro"),              // Instance type
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      &keyName,
	}

	// Run the instance
	result, err := svc.RunInstances(runInput)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("failed to launch instance")
	}

	// Check if there is at least one instance created
	if len(result.Instances) == 0 {
		fmt.Println("No instances created")
		return "", errors.New("no instances created")
	}

	// Extract the instance ID from the first instance in the result
	instanceID := *result.Instances[0].InstanceId

	return instanceID, nil
}

func installDocker(conn *ssh.Client) {
	cmd := "sudo yum update -y && sudo yum install -y docker"
	executeCommand(conn, &cmd)
	fmt.Println("Docker installed successfully")
}

func enableRemoteConnection(conn *ssh.Client) {
	cmd := "sudo mkdir -p /etc/systemd/system/docker.service.d && echo '[Service]\nExecStart=\nExecStart=/usr/bin/dockerd -H tcp://0.0.0.0:2375' | sudo tee /etc/systemd/system/docker.service.d/override.conf && sudo systemctl daemon-reload && sudo systemctl restart docker"
	executeCommand(conn, &cmd)
	fmt.Println("Docker remote connection enabled successfully")
}

func executeCommand(conn *ssh.Client, cmd *string) {
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	defer session.Close()

	_, err = session.CombinedOutput(*cmd)
	if err != nil {
		log.Fatalf("Failed to execute command: %s", err)
	}
	fmt.Println("Command executed successfully")
}

func createSSHAuthMethod(privateKey string) ssh.AuthMethod {
	key, err := os.ReadFile(privateKey)
	if err != nil {
		log.Fatalf("Unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("Unable to parse private key: %v", err)
	}

	return ssh.PublicKeys(signer)
}

func pullImage(client *client.Client, imageName string) error {
	ctx := context.Background()
	out, err := client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	return nil
}

func waitForImage(cli *client.Client, imageName string) error {
	ctx := context.Background()
	events, err := cli.Events(ctx, types.EventsOptions{})
	timeout := time.After(10 * time.Second) // Set your desired timeout

	for {
		select {
		case event := <-events:
			fmt.Println(event)
			if event.Type == "image" && event.Action == "pull" && event.Actor.Attributes["name"] == imageName && event.Status == "downloaded" {
				return nil
			}
		case e := <-err:
			log.Fatal(e)
		case <-timeout:
			return fmt.Errorf("timeout waiting for image %s to be fully downloaded", imageName)
		}
	}
}

func createContainer(client *client.Client, imageName, containerName string) error {
	ctx := context.Background()
	containerConfig := &container.Config{
		Image: imageName,
		// Add other configurations if needed
	}
	hostConfig := &container.HostConfig{
		// Add host configurations if needed
	}
	networkConfig := &network.NetworkingConfig{
		// Add networking config
	}
	resp, err := client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return err
	}
	if err := client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	return nil
}
