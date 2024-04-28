package utils

import (
	"fmt"
	"os"
)

func AssertStringNotEmpty(str string, flag string, errorMessage string) {
	if str != "" {
		return
	}
	if errorMessage == "" {
		errorMessage = "ERROR: The string for flag '%s' was not specified."
	}
	outputMessage := fmt.Sprintf(errorMessage, flag)
	fmt.Println(outputMessage)
	os.Exit(1)
}
