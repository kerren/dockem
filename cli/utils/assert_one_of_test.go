package utils

import (
	"os"
	"os/exec"
	"testing"
)

func TestAssertOneOfRejectsValueNotInList(t *testing.T) {
	if os.Getenv("ASSERT_SHOULD_FAIL") == "1" {
		AssertOneOf("xml", []string{"text", "json"}, "ERROR: The value '%s' is not valid.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertOneOfRejectsValueNotInList")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("TestAssertOneOfRejectsValueNotInList did not fail when the value was not in the allowed list")
}

func TestAssertOneOfAcceptsValueInList(t *testing.T) {
	if os.Getenv("ASSERT_SHOULD_NOT_FAIL") == "1" {
		AssertOneOf("json", []string{"text", "json"}, "ERROR: The value '%s' is not valid.")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestAssertOneOfAcceptsValueInList")
	cmd.Env = append(os.Environ(), "ASSERT_SHOULD_NOT_FAIL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		t.Fatalf("TestAssertOneOfAcceptsValueInList failed when the value WAS in the allowed list")
	}
}
