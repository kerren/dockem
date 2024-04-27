package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"

	"encoding/json"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"golang.org/x/mod/sumdb/dirhash"
)

type Version struct {
	Version string `json:"version"`
}

func ParseVersionFileJson(jsonData []byte) (*Version, error) {
	var version Version
	err := json.Unmarshal(jsonData, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func BuildDockerImage(params BuildDockerImageParams) error {
	// I create a string that I append all of the hashes to
	overallHash := ""

	// This is the function that I'll use to open a SINGLE file
	osOpen := func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	}

	// Hash the watch files if they exist
	if len(params.WatchFile) > 0 {
		// Note: No need to sort the files as they are sorted in the Hash1 function
		watchFileHash, err := dirhash.Hash1(params.WatchFile, osOpen)
		if err != nil {
			print("ERROR: An error ocurred when hashing the watch files, please ensure they all exist, they are listed as follows:\n")
			for _, file := range params.WatchFile {
				print(file + "\n")
			}
			return err
		}
		overallHash += watchFileHash
	}

	// Hash the watch directories if they exist
	if len(params.WatchDirectory) > 0 {
		sort.Strings(params.WatchDirectory)

		for _, directory := range params.WatchDirectory {
			directoryHash, err := dirhash.HashDir(directory, "", dirhash.Hash1)
			if err != nil {
				print("ERROR: An error ocurred when hashing the watch directories, please ensure they all exist, they are listed as follows:\n")
				for _, dir := range params.WatchDirectory {
					print(dir + "\n")
				}
				return err
			}
			overallHash += directoryHash
		}
	}

	// Hash the build directory
	directoryHash, err := dirhash.HashDir(params.Directory, "", dirhash.Hash1)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the build directory, please ensure it exists and is not empty. You specified %s as the directory\n", params.Directory))
		return err
	}
	overallHash += directoryHash

	// Hash the Dockerfile
	dockerfileHash, err := dirhash.Hash1([]string{params.DockerfilePath}, osOpen)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the Dockerfile, please ensure it exists. You specified %s as the Dockerfile\n", params.DockerfilePath))
		return err
	}
	overallHash += dockerfileHash

	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(overallHash))
	imageHash := fmt.Sprintf("%x", sha256Hash.Sum(nil))

	// Now we need to open the version file (JSON file) and pull out the "version" key

	versionFile, versionFileError := os.Open(params.VersionFile)
	defer versionFile.Close()
	if versionFileError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to open the version file: %s\n", params.VersionFile))
		return versionFileError
	}
	bytes, _ := io.ReadAll(versionFile)
	parsedVersionFile, parsedVersionFileError := ParseVersionFileJson(bytes)
	if parsedVersionFileError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to parse the version file: %s\n", params.VersionFile))
		return parsedVersionFileError
	}
	version := "v" + parsedVersionFile.Version
	print(fmt.Sprintf("The version of the image being built is: %s\n", version))

	// Now that we have the hash, we can check if this hash exist on the docker registry already.
	// For this, we'll need regclient because it allows us to interact with the registry instead
	// of just the docker daemon. https://github.com/regclient/regclient

	host := *config.HostNew()
	if params.Registry != "" {
		host.Hostname = params.Registry
	}
	if params.DockerUsername != "" {
		host.User = params.DockerUsername
	}
	if params.DockerPassword != "" {
		host.Pass = params.DockerPassword
	}

	client := regclient.New(regclient.WithConfigHost(host))

	imageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
	r, err := ref.New(imageName)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to parse the image: %s\n", imageName))
		return err
	}

	mOpts := []regclient.ManifestOpts{}
	print(fmt.Sprintf("Checking for the image hash %s on the registry\n", imageHash))
	_, manifestError := client.ManifestHead(context.Background(), r, mOpts...)
	exists := true
	if manifestError != nil {
		print(fmt.Sprintf("The image hash %s does not exist on the registry\n", imageHash))
		exists = false
	}

	if exists {
		print(fmt.Sprintf("The image hash %s already exists on the registry, we can now copy this to the other tags!\n", imageHash))
		// If the image already exists, we just need to copy the tags across
		for _, tag := range params.Tag {
			versionTag := fmt.Sprintf("%s-%s", tag, version)
			targetImageName := GenerateDockerImageName(params.Registry, params.ImageName, versionTag)
			print(fmt.Sprintf("Copying the image to the new tag: %s\n", targetImageName))
			copyError := CopyDockerImage(client, imageName, targetImageName)
			if copyError != nil {
				return copyError
			}
		}
		if len(params.Tag) == 0 && !params.Latest && !params.MainVersion {
			// At this point, we just deploy it straight to the main version
			mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
			print(fmt.Sprintf("WARN: No tags were specified and you have not selected the --latest flag, so the image will be deployed to the main version: %s\n", mainVersionImageName))
			copyError := CopyDockerImage(client, imageName, mainVersionImageName)
			if copyError != nil {
				return copyError
			}
		}
		if params.Latest {
			latestImageName := GenerateDockerImageName(params.Registry, params.ImageName, "latest")
			print(fmt.Sprintf("You have selected the --latest flag, so the image will be deployed to the latest tag: %s\n", latestImageName))
			copyError := CopyDockerImage(client, imageName, latestImageName)
			if copyError != nil {
				return copyError
			}
		}
		if params.MainVersion {
			mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
			print(fmt.Sprintf("You have selected the --main-version flag, so the image will be deployed to the main version: %s\n", mainVersionImageName))
			copyError := CopyDockerImage(client, imageName, mainVersionImageName)
			if copyError != nil {
				return copyError
			}
		}
	} else {
		// We need to build the image and then we push it to the registry
		print(fmt.Sprintf("The image hash %s does not exist on the registry, we will now build the image and push it to the registry\n", imageHash))

	}

	return nil
}
