package utils

/**
 * This struct is used to save the process of the build and any variables as well.
 * I use this in testing to ensure that the expected outcomes are met.
 */
type BuildLog struct {
	imageHash       string
	version         string
	hashExists      bool
	outputTags      []string
	hashedImageName string
	localTag        string
}
