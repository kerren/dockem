package utils

import "testing"

// TestBuildDockerImageSecretsWithClassicBuilderErrorBeforeAnyRegistryWork pins
// the CALL SITE, which the ResolveBuilder truth tables cannot: that
// BuildDockerImage actually passes the requested secrets into the builder
// resolution, and that it does so early enough for an impossible request to
// abort before any work happens.
//
// buildLog.imageHash being empty is the cheap, direct proof of "before any work":
// the hash is computed after builder resolution and before the registry HEAD, so
// an empty hash means the run stopped ahead of both. That ordering is the whole
// reason ResolveBuilder is called first, and it is what keeps a doomed run from
// spending a registry request.
//
// This test needs no registry, no credentials and no Docker daemon: with
// Builder "docker", BuildDockerImage never probes for buildx, and the paths
// below are the real fixtures only so that nothing BUT the secrets is wrong.
func TestBuildDockerImageSecretsWithClassicBuilderErrorBeforeAnyRegistryWork(t *testing.T) {
	testDirectory := "../../testing/e2e/base-test-image"
	directory := testDirectory + "/build"

	params := BuildDockerImageParams{
		Builder:        "docker",
		Directory:      directory,
		DockerfilePath: directory + "/Dockerfile",
		ImageName:      "example-org/never-built",
		Secret:         []string{"id=npmrc,src=./.npmrc"},
		VersionFile:    testDirectory + "/version.json",
	}

	buildLog, err := BuildDockerImage(params)
	if err == nil {
		t.Fatalf("--secret with --builder=docker must fail rather than build without the secret, got nil")
	}
	if buildLog.imageHash != "" {
		t.Errorf("expected the run to abort before hashing (and therefore before the registry HEAD), but an image hash %q was computed", buildLog.imageHash)
	}
}
