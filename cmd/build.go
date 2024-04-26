package cmd

import (
	"dockem/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the new Docker image",
	Long: `Check the files or folders specified and compare the hash to what has already
been built. If it has been built, then skip the build and copy the tag,
otherwise, build the new image and push it to the specified tag(s).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("build called")
		// First we need to check that the required flags are set.
		// 1. Ensure that the directory flag is set and the directory exists
		directory, _ := cmd.Flags().GetString("directory")
		utils.AssertDirectoryExists(directory, "ERROR: The directory '%s' does not exist. Please specify the path to the directory you would like to build.")
		// 2. Ensure that the dockerfile-path flag is set and the file exists
		// 3. Ensure that the image-name flag is set
		// 4. Ensure that the version-file flag is set and the file exists
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringArrayP("watch-file", "w", []string{}, "Watch for changes on a specific file or files")
	buildCmd.Flags().StringArrayP("watch-directory", "W", []string{}, "Watch for changes in a directory or directories")
	buildCmd.Flags().StringP("directory", "d", "./", "(required) The directory that should be used as the context for the Docker build")
	buildCmd.Flags().StringArrayP("docker-build-flags", "b", []string{}, "Any additional build flags you would like to pass directly into the docker build command")
	buildCmd.Flags().StringP("dockerfile-path", "f", "./Dockerfile", "(required) The path to the Dockerfile that should be used to build the image")
	buildCmd.Flags().StringP("image-name", "i", "", "(required) The name of the image you are building")
	buildCmd.Flags().BoolP("latest", "l", false, "Whether to push the latest tag with this image")
	buildCmd.Flags().StringP("version-file", "F", "./package.json", "(required) The name of the JSON file that holds the version to be used in the build. This JSON file must have the 'version' key.")
	buildCmd.Flags().StringP("registry", "r", "", "The registry that should be used when pulling/pushing the image, Dockerhub is used by default")
	buildCmd.Flags().StringArrayP("tag", "t", []string{}, "The tag or tags that should be attached to image")
	buildCmd.Flags().StringP("docker-username", "u", "", "The username that should be used to authenticate the docker client. Ignore if you have already logged in.")
	buildCmd.Flags().StringP("docker-password", "p", "", "The password that should be used to authenticate the docker client. Ignore if you have already logged in.")

	buildCmd.Example = `$ dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable

$ dockem build --directory=./apps/backend --watch-directory=./libs/shared --dockerfile-path=./apps/backend/Dockerfile --image-name=my-repo/backend --tag=dev

$ dockem build --image-name=my-repo/backend --registry=eu.reg.io --docker-username=uname --docker-password=1234 --tag=alpha --tag=test

`
}
