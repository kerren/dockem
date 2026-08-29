package utils

import (
	"reflect"
	"testing"
)

// TestResolveTargetTags exercises every branch of the shared tag-resolution
// rule and their combinations. It is a pure unit test: it needs no registry and
// no credentials. The ordering of the returned slice is load-bearing (callers
// append it to buildLog.outputTags in order), so each case asserts the exact
// slice, including order.
func TestResolveTargetTags(t *testing.T) {
	const (
		registry  = "reg.example.com"
		imageName = "myapp"
		version   = "v1.2.3"
	)

	cases := []struct {
		name     string
		params   BuildDockerImageParams
		expected []ResolvedTag
	}{
		{
			name: "tags only",
			params: BuildDockerImageParams{
				Registry:  registry,
				ImageName: imageName,
				Tag:       []string{"alpha", "beta"},
			},
			expected: []ResolvedTag{
				{ImageName: "reg.example.com/myapp:alpha-v1.2.3", Reason: TagReasonTag},
				{ImageName: "reg.example.com/myapp:beta-v1.2.3", Reason: TagReasonTag},
			},
		},
		{
			name: "latest only",
			params: BuildDockerImageParams{
				Registry:  registry,
				ImageName: imageName,
				Latest:    true,
			},
			expected: []ResolvedTag{
				{ImageName: "reg.example.com/myapp:latest", Reason: TagReasonLatest},
			},
		},
		{
			name: "main-version only",
			params: BuildDockerImageParams{
				Registry:    registry,
				ImageName:   imageName,
				MainVersion: true,
			},
			expected: []ResolvedTag{
				{ImageName: "reg.example.com/myapp:v1.2.3", Reason: TagReasonMainVersion},
			},
		},
		{
			name: "none falls back to main version",
			params: BuildDockerImageParams{
				Registry:  registry,
				ImageName: imageName,
			},
			expected: []ResolvedTag{
				{ImageName: "reg.example.com/myapp:v1.2.3", Reason: TagReasonFallback},
			},
		},
		{
			name: "tags plus latest plus main-version",
			params: BuildDockerImageParams{
				Registry:    registry,
				ImageName:   imageName,
				Tag:         []string{"alpha"},
				Latest:      true,
				MainVersion: true,
			},
			expected: []ResolvedTag{
				{ImageName: "reg.example.com/myapp:alpha-v1.2.3", Reason: TagReasonTag},
				{ImageName: "reg.example.com/myapp:latest", Reason: TagReasonLatest},
				{ImageName: "reg.example.com/myapp:v1.2.3", Reason: TagReasonMainVersion},
			},
		},
		{
			name: "empty registry omits host prefix",
			params: BuildDockerImageParams{
				Registry:  "",
				ImageName: imageName,
				Tag:       []string{"alpha"},
			},
			expected: []ResolvedTag{
				{ImageName: "myapp:alpha-v1.2.3", Reason: TagReasonTag},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveTargetTags(tc.params, version)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ResolveTargetTags(%q) = %#v, want %#v", tc.name, got, tc.expected)
			}
		})
	}
}
