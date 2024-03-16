package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
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

	publicIP := "54.162.183.229"
	sshIntoInstance(&publicIP)

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

func createInstance(region *string) (*ec2.Reservation, error) {
	// Create a new AWS session using default credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(*region),
	},
	)
	if err != nil {
		fmt.Println(err)
		return &ec2.Reservation{}, errors.New("failed to create aws session")
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
		return &ec2.Reservation{}, errors.New("failed to launch instance")
	}

	return result, nil
}

func sshIntoInstance(publicIP *string) {
	// Read the private key file
	keyPath := ".ssh/cloudservers.pem"
	key, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to read private key file: %v", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// SSH connection parameters
	sshConfig := &ssh.ClientConfig{
		User: "ec2-user", // SSH username
		Auth: []ssh.AuthMethod{
			// Use the parsed private key for authentication
			ssh.PublicKeys(signer),
		},
		// Optionally, you can provide HostKeyCallback to verify the server's host key
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the EC2 instance
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", *publicIP, 22), sshConfig)
	if err != nil {
		log.Fatalf("Failed to connect to EC2 instance: %v", err)
	}
	defer conn.Close()

	// Create a new SSH session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Execute a command on the remote instance
	output, err := session.CombinedOutput("ls -l")
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	// Print the output of the command
	fmt.Println("Output of 'ls -l' command:")
	fmt.Println(string(output))
}
