package utils

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/fs"
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
//
// The directory is NOT only a credential store. DOCKER_CONFIG is also where
// buildx keeps its builder instances and its record of which builder is
// currently selected (in the "buildx" subdirectory), so a directory holding
// nothing but a config.json hides every builder the user has configured.
// buildx does not fail on that - it silently falls back to the "default"
// docker-driver instance, which cannot build multi-platform images, so an
// otherwise correct `--platform linux/amd64,linux/arm64` build dies with
// "Multi-platform build is not supported for the docker driver" purely because
// credentials were passed. carryBuildxState therefore reproduces that
// subdirectory in the throwaway directory: credentials stay isolated, builder
// selection survives.
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

	// Carry the buildx builder state across. A failure here is not fatal: the
	// build can still run on buildx's "default" instance, which is correct for
	// every single-platform build. Warn rather than abort so passing
	// credentials never turns a working build into a failed one.
	if carryErr := carryBuildxState(dir); carryErr != nil {
		LogWarn("Unable to carry your buildx builder configuration into the temporary docker config (%v). The build will fall back to the 'default' buildx instance, which cannot build multi-platform images. If you are building for more than one --platform, either drop --docker-username/--docker-password and use an existing 'docker login', or set BUILDX_BUILDER to the builder you want.\n", carryErr)
	}

	return dir, cleanup, nil
}

// sourceDockerConfigDir returns the docker config directory the buildx
// subprocess would have read had dockem not overridden DOCKER_CONFIG for it -
// i.e. dockem's own inherited DOCKER_CONFIG when set, and ~/.docker otherwise,
// which is the same resolution order the docker CLI itself uses. An empty
// return means there is no directory to read from, and the caller skips.
func sourceDockerConfigDir() string {
	if fromEnv := os.Getenv("DOCKER_CONFIG"); fromEnv != "" {
		return fromEnv
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return ""
	}

	return filepath.Join(home, ".docker")
}

// carryBuildxState reproduces the "buildx" subdirectory of the real docker
// config directory inside the throwaway one at dir, so that the subprocess sees
// the builders the user actually has (and the one they have selected) rather
// than an empty config that silently degrades to the "default" docker driver.
// See TempDockerConfig's doc comment for why that matters.
//
// A symlink is preferred: it is a single syscall, it costs nothing, and buildx
// keeps reading and WRITING the real state directory exactly as it would in a
// normal build. Cleanup is safe with it, because os.RemoveAll unlinks a symlink
// rather than following it into the real directory - a property
// TestTempDockerConfigCleanupLeavesRealBuildxStateIntact pins down, since
// getting it wrong would delete the user's builder configuration.
//
// Windows only permits symlink creation with Developer Mode or elevation, so a
// recursive copy is the fallback. The copy is read-only as far as the real
// state is concerned: buildx's writes land in the throwaway directory and are
// discarded with it, which loses nothing dockem is responsible for.
//
// Having no buildx directory at all is not an error - it means no builders are
// configured, and buildx's own "default" is then the correct instance to use.
func carryBuildxState(dir string) error {
	source := sourceDockerConfigDir()
	if source == "" {
		return nil
	}

	buildxDir := filepath.Join(source, "buildx")
	info, statErr := os.Stat(buildxDir)
	if statErr != nil || !info.IsDir() {
		return nil
	}

	target := filepath.Join(dir, "buildx")
	if symlinkErr := os.Symlink(buildxDir, target); symlinkErr == nil {
		return nil
	}

	return copyDirectory(buildxDir, target)
}

// copyDirectory recursively copies the tree at source to target, creating
// target and any directories beneath it 0700 and any files 0600 - the same
// permissions the rest of the throwaway config directory uses, since the tree
// is being copied into it. Symlinks and other irregular entries inside the
// source are skipped rather than followed: buildx's state is plain JSON, and
// following links out of the tree is exactly the behaviour this copy exists to
// avoid.
func copyDirectory(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}

		destination := filepath.Join(target, relative)

		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}

		// Skip anything that is not a regular file (symlinks, sockets, devices).
		if !entry.Type().IsRegular() {
			return nil
		}

		return copyFile(path, destination)
	})
}

// copyFile copies a single regular file to destination with 0600 permissions.
func copyFile(source, destination string) error {
	in, openErr := os.Open(source)
	if openErr != nil {
		return openErr
	}
	defer in.Close()

	out, createErr := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if createErr != nil {
		return createErr
	}
	defer out.Close()

	if _, copyErr := io.Copy(out, in); copyErr != nil {
		return copyErr
	}

	return out.Close()
}
