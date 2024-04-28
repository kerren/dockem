package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func BuildImage(params BuildDockerImageParams, imageHash string, dockerClient *client.Client, buildLog *BuildLog) (string, error) {

	reader, relativeDockerfilePath, readerErr := TarBuildContext(params, dockerClient, buildLog)
	if readerErr != nil {
		return "", readerErr
	}

	localTag := fmt.Sprintf("local:%s", imageHash)

	imageBuildResult, imageBuildError := dockerClient.ImageBuild(context.Background(), reader, types.ImageBuildOptions{
		Dockerfile: relativeDockerfilePath,
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
