package utils

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These are pure unit tests: they operate entirely on a temporary directory and
// never touch a registry, a docker daemon, or any real credentials.

// TestTempDockerConfigWritesExpectedFile checks the whole contract for the
// credentialled case: a 0700 directory holding a 0600 config.json whose single
// auths entry base64-encodes exactly "username:password", and a cleanup that
// removes the directory.
func TestTempDockerConfigWritesExpectedFile(t *testing.T) {
	registry := "eu.reg.io"
	username := "uname"
	password := "s3cr3t-pa55word"

	dir, cleanup, err := TempDockerConfig(registry, username, password)
	if err != nil {
		t.Fatalf("TempDockerConfig returned an unexpected error: %v", err)
	}
	if dir == "" {
		t.Fatalf("TempDockerConfig returned an empty dir for a credentialled call")
	}
	defer cleanup()

	// Directory permissions must be 0700.
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Fatalf("could not stat the temp dir: %v", statErr)
		}
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Fatalf("temp dir permissions = %o, want 0700", perm)
		}
	}

	configPath := filepath.Join(dir, "config.json")

	// File permissions must be 0600.
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(configPath)
		if statErr != nil {
			t.Fatalf("could not stat config.json: %v", statErr)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("config.json permissions = %o, want 0600", perm)
		}
	}

	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("could not read config.json: %v", readErr)
	}

	var parsed struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if unmarshalErr := json.Unmarshal(data, &parsed); unmarshalErr != nil {
		t.Fatalf("config.json is not valid JSON: %v\ncontents: %s", unmarshalErr, data)
	}

	entry, ok := parsed.Auths[registry]
	if !ok {
		t.Fatalf("config.json has no auths entry for %q; got keys %v", registry, keysOf(parsed.Auths))
	}

	decoded, decodeErr := base64.StdEncoding.DecodeString(entry.Auth)
	if decodeErr != nil {
		t.Fatalf("the auth field is not valid base64: %v", decodeErr)
	}
	if want := username + ":" + password; string(decoded) != want {
		t.Fatalf("decoded auth = %q, want %q", decoded, want)
	}
}

// TestTempDockerConfigEmptyRegistryUsesDockerHubKey checks that an empty
// registry (which means Docker Hub) files the credentials under the docker CLI's
// Docker Hub key rather than an empty string.
func TestTempDockerConfigEmptyRegistryUsesDockerHubKey(t *testing.T) {
	dir, cleanup, err := TempDockerConfig("", "uname", "pw")
	if err != nil {
		t.Fatalf("TempDockerConfig returned an unexpected error: %v", err)
	}
	defer cleanup()

	data, readErr := os.ReadFile(filepath.Join(dir, "config.json"))
	if readErr != nil {
		t.Fatalf("could not read config.json: %v", readErr)
	}
	if !strings.Contains(string(data), dockerHubConfigKey) {
		t.Fatalf("config.json does not file the empty registry under %q:\n%s", dockerHubConfigKey, data)
	}
}

// TestTempDockerConfigCleanupRemovesDirectory checks that the returned cleanup
// actually removes the temp directory, so a caller's defer leaves nothing on
// disk (including the password that lived inside it).
func TestTempDockerConfigCleanupRemovesDirectory(t *testing.T) {
	dir, cleanup, err := TempDockerConfig("eu.reg.io", "uname", "pw")
	if err != nil {
		t.Fatalf("TempDockerConfig returned an unexpected error: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("expected the temp dir to exist before cleanup: %v", statErr)
	}

	cleanup()

	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("expected the temp dir to be gone after cleanup, stat err = %v", statErr)
	}

	// cleanup must be idempotent - a second call (eg. from a deferred cleanup on
	// top of one already run) must not panic or error.
	cleanup()
}

// TestTempDockerConfigNoCredentialsIsNoop checks that when either credential is
// empty, nothing is written and the returned dir is empty (the caller's signal
// to leave the subprocess environment - and thus an existing `docker login` -
// untouched). The no-op cleanup must be safe to defer.
func TestTempDockerConfigNoCredentialsIsNoop(t *testing.T) {
	cases := []struct {
		name     string
		username string
		password string
	}{
		{"both empty", "", ""},
		{"username only", "uname", ""},
		{"password only", "", "pw"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup, err := TempDockerConfig("eu.reg.io", tc.username, tc.password)
			if err != nil {
				t.Fatalf("TempDockerConfig returned an unexpected error: %v", err)
			}
			if dir != "" {
				t.Fatalf("expected an empty dir when credentials are incomplete, got %q", dir)
			}
			if cleanup == nil {
				t.Fatalf("expected a non-nil no-op cleanup even with no credentials")
			}
			// Must be safe to call.
			cleanup()
		})
	}
}

