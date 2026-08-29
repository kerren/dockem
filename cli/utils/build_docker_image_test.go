package utils

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"testing"
	"time"
)

func CreateTempFile(dir string, t *testing.T) *os.File {
	tempFile, tempFileErr := os.CreateTemp(dir, "test_file_")
	if tempFileErr != nil {
		t.Fatalf("Error creating the temporary file: %s", tempFileErr)
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := rand.Intn(100000000)
	tempFile.Write([]byte(fmt.Sprintf("This is a test file %d", randomInt)))
	return tempFile
}

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

func TestStandardBuildWhereHashDoesNotExist(t *testing.T) {
	// In this test, I'm going to test a build where the hash is different.
	// In that case, it should trigger a build and push the new image to the
	// registry.
	imageName := os.Getenv("TEST_IMAGE_NAME")
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if imageName == "" || username == "" || password == "" {
		t.Fatal("Unable to run test because environment variables are not set")
	}
	testDirectory := "../../testing/e2e/base-changing-test-image"
	directory := testDirectory + "/build"
	versionPath := testDirectory + "/version.json"

	tempFile := CreateTempFile(directory, t)
	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())

	params := BuildDockerImageParams{
		Directory:            directory,
		DockerPassword:       password,
		DockerUsername:       username,
		DockerfilePath:       directory + "/Dockerfile",
		IgnoreBuildDirectory: false,
		ImageName:            imageName,
		Latest:               false,
		MainVersion:          false,
		Registry:             "",
		Tag:                  []string{"test-hash-does-not-exist"},
		VersionFile:          versionPath,
		WatchDirectory:       []string{},
		WatchFile:            []string{},
	}

	buildLog, err := BuildDockerImage(params)
	if err != nil {
		t.Errorf("Error building the docker image: %s", err)
	}
	if buildLog.hashExists {
		t.Errorf("The hash should not exist")
	}
}

func TestDockerfileOutsideOfBuildContext(t *testing.T) {
	// In this test, I'm going to test a build where the Dockerfile is outside
	// of the build context. I will create a temp file to trigger a change
	// in the hash and rebuild the image to ensure this logic works correctly.

	imageName := os.Getenv("TEST_IMAGE_NAME")
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if imageName == "" || username == "" || password == "" {
		t.Fatal("Unable to run test because environment variables are not set")
	}
	testDirectory := "../../testing/e2e/base-changing-test-image"
	directory := testDirectory + "/build"
	versionPath := testDirectory + "/version.json"
	dockerfilePath := "../../testing/e2e/dockerfile-context/Dockerfile.alpine-3.17"

	tempFile := CreateTempFile(directory, t)
	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())

	params := BuildDockerImageParams{
		Directory:            directory,
		DockerPassword:       password,
		DockerUsername:       username,
		DockerfilePath:       dockerfilePath,
		IgnoreBuildDirectory: false,
		ImageName:            imageName,
		Latest:               false,
		MainVersion:          false,
		Registry:             "",
		Tag:                  []string{"test-hash-does-not-exist"},
		VersionFile:          versionPath,
		WatchDirectory:       []string{},
		WatchFile:            []string{},
	}

	buildLog, err := BuildDockerImage(params)
	if err != nil {
		t.Errorf("Error building the docker image: %s", err)
	}
	if buildLog.hashExists {
		t.Errorf("The hash should not exist")
	}
	if !buildLog.customDockerfile {
		t.Errorf("The custom Dockerfile flag should be set")
	}
}

func TestBuildWithLatestFlag(t *testing.T) {
    // In this test, I'm going to test a build where the latest flag is set.
    // In that case, it should push the image with the latest tag to the registry.
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
        DockerPassword:       password,
        DockerUsername:       username,
        DockerfilePath:       directory + "/Dockerfile",
        IgnoreBuildDirectory: false,
        ImageName:            imageName,
        Latest:               true,
        MainVersion:          false,
        Registry:             "",
        Tag:                  []string{"test-latest"},
        VersionFile:          versionPath,
        WatchDirectory:       []string{},
        WatchFile:            []string{},
    }

    buildLog, err := BuildDockerImage(params)
    if err != nil {
        t.Errorf("Error building the docker image: %s", err)
    }

    r, _ := regexp.Compile("latest$")
    idx := sort.Search(len(buildLog.outputTags), func(i int) bool {
        return r.MatchString(buildLog.outputTags[i])
    })
    if idx == len(buildLog.outputTags) {
        t.Errorf("The latest tag should exist in the output tags")
    }

}

