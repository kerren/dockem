package utils

import (
	"os/exec"
	"strings"
)

// DetectBuildx probes for the docker buildx plugin by running
// `docker buildx version`. It returns whether buildx is available and, when it
// is, the version string the plugin reports (eg. "v0.36.1").
//
// A missing docker binary, a missing buildx plugin, or a non-zero exit from the
// probe are all reported as (false, "", nil) rather than as an error: "buildx
// is unavailable" is a normal, expected outcome that the caller resolves
// against (see ResolveBuilder). It only becomes fatal when the caller
// specifically needs buildx - eg. a multi-platform build - and that decision
// belongs to ResolveBuilder, not here. The error return is reserved for a
// genuinely unexpected failure and is nil in practice.
func DetectBuildx() (bool, string, error) {
	output, err := exec.Command("docker", "buildx", "version").CombinedOutput()
	if err != nil {
		return false, "", nil
	}
	return true, parseBuildxVersion(string(output)), nil
}

// parseBuildxVersion pulls the version token out of `docker buildx version`
// output, which looks like:
//
//	github.com/docker/buildx v0.36.1 d9f8b8f0...
//
// It returns the first whitespace-separated field that looks like a version
// (a 'v' followed by a digit); failing that, the trimmed output as-is so the
// caller still has something to log.
func parseBuildxVersion(output string) string {
	trimmed := strings.TrimSpace(output)
	for _, field := range strings.Fields(trimmed) {
		if len(field) >= 2 && field[0] == 'v' && field[1] >= '0' && field[1] <= '9' {
			return field
		}
	}
	return trimmed
}
