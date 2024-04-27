package utils

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
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

func BuildDockerImage(params BuildDockerImageParams) error {
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
			return err
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
				return err
			}
			overallHash += directoryHash
		}
	}

	// Hash the build directory
	directoryHash, err := dirhash.HashDir(params.Directory, "", dirhash.Hash1)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the build directory, please ensure it exists and is not empty. You specified %s as the directory\n", params.Directory))
		return err
	}
	overallHash += directoryHash

	// Hash the Dockerfile
	dockerfileHash, err := dirhash.Hash1([]string{params.DockerfilePath}, osOpen)
	if err != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when hashing the Dockerfile, please ensure it exists. You specified %s as the Dockerfile\n", params.DockerfilePath))
		return err
	}
	overallHash += dockerfileHash

	// Now that we have the hash, we can check if this hash exist on the docker registry already.
	// For this, we'll need regclient because it allows us to interact with the registry instead
	// of just the docker daemon. https://github.com/regclient/regclient

	host := *config.HostNew()
	if params.Registry != "" {
		host.Hostname = params.Registry
	}
	if params.DockerUsername != "" {
		host.User = params.DockerUsername
	}
	if params.DockerPassword != "" {
		host.Pass = params.DockerPassword
	}

	client := regclient.New(regclient.WithConfigHost(host))

	// TODO: Implement this
	// manifestHead, _ := client.ManifestHead(context.Background(), ref.Ref{},)

}
