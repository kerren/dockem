package utils

import "testing"

// These are pure unit tests: no filesystem, no registry, no credentials.

func TestGenerateDockerImageNameEmptyRegistryOmitsHost(t *testing.T) {
	got := GenerateDockerImageName("", "myapp", "abc123")
	want := "myapp:abc123"
	if got != want {
		t.Fatalf("GenerateDockerImageName(\"\", ...) = %q, want %q", got, want)
	}
}

func TestGenerateDockerImageNameSetRegistryIncludesHost(t *testing.T) {
	got := GenerateDockerImageName("reg.example.com", "myapp", "abc123")
	want := "reg.example.com/myapp:abc123"
	if got != want {
		t.Fatalf("GenerateDockerImageName(reg, ...) = %q, want %q", got, want)
	}
}

// TestGenerateDockerImageNameImageNameWithSlash pins that an image name that
// already contains an org/namespace slash (e.g. "org/myapp") is concatenated
// as-is - GenerateDockerImageName does no parsing or validation of the image
// name, it only decides whether to prepend the registry host.
func TestGenerateDockerImageNameImageNameWithSlash(t *testing.T) {
	got := GenerateDockerImageName("reg.example.com", "org/myapp", "abc123")
	want := "reg.example.com/org/myapp:abc123"
	if got != want {
		t.Fatalf("GenerateDockerImageName(reg, \"org/myapp\", ...) = %q, want %q", got, want)
	}

	gotNoRegistry := GenerateDockerImageName("", "org/myapp", "abc123")
	wantNoRegistry := "org/myapp:abc123"
	if gotNoRegistry != wantNoRegistry {
		t.Fatalf("GenerateDockerImageName(\"\", \"org/myapp\", ...) = %q, want %q", gotNoRegistry, wantNoRegistry)
	}
}
