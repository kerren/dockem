package utils

import "fmt"

// GenerateDockerImageName Generates a docker image name with the format `org/imageName:hash` or `imageName:hash`
func GenerateDockerImageName(registry string, imageName string, tag string) string {
	if registry != "" {
		return fmt.Sprintf("%s/%s:%s", registry, imageName, tag)
	}
	return fmt.Sprintf("%s:%s", imageName, tag)
}
