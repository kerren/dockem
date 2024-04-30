package utils

import (
	"fmt"

	"golang.org/x/mod/sumdb/dirhash"
)

func HashWatchFiles(watchFiles []string) (string, error) {

	if len(watchFiles) > 0 {
		// Note: No need to sort the files as they are sorted in the Hash1 function
		watchFileHash, err := dirhash.Hash1(watchFiles, osOpen)
		if err != nil {
			print("ERROR: An error ocurred when hashing the watch files, please ensure they all exist, they are listed as follows:\n")
			for _, file := range watchFiles {
				fmt.Print(file + "\n")
			}
			return "", err
		}
		return watchFileHash, nil
	}
	return "", nil
}
