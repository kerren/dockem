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
