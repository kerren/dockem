package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/moby/patternmatcher"
	"golang.org/x/mod/sumdb/dirhash"
)

// HashDirectory hashes the contents of dir, dropping every file that matches
// one of excludePatterns. It is an exclusion-aware replacement for
// dirhash.HashDir(dir, "", dirhash.Hash1): with an empty excludePatterns it
// produces a byte-identical hash, so builds that have not opted into
// .dockerignore handling see no change to their cache identity.
//
// The excludePatterns passed here MUST be the very same slice handed to
// TarBuildContext, so that the files that feed the hash are exactly the files
// streamed to the daemon. See ReadDockerignore, which computes that slice once.
//
// Two files are always retained regardless of the patterns:
//   - Dockerfile     - mirrors Docker, which never lets you ignore the
//     Dockerfile out of its own build.
//   - .dockerignore  - it defines what is included, so a change to it must
//     change the hash and invalidate the cache.
//
// Broken (dangling) symlinks are skipped rather than aborting the whole hash:
// today a single dead link anywhere under the build directory makes os.Open
// fail and kills the build.
func HashDirectory(dir string, excludePatterns []string) (string, error) {
	files, err := dirhash.DirFiles(dir, "")
	if err != nil {
		return "", err
	}

	matcher, err := patternmatcher.New(excludePatterns)
	if err != nil {
		return "", err
	}

	kept := make([]string, 0, len(files))
	for _, file := range files {
		if !isAlwaysHashed(file) {
			excluded, matchErr := matcher.MatchesOrParentMatches(file)
			if matchErr != nil {
				return "", matchErr
			}
			if excluded {
				continue
			}
		}

		// Skip broken symlinks (and files that vanished mid-walk): os.Stat
		// follows the link and reports "not exist" when the target is missing,
		// whereas a real file always stats cleanly. Anything else is a genuine
		// error worth surfacing.
		fullPath := filepath.Join(dir, file)
		if _, statErr := os.Stat(fullPath); statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return "", statErr
		}

		kept = append(kept, file)
	}

	// This open closure joins dir exactly the way dirhash.HashDir does
	// internally (with an empty prefix), which is what keeps the empty-pattern
	// hash byte-identical to dirhash.HashDir.
	open := func(name string) (io.ReadCloser, error) {
		return os.Open(filepath.Join(dir, name))
	}

	return dirhash.Hash1(kept, open)
}

// isAlwaysHashed reports whether a context-relative path is one of the two
// files that survive the ignore patterns no matter what.
func isAlwaysHashed(file string) bool {
	return file == "Dockerfile" || file == ".dockerignore"
}
