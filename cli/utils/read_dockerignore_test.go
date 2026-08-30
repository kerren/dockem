package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// These are pure unit tests: they operate entirely inside t.TempDir() and
// never touch a registry or need any credentials.

func writeDockerignore(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// TestReadDockerignoreMissingFileIsNotAnError: with no .dockerignore present
// and no --exclude patterns, the result is an empty (non-nil-shaped but empty)
// slice and no error - the doc comment promises this yields a hash identical
// to pre-.dockerignore behaviour.
func TestReadDockerignoreMissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()

	got, err := ReadDockerignore(dir, "", nil)
	if err != nil {
		t.Fatalf("ReadDockerignore with no .dockerignore present returned an unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ReadDockerignore with no .dockerignore present = %#v, want an empty slice", got)
	}
}

// TestReadDockerignoreMissingFileContributesOnlyExcludePatterns: still no
// error when the file is missing, but --exclude patterns must still come
// through.
func TestReadDockerignoreMissingFileContributesOnlyExcludePatterns(t *testing.T) {
	dir := t.TempDir()

	got, err := ReadDockerignore(dir, "", []string{"*.log", "tmp/"})
	if err != nil {
		t.Fatalf("ReadDockerignore returned an unexpected error: %v", err)
	}
	want := []string{"*.log", "tmp/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDockerignore with missing file = %#v, want %#v", got, want)
	}
}

// TestReadDockerignoreCommentsAndBlankLinesStripped relies on
// ignorefile.ReadAll to strip "#" comments and blank lines - this pins that
// ReadDockerignore passes that behaviour through untouched.
func TestReadDockerignoreCommentsAndBlankLinesStripped(t *testing.T) {
	dir := t.TempDir()
	writeDockerignore(t, filepath.Join(dir, ".dockerignore"), "# a comment\n\nnode_modules\n\n# another comment\n*.log\n")

	got, err := ReadDockerignore(dir, "", nil)
	if err != nil {
		t.Fatalf("ReadDockerignore returned an unexpected error: %v", err)
	}
	want := []string{"node_modules", "*.log"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDockerignore = %#v, want %#v", got, want)
	}
}

// TestReadDockerignoreIgnoreFileOverridesDefaultPath: when ignoreFile is
// supplied, it is used verbatim instead of <contextDir>/.dockerignore - even
// if a real .dockerignore also exists in contextDir.
func TestReadDockerignoreIgnoreFileOverridesDefaultPath(t *testing.T) {
	dir := t.TempDir()
	writeDockerignore(t, filepath.Join(dir, ".dockerignore"), "should-not-be-used\n")

	overridePath := filepath.Join(dir, "custom.ignore")
	writeDockerignore(t, overridePath, "should-be-used\n")

	got, err := ReadDockerignore(dir, overridePath, nil)
	if err != nil {
		t.Fatalf("ReadDockerignore returned an unexpected error: %v", err)
	}
	want := []string{"should-be-used"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDockerignore with ignoreFile override = %#v, want %#v", got, want)
	}
}

// TestReadDockerignoreUnreadableFileErrors pins that a present-but-unreadable
// ignore file surfaces an error rather than being treated like a missing one.
// Meaningless when the test process is root (root ignores mode bits), so we
// skip in that case rather than assert a false failure.
func TestReadDockerignoreUnreadableFileErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads for root, so this edge cannot be exercised here")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".dockerignore")
	writeDockerignore(t, path, "node_modules\n")

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("failed to chmod ignore file: %v", err)
	}
	defer os.Chmod(path, 0o644) // restore so t.TempDir() cleanup can remove it

	_, err := ReadDockerignore(dir, "", nil)
	if err == nil {
		t.Fatal("ReadDockerignore on an unreadable file returned a nil error, want an error")
	}
}

// TestReadDockerignoreExcludeAppearsAfterFilePatterns pins a real behavioural
// contract that is otherwise only implied by the code: --exclude patterns are
// appended AFTER the file's own patterns. That ordering is what lets a
// trailing "!keep-me" passed via --exclude re-include a path the
// .dockerignore file excluded - if the order were reversed, the file's later
// blanket pattern would win instead and the negation would have no effect.
func TestReadDockerignoreExcludeAppearsAfterFilePatterns(t *testing.T) {
	dir := t.TempDir()
	writeDockerignore(t, filepath.Join(dir, ".dockerignore"), "*.txt\n")

	got, err := ReadDockerignore(dir, "", []string{"!keep-me.txt"})
	if err != nil {
		t.Fatalf("ReadDockerignore returned an unexpected error: %v", err)
	}
	want := []string{"*.txt", "!keep-me.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDockerignore = %#v, want %#v (exclude patterns must be appended after file patterns)", got, want)
	}

	// Prove the ordering actually matters for HashDirectory, not just for the
	// returned slice's shape: with the file's "*.txt" followed by the
	// negation, keep-me.txt must survive into the hash while drop-me.txt does
	// not.
	writeDockerignore(t, filepath.Join(dir, "keep-me.txt"), "keep")
	writeDockerignore(t, filepath.Join(dir, "drop-me.txt"), "drop")

	base := hashOrFatal(t, dir, got)

	writeDockerignore(t, filepath.Join(dir, "drop-me.txt"), "drop, changed")
	if after := hashOrFatal(t, dir, got); after != base {
		t.Fatalf("editing drop-me.txt (still excluded by *.txt) changed the hash: base=%q after=%q", base, after)
	}

	writeDockerignore(t, filepath.Join(dir, "keep-me.txt"), "keep, changed")
	if after := hashOrFatal(t, dir, got); after == base {
		t.Fatalf("editing keep-me.txt (re-included by !keep-me.txt) did not change the hash: %q", after)
	}
}
