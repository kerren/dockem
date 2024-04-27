package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
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
	if versionFileError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to open the version file: %s\n", params.VersionFile))
		return versionFileError
	}
	defer versionFile.Close()
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

	host := config.Host{}
	customHost := false
	if params.Registry != "" {
		host.Hostname = params.Registry
		host.Name = params.Registry
		customHost = true
	}
	if params.DockerUsername != "" {
		host.User = params.DockerUsername
		host.RepoAuth = true
		customHost = true
	}
	if params.DockerPassword != "" {
		host.Pass = params.DockerPassword
		host.RepoAuth = true
		customHost = true
	}

	var regclientOpts []regclient.Opt

	if customHost {
		regclientOpts = []regclient.Opt{
			regclient.WithConfigHost(host),
			regclient.WithDockerCreds(),
		}

	} else {
		regclientOpts = []regclient.Opt{}
	}

	client := regclient.New(regclientOpts...)

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
		print(fmt.Sprintf("The image hash %s does not exist on the registry or we were unable to pull it\n", imageHash))
		if strings.Contains(manifestError.Error(), "failed to request manifest head") {
			print("WARN: Unable to pull the details from the registry, please ensure you have the correct credentials.\n")
			print("WARN: The build will continue, but this should be investigated\n")
		}
		print(manifestError.Error())
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
			print(fmt.Sprintf("WARN: No tags were specified and you have not selected the --latest flag, so the image will be copied to the main version: %s\n", mainVersionImageName))
			copyError := CopyDockerImage(client, imageName, mainVersionImageName)
			if copyError != nil {
				return copyError
			}
		}
		if params.Latest {
			latestImageName := GenerateDockerImageName(params.Registry, params.ImageName, "latest")
			print(fmt.Sprintf("You have selected the --latest flag, so the image will be copied to the latest tag: %s\n", latestImageName))
			copyError := CopyDockerImage(client, imageName, latestImageName)
			if copyError != nil {
				return copyError
			}
		}
		if params.MainVersion {
			mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
			print(fmt.Sprintf("You have selected the --main-version flag, so the image will be copied to the main version: %s\n", mainVersionImageName))
			copyError := CopyDockerImage(client, imageName, mainVersionImageName)
			if copyError != nil {
				return copyError
			}
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
		reader, tarErr := archive.TarWithOptions(params.Directory, &archive.TarOptions{})

		if tarErr != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to archive the directory to send to the builder: %s\n", tarErr))
			return tarErr
		}

		localTag := fmt.Sprintf("local:%s", imageHash)

		imageBuildResult, imageBuildError := dockerClient.ImageBuild(context.Background(), reader, types.ImageBuildOptions{
			Dockerfile: params.DockerfilePath,
			Tags:       []string{localTag},
		})

		if imageBuildError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to build the image: %s\n", imageBuildError))
			return imageBuildError
		}
		termFd, isTerm := term.GetFdInfo(os.Stderr)
		jsonmessage.DisplayJSONMessagesStream(imageBuildResult.Body, os.Stderr, termFd, isTerm, nil)
		imageBuildResult.Body.Close()

		print("Docker build complete, pushing the image to the registry\n")

		// First we push the hashed image

		hashedImageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
		tagErr := dockerClient.ImageTag(context.Background(), localTag, hashedImageName)
		if tagErr != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to tag the image hash: %s\n", tagErr))
			return tagErr
		}

		hashedReader, pushError := dockerClient.ImagePush(context.Background(), hashedImageName, pushOptions)
		if pushError != nil {
			print(fmt.Sprintf("ERROR: An error ocurred when trying to push the image hash: %s\n", pushError))
			return pushError
		}
		jsonmessage.DisplayJSONMessagesStream(hashedReader, os.Stderr, termFd, isTerm, nil)
		defer hashedReader.Close()
		print(fmt.Sprintf("The image has been pushed to the registry with the hash %s\n", hashedImageName))

		for _, tag := range params.Tag {
			versionTag := fmt.Sprintf("%s-%s", tag, version)
			targetImageName := GenerateDockerImageName(params.Registry, params.ImageName, versionTag)
			print(fmt.Sprintf("Pushing the image to the new tag: %s\n", targetImageName))
			dockerClient.ImageTag(context.Background(), localTag, targetImageName)
			tagReader, pushError := dockerClient.ImagePush(context.Background(), targetImageName, pushOptions)
			if pushError != nil {
				print(fmt.Sprintf("ERROR: An error ocurred when trying to push the image: %s\n", pushError))
				return pushError
			}
			jsonmessage.DisplayJSONMessagesStream(tagReader, os.Stderr, termFd, isTerm, nil)
			defer tagReader.Close()
		}
		if len(params.Tag) == 0 && !params.Latest && !params.MainVersion {
			// At this point, we just deploy it straight to the main version
			mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
			print(fmt.Sprintf("WARN: No tags were specified and you have not selected the --latest flag, so the image will be deployed to the main version: %s\n", mainVersionImageName))
			dockerClient.ImageTag(context.Background(), localTag, mainVersionImageName)
			warnReader, pushError := dockerClient.ImagePush(context.Background(), mainVersionImageName, pushOptions)
			if pushError != nil {
				print(fmt.Sprintf("ERROR: An error ocurred when trying to push the image: %s\n", pushError))
				return pushError
			}
			jsonmessage.DisplayJSONMessagesStream(warnReader, os.Stderr, termFd, isTerm, nil)
			defer warnReader.Close()
		}
		if params.Latest {
			latestImageName := GenerateDockerImageName(params.Registry, params.ImageName, "latest")
			print(fmt.Sprintf("You have selected the --latest flag, so the image will be deployed to the latest tag: %s\n", latestImageName))
			dockerClient.ImageTag(context.Background(), localTag, latestImageName)
			latestReader, pushError := dockerClient.ImagePush(context.Background(), latestImageName, pushOptions)
			if pushError != nil {
				print(fmt.Sprintf("ERROR: An error ocurred when trying to push the image: %s\n", pushError))
				return pushError
			}
			jsonmessage.DisplayJSONMessagesStream(latestReader, os.Stderr, termFd, isTerm, nil)
			defer latestReader.Close()
		}
		if params.MainVersion {
			mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
			print(fmt.Sprintf("You have selected the --main-version flag, so the image will be deployed to the main version: %s\n", mainVersionImageName))
			dockerClient.ImageTag(context.Background(), localTag, mainVersionImageName)
			mainReader, pushError := dockerClient.ImagePush(context.Background(), mainVersionImageName, pushOptions)
			if pushError != nil {
				print(fmt.Sprintf("ERROR: An error ocurred when trying to push the image: %s\n", pushError))
				return pushError
			}
			jsonmessage.DisplayJSONMessagesStream(mainReader, os.Stderr, termFd, isTerm, nil)
			defer mainReader.Close()
		}

	}

	return nil
}
