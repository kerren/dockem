package utils

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"
)

// These are pure unit tests: they operate entirely inside t.TempDir() and never
// touch a registry or need any credentials.

func writeFileForHashTest(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

func hashOrFatal(t *testing.T, dir string, patterns []string) string {
	t.Helper()
	h, err := HashDirectory(dir, patterns)
	if err != nil {
		t.Fatalf("HashDirectory(%q, %v) returned an unexpected error: %v", dir, patterns, err)
	}
	return h
}

// TestHashDirectoryMatchesDirhashWhenNoPatterns is the load-bearing guarantee
// for this rung: with no exclude patterns, HashDirectory must be byte-identical
// to the dirhash.HashDir call it replaces, so that builds which have not opted
// into .dockerignore handling see no change to their cache identity.
func TestHashDirectoryMatchesDirhashWhenNoPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "README.md", "readme")
	writeFileForHashTest(t, dir, "src/main.go", "package main")
	writeFileForHashTest(t, dir, "src/util/helper.go", "package util")
	writeFileForHashTest(t, dir, "Dockerfile", "FROM scratch")
	writeFileForHashTest(t, dir, ".dockerignore", "node_modules\n")

	want, err := dirhash.HashDir(dir, "", dirhash.Hash1)
	if err != nil {
		t.Fatalf("dirhash.HashDir failed: %v", err)
	}

	for _, patterns := range [][]string{nil, {}} {
		if got := hashOrFatal(t, dir, patterns); got != want {
			t.Fatalf("HashDirectory(dir, %v) = %q, want byte-identical to dirhash.HashDir = %q", patterns, got, want)
		}
	}
}

// TestHashDirectoryExcludedFileDoesNotChangeHash: a file that matches an exclude
// pattern must not feed the hash.
func TestHashDirectoryExcludedFileDoesNotChangeHash(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "app.go", "package app")

	patterns := []string{"*.log"}
	before := hashOrFatal(t, dir, patterns)

	writeFileForHashTest(t, dir, "debug.log", "noise")
	after := hashOrFatal(t, dir, patterns)

	if before != after {
		t.Fatalf("adding an excluded file changed the hash: before=%q after=%q", before, after)
	}
}

// TestHashDirectoryNonMatchingFileChangesHash: a file that does not match any
// pattern must feed the hash.
func TestHashDirectoryNonMatchingFileChangesHash(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "app.go", "package app")

	patterns := []string{"*.log"}
	before := hashOrFatal(t, dir, patterns)

	writeFileForHashTest(t, dir, "data.json", "{}")
	after := hashOrFatal(t, dir, patterns)

	if before == after {
		t.Fatalf("adding a non-excluded file did not change the hash: %q", before)
	}
}

// TestHashDirectoryNegationPatternReincludesFile: a trailing "!keep.txt"
// re-includes a file that an earlier "*.txt" would have excluded.
func TestHashDirectoryNegationPatternReincludesFile(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "keep.txt", "keep me")
	writeFileForHashTest(t, dir, "drop.txt", "drop me")

	patterns := []string{"*.txt", "!keep.txt"}
	base := hashOrFatal(t, dir, patterns)

	// drop.txt stays excluded, so editing it must not move the hash.
	writeFileForHashTest(t, dir, "drop.txt", "drop me differently")
	if got := hashOrFatal(t, dir, patterns); got != base {
		t.Fatalf("editing an excluded file changed the hash: base=%q got=%q", base, got)
	}

	// keep.txt is re-included by the negation, so editing it MUST move the hash.
	writeFileForHashTest(t, dir, "keep.txt", "keep me differently")
	if got := hashOrFatal(t, dir, patterns); got == base {
		t.Fatalf("editing a re-included file did not change the hash: %q", got)
	}
}

// TestHashDirectoryAlwaysKeepsDockerfileAndDockerignore: even a "*" pattern that
// would exclude everything leaves the Dockerfile and .dockerignore in the hash,
// and editing .dockerignore itself must change the hash.
func TestHashDirectoryAlwaysKeepsDockerfileAndDockerignore(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "Dockerfile", "FROM scratch")
	writeFileForHashTest(t, dir, ".dockerignore", "*")
	writeFileForHashTest(t, dir, "app.go", "package app")

	patterns := []string{"*"}
	base := hashOrFatal(t, dir, patterns)

	// A normal file is excluded by "*", so editing it must not move the hash.
	writeFileForHashTest(t, dir, "app.go", "package app // changed")
	if got := hashOrFatal(t, dir, patterns); got != base {
		t.Fatalf("editing an excluded file changed the hash: base=%q got=%q", base, got)
	}

	// Editing .dockerignore MUST move the hash - it defines what is included.
	writeFileForHashTest(t, dir, ".dockerignore", "*\n# changed\n")
	afterIgnore := hashOrFatal(t, dir, patterns)
	if afterIgnore == base {
		t.Fatalf("editing .dockerignore did not change the hash: %q", afterIgnore)
	}

	// Editing the Dockerfile MUST move the hash too.
	writeFileForHashTest(t, dir, "Dockerfile", "FROM scratch\n# changed\n")
	if got := hashOrFatal(t, dir, patterns); got == afterIgnore {
		t.Fatalf("editing the Dockerfile did not change the hash: %q", got)
	}
}

// TestHashDirectorySkipsBrokenSymlink: a dangling symlink used to make os.Open
// fail and abort the whole hash. It must now be skipped, contributing nothing.
func TestHashDirectorySkipsBrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	writeFileForHashTest(t, dir, "real.txt", "content")

	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), filepath.Join(dir, "dangling")); err != nil {
		t.Skipf("symlinks are unsupported on this platform: %v", err)
	}

	got, err := HashDirectory(dir, nil)
	if err != nil {
		t.Fatalf("HashDirectory failed on a directory containing a broken symlink: %v", err)
	}

	// The broken link must contribute nothing, so a directory holding only the
	// real file hashes identically.
	clean := t.TempDir()
	writeFileForHashTest(t, clean, "real.txt", "content")
	want := hashOrFatal(t, clean, nil)

	if got != want {
		t.Fatalf("broken symlink altered the hash: got=%q want=%q", got, want)
	}
}
