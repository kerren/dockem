package cmd

import (
	"dockem/utils"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Build the new Docker image",
	Version: Version,
	Long: `Check the files or folders specified and compare the hash to what has already
been built. If it has been built, then skip the build and copy the tag,
otherwise, build the new image and push it to the specified tag(s).`,
	Run: func(cmd *cobra.Command, args []string) {
		// First we need to check that the required flags are set.
		// 1. Ensure that the directory flag is set and the directory exists
		directory, _ := cmd.Flags().GetString("directory")
		utils.AssertDirectoryExists(directory, "ERROR: The directory '%s' does not exist. Please specify the path to the directory you would like to build.")
		// 2. Ensure that the dockerfile-path flag is set and the file exists
		dockerfilePath, _ := cmd.Flags().GetString("dockerfile-path")
		utils.AssertFileExists(dockerfilePath, "ERROR: The file '%s' does not exist. Please specify the path to the Dockerfile you would like to use to build the image.")
		// 3. Ensure that the image-name flag is set
		imageName, _ := cmd.Flags().GetString("image-name")
		utils.AssertStringNotEmpty(imageName, "image-name", "ERROR: The image-name flag is required. Please specify the name of the image you would like to build, this usually includes the organisation or group as well eg. your-org/image-name.")
		// 4. Ensure that the version-file flag is set and the file exists
		versionFile, _ := cmd.Flags().GetString("version-file")
		utils.AssertFileExists(versionFile, "ERROR: The file '%s' does not exist. Please specify the path to the file that holds the version you would like to use in the build. This is a JSON file that must have the 'version' key.")
		// 5. Ensure that the output-format flag is one of the supported formats
		outputFormat, _ := cmd.Flags().GetString("output-format")
		utils.AssertOneOf(outputFormat, []string{"text", "json"}, "ERROR: The output-format '%s' is not valid. Please specify either 'text' or 'json'.")

		dockerPassword, _ := cmd.Flags().GetString("docker-password")
		dockerUsername, _ := cmd.Flags().GetString("docker-username")
		exclude, _ := cmd.Flags().GetStringArray("exclude")
		ignoreBuildDirectory, _ := cmd.Flags().GetBool("ignore-build-directory")
		ignoreFile, _ := cmd.Flags().GetString("ignore-file")
		latest, _ := cmd.Flags().GetBool("latest")
		// --respect-dockerignore is opt-in on this release (default false); the
		// --no-respect-dockerignore counterpart is provided so a pipeline can
		// pin the old behaviour once the default flips to true in v3.
		respectDockerignore, _ := cmd.Flags().GetBool("respect-dockerignore")
		noRespectDockerignore, _ := cmd.Flags().GetBool("no-respect-dockerignore")
		if noRespectDockerignore {
			respectDockerignore = false
		}
		outputFile, _ := cmd.Flags().GetString("output-file")
		registry, _ := cmd.Flags().GetString("registry")
		strictRegistry, _ := cmd.Flags().GetBool("strict-registry")
		tag, _ := cmd.Flags().GetStringArray("tag")
		watchDirectory, _ := cmd.Flags().GetStringArray("watch-directory")
		watchFile, _ := cmd.Flags().GetStringArray("watch-file")
		mainVersion, _ := cmd.Flags().GetBool("main-version")

		// Now we can create the build docker image params struct
		buildDockerImageParams := utils.BuildDockerImageParams{
			Directory:            directory,
			DockerPassword:       dockerPassword,
			DockerUsername:       dockerUsername,
			DockerfilePath:       dockerfilePath,
			Exclude:              exclude,
			IgnoreBuildDirectory: ignoreBuildDirectory,
			IgnoreFile:           ignoreFile,
			ImageName:            imageName,
			Latest:               latest,
			MainVersion:          mainVersion,
			OutputFile:           outputFile,
			OutputFormat:         outputFormat,
			Registry:             registry,
			RespectDockerignore:  respectDockerignore,
			StrictRegistry:       strictRegistry,
			Tag:                  tag,
			VersionFile:          versionFile,
			WatchDirectory:       watchDirectory,
			WatchFile:            watchFile,
		}

		// Finally, we push this off to the build docker image function
		buildLog, buildErr := utils.BuildDockerImage(buildDockerImageParams)
		buildResult := buildLog.Result()

		// Emit the (possibly partial) machine-readable result before acting on
		// buildErr, so that --output-format=json still reports the hash - and
		// anything else that was computed - even when the build itself failed.
		// WriteBuildOutput is a no-op for the default text format, so it is
		// always safe to call unconditionally here.
		writeErr := utils.WriteBuildOutput(buildResult, outputFormat, outputFile)

		if buildErr != nil {
			// The blank line separating the logs from the panic trace is
			// human-readable output, so it goes to stderr through LogInfo and
			// leaves stdout holding nothing but the JSON result.
			utils.LogInfo("\n\n")
			panic(buildErr)
		}

		if writeErr != nil {
			panic(writeErr)
		}

		if githubOutputErr := utils.WriteGitHubOutput(buildResult); githubOutputErr != nil {
			panic(githubOutputErr)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringArrayP("watch-file", "w", []string{}, "Watch for changes on a specific file or files")
	buildCmd.Flags().StringArrayP("watch-directory", "W", []string{}, "Watch for changes in a directory or directories")
	buildCmd.Flags().StringP("directory", "d", "./", "(required) The directory that should be used as the context for the Docker build")
	buildCmd.Flags().StringP("dockerfile-path", "f", "./Dockerfile", "(required) The path to the Dockerfile that should be used to build the image")
	buildCmd.Flags().StringP("image-name", "i", "", "(required) The name of the image you are building")
	buildCmd.Flags().BoolP("latest", "l", false, "Whether to push the latest tag with this image")
	buildCmd.Flags().StringP("version-file", "F", "./package.json", "(required) The name of the JSON file that holds the version to be used in the build. This JSON file must have the 'version' key.")
	buildCmd.Flags().StringP("registry", "r", "", "The registry that should be used when pulling/pushing the image, Dockerhub is used by default")
	buildCmd.Flags().StringArrayP("tag", "t", []string{}, "The tag or tags that should be attached to image")
	buildCmd.Flags().StringP("docker-username", "u", "", "The username that should be used to authenticate the docker client. Ignore if you have already logged in.")
	buildCmd.Flags().StringP("docker-password", "p", "", "The password that should be used to authenticate the docker client. Ignore if you have already logged in.")
	buildCmd.Flags().BoolP("main-version", "m", false, "Whether to push this as the main version of the repository. This is done automatically if you do not specify tags or the latest flag.")
	buildCmd.Flags().BoolP("ignore-build-directory", "I", false, "Whether to ignore the build directory in the hashing process, this is useful when you are watching a specific file or directory.")
	buildCmd.Flags().BoolP("strict-registry", "s", false, "Whether to abort the build if the registry cannot be reliably checked for the image hash (eg. an authentication failure or a rate limit), instead of logging a warning and continuing with the build.")
	buildCmd.Flags().String("output-format", "text", "The format to print the build result in, either 'text' (the default, today's human-readable logging) or 'json' (an indented JSON BuildResult on stdout, or --output-file if set). $GITHUB_OUTPUT is always written to when set, regardless of this flag.")
	buildCmd.Flags().String("output-file", "", "The path of a file to write the --output-format=json build result to, instead of stdout. Ignored when --output-format is not 'json'.")
	buildCmd.Flags().Bool("respect-dockerignore", false, "Respect the .dockerignore file in the build directory (and any --ignore-file) when hashing the inputs and building the context, excluding matching files from both. Opt-in for now; this default becomes true in v3, which will reset every published hash tag.")
	buildCmd.Flags().Bool("no-respect-dockerignore", false, "Explicitly do NOT respect the .dockerignore file. This is the default today; it exists so a pipeline can pin the old behaviour once the default flips to true in v3.")
	buildCmd.Flags().String("ignore-file", "", "The path to an alternative ignore file to use instead of <directory>/.dockerignore. Only consulted when --respect-dockerignore is set.")
	buildCmd.Flags().StringArray("exclude", []string{}, "Extra .dockerignore-style pattern(s) to exclude from both the input hash and the build context. Repeatable. Always applied, even without --respect-dockerignore.")

	buildCmd.Example = `$ dockem build --directory=./apps/backend --dockerfile-path=./devops/prod/backend/Dockerfile --image-name=my-repo/backend --tag=stable --main-version

$ dockem build --directory=./apps/backend --watch-directory=./libs/shared --dockerfile-path=./apps/backend/Dockerfile --image-name=my-repo/backend --tag=dev --latest

$ dockem build --image-name=my-repo/backend --registry=eu.reg.io --docker-username=uname --docker-password=1234 --tag=alpha --tag=test

$ dockem build --image-name=my-repo/backend --tag=stable --output-format=json | jq -r '.primaryTag'

`
}
