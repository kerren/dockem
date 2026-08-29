package utils

import (
	"os"
	"os/exec"
	"strings"

	"github.com/moby/term"
)

// BuildImageBuildx builds and pushes the image through `docker buildx build`,
// the modern BuildKit path. Unlike the classic builder (BuildImage +
// TagAndPushImage + TagAndPushNewImages), buildx builds every requested
// platform and pushes EVERY tag in a single invocation, so this one function
// covers the whole build-and-push for the buildx path and TagAndPushImage /
// TagAndPushNewImages are never called alongside it.
//
// targetTags is the fully-resolved tag list from ResolveTargetTags (resolved by
// the caller, before the build, precisely because buildx needs every tag up
// front). It carries the resolution Reason for each tag so the per-branch log
// lines - including the "No tags were specified…" warning - are emitted here
// exactly as TagAndPushNewImages emits them on the classic path, and so
// buildLog.outputTags ends up byte-identical to the classic path.
//
// The command assembled is:
//
//	docker buildx build --file <dockerfile>
//	  [--platform <p1,p2,…>]        # omitted entirely when --platform is unset
//	  --tag <image:hash>            # the hash tag, as on the classic path
//	  --tag <each resolved target>  # every tag ResolveTargetTags produced
//	  --push
//	  [--progress plain]            # only when stderr is not a TTY
//	  <context directory>
//
// The Dockerfile is passed by its real path: buildx accepts a --file outside
// the context directory natively, so the out-of-context temp-file dance that
// TarBuildContext performs for the classic builder is not needed here.
//
// The subprocess stdout and stderr are both streamed straight to os.Stderr so
// build progress keeps flowing while dockem's own stdout stays clean for the
// --output-format=json contract.
//
// Credentials: when an explicit --docker-username/--docker-password pair is
// given, a throwaway config.json is written by TempDockerConfig and handed to
// the subprocess via DOCKER_CONFIG on its environment ONLY (never dockem's own
// environment), and removed by the deferred cleanup even if the build fails.
// With no explicit credentials the subprocess inherits dockem's environment
// unchanged, so an existing `docker login` keeps working. The password is never
// placed on the command line, never logged, and never recorded on BuildLog.
func BuildImageBuildx(params BuildDockerImageParams, imageHash string, targetTags []ResolvedTag, buildLog *BuildLog) error {
	hashImageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)

	args := []string{"buildx", "build", "--file", params.DockerfilePath}

	// --platform is omitted entirely when unset, so a single-platform buildx
	// build behaves as buildx's default (the host platform).
	if len(params.Platform) > 0 {
		args = append(args, "--platform", strings.Join(params.Platform, ","))
	}

	// The hash tag first, mirroring the classic path where TagAndPushImage
	// pushes the hash tag before TagAndPushNewImages pushes the rest. The hash
	// tag is a --tag but is deliberately NOT added to buildLog.outputTags,
	// matching the classic path (TagAndPushImage does not record it either).
	args = append(args, "--tag", hashImageName)

	// Every resolved target tag reaches both the argument list and
	// buildLog.outputTags, in ResolveTargetTags order, with the same per-branch
	// log line the classic push path emits.
	for _, resolved := range targetTags {
		switch resolved.Reason {
		case TagReasonTag:
			LogInfo("Pushing the image to the new tag: %s\n", resolved.ImageName)
		case TagReasonFallback:
			LogWarn("No tags were specified and you have not selected the --latest flag, so the image will be deployed to the main version: %s\n", resolved.ImageName)
		case TagReasonLatest:
			LogInfo("You have selected the --latest flag, so the image will be deployed to the latest tag: %s\n", resolved.ImageName)
		case TagReasonMainVersion:
			LogInfo("You have selected the --main-version flag, so the image will be deployed to the main version: %s\n", resolved.ImageName)
		}
		args = append(args, "--tag", resolved.ImageName)
		buildLog.outputTags = append(buildLog.outputTags, resolved.ImageName)
	}

	args = append(args, "--push")

	// buildx renders an interactive progress UI when it thinks it is on a
	// terminal. In CI (the primary use case for dockem) stderr is not a TTY, so
	// ask for the plain, line-based progress that logs cleanly. We test stderr
	// because that is where the build output is streamed.
	if _, isTerminal := term.GetFdInfo(os.Stderr); !isTerminal {
		args = append(args, "--progress", "plain")
	}

	args = append(args, params.Directory)

	// Write the throwaway credentials (if any) and make sure they are removed
	// however this function returns.
	configDir, cleanup, configErr := TempDockerConfig(params.Registry, params.DockerUsername, params.DockerPassword)
	if configErr != nil {
		return configErr
	}
	defer cleanup()

	LogInfo("Building %s for %s through docker buildx and pushing every tag in one step\n", hashImageName, describePlatforms(params.Platform))

	cmd := exec.Command("docker", args...)
	// Stream both streams to stderr: build logs keep flowing and stdout stays
	// clean for --output-format=json.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	// Only touch the subprocess environment when we have a throwaway config to
	// point it at. An empty configDir means "no explicit credentials" - leave
	// cmd.Env nil so the child inherits dockem's environment unchanged and an
	// existing `docker login` (the user's real config.json) is used. When we do
	// have a config, start from the inherited environment, strip any DOCKER_CONFIG
	// already present so ours is unambiguous, and append it last.
	if configDir != "" {
		env := make([]string, 0, len(os.Environ())+1)
		for _, entry := range os.Environ() {
			if strings.HasPrefix(entry, "DOCKER_CONFIG=") {
				continue
			}
			env = append(env, entry)
		}
		env = append(env, "DOCKER_CONFIG="+configDir)
		cmd.Env = env
	}

	if runErr := cmd.Run(); runErr != nil {
		LogError("The docker buildx build failed for the image hash %s. Please check the buildx output above, and that 'docker buildx' works and you are authenticated to the registry (--docker-username / --docker-password or an existing 'docker login').\n", imageHash)
		return runErr
	}

	LogInfo("The image has been built and every tag pushed to the registry through docker buildx\n")
	return nil
}

// describePlatforms is a small logging convenience: it renders the requested
// platform list for a human, saying "the host platform" when none was given.
func describePlatforms(platforms []string) string {
	if len(platforms) == 0 {
		return "the host platform"
	}
	return strings.Join(platforms, ", ")
}
