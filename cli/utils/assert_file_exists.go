package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func AssertFileExists(path string, errorMessage string) {

	if errorMessage == "" {
		errorMessage = "ERROR: The file '%s' does not exist."
	}
	absFilePath, _ := filepath.Abs(path)
	if exists, _ := FileExists(absFilePath); !exists {
		outputMessage := fmt.Sprintf(errorMessage, absFilePath)
		fmt.Println(outputMessage)
		os.Exit(1)
	}
}
