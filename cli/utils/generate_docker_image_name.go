package utils

import "fmt"

func GenerateDockerImageName(registry string, imageName string, tag string) string {
	if registry != "" {
		return fmt.Sprintf("%s/%s:%s", registry, imageName, tag)
	}
	return fmt.Sprintf("%s:%s", imageName, tag)
}
