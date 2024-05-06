package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/regclient/regclient/config"
)

func CreateDockerClient(username string, password string, registryName string) (*client.Client, image.PushOptions, error) {
	if username != "" && password != "" {
		authConfig := registry.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: registryName,
		}
		return CreateClientWithAuthConfig(authConfig)
	} else {
		hosts, _ := config.DockerLoad()
		for _, h := range hosts {
			if strings.Contains(h.Name, registryName) || (registryName == "" && h.Name == config.DockerRegistry) {
				// At this point, we can configure the auth
				authConfig := registry.AuthConfig{
					Username:      h.User,
					Password:      h.Pass,
					ServerAddress: h.Hostname,
					IdentityToken: h.Token,
				}
				return CreateClientWithAuthConfig(authConfig)
			}
		}

		// At this point, there is no auth and we can return an empty client
		// Create a new docker client
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

		if err != nil {
			fmt.Print("ERROR: An error occurred when creating the docker client. Please ensure that the docker daemon is running and that you have the correct permissions to access it.\n")
			return nil, image.PushOptions{}, err
		}

		// Print the version of the docker client
		fmt.Printf("Docker client version: %s\n", cli.ClientVersion())
		registryPushOptions := image.PushOptions{
			All: true,
		}

		return cli, registryPushOptions, nil

	}
}

func CreateClientWithAuthConfig(authConfig registry.AuthConfig) (*client.Client, image.PushOptions, error) {
	// Create a new docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		fmt.Print("ERROR: An error occurred when creating the docker client. Please ensure that the docker daemon is running and that you have the correct permissions to access it.\n")
		return nil, image.PushOptions{}, err
	}

	// Print the version of the docker client
	fmt.Printf("Docker client version: %s\n", cli.ClientVersion())
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return nil, image.PushOptions{}, err
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	registryPushOptions := image.PushOptions{
		RegistryAuth: authStr,
	}

	return cli, registryPushOptions, nil
}
