package utils

import "fmt"

// ResolveBuilder decides which build backend a run will use, given the user's
// --builder preference ("auto", "buildx" or "docker"; "" is treated as "auto"),
// the requested --platform list, and whether buildx was detected (see
// DetectBuildx). It returns the resolved builder, either "buildx" or "docker",
// and logs the decision.
//
// It is a pure function - buildx availability is passed in rather than probed -
// so the multi-platform error path can be unit-tested without docker installed.
//
// The one rule that must never bend: a build that names more than one platform
// cannot be served by the classic docker builder, which produces a single-arch
// image. Publishing that single-arch image under a multi-arch-looking tag is
// the worst outcome this tool can produce, so any request that would do so is a
// hard error here, never a silent fallback.
func ResolveBuilder(builderPreference string, platforms []string, buildxAvailable bool, buildxVersion string) (string, error) {
	multiPlatform := len(platforms) > 1

	switch builderPreference {
	case "docker":
		if multiPlatform {
			LogError("--builder=docker was requested, but the classic docker builder cannot build the %d platforms you asked for (%v) - it only produces a single-arch image. Re-run with --builder=auto or --builder=buildx on a machine that has 'docker buildx', or drop the extra --platform values.\n", len(platforms), platforms)
			return "", fmt.Errorf("--builder=docker cannot build multiple platforms %v", platforms)
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
		LogInfo("Builder auto-resolved to 'docker' (buildx not detected, falling back to the classic daemon builder).\n")
		return "docker", nil

	default:
		// cmd/build.go validates --builder with AssertOneOf, so this is only
		// reachable if BuildDockerImage is called directly with a bad value.
		LogError("Unknown --builder value '%s'; expected one of auto, buildx or docker.\n", builderPreference)
		return "", fmt.Errorf("unknown builder %q", builderPreference)
	}
}
