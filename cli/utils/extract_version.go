package utils

import (
	"io"
	"os"
)

func ExtractVersion(versionFilePath string) (string, error) {

	versionFile, versionFileError := os.Open(versionFilePath)
	if versionFileError != nil {
		LogError("An error ocurred when trying to open the version file: %s\n", versionFilePath)
		return "", versionFileError
	}
	defer versionFile.Close()
	bytes, _ := io.ReadAll(versionFile)
	parsedVersionFile, parsedVersionFileError := ParseVersionFileJson(bytes)
	if parsedVersionFileError != nil {
		LogError("An error ocurred when trying to parse the version file: %s\n", versionFilePath)
		return "", parsedVersionFileError
	}
	version := "v" + parsedVersionFile.Version
	LogInfo("The version of the image being built is: %s\n", version)
	return version, nil
}
