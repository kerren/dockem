package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// These tests exercise BuildImageBuildx itself, rather than the pure
// assembleBuildxArgs helper covered in build_image_buildx_test.go. They pin
// every rule in the "Subprocess credentials (buildx)" section of CLAUDE.md by
// putting a fake `docker` first on PATH: a POSIX shell script that records its
// full argv, its complete environment and its working directory to files, then
// exits with a chosen status. None of this touches a real docker daemon,
// buildx installation or registry.
//
// None of these tests use t.Parallel(): every one of them manipulates
// process-global state (PATH, the parent's real environment via t.Setenv,
// os.Stdout/os.Stderr) that a concurrently running test could observe or
// stomp on.

// fakeDockerRecording is where one invocation of the fake docker script wrote
// what it saw.
type fakeDockerRecording struct {
	dir string
}

func (r fakeDockerRecording) argv(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, "argv.txt"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("could not read the fake docker's recorded argv: %v", err)
	}
	trimmed := strings.TrimSuffix(string(data), "\n")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func (r fakeDockerRecording) env(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, "env.txt"))
	if err != nil {
		t.Fatalf("could not read the fake docker's recorded environment: %v", err)
	}
	trimmed := strings.TrimSuffix(string(data), "\n")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func (r fakeDockerRecording) envValue(t *testing.T, key string) (string, bool) {
	t.Helper()
	prefix := key + "="
	for _, entry := range r.env(t) {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}
	return "", false
}

func (r fakeDockerRecording) envCount(t *testing.T, key string) int {
	t.Helper()
	prefix := key + "="
	count := 0
	for _, entry := range r.env(t) {
		if strings.HasPrefix(entry, prefix) {
			count++
		}
	}
	return count
}

func (r fakeDockerRecording) cwd(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, "cwd.txt"))
	if err != nil {
		t.Fatalf("could not read the fake docker's recorded cwd: %v", err)
	}
	return strings.TrimSuffix(string(data), "\n")
}

