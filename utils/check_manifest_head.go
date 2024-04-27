package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

func CheckManifestHead(tag string, ref ref.Ref, client *regclient.RegClient) bool {
	mOpts := []regclient.ManifestOpts{}
	print(fmt.Sprintf("Checking for the image hash %s on the registry\n", tag))
	_, manifestError := client.ManifestHead(context.Background(), ref, mOpts...)
	if manifestError != nil {
		print(fmt.Sprintf("The image hash %s does not exist on the registry or we were unable to pull it\n", tag))
		if strings.Contains(manifestError.Error(), "failed to request manifest head") {
			print("WARN: Unable to pull the details from the registry, please ensure you have the correct credentials.\n")
			print("WARN: The build will continue, but this should be investigated\n")
		}
		print(manifestError.Error())
		return false
	}
	return true
}
