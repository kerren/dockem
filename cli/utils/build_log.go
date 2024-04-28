package utils

/**
 * This struct is used to save the process of the build and any variables as well.
 * I use this in testing to ensure that the expected outcomes are met.
 */
type BuildLog struct {
	customDockerfile bool
	customHost       bool
	dockerPassword   string
	dockerRegistry   string
	dockerUsername   string
	hashExists       bool
	hashedImageName  string
	imageHash        string
	localTag         string
	outputTags       []string
	version          string
}
