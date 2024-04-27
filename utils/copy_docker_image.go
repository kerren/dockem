package utils

import (
	"context"
	"fmt"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

func CopyDockerImage(client *regclient.RegClient, fromImageName string, toImageName string) error {
	rFrom, errFrom := ref.New(fromImageName)
	if errFrom != nil {
		print(fmt.Sprintf("ERROR: Could not parse the from image name '%s'. Please ensure that the image name is correct.", fromImageName))
	}

	rTo, errTo := ref.New(toImageName)
	if errTo != nil {
		print(fmt.Sprintf("ERROR: Could not parse the to image name '%s'. Please ensure that the image name is correct.", toImageName))
	}

	opts := []regclient.ImageOpts{}

	error := client.ImageCopy(context.Background(), rFrom, rTo, opts...)

	if error != nil {
		print(fmt.Sprintf("ERROR: An error occurred when copying the image from '%s' to '%s'.", fromImageName, toImageName))
		return error
	}
	return nil
}
