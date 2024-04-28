package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAssertFileDoesNotExist(t *testing.T) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := rand.Intn(100000000)
	fileName := "test_file_" + fmt.Sprintf("%d", randomInt)
	if os.Getenv("ASSERT_SHOULD_FAIL") == "1" {
		AssertFileExists("test", "ERROR: The file '%s' does not exist.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertFileDoesNotExist")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf(fmt.Sprintf("AssertFileExists did not fail when a file '%s' did not exist", fileName))
}

func TestAssertFileExists(t *testing.T) {
	file, _ := os.CreateTemp("", "test_file")
	if os.Getenv("ASSERT_SHOULD_NOT_FAIL") == "1" {
		AssertFileExists(file.Name(), "ERROR: The file '%s' does not exist.")
		os.RemoveAll(file.Name())
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertFileExists")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_NOT_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		t.Fatalf(fmt.Sprintf("AssertFileExists failed when a file '%s' did exist", file.Name()))
	}
}
