package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These are pure unit tests: no registry, no credentials. They do use
// t.Setenv("PATH", ...) to point at a fake `docker` shim, which is why none of
// them call t.Parallel() - Go panics if a test calling t.Setenv is parallel.

// writeDockerShim writes an executable `docker` script into dir that ignores
// its arguments and reproduces the given exit code and combined stdout.
func writeDockerShim(t *testing.T, dir string, exitCode int, output string) {
	t.Helper()
	escaped := strings.ReplaceAll(output, `'`, `'\''`)
	script := fmt.Sprintf("#!/bin/sh\nprintf '%s'\nexit %s\n", escaped, strconv.Itoa(exitCode))
	path := filepath.Join(dir, "docker")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake docker shim: %v", err)
	}
}

// --- parseBuildxVersion (unexported, called directly - same package) ---

func TestParseBuildxVersionNormalOutput(t *testing.T) {
	got := parseBuildxVersion("github.com/docker/buildx v0.36.1 d9f8b8f0abcdef\n")
	want := "v0.36.1"
	if got != want {
		t.Fatalf("parseBuildxVersion(normal output) = %q, want %q", got, want)
	}
}

// TestParseBuildxVersionNoVersionLikeToken pins the fallback: when nothing in
// the output looks like a version token, the trimmed output is returned as-is
// so the caller still has something to log.
func TestParseBuildxVersionNoVersionLikeToken(t *testing.T) {
	input := "  buildx plugin not found  \n"
	got := parseBuildxVersion(input)
	want := "buildx plugin not found"
	if got != want {
		t.Fatalf("parseBuildxVersion(no version token) = %q, want %q", got, want)
	}
}

func TestParseBuildxVersionEmptyOutput(t *testing.T) {
	got := parseBuildxVersion("")
	if got != "" {
		t.Fatalf("parseBuildxVersion(\"\") = %q, want \"\"", got)
	}
}

// TestParseBuildxVersionSuffixedVersion: a version like v0.36.1-desktop.1
// still starts with 'v' followed by a digit, so it must be picked out whole,
// suffix included, rather than truncated at the hyphen.
func TestParseBuildxVersionSuffixedVersion(t *testing.T) {
	got := parseBuildxVersion("github.com/docker/buildx v0.36.1-desktop.1 abcdef123456\n")
	want := "v0.36.1-desktop.1"
	if got != want {
		t.Fatalf("parseBuildxVersion(suffixed version) = %q, want %q", got, want)
	}
}

// --- DetectBuildx ---

func TestDetectBuildxSuccess(t *testing.T) {
	dir := t.TempDir()
	writeDockerShim(t, dir, 0, "github.com/docker/buildx v0.36.1 d9f8b8f0abcdef\n")
	t.Setenv("PATH", dir)

	ok, version, err := DetectBuildx()
	if err != nil {
		t.Fatalf("DetectBuildx returned an unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("DetectBuildx returned ok=false for a shim that exits 0, want true")
	}
	if version != "v0.36.1" {
		t.Fatalf("DetectBuildx version = %q, want %q", version, "v0.36.1")
	}
}

// TestDetectBuildxNonZeroExit pins the contract from docs/testing-plan.md
// Phase T2.4: every failure mode - including a docker binary that exists but
// exits non-zero (e.g. no buildx plugin installed) - returns exactly
// (false, "", nil). The error return is never used for this; that decision is
// ResolveBuilder's, not DetectBuildx's.
func TestDetectBuildxNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	writeDockerShim(t, dir, 1, "docker: 'buildx' is not a docker command.\n")
	t.Setenv("PATH", dir)

	ok, version, err := DetectBuildx()
	if ok != false || version != "" || err != nil {
		t.Fatalf("DetectBuildx(non-zero exit) = (%v, %q, %v), want (false, \"\", nil)", ok, version, err)
	}
}

// TestDetectBuildxAbsentFromPath pins the same (false, "", nil) contract when
// there is no `docker` executable on PATH at all.
func TestDetectBuildxAbsentFromPath(t *testing.T) {
	dir := t.TempDir() // empty directory, no docker binary
	t.Setenv("PATH", dir)

	ok, version, err := DetectBuildx()
	if ok != false || version != "" || err != nil {
		t.Fatalf("DetectBuildx(no docker on PATH) = (%v, %q, %v), want (false, \"\", nil)", ok, version, err)
	}
}
