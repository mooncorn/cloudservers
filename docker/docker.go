package docker

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerService struct {
	client *client.Client
}

func NewDockerService(ip string) (*DockerService, error) {
	cmd := exec.Command("ssh", "-L", "2375:localhost:2375", "ec2-user@"+ip)

	_, err := cmd.Output()
	if err != nil {
		return &DockerService{}, err
	}

	cli, err := client.NewClientWithOpts(client.WithHost("tcp://localhost:2375"))
	if err != nil {
		return &DockerService{}, err
	}

	return &DockerService{
		client: cli,
	}, nil
}

func (ds *DockerService) GetContainer() (types.ContainerJSON, error) {
	ctx := context.Background()

	// Get a list of all containers
	containers, err := ds.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return types.ContainerJSON{}, err
	}

	// Check if there are any containers
	if len(containers) == 0 {
		return types.ContainerJSON{}, errors.New("no containers found")
	}

	// Get detailed information about the first container
	containerJSON, err := ds.client.ContainerInspect(ctx, containers[0].ID)
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return containerJSON, nil
}

func (ds *DockerService) CreateContainer(containerConfig *container.Config, hostConfig *container.HostConfig) (types.ContainerJSON, error) {
	ctx := context.Background()

	// Check if there is already a container running
	containers, err := ds.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return types.ContainerJSON{}, err
	}

	if len(containers) > 0 {
		return types.ContainerJSON{}, errors.New("maximum number of containers exceeded")
	}

	_, err = ds.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return types.ContainerJSON{}, err
	}

	containerJSON, err := ds.GetContainer()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return containerJSON, nil
}

func (ds *DockerService) UpdateContainerEnv(newEnv map[string]string) (types.ContainerJSON, error) {
	ctx := context.Background()

	container, err := ds.GetContainer()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	// Get the current container's configuration
	containerInfo, _, err := ds.client.ContainerInspectWithRaw(ctx, container.ID, false)
	if err != nil {
		return types.ContainerJSON{}, err
	}

	// Update only the environment variables in the container's configuration
	containerInfo.Config.Env = mergeEnv(containerInfo.Config.Env, newEnv)

	_, err = ds.RemoveContainer()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	updatedContainer, err := ds.CreateContainer(containerInfo.Config, containerInfo.HostConfig)
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return updatedContainer, nil
}

func (ds *DockerService) UpdateContainerResources(updateConfig container.UpdateConfig) error {
	return errors.New("function not implemented")
}

func (ds *DockerService) RemoveContainer() (types.ContainerJSON, error) {
	ctx := context.Background()

	container, err := ds.GetContainer()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	err = ds.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return container, nil
}

func mergeEnv(existingEnv []string, newEnv map[string]string) []string {
	// Convert existing environment variables to map for easy lookup
	existingEnvMap := make(map[string]string)
	for _, env := range existingEnv {
		key, value := splitEnv(env)
		existingEnvMap[key] = value
	}

	// Update existing environment variables with new values
	for key, value := range newEnv {
		existingEnvMap[key] = value
	}

	// Convert back to slice of environment variables
	updatedEnv := make([]string, 0, len(existingEnvMap))
	for key, value := range existingEnvMap {
		updatedEnv = append(updatedEnv, key+"="+value)
	}

	return updatedEnv
}

func splitEnv(env string) (string, string) {
	pair := strings.SplitN(env, "=", 2)
	if len(pair) == 2 {
		return pair[0], pair[1]
	}
	return "", ""
}

func (ds *DockerService) CloseDockerClient() {
	if ds.client != nil {
		ds.client.Close()
	}
}
