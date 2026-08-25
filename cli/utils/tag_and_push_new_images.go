package utils

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func TagAndPushNewImages(params BuildDockerImageParams, version string, localTag string, dockerClient *client.Client, pushOptions types.ImagePushOptions, buildLog *BuildLog) error {
	for _, resolved := range ResolveTargetTags(params, version) {
		switch resolved.Reason {
		case TagReasonTag:
			LogInfo("Pushing the image to the new tag: %s\n", resolved.ImageName)
		case TagReasonFallback:
			LogWarn("No tags were specified and you have not selected the --latest flag, so the image will be deployed to the main version: %s\n", resolved.ImageName)
		case TagReasonLatest:
			LogInfo("You have selected the --latest flag, so the image will be deployed to the latest tag: %s\n", resolved.ImageName)
		case TagReasonMainVersion:
			LogInfo("You have selected the --main-version flag, so the image will be deployed to the main version: %s\n", resolved.ImageName)
		}
		err := TagAndPushImage(localTag, resolved.ImageName, dockerClient, pushOptions)
		if err != nil {
			return err
		}
		buildLog.outputTags = append(buildLog.outputTags, resolved.ImageName)
	}
	return nil
}