// installFakeDocker puts a fake `docker` executable first on PATH (ahead of
// whatever real PATH the test process already has, so other POSIX tools the
// script itself needs - env, pwd, printf - are still resolvable). The script
// records its full argv, environment and cwd into a fresh directory, echoes a
// distinctive line to its own stdout and its own stderr (for the stdout/stderr
// plumbing test), then exits with exitCode.
//
// Returns the fakeDockerRecording that will hold whatever the NEXT invocation
// writes. A fresh recording directory is used per install so consecutive
// subtests never read stale output from a previous run.
func installFakeDocker(t *testing.T, exitCode int) fakeDockerRecording {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("the fake docker executable is a POSIX shell script")
	}

	binDir := t.TempDir()
	recordDir := t.TempDir()

	argvFile := filepath.Join(recordDir, "argv.txt")
	envFile := filepath.Join(recordDir, "env.txt")
	cwdFile := filepath.Join(recordDir, "cwd.txt")

	script := "#!/bin/sh\n" +
		"rm -f \"" + argvFile + "\"\n" +
		"touch \"" + argvFile + "\"\n" +
		"for a in \"$@\"; do\n" +
		"  printf '%s\\n' \"$a\" >> \"" + argvFile + "\"\n" +
		"done\n" +
		"env > \"" + envFile + "\"\n" +
		"pwd > \"" + cwdFile + "\"\n" +
		"echo fake-docker-stdout-chatter\n" +
		"echo fake-docker-stderr-chatter 1>&2\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"

	dockerPath := filepath.Join(binDir, "docker")
	if err := os.WriteFile(dockerPath, []byte(script), 0o755); err != nil {
		t.Fatalf("could not write the fake docker script: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

	return fakeDockerRecording{dir: recordDir}
}

// runCapturingStreams swaps os.Stdout and os.Stderr for pipes for the duration
// of run, then returns everything written to each. Every LogInfo/LogWarn/
// LogError call inside BuildImageBuildx writes to os.Stderr at call time (see
// log.go), so this is also how TestBuildImageBuildxStreamsSubprocessOutputToStderr
// distinguishes dockem's own chatter (which is supposed to land on stderr
// anyway) from anything that leaked onto stdout.
func runCapturingStreams(t *testing.T, run func() error) (stdout string, stderr string, runErr error) {
	t.Helper()

	origStdout, origStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create a stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	runErr = run()

	wOut.Close()
	wErr.Close()

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	rOut.Close()
	rErr.Close()

	return string(outBytes), string(errBytes), runErr
}

// basicBuildxParams returns a minimal, valid set of params/tags/hash for
// exercising BuildImageBuildx without caring about the argument-assembly
// details already covered by build_image_buildx_test.go.
func basicBuildxParams(t *testing.T) (BuildDockerImageParams, string, []ResolvedTag) {
	t.Helper()
	contextDir := t.TempDir()
	params := BuildDockerImageParams{
		DockerfilePath: filepath.Join(contextDir, "Dockerfile"),
		Directory:      contextDir,
		ImageName:      "example/repo",
	}
	targetTags := []ResolvedTag{
		{ImageName: "example/repo:v1.2.3", Reason: TagReasonMainVersion},
	}
	return params, "deadbeef", targetTags
}

// TestBuildImageBuildxArgvMatchesAssembleBuildxArgs pins that the argv the
// child actually receives is exactly what assembleBuildxArgs computes for the
// same inputs - ie. that BuildImageBuildx does not add, drop or reorder
// anything between assembling the arguments and executing them.
func TestBuildImageBuildxArgvMatchesAssembleBuildxArgs(t *testing.T) {
	recording := installFakeDocker(t, 0)

	params, imageHash, targetTags := basicBuildxParams(t)
	params.Platform = []string{"linux/amd64", "linux/arm64"}
	params.CacheFrom = []string{"type=gha"}
	params.CacheTo = []string{"type=gha,mode=max"}

	expectedLog := &BuildLog{}
	hashImageName := GenerateDockerImageName(params.Registry, params.ImageName, imageHash)
	var wantArgs []string
	runCapturingStreams(t, func() error {
		wantArgs = assembleBuildxArgs(params, hashImageName, targetTags, expectedLog)
		return nil
	})

	actualLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, actualLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	gotArgs := recording.argv(t)
	if strings.Join(gotArgs, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("child argv did not match assembleBuildxArgs:\ngot:  %#v\nwant: %#v", gotArgs, wantArgs)
	}
}

// TestBuildImageBuildxSetsDockerConfigOnChildOnly protects the two halves of
// the same CLAUDE.md rule at once: DOCKER_CONFIG must reach the CHILD pointing
// at the throwaway config dir, and the PARENT process's own DOCKER_CONFIG (ie.
// dockem's own environment) must be completely unaffected by the call - it is
// set on cmd.Env only, never via os.Setenv. This is the rule most likely to
// regress, since it is one line away from leaking into dockem's own process.
func TestBuildImageBuildxSetsDockerConfigOnChildOnly(t *testing.T) {
	recording := installFakeDocker(t, 0)

	if _, isSet := os.LookupEnv("DOCKER_CONFIG"); isSet {
		t.Fatalf("test precondition failed: DOCKER_CONFIG is already set in the test process")
	}

	params, imageHash, targetTags := basicBuildxParams(t)
	params.DockerUsername = "uname"
	params.DockerPassword = "s3cr3t-buildx-password"

	buildLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	childConfig, ok := recording.envValue(t, "DOCKER_CONFIG")
	if !ok || childConfig == "" {
		t.Fatalf("expected the child to see a non-empty DOCKER_CONFIG, got ok=%v value=%q", ok, childConfig)
	}

	// The parent (this test process, standing in for dockem's own process) must
	// come out exactly as it went in: no DOCKER_CONFIG at all.
	if _, isSet := os.LookupEnv("DOCKER_CONFIG"); isSet {
		t.Fatalf("DOCKER_CONFIG leaked into dockem's own process: %q", os.Getenv("DOCKER_CONFIG"))
	}
}

// TestBuildImageBuildxPassesThroughArbitraryParentEnvVar guards a real
// regression: if cmd.Env were ever narrowed to an allowlist of "the variables
// dockem cares about", --secret id=x,env=VAR would silently stop working,
// because VAR would never reach the child. An arbitrary FOO must survive
// untouched, both with and without explicit credentials (the two branches that
// build cmd.Env take different paths - nil vs a full copy - and both must
// pass arbitrary vars through).
func TestBuildImageBuildxPassesThroughArbitraryParentEnvVar(t *testing.T) {
	t.Run("with credentials", func(t *testing.T) {
		recording := installFakeDocker(t, 0)
		t.Setenv("FOO", "bar")

		params, imageHash, targetTags := basicBuildxParams(t)
		params.DockerUsername = "uname"
		params.DockerPassword = "pw"

		buildLog := &BuildLog{}
		_, _, err := runCapturingStreams(t, func() error {
			return BuildImageBuildx(params, imageHash, targetTags, buildLog)
		})
		if err != nil {
			t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
		}

		if got, ok := recording.envValue(t, "FOO"); !ok || got != "bar" {
			t.Fatalf("expected the child to see FOO=bar, got ok=%v value=%q", ok, got)
		}
	})

	t.Run("without credentials", func(t *testing.T) {
		recording := installFakeDocker(t, 0)
		t.Setenv("FOO", "bar")

		params, imageHash, targetTags := basicBuildxParams(t)

		buildLog := &BuildLog{}
		_, _, err := runCapturingStreams(t, func() error {
			return BuildImageBuildx(params, imageHash, targetTags, buildLog)
		})
		if err != nil {
			t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
		}

		if got, ok := recording.envValue(t, "FOO"); !ok || got != "bar" {
			t.Fatalf("expected the child to see FOO=bar, got ok=%v value=%q", ok, got)
		}
	})
}

// TestBuildImageBuildxChildCwdMatchesParent pins that cmd.Dir is never set: the
// child's working directory must be dockem's own cwd, so a relative
// `--secret id=x,src=./relative/path` resolves against dockem's cwd rather than
// the build context or anywhere else.
func TestBuildImageBuildxChildCwdMatchesParent(t *testing.T) {
	recording := installFakeDocker(t, 0)

	wantCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not determine this test's own cwd: %v", err)
	}
	wantCwdResolved, err := filepath.EvalSymlinks(wantCwd)
	if err != nil {
		t.Fatalf("could not resolve this test's own cwd: %v", err)
	}

	params, imageHash, targetTags := basicBuildxParams(t)

	buildLog := &BuildLog{}
	_, _, runErr := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if runErr != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", runErr)
	}

	gotCwd := recording.cwd(t)
	gotCwdResolved, err := filepath.EvalSymlinks(gotCwd)
	if err != nil {
		t.Fatalf("could not resolve the child's reported cwd %q: %v", gotCwd, err)
	}

	if gotCwdResolved != wantCwdResolved {
		t.Fatalf("child cwd = %q, want %q (dockem's own cwd)", gotCwdResolved, wantCwdResolved)
	}
}

