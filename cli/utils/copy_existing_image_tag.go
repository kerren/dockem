package utils

import (
	"fmt"

	"github.com/regclient/regclient/regclient"
)

func CopyExistingImageTag(params BuildDockerImageParams, version string, imageNameWithHash string, client regclient.RegClient) error {
	for _, tag := range params.Tag {
		tagVersion := fmt.Sprintf("%s-%s", tag, version)
		targetImageName := GenerateDockerImageName(params.Registry, params.ImageName, tagVersion)
		print(fmt.Sprintf("Copying the image to the new tag: %s\n", targetImageName))
		copyError := CopyDockerImage(client, imageNameWithHash, targetImageName)
		if copyError != nil {
			return copyError
		}
	}
	if len(params.Tag) == 0 && !params.Latest && !params.MainVersion {
		// At this point, we just deploy it straight to the main version
		mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
		print(fmt.Sprintf("WARN: No tags were specified and you have not selected the --latest flag, so the image will be copied to the main version: %s\n", mainVersionImageName))
		copyError := CopyDockerImage(client, imageNameWithHash, mainVersionImageName)
		if copyError != nil {
			return copyError
		}
	}
	if params.Latest {
		latestImageName := GenerateDockerImageName(params.Registry, params.ImageName, "latest")
		print(fmt.Sprintf("You have selected the --latest flag, so the image will be copied to the latest tag: %s\n", latestImageName))
		copyError := CopyDockerImage(client, imageNameWithHash, latestImageName)
		if copyError != nil {
			return copyError
		}
	}
	if params.MainVersion {
		mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
		print(fmt.Sprintf("You have selected the --main-version flag, so the image will be copied to the main version: %s\n", mainVersionImageName))
		copyError := CopyDockerImage(client, imageNameWithHash, mainVersionImageName)
		if copyError != nil {
			return copyError
		}
	}
	return nil
}
