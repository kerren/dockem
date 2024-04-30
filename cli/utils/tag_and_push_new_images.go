package utils

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func TagAndPushNewImages(params BuildDockerImageParams, version string, localTag string, dockerClient *client.Client, pushOptions types.ImagePushOptions, buildLog *BuildLog) error {
	for _, tag := range params.Tag {
		versionTag := fmt.Sprintf("%s-%s", tag, version)
		targetImageName := GenerateDockerImageName(params.Registry, params.ImageName, versionTag)
		fmt.Printf("Pushing the image to the new tag: %s\n", targetImageName)
		err := TagAndPushImage(localTag, targetImageName, dockerClient, pushOptions)
		if err != nil {
			return err
		}
		buildLog.outputTags = append(buildLog.outputTags, targetImageName)
	}
	if len(params.Tag) == 0 && !params.Latest && !params.MainVersion {
		// At this point, we just deploy it straight to the main version
		mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
		fmt.Printf("WARN: No tags were specified and you have not selected the --latest flag, so the image will be deployed to the main version: %s\n", mainVersionImageName)
		err := TagAndPushImage(localTag, mainVersionImageName, dockerClient, pushOptions)
		if err != nil {
			return err
		}
		buildLog.outputTags = append(buildLog.outputTags, mainVersionImageName)
	}
	if params.Latest {
		latestImageName := GenerateDockerImageName(params.Registry, params.ImageName, "latest")
		fmt.Printf("You have selected the --latest flag, so the image will be deployed to the latest tag: %s\n", latestImageName)
		err := TagAndPushImage(localTag, latestImageName, dockerClient, pushOptions)
		if err != nil {
			return err
		}
		buildLog.outputTags = append(buildLog.outputTags, latestImageName)
	}
	if params.MainVersion {
		mainVersionImageName := GenerateDockerImageName(params.Registry, params.ImageName, version)
		fmt.Printf("You have selected the --main-version flag, so the image will be deployed to the main version: %s\n", mainVersionImageName)
		err := TagAndPushImage(localTag, mainVersionImageName, dockerClient, pushOptions)
		if err != nil {
			return err
		}
		buildLog.outputTags = append(buildLog.outputTags, mainVersionImageName)
	}
	return nil
}
