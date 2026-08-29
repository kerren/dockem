package utils

import (
	"context"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

// TagAndPushImage Tags and pushes a Docker image to a registry.
// It will print updates to the console and wait for the stream to complete or fail.
func TagAndPushImage(fromImage string, toImage string, dockerClient *client.Client, pushOptions types.ImagePushOptions) error {

	tagErr := dockerClient.ImageTag(context.Background(), fromImage, toImage)
	if tagErr != nil {
		LogError("An error ocurred when trying to tag the image from %s to %s\n", fromImage, toImage)
		return tagErr
	}

	reader, pushError := dockerClient.ImagePush(context.Background(), toImage, pushOptions)
	if pushError != nil {
		LogError("An error ocurred when trying to push the image: %s\n", toImage)
		return pushError
	}
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)
	defer reader.Close()
	return nil
}
