package utils

import (
	"context"
	"fmt"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

// CopyDockerImage Copies a Docker image within the same registry or across registries.
// If the source and destination are in the same registry, it uses manifest re-tagging
// to optimize the copy process without pulling and pushing layers.
func CopyDockerImage(client *regclient.RegClient, fromImageName string, toImageName string) error {
	rFrom, errFrom := ref.New(fromImageName)
	if errFrom != nil {
		fmt.Printf("ERROR: Could not parse the from image name '%s'. Please ensure that the image name is correct.\n", fromImageName)
		return errFrom
	}

	rTo, errTo := ref.New(toImageName)
	if errTo != nil {
		fmt.Printf("ERROR: Could not parse the to image name '%s'. Please ensure that the image name is correct.\n", toImageName)
		return errTo
	}

	opts := []regclient.ImageOpts{}

	err := client.ImageCopy(context.Background(), rFrom, rTo, opts...)

	if err != nil {
		fmt.Printf("ERROR: An error occurred when copying the image from '%s' to '%s'\n", fromImageName, toImageName)
		return err
	}
	return nil
}