// TestBuildImageBuildxNoCredentialsLeavesEnvNil checks the no-credentials
// branch directly: with no --docker-username/--docker-password, cmd.Env must
// stay nil so the child inherits dockem's environment completely unchanged -
// which is what makes an existing `docker login` keep working. We cannot
// observe cmd.Env itself from outside the function, so instead we assert on
// its externally-visible consequence: every variable the parent has (PATH,
// and a distinctive marker) reaches the child with the exact values the parent
// had, and no DOCKER_CONFIG is introduced.
func TestBuildImageBuildxNoCredentialsLeavesEnvNil(t *testing.T) {
	recording := installFakeDocker(t, 0)
	t.Setenv("DOCKEM_MARKER", "present")

	params, imageHash, targetTags := basicBuildxParams(t)

	buildLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	if got, ok := recording.envValue(t, "DOCKEM_MARKER"); !ok || got != "present" {
		t.Fatalf("expected the child to inherit DOCKEM_MARKER unchanged, got ok=%v value=%q", ok, got)
	}
	if _, ok := recording.envValue(t, "DOCKER_CONFIG"); ok {
		t.Fatalf("expected no DOCKER_CONFIG on the child when no credentials were given")
	}
}

// TestBuildImageBuildxStripsPreexistingDockerConfig checks that a
// DOCKER_CONFIG already present in dockem's own environment (eg. because the
// user exported one for their own `docker login` setup) is stripped rather
// than duplicated when credentials are supplied: the child must see exactly
// ONE DOCKER_CONFIG entry, and it must be dockem's throwaway one, not the
// user's original value.
func TestBuildImageBuildxStripsPreexistingDockerConfig(t *testing.T) {
	recording := installFakeDocker(t, 0)

	preexisting := t.TempDir()
	t.Setenv("DOCKER_CONFIG", preexisting)

	params, imageHash, targetTags := basicBuildxParams(t)
	params.DockerUsername = "uname"
	params.DockerPassword = "pw"

	buildLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	if count := recording.envCount(t, "DOCKER_CONFIG"); count != 1 {
		t.Fatalf("expected exactly one DOCKER_CONFIG entry on the child, found %d: %v", count, recording.env(t))
	}

	got, ok := recording.envValue(t, "DOCKER_CONFIG")
	if !ok {
		t.Fatalf("expected a DOCKER_CONFIG entry on the child")
	}
	if got == preexisting {
		t.Fatalf("expected DOCKER_CONFIG to be dockem's throwaway dir, got the preexisting value %q", got)
	}
}

