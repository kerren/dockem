package utils

import (
	"fmt"
	"os"
	"slices"
)

// AssertOneOf exits the process with an error message if value is not one
// of allowed. It mirrors the other Assert* flag validators called from
// cmd/build.go before a build starts (see AssertStringNotEmpty,
// AssertFileExists, AssertDirectoryExists).
func AssertOneOf(value string, allowed []string, errorMessage string) {
	if slices.Contains(allowed, value) {
		return
	}
	if errorMessage == "" {
		errorMessage = "ERROR: The value '%s' is not one of the allowed values."
	}
	outputMessage := fmt.Sprintf(errorMessage, value)
	LogInfo("%s\n", outputMessage)
	os.Exit(1)
}
