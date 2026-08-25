package utils

import (
	"fmt"
	"os"
)

// LogInfo writes a formatted, human-readable informational message to stderr.
func LogInfo(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

// LogWarn writes a formatted, human-readable warning message to stderr,
// prepending the "WARN: " prefix that callers previously included in the
// message itself.
func LogWarn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format, a...)
}

// LogError writes a formatted, human-readable error message to stderr,
// prepending the "ERROR: " prefix that callers previously included in the
// message itself.
func LogError(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format, a...)
}
