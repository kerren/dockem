package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func AssertDirectoryExists(path string, errorMessage string) {

	if errorMessage == "" {
		errorMessage = "ERROR: The directory '%s' does not exist."
	}
	absDirectory, _ := filepath.Abs(path)
	if exists, _ := DirectoryExists(absDirectory); !exists {
		outputMessage := fmt.Sprintf(errorMessage, absDirectory)
		fmt.Println(outputMessage)
		os.Exit(1)
	}
}
