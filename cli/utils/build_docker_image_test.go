package utils

import (
	"os"
	"testing"
)

func TestStandardBuildWhereHashExists(t *testing.T) {
	// In this test, I'm going to test a build where the hash is the same.
	// In that case, it should not trigger a build but should rather copy
	// the tag from the existing image to the new image.
	imageName := os.Getenv("TEST_IMAGE_NAME")
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if imageName == "" || username == "" || password == "" {
		t.Fatal("Unable to run test because environment variables are not set")
	}
	testDirectory := "../../testing/e2e/base-test-image"
	directory := testDirectory + "/build"
	versionPath := testDirectory + "/version.json"

	params := BuildDockerImageParams{
		Directory:            directory,
		DockerBuildFlags:     []string{},
		DockerPassword:       password,
		DockerUsername:       username,
		DockerfilePath:       directory + "/Dockerfile",
		IgnoreBuildDirectory: false,
		ImageName:            imageName,
		Latest:               false,
		MainVersion:          false,
		Registry:             "",
		Tag:                  []string{"test-hash-exists"},
		VersionFile:          versionPath,
		WatchDirectory:       []string{},
		WatchFile:            []string{},
	}

	buildLog, err := BuildDockerImage(params)
	if err != nil {
		t.Errorf("Error building the docker image: %s", err)
	}
	if !buildLog.hashExists {
		t.Errorf("The hash should exist")
	}

}
