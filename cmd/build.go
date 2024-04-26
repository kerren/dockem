package cmd

import (
	"fmt"

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
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	buildCmd.Flags().StringArrayP("watch-file", "w", []string{}, "Watch for changes on a specific file or files")
	buildCmd.Flags().StringArrayP("watch-directory", "W", []string{}, "Watch for changes in a directory or directories")
	buildCmd.Flags().StringP("directory", "d", "./", "(required) The directory that should be used as the context for the Docker build")
	buildCmd.Flags().StringArrayP("docker-build-flags", "b", []string{}, "Any additional build flags you would like to pass directly into the docker build command")
	buildCmd.Flags().StringP("dockerfile-path", "f", "./Dockerfile", "(required) The path to the Dockerfile that should be used to build the image")
	buildCmd.Flags().StringP("image-name", "i", "", "(required) The name of the image you are building")
	buildCmd.Flags().BoolP("latest", "l", false, "Whether to push the latest tag with this image")
	buildCmd.Flags().StringP("version-file", "v", "./package.json", "(required) The name of the JSON file that holds the version to be used in the build. This JSON file must have the 'version' key.")
	buildCmd.Flags().StringP("registry", "r", "", "The registry that should be used when pulling/pushing the image, Dockerhub is used by default")
	buildCmd.Flags().StringArrayP("tag", "t", []string{}, "The tag or tags that should be attached to image")
	buildCmd.Flags().StringP("docker-username", "u", "", "The username that should be used to authenticate the docker client. Ignore if you have already logged in.")
	buildCmd.Flags().StringP("docker-password", "p", "", "The password that should be used to authenticate the docker client. Ignore if you have already logged in.")

	buildCmd.Example = `$ dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable

$ dockem build --directory=./apps/backend --watch-directory=./libs/shared --dockerfile-path=./apps/backend/Dockerfile --image-name=my-repo/backend --tag=dev

$ dockem build --image-name=my-repo/backend --registry=eu.reg.io --docker-username=uname --docker-password=1234 --tag=alpha --tag=test

`
}
