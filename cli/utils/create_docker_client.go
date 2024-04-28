package utils

import (
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

func CreateDockerClient(username string, password string, registryName string) (*client.Client, image.PushOptions, error) {
	// Create a new docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		print("ERROR: An error occurred when creating the docker client. Please ensure that the docker daemon is running and that you have the correct permissions to access it.\n")
		return nil, image.PushOptions{}, err
	}

	// Print the version of the docker client
	cli.ClientVersion()

	if username != "" && password != "" {
		// See https://docs.docker.com/engine/api/sdk/examples/#pull-an-image-with-authentication for details
		authConfig := registry.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: registryName,
		}
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

	return cli, image.PushOptions{}, nil
}
