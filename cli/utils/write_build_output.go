package utils

import (
	"encoding/json"
	"os"
)

// WriteBuildOutput emits the machine-readable result of a build.
//
// format="json" marshals result as indented JSON to stdout, or to the file
// at path instead if one is given. This is the "pipe to jq" contract: only
// JSON may reach stdout in this mode, so this function must never call any
// of the LogInfo/LogWarn/LogError helpers (they already target stderr, see
// log.go) or otherwise write anything else to stdout.
//
// format="text" (the default) is a no-op - today's behaviour, where the
// human-readable log lines already written to stderr during the build are
// the only output.
func WriteBuildOutput(result BuildResult, format string, path string) error {
	if format != "json" {
		return nil
	}

	encoded, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		LogError("An error ocurred when marshalling the build result to JSON: %s\n", marshalErr)
		return marshalErr
	}
	encoded = append(encoded, '\n')

	if path == "" {
		if _, writeErr := os.Stdout.Write(encoded); writeErr != nil {
			LogError("An error ocurred when writing the build result to stdout: %s\n", writeErr)
			return writeErr
		}
		return nil
	}

	if writeErr := os.WriteFile(path, encoded, 0644); writeErr != nil {
		LogError("An error ocurred when writing the build result to %s: %s\n", path, writeErr)
		return writeErr
	}
	return nil
}
