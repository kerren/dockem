package utils

import (
	"fmt"

	"github.com/regclient/regclient/types/ref"
	"golang.org/x/mod/sumdb/dirhash"
)

func BuildDockerImage(params BuildDockerImageParams) error {
	// I create a string that I append all of the hashes to
	overallHash := ""

	// Hash the watch files if they exist
	hashWatchFileResult, hashWatchFileError := HashWatchFiles(params.WatchFile)
	if hashWatchFileError != nil {
		return hashWatchFileError
	}
	overallHash += hashWatchFileResult

	// Hash the watch directories if they exist
	watchDirectoriesHash, watchDirectoriesHashError := HashWatchDirectories(params.WatchDirectory)
	if watchDirectoriesHashError != nil {
		return watchDirectoriesHashError
	}
	overallHash += watchDirectoriesHash

	// Hash the build directory if the ignore flag has not been specified
	if !params.IgnoreBuildDirectory {
		directoryHash, err := dirhash.HashDir(params.Directory, "", dirhash.Hash1)
		if err != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when hashing the build directory, please ensure it exists and is not empty. You specified %s as the directory\n", params.Directory))
			return err
		}
		overallHash += directoryHash
	}

	// Hash the Dockerfile
	dockerfileHash, err := dirhash.Hash1([]string{params.DockerfilePath}, osOpen)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the Dockerfile, please ensure it exists. You specified %s as the Dockerfile\n", params.DockerfilePath))
		return err
	}
	overallHash += dockerfileHash

	// We now have the hash of all of the different files combined into one (unique) string. We
	// can now hash this string to create a unique hash for the image.
	imageHash := HashString(overallHash)

	// Now we need to open the version file (JSON file) and pull out the "version" key
	version, versionError := ExtractVersion(params.VersionFile)
	if versionError != nil {
		return versionError
	}

	// Now that we have the hash, we can check if this hash exists on the docker registry already.
	// For this, we'll need regclient because it allows us to interact with the registry instead
	// of just the docker daemon. https://github.com/regclient/regclient
	client := CreateRegclientClient(params.Registry, params.DockerUsername, params.DockerPassword)

	// Now we create the image name of the image that should exist on the registry if it has
	// been built before. This would look like this:
	//
	// 		org/image-name:hash
	//
	imageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
	r, err := ref.New(imageName)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to parse the image: %s\n", imageName))
		return err
	}

	// Now we do a HEAD request to see if the image exists on the registry already. This is
	// really good for registries that have a limit on image pulls per day.
	exists := CheckManifestHead(imageHash, r, client)

	if exists {
		print(fmt.Sprintf("The image hash %s already exists on the registry, we can now copy this to the other tags!\n", imageHash))
		// If the image already exists, we just need to copy the tags across
		copyError := CopyExistingImageTag(params, version, imageName, client)
		if copyError != nil {
			return copyError
		}
	} else {
		// We need to build the image and then we push it to the registry
		print(fmt.Sprintf("The image hash %s does not exist on the registry, we will now build the image and push it to the registry\n", imageHash))

		dockerClient, pushOptions, err := CreateDockerClient(params.DockerUsername, params.DockerPassword, params.Registry)
		if err != nil {
			return err
		}
		defer dockerClient.Close()

		// Build the image
		localTag, dockerImageBuildError := BuildImage(params, imageHash, dockerClient)

		if dockerImageBuildError != nil {
			return dockerImageBuildError
		}

		print("Docker build complete, pushing the image to the registry\n")

		// Now we push the hashed image and then all of the other tags that the
		// user has specified
		hashedImageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
		hashedImageNameError := TagAndPushImage(localTag, hashedImageName, dockerClient, pushOptions)
		if hashedImageNameError != nil {
			return hashedImageNameError
		}

		print(fmt.Sprintf("The image has been pushed to the registry with the hash %s\n", hashedImageName))

		// Now that the hashed image has been pushed, we can push all of the other tags
		tagAndPushImagesError := TagAndPushNewImages(params, version, localTag, dockerClient, pushOptions)
		if tagAndPushImagesError != nil {
			return tagAndPushImagesError
		}
	}

	return nil
}