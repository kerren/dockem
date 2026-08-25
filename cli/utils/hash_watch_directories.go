package utils

import (
	"sort"
)

// HashWatchDirectories hashes each watch directory, applying the same
// excludePatterns as the build directory so that watch directories behave
// consistently with it. The patterns are resolved once in BuildDockerImage and
// threaded through unchanged.
func HashWatchDirectories(watchDirectories []string, excludePatterns []string) (string, error) {
	finalHash := ""
	if len(watchDirectories) > 0 {
		sort.Strings(watchDirectories)

		for _, directory := range watchDirectories {
			directoryHash, err := HashDirectory(directory, excludePatterns)
			if err != nil {
				LogError("An error ocurred when hashing the watch directories, please ensure they all exist, they are listed as follows:\n")
				for _, dir := range watchDirectories {
					LogInfo("%s", dir+"\n")
				}
				return "", err
			}
			finalHash += directoryHash
		}
		return finalHash, nil
	}
	return "", nil
}
