package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAssertDirectoryDoesNotExist(t *testing.T) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := rand.Intn(100000000)
	directory := "test_dir_" + fmt.Sprintf("%d", randomInt)
	if os.Getenv("ASSERT_SHOULD_FAIL") == "1" {
		AssertDirectoryExists("test", "ERROR: The directory '%s' does not exist. Please specify the path to the directory you would like to build.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertDirectoryDoesNotExist")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf(fmt.Sprintf("AssertDirectoryExists did not fail when a directory '%s' did not exist", directory))
}

func TestAssertDirectoryExists(t *testing.T) {
	directory, _ := os.MkdirTemp("", "test_dir_")
	if os.Getenv("ASSERT_SHOULD_NOT_FAIL") == "1" {
		AssertDirectoryExists(directory, "ERROR: The directory '%s' does not exist. Please specify the path to the directory you would like to build.")
		os.RemoveAll(directory)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertDirectoryExists")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_NOT_FAIL=1")
	err := cmd.Run()
	os.RemoveAll(directory)
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		t.Fatalf(fmt.Sprintf("AssertDirectoryExists failed when a directory '%s' did exist", directory))
	}
}