func keysOf(m map[string]struct {
	Auth string `json:"auth"`
}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// buildxStateFixture creates a directory that stands in for a real docker config
// directory, with a buildx state subdirectory holding a selected builder and one
// instance, and returns the config directory's path. It mirrors the layout
// buildx actually writes: a "current" file naming the selected builder and an
// "instances" directory describing it.
func buildxStateFixture(t *testing.T) string {
	t.Helper()

	configDir := t.TempDir()
	instancesDir := filepath.Join(configDir, "buildx", "instances")
	if err := os.MkdirAll(instancesDir, 0o700); err != nil {
		t.Fatalf("Unable to create the buildx fixture: %s", err)
	}

	current := filepath.Join(configDir, "buildx", "current")
	if err := os.WriteFile(current, []byte(`{"Name":"builder-under-test"}`), 0o600); err != nil {
		t.Fatalf("Unable to write the buildx current file: %s", err)
	}

	instance := filepath.Join(instancesDir, "builder-under-test")
	if err := os.WriteFile(instance, []byte(`{"Driver":"docker-container"}`), 0o600); err != nil {
		t.Fatalf("Unable to write the buildx instance file: %s", err)
	}

	return configDir
}

func TestTempDockerConfigCarriesBuildxStateFromDockerConfig(t *testing.T) {
	// The regression this pins: without the buildx state, the subprocess sees no
	// builders and falls back to the "default" docker driver, which cannot build
	// multi-platform images - so passing credentials broke --platform builds.
	configDir := buildxStateFixture(t)
	t.Setenv("DOCKER_CONFIG", configDir)

	dir, cleanup, err := TempDockerConfig("", "user", "password")
	if err != nil {
		t.Fatalf("TempDockerConfig returned an error: %s", err)
	}
	defer cleanup()

	// Read through whatever carryBuildxState produced (symlink or copy): what
	// matters is that the subprocess reading DOCKER_CONFIG=dir sees the builder.
	current, readErr := os.ReadFile(filepath.Join(dir, "buildx", "current"))
	if readErr != nil {
		t.Fatalf("The buildx current file is not readable through the temporary config: %s", readErr)
	}
	if string(current) != `{"Name":"builder-under-test"}` {
		t.Errorf("The buildx current file has the wrong contents: %s", string(current))
	}

	instance, instanceErr := os.ReadFile(filepath.Join(dir, "buildx", "instances", "builder-under-test"))
	if instanceErr != nil {
		t.Fatalf("The buildx instance file is not readable through the temporary config: %s", instanceErr)
	}
	if string(instance) != `{"Driver":"docker-container"}` {
		t.Errorf("The buildx instance file has the wrong contents: %s", string(instance))
	}

	// The credentials must still be there - carrying the state must not have
	// replaced or hidden the config.json.
	if _, configErr := os.Stat(filepath.Join(dir, "config.json")); configErr != nil {
		t.Errorf("The config.json is missing after the buildx state was carried: %s", configErr)
	}
}

func TestTempDockerConfigCleanupLeavesRealBuildxStateIntact(t *testing.T) {
	// carryBuildxState prefers a symlink, so cleanup runs os.RemoveAll over a
	// directory containing a link to the user's REAL buildx state. os.RemoveAll
	// unlinks a symlink rather than following it, but that is the difference
	// between removing a link and deleting every builder the user has, so it is
	// worth a test rather than an assumption.
	configDir := buildxStateFixture(t)
	t.Setenv("DOCKER_CONFIG", configDir)

	dir, cleanup, err := TempDockerConfig("", "user", "password")
	if err != nil {
		t.Fatalf("TempDockerConfig returned an error: %s", err)
	}

	cleanup()

	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("The temporary directory should have been removed, got: %v", statErr)
	}

	for _, path := range []string{
		filepath.Join(configDir, "buildx", "current"),
		filepath.Join(configDir, "buildx", "instances", "builder-under-test"),
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("Cleanup deleted the real buildx state at %s: %s", path, statErr)
		}
	}
}

func TestTempDockerConfigWithoutBuildxStateIsNotAnError(t *testing.T) {
	// No buildx directory means no builders are configured, and buildx's own
	// "default" instance is then the right one to use. That is not a failure and
	// must not create an empty buildx directory that misrepresents the state.
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	dir, cleanup, err := TempDockerConfig("", "user", "password")
	if err != nil {
		t.Fatalf("TempDockerConfig returned an error: %s", err)
	}
	defer cleanup()

	if _, statErr := os.Lstat(filepath.Join(dir, "buildx")); !os.IsNotExist(statErr) {
		t.Errorf("No buildx directory should have been created, got: %v", statErr)
	}
}

func TestCopyDirectoryReproducesTreeAndSkipsSymlinks(t *testing.T) {
	// The Windows fallback path, exercised directly since symlink() succeeds on
	// the test platform and would otherwise shadow it entirely.
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "instances"), 0o700); err != nil {
		t.Fatalf("Unable to build the source tree: %s", err)
	}
	if err := os.WriteFile(filepath.Join(source, "current"), []byte("selected"), 0o600); err != nil {
		t.Fatalf("Unable to write the source file: %s", err)
	}
	if err := os.WriteFile(filepath.Join(source, "instances", "one"), []byte("instance"), 0o600); err != nil {
		t.Fatalf("Unable to write the nested source file: %s", err)
	}

	// A symlink pointing outside the tree: copyDirectory must skip it rather
	// than follow it and copy in whatever it targets.
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("must not be copied"), 0o600); err != nil {
		t.Fatalf("Unable to write the outside file: %s", err)
	}
	if err := os.Symlink(outside, filepath.Join(source, "escape")); err != nil {
		t.Skipf("This platform does not permit symlink creation: %s", err)
	}

	target := filepath.Join(t.TempDir(), "copied")
	if err := copyDirectory(source, target); err != nil {
		t.Fatalf("copyDirectory returned an error: %s", err)
	}

	current, readErr := os.ReadFile(filepath.Join(target, "current"))
	if readErr != nil || string(current) != "selected" {
		t.Errorf("The top-level file was not copied correctly: %s (%v)", string(current), readErr)
	}

	nested, nestedErr := os.ReadFile(filepath.Join(target, "instances", "one"))
	if nestedErr != nil || string(nested) != "instance" {
		t.Errorf("The nested file was not copied correctly: %s (%v)", string(nested), nestedErr)
	}

	if _, statErr := os.Lstat(filepath.Join(target, "escape")); !os.IsNotExist(statErr) {
		t.Errorf("The symlink should have been skipped, got: %v", statErr)
	}
}
