package main

import (
	mydocker "dasior/cloudservers/docker"
	myec2 "dasior/cloudservers/ec2"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/joho/godotenv"
)

type MinecraftServerConfig struct {
	Name         string
	InstanceType string
	InitMemory   string
	MaxMemory    string
}

var AWS_KEY_PAIR_NAME = "cloudservers"
var AWS_IMAGE_ID = "ami-0c101f26f147fa7fd"

// Configurations for small, medium, and large Minecraft servers
var (
	MINECRAFT_PLAN_SMALL = MinecraftServerConfig{
		Name:         "small",
		InstanceType: "t3.small",
		InitMemory:   "1G",
		MaxMemory:    "1G",
	}
	MINECRAFT_PLAN_MEDIUM = MinecraftServerConfig{
		Name:         "medium",
		InstanceType: "t3.medium",
		InitMemory:   "2G",
		MaxMemory:    "3G",
	}
	MINECRAFT_PLAN_LARGE = MinecraftServerConfig{
		Name:         "large",
		InstanceType: "t3.large",
		InitMemory:   "2G",
		MaxMemory:    "7G",
	}
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	NewMinecraftServer(&MINECRAFT_PLAN_SMALL)
}

func NewMinecraftServer(plan *MinecraftServerConfig) {
	ec2s := myec2.NewEC2Service()

	fmt.Printf("Creating instance... (%s)\n", plan.Name)

	instance, err := ec2s.CreateInstance(&ec2.RunInstancesInput{
		ImageId:      aws.String(AWS_IMAGE_ID),
		InstanceType: aws.String(plan.InstanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      &AWS_KEY_PAIR_NAME,
	})
	if err != nil {
		fmt.Println("Could not create instance: ", err)
		return
	}
	defer ec2s.DeleteInstance(*instance.InstanceId)

	fmt.Println("Instance created: ", *instance.InstanceId)
	fmt.Println("Setting up instance...")

	setupScript := "./install-configure-docker.sh"
	err = ExecuteScriptThroughSSHRealTimeRetrying(instance.PublicIpAddress, &setupScript, 30)
	if err != nil {
		fmt.Println("Failed to execute script through ssh: ", err)
		return
	}

	fmt.Println("Creating container...")

	dockerService, err := mydocker.NewDockerService(*instance.PublicIpAddress)
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
				"INIT_MEMORY=" + plan.InitMemory,
				"MAX_MEMORY=" + plan.MaxMemory,
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
		return
	}

	fmt.Println("Created container: ", container.ID, container.Name)

	err = dockerService.Start()
	if err != nil {
		fmt.Println("Could not start container: ", container.ID)
		return
	}

	fmt.Println("Container started.")

	err = dockerService.StreamLogs()
	if err != nil {
		fmt.Println("Failed to stream logs: ", err)
		return
	}
}

func ExecuteScriptThroughSSH(publicIP *string, scriptPath *string) (string, error) {
	cmd := exec.Command("ssh", "-o StrictHostKeyChecking=no", fmt.Sprintf("ec2-user@%s", *publicIP), "bash", "-s")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	// Open script file
	scriptFile, err := os.Open(*scriptPath)
	if err != nil {
		return "", err
	}
	defer scriptFile.Close()

	// Write script content to stdin
	_, err = io.Copy(stdin, scriptFile)
	if err != nil {
		return "", err
	}
	stdin.Close()

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func ExecuteScriptThroughSSHRealTime(publicIP *string, scriptPath *string) error {
	// Open script file
	scriptFile, err := os.Open(*scriptPath)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	// Create SSH command
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("ec2-user@%s", *publicIP), "bash", "-s")

	// Set command's stdin to script file
	cmd.Stdin = scriptFile

	// Redirect command's stdout and stderr to current process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command asynchronously
	if err := cmd.Start(); err != nil {
		return err
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func ExecuteScriptThroughSSHRealTimeRetrying(publicIP *string, scriptPath *string, maxAttempts int) error {
	// Open script file
	scriptFile, err := os.Open(*scriptPath)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	// Attempt to establish SSH connection
	var sshCmd *exec.Cmd
	for attempt := 0; attempt < maxAttempts; attempt++ {
		fmt.Printf("Attempting SSH connection (Attempt %d/%d)...\n", attempt+1, maxAttempts)
		sshCmd = exec.Command("ssh", "-v", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("ec2-user@%s", *publicIP), "bash", "-s")

		// Set command's stdin to script file
		sshCmd.Stdin = scriptFile

		// Redirect command's stdout and stderr to current process
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		// Start the command asynchronously
		if err := sshCmd.Start(); err != nil {
			fmt.Printf("Failed to start SSH connection: %v\n", err)
			time.Sleep(time.Second) // Wait before retrying
			continue
		}

		// Wait for the command to finish
		if err := sshCmd.Wait(); err != nil {
			fmt.Printf("SSH connection failed: %v\n", err)
			time.Sleep(time.Second) // Wait before retrying
			continue
		}

		// SSH connection succeeded
		fmt.Println("SSH connection established successfully.")
		return nil
	}

	// All attempts failed
	return fmt.Errorf("unable to establish SSH connection after %d attempts", maxAttempts)
}
