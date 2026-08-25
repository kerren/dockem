package utils

import (
	"github.com/regclient/regclient/regclient"
)

func CopyExistingImageTag(params BuildDockerImageParams, version string, imageNameWithHash string, client *regclient.RegClient, buildLog *BuildLog) error {
	for _, resolved := range ResolveTargetTags(params, version) {
		switch resolved.Reason {
		case TagReasonTag:
			LogInfo("Copying the image to the new tag: %s\n", resolved.ImageName)
		case TagReasonFallback:
			LogWarn("No tags were specified and you have not selected the --latest flag, so the image will be copied to the main version: %s\n", resolved.ImageName)
		case TagReasonLatest:
			LogInfo("You have selected the --latest flag, so the image will be copied to the latest tag: %s\n", resolved.ImageName)
		case TagReasonMainVersion:
			LogInfo("You have selected the --main-version flag, so the image will be copied to the main version: %s\n", resolved.ImageName)
		}
		copyError := CopyDockerImage(*client, imageNameWithHash, resolved.ImageName)
		if copyError != nil {
			return copyError
		}
		buildLog.outputTags = append(buildLog.outputTags, resolved.ImageName)
	}
	return nil
}