// TestBuildImageBuildxRemovesTempConfigDirAfterReturn checks that the
// throwaway config directory TempDockerConfig writes credentials into is
// removed once BuildImageBuildx returns, on both a clean exit and a failing
// one - the deferred cleanup must run regardless of how the subprocess exits.
func TestBuildImageBuildxRemovesTempConfigDirAfterReturn(t *testing.T) {
	cases := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{"clean exit", 0, false},
		{"non-zero exit", 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recording := installFakeDocker(t, tc.exitCode)

			params, imageHash, targetTags := basicBuildxParams(t)
			params.DockerUsername = "uname"
			params.DockerPassword = "pw"

			buildLog := &BuildLog{}
			_, _, err := runCapturingStreams(t, func() error {
				return BuildImageBuildx(params, imageHash, targetTags, buildLog)
			})
			if tc.wantErr && err == nil {
				t.Fatalf("expected BuildImageBuildx to return an error for exit code %d", tc.exitCode)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
			}

			configDir, ok := recording.envValue(t, "DOCKER_CONFIG")
			if !ok || configDir == "" {
				t.Fatalf("expected the child to have recorded a DOCKER_CONFIG value to check")
			}

			if _, statErr := os.Stat(configDir); !os.IsNotExist(statErr) {
				t.Fatalf("expected the temp config dir %q to be removed after BuildImageBuildx returned, stat err = %v", configDir, statErr)
			}
		})
	}
}

// TestBuildImageBuildxNonZeroExitIsError checks the basic error-surfacing
// contract: a failing `docker buildx build` must come back as a non-nil error
// from BuildImageBuildx, so BuildDockerImage propagates the failure instead of
// reporting a successful build that never happened.
func TestBuildImageBuildxNonZeroExitIsError(t *testing.T) {
	installFakeDocker(t, 17)

	params, imageHash, targetTags := basicBuildxParams(t)

	buildLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err == nil {
		t.Fatalf("expected BuildImageBuildx to return an error for a non-zero subprocess exit")
	}
}

// TestBuildImageBuildxStreamsSubprocessOutputToStderr protects the
// --output-format=json contract: BuildImageBuildx sets BOTH cmd.Stdout and
// cmd.Stderr to os.Stderr, so anything the subprocess prints - on either of
// its own streams - must land on dockem's stderr, and dockem's stdout must
// stay completely empty.
func TestBuildImageBuildxStreamsSubprocessOutputToStderr(t *testing.T) {
	installFakeDocker(t, 0)

	params, imageHash, targetTags := basicBuildxParams(t)

	buildLog := &BuildLog{}
	stdout, stderr, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected dockem's stdout to stay empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "fake-docker-stdout-chatter") {
		t.Fatalf("expected the child's own stdout chatter to land on dockem's stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "fake-docker-stderr-chatter") {
		t.Fatalf("expected the child's own stderr chatter to land on dockem's stderr, got %q", stderr)
	}
}

// TestBuildImageBuildxPasswordNeverLeaks uses a distinctive sentinel password
// and greps everywhere it could conceivably leak: the child's argv (buildx's
// --secret/--tag syntax never carries a raw password, and DOCKER_CONFIG only
// ever carries a PATH to the credentials file, not the credentials
// themselves), the populated BuildLog (inspected directly - this test lives in
// package utils, so its unexported fields are visible here), and the JSON
// BuildResult a caller would actually see.
func TestBuildImageBuildxPasswordNeverLeaks(t *testing.T) {
	recording := installFakeDocker(t, 0)

	const sentinelPassword = "sentinel-pw-4f8c9d2a-must-not-leak"

	params, imageHash, targetTags := basicBuildxParams(t)
	params.DockerUsername = "uname"
	params.DockerPassword = sentinelPassword

	buildLog := &BuildLog{}
	_, _, err := runCapturingStreams(t, func() error {
		return BuildImageBuildx(params, imageHash, targetTags, buildLog)
	})
	if err != nil {
		t.Fatalf("BuildImageBuildx returned an unexpected error: %v", err)
	}

	for _, a := range recording.argv(t) {
		if strings.Contains(a, sentinelPassword) {
			t.Fatalf("the password leaked into the child's argv: %q", a)
		}
	}

	// BuildLog itself: dump every field's value via fmt (this test is in the
	// same package, so unexported fields are directly readable) and confirm
	// the sentinel is nowhere in it.
	buildLogDump := fmt.Sprintf("%#v", *buildLog)
	if strings.Contains(buildLogDump, sentinelPassword) {
		t.Fatalf("the password leaked into BuildLog: %s", buildLogDump)
	}

	result := buildLog.Result()
	resultDump := fmt.Sprintf("%#v", result)
	if strings.Contains(resultDump, sentinelPassword) {
		t.Fatalf("the password leaked into BuildResult: %s", resultDump)
	}
}

// TestDescribePlatforms covers the small logging helper directly: no
// platforms reads as "the host platform", one or more platforms are rendered
// joined with ", ".
func TestDescribePlatforms(t *testing.T) {
	cases := []struct {
		name     string
		input    []string
		expected string
	}{
		{"unset", nil, "the host platform"},
		{"empty slice", []string{}, "the host platform"},
		{"single platform", []string{"linux/amd64"}, "linux/amd64"},
		{"multiple platforms", []string{"linux/amd64", "linux/arm64"}, "linux/amd64, linux/arm64"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describePlatforms(tc.input); got != tc.expected {
				t.Errorf("describePlatforms(%#v) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
