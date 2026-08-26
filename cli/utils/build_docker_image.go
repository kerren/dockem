package utils

import (
	"sort"
	"strings"
	"time"

	"github.com/regclient/regclient/types/ref"
	"golang.org/x/mod/sumdb/dirhash"
)

// hashVersion prefixes every image hash this tool produces. It exists so that
// a change to the *composition* of the hash - what feeds it, or the order/
// form in which it does - can reset cache identity deliberately and visibly,
// rather than as an accidental side effect of some unrelated change landing
// in the same release.
//
// Bump this constant (eg. "dockem-hash-v2" -> "dockem-hash-v3") whenever you
// change what HashWatchFiles, HashWatchDirectories, HashDirectory or the
// Dockerfile hashing below feed into overallHash, or the order they're
// concatenated in. Doing so invalidates every hash tag ever published under
// the old prefix - see CLAUDE.md's Architecture section and the README's
// "Cache identity" section for the consequences.
const hashVersion = "dockem-hash-v2"

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

	// I create a string that I append all of the hashes to. Seeding it with
	// hashVersion namespaces every hash this build of dockem produces to that
	// hash format - see the comment on hashVersion above.
	overallHash := hashVersion

	// Filter out any empty tags
	params.Tag = RemoveEmptyStringsFromArray(params.Tag)

	buildLog = BuildLog{}
	buildLog.respectDockerignore = params.RespectDockerignore

	// Resolve which build backend this run will use before doing any hashing or
	// registry work, so an impossible request (eg. multiple --platform values on
	// a machine without buildx) fails fast rather than after a HEAD check. Only
	// probe for buildx when it could actually be chosen - --builder=docker never
	// needs it. NOTE: this rung only *resolves and records* the builder; the
	// buildx build path itself is wired up in a later rung, so the classic
	// daemon builder below is still used regardless of the decision here.
	builderPreference := params.Builder
	if builderPreference == "" {
		builderPreference = "auto"
	}
	buildxAvailable := false
	buildxVersion := ""
	if builderPreference != "docker" {
		buildxAvailable, buildxVersion, _ = DetectBuildx()
	}
	resolvedBuilder, resolveBuilderError := ResolveBuilder(builderPreference, params.Platform, buildxAvailable, buildxVersion)
	if resolveBuilderError != nil {
		return buildLog, resolveBuilderError
	}
	buildLog.builder = resolvedBuilder
	buildLog.platforms = params.Platform

	// Resolve the ignore/exclude patterns ONCE. The exact same slice feeds the
	// input hash (HashDirectory / HashWatchDirectories) and the build-context
	// tar (TarBuildContext). If those two ever derived from different lists,
	// dockem would build something other than what it hashed - the worst
	// failure this tool can have.
	var excludePatterns []string
	if params.RespectDockerignore {
		resolvedPatterns, ignoreError := ReadDockerignore(params.Directory, params.IgnoreFile, params.Exclude)
		if ignoreError != nil {
			return buildLog, ignoreError
		}
		excludePatterns = resolvedPatterns
	} else {
		// Respecting .dockerignore is opt-in for now (this release defaults it
		// off), but an explicit --exclude is always honoured.
		excludePatterns = append(excludePatterns, params.Exclude...)
	}
	buildLog.excludePatterns = excludePatterns

	// Hash the watch files if they exist
	hashWatchFileResult, hashWatchFileError := HashWatchFiles(params.WatchFile)
	if hashWatchFileError != nil {
		return buildLog, hashWatchFileError
	}
	overallHash += hashWatchFileResult

	// Hash the watch directories if they exist
	watchDirectoriesHash, watchDirectoriesHashError := HashWatchDirectories(params.WatchDirectory, excludePatterns)
	if watchDirectoriesHashError != nil {
		return buildLog, watchDirectoriesHashError
	}
	overallHash += watchDirectoriesHash

	// Hash the build directory if the ignore flag has not been specified
	if !params.IgnoreBuildDirectory {
		directoryHash, err := HashDirectory(params.Directory, excludePatterns)
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

	// Fold the requested platform list into the hash, immediately after the
	// Dockerfile hash. The same inputs built for a different set of target
	// architectures are a *different required output*, so the platform list is
	// part of the cache identity: without this, dockem would keep copying, say,
	// a linux/amd64 image forward even after the caller started asking for
	// linux/amd64,linux/arm64. hashPlatforms appends nothing when --platform is
	// unset, so single-platform users who never adopt the flag see no extra
	// invalidation beyond the .dockerignore reset - see its comment below.
	overallHash += hashPlatforms(params.Platform)

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

		// Two build backends, chosen by the builder resolved above.
		//
		// buildx builds every requested platform and pushes ALL tags - the hash
		// tag and every resolved target tag - in a single `docker buildx build
		// --push`. It therefore resolves the full tag list up front (via
		// ResolveTargetTags) and does its own tag logging / outputTags
		// recording; TagAndPushImage and TagAndPushNewImages are NOT called on
		// this path.
		//
		// The classic daemon builder below can only ever produce a single-arch
		// image (ResolveBuilder guarantees we never reach it with more than one
		// platform), so it builds to a local:<hash> tag, pushes the hash tag,
		// then pushes each target tag one at a time - exactly as before.
		if buildLog.builder == "buildx" {
			buildxError := BuildImageBuildx(params, imageHash, ResolveTargetTags(params, version), &buildLog)
			if buildxError != nil {
				return buildLog, buildxError
			}
		} else {
			dockerClient, pushOptions, err := CreateDockerClient(params.DockerUsername, params.DockerPassword, params.Registry)
			if err != nil {
				return buildLog, err
			}
			defer dockerClient.Close()

			// Build the image
			localTag, dockerImageBuildError := BuildImage(params, imageHash, excludePatterns, dockerClient, &buildLog)

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
	}

	return buildLog, nil
}

// hashPlatforms turns the requested --platform list into the string that feeds
// overallHash (see the call site above). It returns the empty string for an
// unset list so that, byte for byte, a build with no --platform hashes exactly
// as a pre-platform-support dockem did - existing single-platform users get no
// surprise cache invalidation. For a non-empty list it sorts the platforms
// before joining them, so the cache identity depends on the *set* of target
// platforms, not the order they happened to be passed on the command line
// (`--platform linux/amd64,linux/arm64` and `--platform linux/arm64,linux/amd64`
// are the same build and must share a hash). The input slice is not mutated.
func hashPlatforms(platforms []string) string {
	if len(platforms) == 0 {
		return ""
	}
	sorted := append([]string(nil), platforms...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}
