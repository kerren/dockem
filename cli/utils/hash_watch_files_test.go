package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// These are pure unit tests: they operate entirely inside t.TempDir() and
// never touch a registry or need any credentials.

func writeWatchFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

// TestHashWatchFilesEmptyListReturnsEmptyString is load-bearing for cache
// identity: users who never adopt --watch-file must see the exact same
// "contributes nothing" behaviour as hashPlatforms does for --platform, so
// that overallHash is unaffected by a feature they never opted into.
func TestHashWatchFilesEmptyListReturnsEmptyString(t *testing.T) {
	got, err := HashWatchFiles(nil)
	if err != nil {
		t.Fatalf("HashWatchFiles(nil) returned an unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("HashWatchFiles(nil) = %q, want exactly \"\"", got)
	}

	got, err = HashWatchFiles([]string{})
	if err != nil {
		t.Fatalf("HashWatchFiles([]string{}) returned an unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("HashWatchFiles([]string{}) = %q, want exactly \"\"", got)
	}
}

// TestHashWatchFilesOrderInvariance: dirhash.Hash1 sorts internally, so the
// order the caller lists the files in must not affect the resulting hash.
func TestHashWatchFilesOrderInvariance(t *testing.T) {
	dir := t.TempDir()
	a := writeWatchFile(t, dir, "a.txt", "alpha")
	b := writeWatchFile(t, dir, "b.txt", "beta")

	forward, err := HashWatchFiles([]string{a, b})
	if err != nil {
		t.Fatalf("HashWatchFiles(forward order) returned an unexpected error: %v", err)
	}
	reverse, err := HashWatchFiles([]string{b, a})
	if err != nil {
		t.Fatalf("HashWatchFiles(reverse order) returned an unexpected error: %v", err)
	}

	if forward != reverse {
		t.Fatalf("HashWatchFiles is order-dependent: forward=%q reverse=%q", forward, reverse)
	}
}

func TestHashWatchFilesMissingFileErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := HashWatchFiles([]string{filepath.Join(dir, "does-not-exist.txt")})
	if err == nil {
		t.Fatal("HashWatchFiles with a missing file returned a nil error, want an error")
	}
}

func TestHashWatchFilesContentChangeChangesHash(t *testing.T) {
	dir := t.TempDir()
	path := writeWatchFile(t, dir, "watched.txt", "version one")

	before, err := HashWatchFiles([]string{path})
	if err != nil {
		t.Fatalf("HashWatchFiles returned an unexpected error: %v", err)
	}

	writeWatchFile(t, dir, "watched.txt", "version two")
	after, err := HashWatchFiles([]string{path})
	if err != nil {
		t.Fatalf("HashWatchFiles returned an unexpected error: %v", err)
	}

	if before == after {
		t.Fatalf("changing the content of a watched file did not change the hash: %q", before)
	}
}
