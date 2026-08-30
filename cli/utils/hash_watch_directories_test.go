package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// These are pure unit tests: they operate entirely inside t.TempDir() and
// never touch a registry or need any credentials.

func writeWatchDirFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

// TestHashWatchDirectoriesEmptyListReturnsEmptyString mirrors HashWatchFiles'
// "contributes nothing" contract: users who never adopt --watch-directory must
// see no effect on overallHash.
func TestHashWatchDirectoriesEmptyListReturnsEmptyString(t *testing.T) {
	got, err := HashWatchDirectories(nil, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(nil, nil) returned an unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("HashWatchDirectories(nil, nil) = %q, want exactly \"\"", got)
	}

	got, err = HashWatchDirectories([]string{}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories([]string{}, nil) returned an unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("HashWatchDirectories([]string{}, nil) = %q, want exactly \"\"", got)
	}
}

// TestHashWatchDirectoriesSortInvariance: the function sorts the slice before
// hashing, so the caller's original ordering must not affect the result.
func TestHashWatchDirectoriesSortInvariance(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeWatchDirFile(t, dirA, "a.txt", "alpha")
	writeWatchDirFile(t, dirB, "b.txt", "beta")

	forward, err := HashWatchDirectories([]string{dirA, dirB}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(forward order) returned an unexpected error: %v", err)
	}
	reverse, err := HashWatchDirectories([]string{dirB, dirA}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(reverse order) returned an unexpected error: %v", err)
	}

	if forward != reverse {
		t.Fatalf("HashWatchDirectories is order-dependent: forward=%q reverse=%q", forward, reverse)
	}
}

// TestHashWatchDirectoriesMultipleDirsConcatenate: the combined hash of two
// directories must differ from the hash of either directory alone - proving
// the per-directory hashes are actually concatenated rather than, say, the
// last one winning.
func TestHashWatchDirectoriesMultipleDirsConcatenate(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeWatchDirFile(t, dirA, "a.txt", "alpha")
	writeWatchDirFile(t, dirB, "b.txt", "beta")

	both, err := HashWatchDirectories([]string{dirA, dirB}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(both) returned an unexpected error: %v", err)
	}
	onlyA, err := HashWatchDirectories([]string{dirA}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(onlyA) returned an unexpected error: %v", err)
	}
	onlyB, err := HashWatchDirectories([]string{dirB}, nil)
	if err != nil {
		t.Fatalf("HashWatchDirectories(onlyB) returned an unexpected error: %v", err)
	}

	if both == onlyA || both == onlyB {
		t.Fatalf("combined hash %q collided with a single-directory hash (onlyA=%q onlyB=%q)", both, onlyA, onlyB)
	}
}

func TestHashWatchDirectoriesMissingDirectoryErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := HashWatchDirectories([]string{filepath.Join(dir, "does-not-exist")}, nil)
	if err == nil {
		t.Fatal("HashWatchDirectories with a missing directory returned a nil error, want an error")
	}
}

// TestHashWatchDirectoriesExcludePatternsMatchHashDirectory: HashWatchDirectories
// must apply excludePatterns the same way HashDirectory does directly, since
// the doc comment promises watch directories "behave consistently with" the
// build directory.
func TestHashWatchDirectoriesExcludePatternsMatchHashDirectory(t *testing.T) {
	dir := t.TempDir()
	writeWatchDirFile(t, dir, "app.go", "package app")

	patterns := []string{"*.log"}
	want, err := HashDirectory(dir, patterns)
	if err != nil {
		t.Fatalf("HashDirectory returned an unexpected error: %v", err)
	}
	got, err := HashWatchDirectories([]string{dir}, patterns)
	if err != nil {
		t.Fatalf("HashWatchDirectories returned an unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("HashWatchDirectories(single dir) = %q, want identical to HashDirectory = %q", got, want)
	}

	// An excluded file must not move HashWatchDirectories' result either.
	writeWatchDirFile(t, dir, "debug.log", "noise")
	after, err := HashWatchDirectories([]string{dir}, patterns)
	if err != nil {
		t.Fatalf("HashWatchDirectories returned an unexpected error: %v", err)
	}
	if after != got {
		t.Fatalf("adding an excluded file changed HashWatchDirectories' result: before=%q after=%q", got, after)
	}
}

// TestHashWatchDirectoriesMutatesCallerSliceInPlace pins a known finding from
// docs/testing-plan.md Phase T2.2: HashWatchDirectories calls sort.Strings
// directly on the slice it is given, so the caller's own slice - here,
// standing in for BuildDockerImageParams.WatchDirectory - comes back sorted
// even though the function only returns a hash string. This is documented as
// existing behaviour, not fixed, per the task's constraint against changing
// production behaviour.
func TestHashWatchDirectoriesMutatesCallerSliceInPlace(t *testing.T) {
	// Use literal path-like strings, not t.TempDir()'s unpredictable naming,
	// so the "before" order is known and the mutation is unambiguous.
	watchDirs := []string{"/z-should-end-up-second", "/a-should-end-up-first"}
	original := append([]string(nil), watchDirs...)

	// HashWatchDirectories will error because these paths don't exist, but
	// sort.Strings runs on the caller's slice before HashDirectory is ever
	// called, so the mutation happens regardless of the error.
	_, _ = HashWatchDirectories(watchDirs, nil)

	if watchDirs[0] != "/a-should-end-up-first" || watchDirs[1] != "/z-should-end-up-second" {
		t.Fatalf("expected sort.Strings to have reordered the caller's slice, got %v", watchDirs)
	}
	if watchDirs[0] == original[0] {
		t.Fatal("expected the caller's slice to have been reordered, but it matches the original order")
	}
}
