package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestStdoutStaysPureOutsideWriteBuildOutput is a Phase T0 source-reading
// guard, in the same spirit as TestCacheFromCacheToExcludedFromImageHash in
// build_docker_image_cache_hash_test.go: the invariant it protects cannot be
// reached by any runtime assertion, only by reading dockem's own source.
//
// With --output-format=json, stdout must carry nothing but the JSON
// BuildResult (see write_build_output.go) - anything a piped consumer like
// `dockem build --output-format=json | jq ...` isn't expecting would corrupt
// the JSON it's trying to parse. That is exactly why CLAUDE.md mandates
// LogInfo/LogWarn/LogError (cli/utils/log.go, all writing to os.Stderr)
// instead of raw fmt.Print*/os.Stdout for every other message dockem emits.
// This test enforces that convention mechanically: it fails if fmt.Print,
// fmt.Printf, fmt.Println, or a direct write to os.Stdout appears anywhere
// in cli/utils/ or cli/cmd/, in a non-test source file, outside the one file
// that is deliberately allowed to write to stdout.
//
// Only non-test (*.go, not *_test.go) files are scanned. Test files never
// run as part of the built dockem binary, so a *_test.go file redirecting
// os.Stdout to capture output for an assertion (see write_build_output_test.go's
// captureStdout helper) is a test technique, not a stdout-purity violation -
// scoping to non-test files avoids having to special-case that pattern here.
func TestStdoutStaysPureOutsideWriteBuildOutput(t *testing.T) {
	// The one file allowed to write to stdout: it implements the JSON
	// result output itself, and is the sole reason stdout exists as a
	// dockem output channel at all.
	allowlist := map[string]bool{
		"write_build_output.go": true,
	}

	// Matches fmt.Print(, fmt.Printf(, fmt.Println(, and any direct
	// reference to os.Stdout (covering os.Stdout.Write(...), and any
	// fmt.Fprint*(os.Stdout, ...) call).
	violationPattern := regexp.MustCompile(`fmt\.Print(ln|f)?\(|os\.Stdout`)

	dirs := []string{
		"../utils",
		"../cmd",
	}

	var violations []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("could not read %s to scan for stdout-purity violations: %s", dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			if allowlist[name] {
				continue
			}
			path := filepath.Join(dir, name)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("could not read %s to scan for stdout-purity violations: %s", path, err)
			}
			for i, line := range strings.Split(string(src), "\n") {
				if violationPattern.MatchString(line) {
					violations = append(violations, path+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("found fmt.Print*/os.Stdout usage outside write_build_output.go - with --output-format=json, stdout must carry nothing but the JSON result, so use LogInfo/LogWarn/LogError (cli/utils/log.go) instead:\n%s", strings.Join(violations, "\n"))
	}
}

