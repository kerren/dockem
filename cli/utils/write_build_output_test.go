package utils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written to it.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Error creating the pipe used to capture stdout: %s", err)
	}
	os.Stdout = writer

	fn()

	writer.Close()
	os.Stdout = original

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Error reading the captured stdout: %s", err)
	}
	return output
}

func TestWriteBuildOutputJSONToStdoutIsParseable(t *testing.T) {
	result := BuildResult{
		Hash:       "abc123",
		CacheHit:   true,
		Image:      "org/image:abc123",
		Version:    "v1.0.0",
		Tags:       []string{"org/image:stable-v1.0.0"},
		PrimaryTag: "org/image:stable-v1.0.0",
		Platforms:  []string{},
		Registry:   "",
		DurationMs: 42,
	}

	var writeErr error
	output := captureStdout(t, func() {
		writeErr = WriteBuildOutput(result, "json", "")
	})
	if writeErr != nil {
		t.Fatalf("Error writing the build output: %s", writeErr)
	}

	var decoded BuildResult
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("Expected valid JSON on stdout, got an error decoding it: %s\noutput was: %s", err, output)
	}
	if decoded.Hash != result.Hash || decoded.PrimaryTag != result.PrimaryTag {
		t.Errorf("Decoded JSON does not match the input result.\nexpected: %+v\ngot: %+v", result, decoded)
	}
}

func TestWriteBuildOutputTextEmitsNothing(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "should-not-be-created.json")

	var writeErr error
	output := captureStdout(t, func() {
		writeErr = WriteBuildOutput(BuildResult{Hash: "abc123"}, "text", path)
	})
	if writeErr != nil {
		t.Fatalf("Expected no error for format=text, got: %s", writeErr)
	}
	if len(output) != 0 {
		t.Errorf("Expected format=text to write nothing to stdout, got: %q", output)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("Expected format=text to not create an output file, but %s exists", path)
	}
}

func TestWriteBuildOutputJSONToFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "result.json")

	result := BuildResult{
		Hash:     "abc123",
		CacheHit: false,
		Tags:     []string{"org/image:v1.0.0"},
	}

	var writeErr error
	output := captureStdout(t, func() {
		writeErr = WriteBuildOutput(result, "json", path)
	})
	if writeErr != nil {
		t.Fatalf("Error writing the build output: %s", writeErr)
	}
	if len(output) != 0 {
		t.Errorf("Expected nothing written to stdout when --output-file is set, got: %q", output)
	}

	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Error reading the output file: %s", err)
	}

	var decoded BuildResult
	if err := json.Unmarshal(fileContents, &decoded); err != nil {
		t.Fatalf("Expected valid JSON in the output file, got an error decoding it: %s", err)
	}
	if decoded.Hash != result.Hash {
		t.Errorf("Expected the decoded hash to be %q, got %q", result.Hash, decoded.Hash)
	}
}
