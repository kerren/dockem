package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func AssertFileExists(path string, error_message string) {

	if error_message == "" {
		error_message = "ERROR: The file '%s' does not exist."
	}
	abs_directory, _ := filepath.Abs(path)
	if exists, _ := FileExists(abs_directory); !exists {
		error_message := fmt.Sprintf(error_message, abs_directory)
		fmt.Println(error_message)
		os.Exit(1)
	}
}
