package utils

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
)

// dockerAuthEntry is one registry's credentials inside a docker config.json.
// The "auth" field is the base64 of "username:password" - the same encoding the
// docker CLI writes for `docker login`.
type dockerAuthEntry struct {
	Auth string `json:"auth"`
}

// dockerConfigFile is the minimal subset of ~/.docker/config.json that buildx
// needs to authenticate a push: just the "auths" map.
type dockerConfigFile struct {
	Auths map[string]dockerAuthEntry `json:"auths"`
}

// dockerHubConfigKey is the config.json auths key the docker CLI uses for
// Docker Hub. An empty --registry means Docker Hub (see GenerateDockerImageName),
// so the credentials must be filed under this key rather than an empty string.
const dockerHubConfigKey = "https://index.docker.io/v1/"

// TempDockerConfig writes a throwaway docker config.json holding a single
// registry's credentials into a fresh temporary directory, and returns that
// directory's path plus a cleanup func that removes it.
//
// It exists for the buildx build path (wired in a later rung): `docker buildx`
// reads credentials from a config.json and cannot take a username/password on
// the command line, so when they are passed explicitly to dockem they must be
// handed to the subprocess without touching the user's real ~/.docker/config.json.
//
// Contract for the caller:
//
//   - Point the SUBPROCESS at the returned directory by adding
//     "DOCKER_CONFIG=<dir>" to exec.Cmd.Env only. Never os.Setenv it and never
//     export it into dockem's own process - that would leak the throwaway
//     config into everything dockem does after the build.
//   - Always `defer cleanup()` immediately, so the directory (and the password
//     on disk inside it) is removed even if the build fails or panics.
//
// When no explicit credentials are given (either username or password empty),
// there is nothing to write: TempDockerConfig returns an empty dir and a no-op
// cleanup. The caller must treat an empty dir as "leave the subprocess
// environment unchanged", so that an existing `docker login` - i.e. the user's
// real config.json, found via the inherited DOCKER_CONFIG/HOME - keeps working.
//
// The password is written only into the 0600 config.json inside the 0700
// directory. It is never logged, never placed on BuildLog, and never surfaced
// in the BuildResult JSON.
func TempDockerConfig(registry, username, password string) (string, func(), error) {
	noop := func() {}

	// No explicit credentials -> write nothing and signal "inherit the
	// environment unchanged" with an empty dir and a no-op cleanup.
	if username == "" || password == "" {
		return "", noop, nil
	}

	dir, err := os.MkdirTemp("", "dockem-docker-config-")
	if err != nil {
		LogError("Unable to create a temporary directory for the docker credentials. Please check the permissions of your system's temp directory.\n")
		return "", noop, err
	}

	// MkdirTemp already creates the directory 0700, but set it explicitly so the
	// guarantee is visible here rather than assumed of the stdlib.
	if chmodErr := os.Chmod(dir, 0o700); chmodErr != nil {
		os.RemoveAll(dir)
		return "", noop, chmodErr
	}

	// From here on the directory exists, so cleanup must be able to remove it.
	// os.RemoveAll is idempotent, so it is safe to call from both the error
	// paths below and the caller's deferred cleanup.
	cleanup := func() { os.RemoveAll(dir) }

	registryKey := registry
	if registryKey == "" {
		registryKey = dockerHubConfigKey
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	config := dockerConfigFile{
		Auths: map[string]dockerAuthEntry{
			registryKey: {Auth: auth},
		},
	}

	data, marshalErr := json.MarshalIndent(config, "", "  ")
	if marshalErr != nil {
		cleanup()
		return "", noop, marshalErr
	}

	configPath := filepath.Join(dir, "config.json")
	if writeErr := os.WriteFile(configPath, data, 0o600); writeErr != nil {
		LogError("Unable to write the temporary docker credentials file. Please check the permissions of your system's temp directory.\n")
		cleanup()
		return "", noop, writeErr
	}

	return dir, cleanup, nil
}
