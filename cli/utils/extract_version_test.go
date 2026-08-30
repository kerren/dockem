package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// These are pure unit tests: they operate entirely inside t.TempDir() and
// never touch a registry or need any credentials.

func writeVersionFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

func TestExtractVersionValid(t *testing.T) {
	dir := t.TempDir()
	path := writeVersionFile(t, dir, "version.json", `{"version": "1.0.0"}`)

	got, err := ExtractVersion(path)
	if err != nil {
		t.Fatalf("ExtractVersion returned an unexpected error: %v", err)
	}
	if got != "v1.0.0" {
		t.Fatalf("ExtractVersion() = %q, want %q", got, "v1.0.0")
	}
}

func TestExtractVersionMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ExtractVersion(filepath.Join(dir, "does-not-exist.json"))
	if err == nil {
		t.Fatal("ExtractVersion on a missing file returned a nil error, want an error")
	}
}

// TestExtractVersionUnreadableFile pins that a version file whose permissions
// deny read access surfaces an error rather than being silently skipped. This
// subtest is meaningless when the test process itself is root, since root
// ignores the file mode bits and the open succeeds anyway - in that case we
// skip rather than assert a false failure.
func TestExtractVersionUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads for root, so this edge cannot be exercised here")
	}

	dir := t.TempDir()
	path := writeVersionFile(t, dir, "version.json", `{"version": "1.0.0"}`)

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("failed to chmod version file: %v", err)
	}
	defer os.Chmod(path, 0o644) // restore so t.TempDir() cleanup can remove it

	_, err := ExtractVersion(path)
	if err == nil {
		t.Fatal("ExtractVersion on an unreadable file returned a nil error, want an error")
	}
}

// TestExtractVersionEmptyObjectYieldsVPrefix pins a currently-silent edge
// (see docs/testing-plan.md Phase T2.1): a version file that is valid JSON but
// has no `version` key at all does not error - it produces the tag-looking
// string "v". This is documented as a deliberate, known edge, not a fix
// target: ExtractVersion has no way to distinguish "key absent" from "key
// present but empty" once json.Unmarshal has run.
func TestExtractVersionEmptyObjectYieldsVPrefix(t *testing.T) {
	dir := t.TempDir()
	path := writeVersionFile(t, dir, "version.json", `{}`)

	got, err := ExtractVersion(path)
	if err != nil {
		t.Fatalf("ExtractVersion returned an unexpected error: %v", err)
	}
	if got != "v" {
		t.Fatalf("ExtractVersion(\"{}\") = %q, want %q (pinned known edge)", got, "v")
	}
}

// TestExtractVersionDoubleVPrefix pins the other half of the same known edge:
// a version value that already carries a leading "v" (e.g. copied from a git
// tag) is not detected or stripped, so the returned tag doubles up as "vv...".
// Either behaviour is a plausible tag name and neither fails loudly today, so
// this test documents the current output rather than asserting it is correct.
func TestExtractVersionDoubleVPrefix(t *testing.T) {
	dir := t.TempDir()
	path := writeVersionFile(t, dir, "version.json", `{"version": "v1.0.0"}`)

	got, err := ExtractVersion(path)
	if err != nil {
		t.Fatalf("ExtractVersion returned an unexpected error: %v", err)
	}
	if got != "vv1.0.0" {
		t.Fatalf("ExtractVersion(%q) = %q, want %q (pinned known edge)", `{"version": "v1.0.0"}`, got, "vv1.0.0")
	}
}
