package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func BuildImage(params BuildDockerImageParams, imageHash string, dockerClient *client.Client) (string, error) {

	absDirectoryPath, absDirectoryPathError := filepath.Abs(params.Directory)
	if absDirectoryPathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the absolute path of the directory, please check your entry for %s\n", params.Directory))
		return "", absDirectoryPathError
	}

	reader, tarErr := archive.TarWithOptions(absDirectoryPath, &archive.TarOptions{})

	if tarErr != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to archive the directory to send to the builder: %s\n", tarErr))
		return "", tarErr
	}

	localTag := fmt.Sprintf("local:%s", imageHash)

	absDockerfilePath, absDockerfilePathError := filepath.Abs(params.DockerfilePath)
	if absDockerfilePathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the absolute path of the Dockerfile, please check your entry for %s\n", params.DockerfilePath))
		return "", absDockerfilePathError
	}

	relativeDockerfilePath, relativeDockerfilePathError := filepath.Rel(absDirectoryPath, absDockerfilePath)
	if relativeDockerfilePathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the relative path of the Dockerfile from the build directory, please check your entry for:\nDirectory %s\nDockerfile %s", absDirectoryPath, absDockerfilePath))
		return "", relativeDockerfilePathError
	}

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
