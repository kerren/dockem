package utils

import (
	"time"

	"github.com/regclient/regclient/types/ref"
	"golang.org/x/mod/sumdb/dirhash"
)

// BuildDockerImage returns named results (rather than the usual unnamed
// (BuildLog, error)) purely so the deferred func below can record the
// duration on buildLog no matter which of the many return statements below
// is hit - including the early-return error paths, so BuildResult.DurationMs
// (see build_result.go) reflects a failed run's duration too, not just a
// successful one.
func BuildDockerImage(params BuildDockerImageParams) (buildLog BuildLog, err error) {
	startTime := time.Now()
	defer func() {
		buildLog.durationMs = time.Since(startTime).Milliseconds()
	}()

	// I create a string that I append all of the hashes to
	overallHash := ""

    // Filter out any empty tags
    params.Tag = RemoveEmptyStringsFromArray(params.Tag)

	buildLog = BuildLog{}

	// Hash the watch files if they exist
	hashWatchFileResult, hashWatchFileError := HashWatchFiles(params.WatchFile)
	if hashWatchFileError != nil {
		return buildLog, hashWatchFileError
	}
	overallHash += hashWatchFileResult

	// Hash the watch directories if they exist
	watchDirectoriesHash, watchDirectoriesHashError := HashWatchDirectories(params.WatchDirectory)
	if watchDirectoriesHashError != nil {
		return buildLog, watchDirectoriesHashError
	}
	overallHash += watchDirectoriesHash

	// Hash the build directory if the ignore flag has not been specified
	if !params.IgnoreBuildDirectory {
		directoryHash, err := dirhash.HashDir(params.Directory, "", dirhash.Hash1)
		if err != nil {
			LogError("An error ocurred when hashing the build directory, please ensure it exists and is not empty. You specified %s as the directory\n", params.Directory)
			return buildLog, err
		}
		overallHash += directoryHash
	}

	// Hash the Dockerfile
	dockerfileHash, err := dirhash.Hash1([]string{params.DockerfilePath}, osOpen)
	if err != nil {
		LogError("An error ocurred when hashing the Dockerfile, please ensure it exists. You specified %s as the Dockerfile\n", params.DockerfilePath)
		return buildLog, err
	}
	overallHash += dockerfileHash

	// We now have the hash of all of the different files combined into one (unique) string. We
	// can now hash this string to create a unique hash for the image.
	imageHash := HashString(overallHash)
	buildLog.imageHash = imageHash

	// Now we need to open the version file (JSON file) and pull out the "version" key
	version, versionError := ExtractVersion(params.VersionFile)
	if versionError != nil {
		return buildLog, versionError
	}
	buildLog.version = version

	// Now that we have the hash, we can check if this hash exists on the docker registry already.
	// For this, we'll need regclient because it allows us to interact with the registry instead
	// of just the docker daemon. https://github.com/regclient/regclient
	client := CreateRegclientClient(params.Registry, params.DockerUsername, params.DockerPassword, &buildLog)

	// Now we create the image name of the image that should exist on the registry if it has
	// been built before. This would look like this:
	//
	// 		org/image-name:hash
	//
	imageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
	r, err := ref.New(imageName)
	if err != nil {
		LogError("An error ocurred when trying to parse the image: %s\n", imageName)
		return buildLog, err
	}
	buildLog.hashedImageName = imageName

	// Now we do a HEAD request to see if the image exists on the registry already. This is
	// really good for registries that have a limit on image pulls per day.
	exists, headCheckError := CheckManifestHead(imageHash, r, client)
	buildLog.hashExists = exists
	buildLog.headCheckError = headCheckError

	// A non-nil error here means the check itself failed - eg. an auth failure or a
	// rate limit - as opposed to a clean "the hash does not exist yet". exists is
	// always false alongside such an error. By default we log a warning and fall
	// through to the build below exactly as before this check existed; passing
	// --strict-registry makes this fatal instead, since building and pushing after
	// an unreliable check can silently mask an auth problem until the push fails too.
	if headCheckError != nil {
		if params.StrictRegistry {
			return buildLog, headCheckError
		}
		buildLog.headCheckSkipped = true
		LogWarn("Unable to reliably check whether the image hash %s exists on the registry: %s -- the build will continue, but this should be investigated\n", imageHash, headCheckError)
	}

	if exists {
		LogInfo("The image hash %s already exists on the registry, we can now copy this to the other tags!\n", imageHash)
		// If the image already exists, we just need to copy the tags across
		copyError := CopyExistingImageTag(params, version, imageName, &client, &buildLog)
		if copyError != nil {
			return buildLog, copyError
		}
	} else {
		// We need to build the image and then we push it to the registry
		LogInfo("The image hash %s does not exist on the registry, we will now build the image and push it to the registry\n", imageHash)

		dockerClient, pushOptions, err := CreateDockerClient(params.DockerUsername, params.DockerPassword, params.Registry)
		if err != nil {
			return buildLog, err
		}
		defer dockerClient.Close()

		// Build the image
		localTag, dockerImageBuildError := BuildImage(params, imageHash, dockerClient, &buildLog)

		if dockerImageBuildError != nil {
			return buildLog, dockerImageBuildError
		}
		buildLog.localTag = localTag

		LogInfo("Docker build complete, pushing the image to the registry\n")

		// Now we push the hashed image and then all of the other tags that the
		// user has specified
		hashedImageNameError := TagAndPushImage(localTag, imageName, dockerClient, pushOptions)
		if hashedImageNameError != nil {
			return buildLog, hashedImageNameError
		}

		LogInfo("The image has been pushed to the registry with the hash %s\n", imageName)

		// Now that the hashed image has been pushed, we can push all of the other tags
		tagAndPushImagesError := TagAndPushNewImages(params, version, localTag, dockerClient, pushOptions, &buildLog)
		if tagAndPushImagesError != nil {
			return buildLog, tagAndPushImagesError
		}
	}

	return buildLog, nil
}
