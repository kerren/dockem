package utils

import (
	"os"
	"os/exec"
	"path/filepath"
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
// The command assembled (see assembleBuildxArgs) is:
//
//	docker buildx build --file <dockerfile>
//	  [--platform <p1,p2,…>]        # omitted entirely when --platform is unset
//	  [--cache-from <value>]…       # one flag per --cache-from value, omitted entirely when unset
//	  [--cache-to <value>]…         # one flag per --cache-to value, omitted entirely when unset
//	  [--secret <value>]…           # one flag per --secret value, omitted entirely when unset
//	  --tag <image:hash>            # the hash tag, as on the classic path
//	  --tag <each resolved target>  # every tag ResolveTargetTags produced
//	  --push
//	  [--progress plain]            # only when stderr is not a TTY
//	  <context directory>
//
// --cache-from/--cache-to are passed through to buildx VERBATIM, exactly as the
// caller wrote them - dockem never interprets or rewrites their contents. They
// select a BuildKit cache backend (eg. type=gha for GitHub Actions) and change
// how fast a build runs, never what it produces, so - unlike --platform - they
// are deliberately excluded from the image hash (see build_docker_image.go).
// When the classic (docker) builder is resolved instead of buildx,
// BuildDockerImage logs a warning that these flags are being ignored, since
// this function - and therefore the only place they take effect - never runs.
//
// --secret is passed through VERBATIM in the same way, and is likewise excluded
// from the image hash - a build secret is a credential that routinely rotates
// (a CI token changes on every run), so hashing it would invalidate every tag on
// every build. It differs from the cache flags in one important way: on the
// classic path it is a HARD ERROR in ResolveBuilder rather than a warning here,
// because the classic builder has no BuildKit session and would hand the
// Dockerfile an empty /run/secrets/<id>, publishing an image built without the
// secret. Note the secret VALUE never reaches this argument list: buildx's
// --secret syntax carries only an id and a reference to the value (env=NAME or
// src=PATH), so nothing sensitive lands in the subprocess argv.
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

	args := assembleBuildxArgs(params, hashImageName, targetTags, buildLog)

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

// assembleBuildxArgs builds the full `docker buildx build` argument list (ie.
// everything that follows `docker` on the command line) described on
// BuildImageBuildx. It is split out from BuildImageBuildx as a pure function -
// no subprocess, no filesystem, no network - purely so the argument assembly,
// in particular that --cache-from/--cache-to/--secret reach the list VERBATIM
// and one flag per value, can be unit tested directly (see
// build_image_buildx_test.go)
// without needing docker or a registry.
//
// It is not otherwise side-effect-free: exactly as before this was split out,
// it emits the same per-branch log line TagAndPushNewImages emits for each
// target tag (via LogInfo/LogWarn) and appends every target tag's fully
// qualified image name to buildLog.outputTags, in ResolveTargetTags order.
// It also records buildLog.customDockerfile for an out-of-context Dockerfile,
// which is the one piece of BuildLog bookkeeping the classic path performs
// inside TarBuildContext - a function this path deliberately never calls.
// All three are depended on elsewhere (see BuildImageBuildx's doc comment),
// so preserve them if you touch this function.
func assembleBuildxArgs(params BuildDockerImageParams, hashImageName string, targetTags []ResolvedTag, buildLog *BuildLog) []string {
	args := []string{"buildx", "build", "--file", params.DockerfilePath}

	// Record the out-of-context Dockerfile on BuildLog. buildx needs no
	// workaround for one - the real path goes straight to --file above, which
	// is why TarBuildContext, where the classic path sets this flag, is never
	// reached here. The flag still has to be set: BuildLog is how the e2e tests
	// observe which branch ran, so leaving it unset on this path makes the two
	// builders report differently about an identical build.
	if dockerfileOutsideContext(params.Directory, params.DockerfilePath) {
		buildLog.customDockerfile = true
	}

	// --platform is omitted entirely when unset, so a single-platform buildx
	// build behaves as buildx's default (the host platform).
	if len(params.Platform) > 0 {
		args = append(args, "--platform", strings.Join(params.Platform, ","))
	}

	// --cache-from/--cache-to: one --cache-from/--cache-to flag per value,
	// verbatim, in the order given, omitted entirely when unset (the same
	// "no flag, no behaviour change" shape as --platform above). dockem does
	// not parse or validate the value - eg. "type=gha" or
	// "type=registry,ref=example.com/repo:cache" pass straight through so any
	// buildx cache backend works without dockem needing to know about it.
	for _, cacheFrom := range params.CacheFrom {
		args = append(args, "--cache-from", cacheFrom)
	}
	for _, cacheTo := range params.CacheTo {
		args = append(args, "--cache-to", cacheTo)
	}

	// --secret: one --secret flag per value, verbatim, in the order given,
	// omitted entirely when unset - the same shape as the cache flags above.
	// dockem does not parse the value, so id=, env=, src=, type= and any future
	// buildx secret syntax pass straight through. In particular the value is
	// never re-split on commas: "id=npmrc,src=/path" is ONE secret.
	//
	// Unlike the cache flags, reaching this loop on the classic path is
	// impossible rather than merely unhelpful: ResolveBuilder hard-errors when
	// secrets are requested and the resolved builder is not buildx, because a
	// classic build would mount an empty secret file and publish a wrong image.
	for _, secret := range params.Secret {
		args = append(args, "--secret", secret)
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

	return args
}

// describePlatforms is a small logging convenience: it renders the requested
// platform list for a human, saying "the host platform" when none was given.
func describePlatforms(platforms []string) string {
	if len(platforms) == 0 {
		return "the host platform"
	}
	return strings.Join(platforms, ", ")
}

// dockerfileOutsideContext reports whether dockerfilePath resolves outside the
// build context at directory, which is the condition the classic path detects
// in TarBuildContext (there, to copy the Dockerfile into the context before
// tarring it; here, only to record it on BuildLog).
//
// It applies the same rule TarBuildContext does - the Dockerfile's path
// relative to the context escapes upwards - but tests it against the platform's
// own separator rather than a literal "../", so it holds on Windows too.
//
// A path that cannot be resolved is reported as inside the context: this
// function only feeds a BuildLog field, so it must never be the thing that
// fails a build. An unresolvable path will surface a real error from buildx
// itself moments later, with a far better message than anything derivable here.
func dockerfileOutsideContext(directory, dockerfilePath string) bool {
	absDirectory, directoryErr := filepath.Abs(directory)
	if directoryErr != nil {
		return false
	}

	absDockerfile, dockerfileErr := filepath.Abs(dockerfilePath)
	if dockerfileErr != nil {
		return false
	}

	relative, relativeErr := filepath.Rel(absDirectory, absDockerfile)
	if relativeErr != nil {
		return false
	}

	return relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
