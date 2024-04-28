package utils

import (
	"os"
	"os/exec"
	"testing"
)

func TestAssertStringEmpty(t *testing.T) {
	if os.Getenv("ASSERT_SHOULD_FAIL") == "1" {
		AssertStringNotEmpty("", "flag", "ERROR: The flag '%s' cannot be empty.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertStringEmpty")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("TestAssertStringEmpty did not fail when a string was empty")
}

func TestAssertStringNotEmpty(t *testing.T) {
	if os.Getenv("ASSERT_SHOULD_NOT_FAIL") == "1" {
		AssertStringNotEmpty("NOT EMPTY STRING", "flag", "ERROR: The flag '%s' cannot be empty.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertStringNotEmpty")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_NOT_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		t.Fatalf("TestAssertStringEmpty failed when a string was NOT empty")
	}
}
