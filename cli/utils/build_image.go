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

// BuildImage builds a Docker image using the provided build context tarball. It
// will name the image local:imageHash. The excludePatterns are resolved once in
// BuildDockerImage and threaded through to TarBuildContext so that the files
// streamed to the daemon are exactly the files that fed the hash.
func BuildImage(params BuildDockerImageParams, imageHash string, excludePatterns []string, dockerClient *client.Client, buildLog *BuildLog) (string, error) {

	reader, relativeDockerfilePath, readerErr := TarBuildContext(params, excludePatterns, dockerClient, buildLog)
	if readerErr != nil {
		return "", readerErr
	}

	localTag := fmt.Sprintf("local:%s", imageHash)

	imageBuildResult, imageBuildError := dockerClient.ImageBuild(context.Background(), reader, types.ImageBuildOptions{
		Dockerfile: relativeDockerfilePath,
		Tags:       []string{localTag},
	})

	if imageBuildError != nil {
		LogError("An error ocurred when trying to build the image: %s\n", imageBuildError)
		return "", imageBuildError
	}
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(imageBuildResult.Body, os.Stderr, termFd, isTerm, nil)
	imageBuildResult.Body.Close()

	return localTag, nil
}
