package utils

import (
	"golang.org/x/mod/sumdb/dirhash"
)

// HashWatchFiles Hashes the given list of files and returns a combined hash. It will sort the list of watch files before hashing to
// guarantee consistency. If a file does not exist it will throw an error.
func HashWatchFiles(watchFiles []string) (string, error) {

	if len(watchFiles) == 0 {
		return "", nil
	}

	// Note: No need to sort the files as they are sorted in the Hash1 function
	watchFileHash, err := dirhash.Hash1(watchFiles, osOpen)
	if err != nil {
		print("ERROR: An error ocurred when hashing the watch files, please ensure they all exist, they are listed as follows:\n")
		for _, file := range watchFiles {
			LogInfo("%s", file+"\n")
		}
		return "", err
	}

	return watchFileHash, nil
}
