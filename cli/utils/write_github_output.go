package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// WriteGitHubOutput appends the build result to the file at $GITHUB_OUTPUT,
// the mechanism a GitHub Actions step uses to expose values to later steps
// as `steps.<id>.outputs.<name>`.
//
// It is a no-op - not an error - when $GITHUB_OUTPUT is unset, which is the
// normal case for every environment other than a GitHub Actions runner (eg.
// running dockem locally, or under another CI provider).
//
// The file is opened append-only and is never truncated: other steps in the
// same job read and write their own outputs via the same file, both before
// and after dockem runs, and none of that may be destroyed.
func WriteGitHubOutput(result BuildResult) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}

	file, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if openErr != nil {
		LogError("An error ocurred when opening $GITHUB_OUTPUT ('%s') to write the build result: %s\n", path, openErr)
		return openErr
	}
	defer file.Close()

	var output strings.Builder
	writeGitHubOutputLine(&output, "hash", result.Hash)
	writeGitHubOutputLine(&output, "cache-hit", strconv.FormatBool(result.CacheHit))
	writeGitHubOutputLine(&output, "image", result.Image)
	writeGitHubOutputLine(&output, "version", result.Version)
	writeGitHubOutputLine(&output, "primary-tag", result.PrimaryTag)
	writeGitHubOutputLine(&output, "tags", strings.Join(result.Tags, ","))
	writeGitHubOutputLine(&output, "platforms", strings.Join(result.Platforms, ","))

	if _, writeErr := file.WriteString(output.String()); writeErr != nil {
		LogError("An error ocurred when writing the build result to $GITHUB_OUTPUT ('%s'): %s\n", path, writeErr)
		return writeErr
	}
	return nil
}

// writeGitHubOutputLine appends one key/value pair in the format the
// $GITHUB_OUTPUT file expects. A value containing a newline must use the
// heredoc form (`key<<EOF` ... `EOF`) instead of plain `key=value`, since a
// bare newline inside the value would otherwise be parsed as the start of an
// unrelated key.
func writeGitHubOutputLine(output *strings.Builder, key string, value string) {
	if strings.Contains(value, "\n") {
		fmt.Fprintf(output, "%s<<EOF\n%s\nEOF\n", key, value)
		return
	}
	fmt.Fprintf(output, "%s=%s\n", key, value)
}
