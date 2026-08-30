package utils

import (
	"os"
	"path/filepath"

	"github.com/moby/patternmatcher/ignorefile"
)

// ReadDockerignore resolves the single ignore/exclude pattern list that drives
// BOTH the input hash (HashDirectory / HashWatchDirectories) and the
// build-context tar (TarBuildContext). Computing it in one place is what keeps
// those two in lock-step: if the hash and the tar ever derived from different
// pattern lists, dockem would build something other than what it hashed, which
// is the worst failure this tool can have.
//
// It reads the .dockerignore at the root of contextDir - or the file named by
// ignoreFile when that override is supplied - parses it with
// ignorefile.ReadAll, then appends any extra patterns passed via --exclude. A
// missing ignore file is not an error: it simply contributes no patterns, so
// with no file and no --exclude the returned slice is empty and the resulting
// hash is identical to the pre-.dockerignore behaviour.
func ReadDockerignore(contextDir string, ignoreFile string, exclude []string) ([]string, error) {
	ignorePath := ignoreFile
	if ignorePath == "" {
		ignorePath = filepath.Join(contextDir, ".dockerignore")
	}

	patterns := []string{}

	file, openErr := os.Open(ignorePath)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			// A missing ignore file is normal (most build contexts do not have
			// one). Carry on with only the explicit --exclude patterns, if any.
			return append(patterns, exclude...), nil
		}
		LogError("An error ocurred when trying to open the ignore file, please check that it exists and is readable. You specified %s as the ignore file\n", ignorePath)
		return nil, openErr
	}
	defer file.Close()

	parsed, parseErr := ignorefile.ReadAll(file)
	if parseErr != nil {
		LogError("An error ocurred when trying to parse the ignore file %s\n", ignorePath)
		return nil, parseErr
	}

	patterns = append(patterns, parsed...)
	patterns = append(patterns, exclude...)

	return patterns, nil
}
