package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

func TarBuildContext(params BuildDockerImageParams, dockerClient *client.Client) (io.Reader, string, error) {

	absDirectoryPath, absDirectoryPathError := filepath.Abs(params.Directory)
	if absDirectoryPathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the absolute path of the directory, please check your entry for %s\n", params.Directory))
		return nil, "", absDirectoryPathError
	}
	absDockerfilePath, absDockerfilePathError := filepath.Abs(params.DockerfilePath)
	if absDockerfilePathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the absolute path of the Dockerfile, please check your entry for %s\n", params.DockerfilePath))
		return nil, "", absDockerfilePathError
	}
	relativeDockerfilePath, relativeDockerfilePathError := filepath.Rel(absDirectoryPath, absDockerfilePath)
	if relativeDockerfilePathError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to get the relative path of the Dockerfile from the build directory, please check your entry for:\nDirectory %s\nDockerfile %s", absDirectoryPath, absDockerfilePath))
		return nil, "", relativeDockerfilePathError
	}
	var tempDockerfileRef *os.File
	// Check if the dockerfile is in the directory or in a child folder
	if strings.HasPrefix(relativeDockerfilePath, "../") {
		// We'll make a temporary Dockerfile in the directory and copy the contents
		// of the orignal into here
		tempDockerfile, tempDockerfileError := os.CreateTemp(absDirectoryPath, "Dockerfile.")
		if tempDockerfileError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to create a temporary Dockerfile for the build, please check that the directory is writable %s\n", absDirectoryPath))
			return nil, "", tempDockerfileError
		}
		tempDockerfileRef = tempDockerfile
		defer tempDockerfile.Close()
		dockerfileContent, dockerfileContentError := os.ReadFile(absDockerfilePath)
		if dockerfileContentError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to read the Dockerfile, please check your entry for %s\n", absDockerfilePath))
			return nil, "", dockerfileContentError
		}
		_, writeError := tempDockerfile.Write(dockerfileContent)
		if writeError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to write the Dockerfile to a temporary one for the build, please check that the directory is writable %s\n", absDirectoryPath))
			return nil, "", writeError
		}
		relativeDockerfilePath, relativeDockerfilePathError = filepath.Rel(absDirectoryPath, tempDockerfile.Name())
		if relativeDockerfilePathError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to get the relative path of the Dockerfile from the build directory, please check your entry for:\nDirectory %s\nDockerfile %s", absDirectoryPath, tempDockerfile.Name()))
			return nil, "", relativeDockerfilePathError
		}
	}

	reader, tarErr := archive.TarWithOptions(absDirectoryPath, &archive.TarOptions{})
	if tarErr != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to archive the directory to send to the builder: %s\n", tarErr))
		return nil, "", tarErr
	}

	// Right now, we have to read the tar stream. The problem is that if the Dockerfile
	// is outside of the build context, I can't remove it yet. This would mean that the
	// docker build command would need to finish before I close it. And I expect people
	// would kill the process using Ctrl-C before it finishes. That would leave the
	// temporary Dockerfile in the directory. So I'm going to stream the tar into a new
	// reader and then delete the file before going to the build process.

	outputReader := reader
	if tempDockerfileRef != nil {
		// Now we read everything and put it into a new stream so that we can delete
		// the temporary file before going to the build process.
		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to read the tar stream: %s\n", readErr))
			return nil, "", readErr
		}
		outputReader = io.NopCloser(bytes.NewReader(data))
		os.Remove(tempDockerfileRef.Name())
	}

	return outputReader, relativeDockerfilePath, nil

}
