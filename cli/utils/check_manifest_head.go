package utils

import (
	"context"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

func CheckManifestHead(tag string, ref ref.Ref, client *regclient.RegClient) bool {
	mOpts := []regclient.ManifestOpts{}
	LogInfo("Checking for the image hash %s on the registry\n", tag)
	_, manifestError := client.ManifestHead(context.Background(), ref, mOpts...)
	if manifestError != nil {
		LogInfo("The image hash %s does not exist on the registry or we were unable to pull it\n", tag)
		if strings.Contains(manifestError.Error(), "failed to request manifest head") {
			LogWarn("Unable to pull the details from the registry, please ensure you have the correct credentials.\n")
			LogWarn("The build will continue, but this should be investigated\n")
		}
		LogInfo("%s", manifestError.Error())
		return false
	}
	return true
}