func TestBuildWithStableTag(t *testing.T) {
    // In this test, I'm going to test a build where the stable flag is set.
    // In that case, it should push the image with the stable tag to the registry.
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
        DockerPassword:       password,
        DockerUsername:       username,
        DockerfilePath:       directory + "/Dockerfile",
        IgnoreBuildDirectory: false,
        ImageName:            imageName,
        Latest:               false,
        MainVersion:          false,
        Registry:             "",
        Tag:                  []string{"stable"},
        VersionFile:          versionPath,
        WatchDirectory:       []string{},
        WatchFile:            []string{},
    }

    buildLog, err := BuildDockerImage(params)
    if err != nil {
        t.Errorf("Error building the docker image: %s", err)
    }

    r, _ := regexp.Compile("stable-v0.1.2$")
    idx := sort.Search(len(buildLog.outputTags), func(i int) bool {
        return r.MatchString(buildLog.outputTags[i])
    })
    if idx == len(buildLog.outputTags) {
        t.Errorf("The stable tag should exist in the output tags")
    }

}

func TestMainVersion(t *testing.T) {
    // In this test, I'm going to test a build where the main version flag is set.
    // In that case, it should push the image with the main version tag to the registry.
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
        DockerPassword:       password,
        DockerUsername:       username,
        DockerfilePath:       directory + "/Dockerfile",
        IgnoreBuildDirectory: false,
        ImageName:            imageName,
        Latest:               false,
        MainVersion:          true,
        Registry:             "",
        Tag:                  []string{},
        VersionFile:          versionPath,
        WatchDirectory:       []string{},
        WatchFile:            []string{},
    }

    buildLog, err := BuildDockerImage(params)
    if err != nil {
        t.Errorf("Error building the docker image: %s", err)
    }

    r, _ := regexp.Compile(":v0.1.2$")
    idx := sort.Search(len(buildLog.outputTags), func(i int) bool {
        return r.MatchString(buildLog.outputTags[i])
    })
    if idx == len(buildLog.outputTags) {
        t.Errorf("The main version tag should exist in the output tags")
    }

}

func TestBuildWhereDockerignoreExcludesChangingDirectory(t *testing.T) {
	// In this test, I'm going to test a build with --respect-dockerignore set
	// against a fixture whose .dockerignore excludes build/ignored/. I write
	// a random temp file into that ignored directory on every run - the same
	// way TestStandardBuildWhereHashDoesNotExist writes one straight into the
	// build directory to force a new hash - but because this file lands
	// somewhere .dockerignore excludes, it must NOT feed the hash. So unlike
	// that test, the hash should already exist and we should hit the copy
	// path, exactly like TestStandardBuildWhereHashExists.
	imageName := os.Getenv("TEST_IMAGE_NAME")
	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	if imageName == "" || username == "" || password == "" {
		t.Fatal("Unable to run test because environment variables are not set")
	}
	testDirectory := "../../testing/e2e/dockerignore-test-image"
	directory := testDirectory + "/build"
	versionPath := testDirectory + "/version.json"
	ignoredDirectory := directory + "/ignored"

	tempFile := CreateTempFile(ignoredDirectory, t)
	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())

	params := BuildDockerImageParams{
		Directory:            directory,
		DockerPassword:       password,
		DockerUsername:       username,
		DockerfilePath:       directory + "/Dockerfile",
		IgnoreBuildDirectory: false,
		ImageName:            imageName,
		Latest:               false,
		MainVersion:          false,
		Registry:             "",
		RespectDockerignore:  true,
		Tag:                  []string{"test-dockerignore-hash-exists"},
		VersionFile:          versionPath,
		WatchDirectory:       []string{},
		WatchFile:            []string{},
	}

	buildLog, err := BuildDockerImage(params)
	if err != nil {
		t.Errorf("Error building the docker image: %s", err)
	}
	if !buildLog.respectDockerignore {
		t.Errorf("The build log should record that .dockerignore was respected")
	}
	if !buildLog.hashExists {
		t.Errorf("The hash should exist because the changed file lives in a directory excluded by .dockerignore")
	}
}
