package utils

import (
	"fmt"
	"sort"

	"golang.org/x/mod/sumdb/dirhash"
)

// HashWatchDirectories Hashes the given list of directories and returns a combined hash. It will sort the list of watch directories before hashing to
// guarantee consistency. If a directory does not exist it will throw an error.
func HashWatchDirectories(watchDirectories []string) (string, error) {
	finalHash := ""
	if len(watchDirectories) > 0 {
		sort.Strings(watchDirectories)

		for _, directory := range watchDirectories {
			directoryHash, err := dirhash.HashDir(directory, "", dirhash.Hash1)
			if err != nil {
				fmt.Print("ERROR: An error ocurred when hashing the watch directories, please ensure they all exist, they are listed as follows:\n")
				for _, dir := range watchDirectories {
					fmt.Print(dir + "\n")
				}
				return "", err
			}
			finalHash += directoryHash
		}
		return finalHash, nil
	}
	return "", nil
}
