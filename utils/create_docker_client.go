package utils

import (
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func CreateDockerClient(username string, password string, registry string) (*client.Client, error) {
	// Create a new docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)

	if err != nil {
		print("ERROR: An error occurred when creating the docker client. Please ensure that the docker daemon is running and that you have the correct permissions to access it.\n")
		return nil, err
	}

	if username != "" && password != "" {
		authConfig := types.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: registry,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return nil, err
		}

		authStr := base64.URLEncoding.EncodeToString(encodedJSON)

		cli.RegistryAuth = authStr
	}
	// Print the version of the docker client
	cli.ClientVersion()

	return cli, nil
}
