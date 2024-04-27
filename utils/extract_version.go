package utils

import (
	"fmt"
	"io"
	"os"
)

func ExtractVersion(versionFilePath string) (string, error) {

	versionFile, versionFileError := os.Open(versionFilePath)
	if versionFileError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to open the version file: %s\n", versionFilePath))
		return "", versionFileError
	}
	defer versionFile.Close()
	bytes, _ := io.ReadAll(versionFile)
	parsedVersionFile, parsedVersionFileError := ParseVersionFileJson(bytes)
	if parsedVersionFileError != nil {
		print(fmt.Sprintf("ERROR: An error ocurred when trying to parse the version file: %s\n", versionFilePath))
		return "", parsedVersionFileError
	}
	version := "v" + parsedVersionFile.Version
	print(fmt.Sprintf("The version of the image being built is: %s\n", version))
	return version, nil
}
