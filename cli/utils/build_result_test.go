package utils

import (
	"reflect"
	"testing"
)

// TestBuildLogResultMapsFields is a pure unit test - it constructs a BuildLog
// directly (this test file is in package utils, so the unexported fields are
// reachable) rather than driving a real build, so it needs no registry.
func TestBuildLogResultMapsFields(t *testing.T) {
	buildLog := BuildLog{
		dockerRegistry:  "reg.example.com",
		durationMs:      1234,
		hashExists:      true,
		hashedImageName: "reg.example.com/myapp:abc123",
		imageHash:       "abc123",
		outputTags:      []string{"reg.example.com/myapp:stable-v1.0.0", "reg.example.com/myapp:latest"},
		version:         "v1.0.0",
	}

	result := buildLog.Result()

	expected := BuildResult{
		Hash:       "abc123",
		CacheHit:   true,
		Image:      "reg.example.com/myapp:abc123",
		Version:    "v1.0.0",
		Tags:       []string{"reg.example.com/myapp:stable-v1.0.0", "reg.example.com/myapp:latest"},
		PrimaryTag: "reg.example.com/myapp:stable-v1.0.0",
		Platforms:  []string{},
		Registry:   "reg.example.com",
		DurationMs: 1234,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Result() mismatch.\nexpected: %+v\ngot:      %+v", expected, result)
	}
}

// TestBuildLogResultHandlesNoTags exercises the partial-result shape a
// failed build leaves behind: BuildDockerImage can return an error before
// any tag was ever resolved, and callers (WriteBuildOutput /
// WriteGitHubOutput) must still get a well-formed result rather than a nil
// slice or an out-of-bounds panic.
func TestBuildLogResultHandlesNoTags(t *testing.T) {
	buildLog := BuildLog{
		hashExists: false,
		imageHash:  "abc123",
	}

	result := buildLog.Result()

	if result.PrimaryTag != "" {
		t.Errorf("Expected an empty PrimaryTag when no tags were resolved, got: %q", result.PrimaryTag)
	}
	if result.Tags == nil {
		t.Errorf("Expected Tags to be an empty slice (not nil) when no tags were resolved, so it marshals to [] rather than null")
	}
	if len(result.Tags) != 0 {
		t.Errorf("Expected no tags, got: %v", result.Tags)
	}
	if result.Platforms == nil || len(result.Platforms) != 0 {
		t.Errorf("Expected Platforms to be present but empty, got: %v", result.Platforms)
	}
}
