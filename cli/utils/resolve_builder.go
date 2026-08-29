package utils

import "fmt"

// ResolveBuilder decides which build backend a run will use, given the user's
// --builder preference ("auto", "buildx" or "docker"; "" is treated as "auto"),
// the requested --platform list, whether any --secret was requested, and whether
// buildx was detected (see DetectBuildx). It returns the resolved builder,
// either "buildx" or "docker", and logs the decision.
//
// It is a pure function - buildx availability is passed in rather than probed -
// so both error paths can be unit-tested without docker installed.
// secretsRequested is deliberately the LAST parameter rather than sitting next
// to platforms with the other request-side inputs: that keeps it separated from
// buildxAvailable by a string, so transposing the two bools is a compile error
// rather than a silently inverted rule.
//
// Two rules must never bend, and both are hard errors here rather than silent
// fallbacks:
//
//  1. A build that names more than one platform cannot be served by the classic
//     docker builder, which produces a single-arch image. Publishing that
//     single-arch image under a multi-arch-looking tag is the worst outcome this
//     tool can produce.
//  2. A build that supplies --secret cannot be served by the classic docker
//     builder either. BuildImage builds through the Engine API's
//     types.ImageBuildOptions, which has no secrets field at all - BuildKit
//     secrets need a buildkitd session it cannot carry. Dropping the secret
//     would hand the Dockerfile an EMPTY /run/secrets/<id> file and publish an
//     image built without it, so unlike --cache-from/--cache-to (which only cost
//     speed and are merely warned about) this can never be ignored. The cost is
//     not a slow build but a poisoned cache: that wrong image would be published
//     under its hash tag and copied forward by every later run, forever.
func ResolveBuilder(builderPreference string, platforms []string, buildxAvailable bool, buildxVersion string, secretsRequested bool) (string, error) {
	multiPlatform := len(platforms) > 1

	switch builderPreference {
	case "docker":
		if multiPlatform {
			LogError("--builder=docker was requested, but the classic docker builder cannot build the %d platforms you asked for (%v) - it only produces a single-arch image. Re-run with --builder=auto or --builder=buildx on a machine that has 'docker buildx', or drop the extra --platform values.\n", len(platforms), platforms)
			return "", fmt.Errorf("--builder=docker cannot build multiple platforms %v", platforms)
		}
		if secretsRequested {
			LogError("--builder=docker was requested, but the classic docker builder cannot mount the --secret values you supplied - BuildKit secrets are a buildx feature and the classic builder would silently build against an empty secret file, producing a different image. Re-run with --builder=auto or --builder=buildx on a machine that has 'docker buildx', or drop the --secret values.\n")
			return "", fmt.Errorf("--builder=docker cannot mount build secrets")
		}
		LogInfo("Builder resolved to 'docker' (the classic daemon builder, --builder=docker).\n")
		return "docker", nil

	case "buildx":
		if !buildxAvailable {
			LogError("--builder=buildx was requested, but 'docker buildx version' did not succeed, so buildx is not available. Install the buildx plugin, or use --builder=auto to fall back to the classic builder for single-platform builds.\n")
			return "", fmt.Errorf("--builder=buildx requested but buildx is not available")
		}
		LogInfo("Builder resolved to 'buildx' %s (--builder=buildx).\n", buildxVersion)
		return "buildx", nil

	case "auto", "":
		if buildxAvailable {
			LogInfo("Builder auto-resolved to 'buildx' %s.\n", buildxVersion)
			return "buildx", nil
		}
		if multiPlatform {
			LogError("buildx is not available ('docker buildx version' did not succeed), but you requested %d platforms (%v). The classic docker builder can only produce a single-arch image, which must never be published under a multi-arch tag. Install the buildx plugin, or drop the extra --platform values.\n", len(platforms), platforms)
			return "", fmt.Errorf("multi-platform build requested (%v) but buildx is not available", platforms)
		}
		if secretsRequested {
			LogError("buildx is not available ('docker buildx version' did not succeed), but you supplied --secret. BuildKit secrets are a buildx feature; falling back to the classic docker builder would build against an empty secret file and publish an image that was never given the secret. Install the buildx plugin, or drop the --secret values.\n")
			return "", fmt.Errorf("build secrets requested but buildx is not available")
		}
		LogInfo("Builder auto-resolved to 'docker' (buildx not detected, falling back to the classic daemon builder).\n")
		return "docker", nil

	default:
		// cmd/build.go validates --builder with AssertOneOf, so this is only
		// reachable if BuildDockerImage is called directly with a bad value.
		LogError("Unknown --builder value '%s'; expected one of auto, buildx or docker.\n", builderPreference)
		return "", fmt.Errorf("unknown builder %q", builderPreference)
	}
}
