package utils

import (
	"fmt"
	"io"
	"os"
	"sort"

	"golang.org/x/mod/sumdb/dirhash"
)

type BuildDockerImageParams struct {
	WatchFile        []string
	WatchDirectory   []string
	Directory        string
	DockerBuildFlags []string
	DockerfilePath   string
	ImageName        string
	Latest           bool
	VersionFile      string
	Registry         string
	Tag              []string
	DockerUsername   string
	DockerPassword   string
}

func BuildDockerImage(params BuildDockerImageParams) {
	// I create a string that I append all of the hashes to
	overallHash := ""

	// This is the function that I'll use to open a SINGLE file
	osOpen := func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	}

	// Hash the watch files if they exist
	if len(params.WatchFile) > 0 {
		// Note: No need to sort the files as they are sorted in the Hash1 function
		watchFileHash, err := dirhash.Hash1(params.WatchFile, osOpen)
		if err != nil {
			print("ERROR: An error ocurred when hashing the watch files, please ensure they all exist, they are listed as follows:\n")
			for _, file := range params.WatchFile {
				print(file + "\n")
			}
			panic(err)
		}
		overallHash += watchFileHash
	}

	// Hash the watch directories if they exist
	if len(params.WatchDirectory) > 0 {
		sort.Strings(params.WatchDirectory)

		for _, directory := range params.WatchDirectory {
			directoryHash, err := dirhash.HashDir(directory, "", dirhash.Hash1)
			if err != nil {
				print("ERROR: An error ocurred when hashing the watch directories, please ensure they all exist, they are listed as follows:\n")
				for _, dir := range params.WatchDirectory {
					print(dir + "\n")
				}
				panic(err)
			}
			overallHash += directoryHash
		}
	}

	// Hash the build directory
	directoryHash, err := dirhash.HashDir(params.Directory, "", dirhash.Hash1)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the build directory, please ensure it exists and is not empty. You specified %s as the directory\n", params.Directory))
		panic(err)
	}
	overallHash += directoryHash

}
