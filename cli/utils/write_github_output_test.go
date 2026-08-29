package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newGitHubOutputTempFile creates an empty temp file to stand in for the
// file $GITHUB_OUTPUT normally points at (a real GitHub Actions runner
// always pre-creates it), seeded with the given content so tests can assert
// that WriteGitHubOutput appends rather than truncates.
func newGitHubOutputTempFile(t *testing.T, seed string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(path, []byte(seed), 0644); err != nil {
		t.Fatalf("Error seeding the temp $GITHUB_OUTPUT file: %s", err)
	}
	return path
}

func TestWriteGitHubOutputNoopWhenUnset(t *testing.T) {
	if _, isSet := os.LookupEnv("GITHUB_OUTPUT"); isSet {
		t.Setenv("GITHUB_OUTPUT", "")
		os.Unsetenv("GITHUB_OUTPUT")
	}

	err := WriteGitHubOutput(BuildResult{Hash: "abc123"})
	if err != nil {
		t.Fatalf("Expected no error when $GITHUB_OUTPUT is unset, got: %s", err)
	}
}

func TestWriteGitHubOutputAppendsExactKeyValueLines(t *testing.T) {
	preExisting := "SOME_EARLIER_STEP=already-here\n"
	path := newGitHubOutputTempFile(t, preExisting)
	t.Setenv("GITHUB_OUTPUT", path)

	result := BuildResult{
		Hash:       "abc123def456",
		CacheHit:   true,
		Image:      "my-registry.io/my-org/my-image:abc123def456",
		Version:    "v1.2.3",
		Tags:       []string{"my-org/my-image:stable-v1.2.3", "my-org/my-image:latest"},
		PrimaryTag: "my-org/my-image:stable-v1.2.3",
		Platforms:  []string{},
		Registry:   "my-registry.io",
		DurationMs: 4200,
	}

	if err := WriteGitHubOutput(result); err != nil {
		t.Fatalf("Error writing the GitHub output: %s", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Error reading the temp $GITHUB_OUTPUT file: %s", err)
	}

	expected := preExisting +
		"hash=abc123def456\n" +
		"cache-hit=true\n" +
		"image=my-registry.io/my-org/my-image:abc123def456\n" +
		"version=v1.2.3\n" +
		"primary-tag=my-org/my-image:stable-v1.2.3\n" +
		"tags=my-org/my-image:stable-v1.2.3,my-org/my-image:latest\n" +
		"platforms=\n"

	if string(contents) != expected {
		t.Errorf("Unexpected $GITHUB_OUTPUT contents.\n--- expected ---\n%s\n--- got ---\n%s", expected, contents)
	}
}

func TestWriteGitHubOutputDoesNotTruncateExistingContent(t *testing.T) {
	preExisting := "FIRST_STEP_KEY=first-step-value\nSECOND_STEP_KEY=second-step-value\n"
	path := newGitHubOutputTempFile(t, preExisting)
	t.Setenv("GITHUB_OUTPUT", path)

	if err := WriteGitHubOutput(BuildResult{Hash: "abc123"}); err != nil {
		t.Fatalf("Error writing the GitHub output: %s", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Error reading the temp $GITHUB_OUTPUT file: %s", err)
	}

	if !strings.HasPrefix(string(contents), preExisting) {
		t.Errorf("Expected the pre-existing content to be preserved (not truncated) at the start of the file, got:\n%s", contents)
	}
}

func TestWriteGitHubOutputUsesHeredocForMultilineValues(t *testing.T) {
	path := newGitHubOutputTempFile(t, "")
	t.Setenv("GITHUB_OUTPUT", path)

	result := BuildResult{
		Hash:  "abc123",
		Image: "org/image:abc123\nwith-a-second-line",
	}

	if err := WriteGitHubOutput(result); err != nil {
		t.Fatalf("Error writing the GitHub output: %s", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Error reading the temp $GITHUB_OUTPUT file: %s", err)
	}

	if !strings.Contains(string(contents), "image<<EOF\norg/image:abc123\nwith-a-second-line\nEOF\n") {
		t.Errorf("Expected the image output to use the key<<EOF heredoc form, got:\n%s", contents)
	}
	if strings.Contains(string(contents), "image=org/image:abc123") {
		t.Errorf("Expected the image output to NOT use the plain key=value form when the value contains a newline, got:\n%s", contents)
	}
}
