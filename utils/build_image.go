package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func BuildImage(params BuildDockerImageParams, imageHash string, dockerClient *client.Client) (string, error) {

	reader, tarErr := archive.TarWithOptions(params.Directory, &archive.TarOptions{})

	if tarErr != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to archive the directory to send to the builder: %s\n", tarErr))
		return "", tarErr
	}

	localTag := fmt.Sprintf("local:%s", imageHash)

	imageBuildResult, imageBuildError := dockerClient.ImageBuild(context.Background(), reader, types.ImageBuildOptions{
		Dockerfile: params.DockerfilePath,
		Tags:       []string{localTag},
	})

	if imageBuildError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to build the image: %s\n", imageBuildError))
		return "", imageBuildError
	}
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(imageBuildResult.Body, os.Stderr, termFd, isTerm, nil)
	imageBuildResult.Body.Close()

	return localTag, nil
}
