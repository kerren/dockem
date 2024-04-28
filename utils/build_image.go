package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// Check if the dockerfile is in the directory or in a child folder
	if strings.HasPrefix(relativeDockerfilePath, "../") {
		// We'll make a temporary Dockerfile in the directory and copy the contents
		// of the orignal into here
		tempDockerfile, tempDockerfileError := os.CreateTemp(absDirectoryPath, "Dockerfile.")
		if tempDockerfileError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to create a temporary Dockerfile for the build, please check that the directory is writable %s\n", absDirectoryPath))
			return "", tempDockerfileError
		}
		dockerfileContent, dockerfileContentError := os.ReadFile(absDockerfilePath)
		if dockerfileContentError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to read the Dockerfile, please check your entry for %s\n", absDockerfilePath))
			return "", dockerfileContentError
		}
		_, writeError := tempDockerfile.Write(dockerfileContent)
		if writeError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to write the Dockerfile to a temporary one for the build, please check that the directory is writable %s\n", absDirectoryPath))
			return "", writeError
		}
		relativeDockerfilePath, relativeDockerfilePathError = filepath.Rel(absDirectoryPath, tempDockerfile.Name())
		if relativeDockerfilePathError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to get the relative path of the Dockerfile from the build directory, please check your entry for:\nDirectory %s\nDockerfile %s", absDirectoryPath, tempDockerfile.Name()))
			return "", relativeDockerfilePathError
		}
		defer tempDockerfile.Close()
	}

	reader, tarErr := archive.TarWithOptions(absDirectoryPath, &archive.TarOptions{})
	if tarErr != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to archive the directory to send to the builder: %s\n", tarErr))
		return "", tarErr
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
