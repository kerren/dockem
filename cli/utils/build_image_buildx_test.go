package utils

import (
	"os"
	"reflect"
	"testing"

	"github.com/moby/term"
)

// These are pure unit tests for assembleBuildxArgs: no subprocess, no
// filesystem beyond the paths given (which are never opened), no network, no
// credentials. They exist for Phase 4.7 - to prove --cache-from/--cache-to
// reach the docker buildx build argument list VERBATIM, one flag per value -
// and are the reason assembleBuildxArgs was split out of BuildImageBuildx in
// the first place (see its doc comment in build_image_buildx.go).

// wantProgressPlain mirrors the exact TTY check assembleBuildxArgs uses for
// --progress plain, so these tests assert against reality in whatever
// environment they run rather than hard-coding one branch and risking flakes
// when stderr happens to be a real terminal.
func wantProgressPlain() bool {
	_, isTerminal := term.GetFdInfo(os.Stderr)
	return !isTerminal
}

// TestAssembleBuildxArgsCacheFlagsPassedVerbatim asserts the full argument
// list produced for a single --cache-from and a single --cache-to value,
// including their exact position (alongside --platform, ahead of the --tag
// list) and that dockem does not rewrite, split, quote or otherwise touch the
// value the caller supplied.
func TestAssembleBuildxArgsCacheFlagsPassedVerbatim(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
		CacheFrom:      []string{"type=gha"},
		CacheTo:        []string{"type=gha,mode=max"},
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	want := []string{
		"buildx", "build",
		"--file", "Dockerfile",
		"--cache-from", "type=gha",
		"--cache-to", "type=gha,mode=max",
		"--tag", "example.com/repo:hash123",
		"--push",
	}
	if wantProgressPlain() {
		want = append(want, "--progress", "plain")
	}
	want = append(want, "./context")

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assembleBuildxArgs cache flags not verbatim/positioned as expected:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// TestAssembleBuildxArgsMultipleCacheValuesEachGetOwnFlag confirms --cache-from
// and --cache-to are repeatable: every value gets its own flag, in order, and a
// value containing "=" and "," (the shape every real buildx cache backend
// string takes, eg. type=registry,ref=example.com/repo:cache) survives intact
// rather than being re-split on those characters.
func TestAssembleBuildxArgsMultipleCacheValuesEachGetOwnFlag(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
		CacheFrom:      []string{"type=gha", "type=registry,ref=example.com/repo:cache"},
		CacheTo:        []string{"type=gha,mode=max"},
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	wantPairs := [][2]string{
		{"--cache-from", "type=gha"},
		{"--cache-from", "type=registry,ref=example.com/repo:cache"},
		{"--cache-to", "type=gha,mode=max"},
	}
	for _, pair := range wantPairs {
		found := false
		for i := 0; i+1 < len(got); i++ {
			if got[i] == pair[0] && got[i+1] == pair[1] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("assembleBuildxArgs did not contain the verbatim flag pair %q %q; got %#v", pair[0], pair[1], got)
		}
	}

	// Order must be preserved and every --cache-from must precede every
	// --cache-to, matching the order documented on BuildImageBuildx.
	firstCacheFrom := indexOfArg(got, "--cache-from")
	lastCacheFrom := lastIndexOfArg(got, "--cache-from")
	firstCacheTo := indexOfArg(got, "--cache-to")
	if firstCacheFrom == -1 || lastCacheFrom == -1 || firstCacheTo == -1 {
		t.Fatalf("expected both --cache-from and --cache-to present, got %#v", got)
	}
	if lastCacheFrom > firstCacheTo {
		t.Errorf("expected all --cache-from flags before --cache-to, got %#v", got)
	}

	// Exactly two --cache-from and one --cache-to flag - not more, not fewer.
	if n := countArg(got, "--cache-from"); n != 2 {
		t.Errorf("expected exactly 2 --cache-from flags, got %d in %#v", n, got)
	}
	if n := countArg(got, "--cache-to"); n != 1 {
		t.Errorf("expected exactly 1 --cache-to flag, got %d in %#v", n, got)
	}
}

// TestAssembleBuildxArgsOmitsCacheFlagsWhenUnset: exactly like --platform,
// --cache-from/--cache-to must be entirely absent from the argument list when
// not supplied, so a caller who never adopts them sees byte-identical buildx
// invocations to before this rung.
func TestAssembleBuildxArgsOmitsCacheFlagsWhenUnset(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	for _, a := range got {
		if a == "--cache-from" || a == "--cache-to" {
			t.Fatalf("assembleBuildxArgs must omit --cache-from/--cache-to entirely when unset, got %#v", got)
		}
	}
}

// TestAssembleBuildxArgsSecretFlagsPassedVerbatim pins both the content and the
// POSITION of --secret in one whole-slice comparison. Position is the thing the
// pair-scanning test below cannot prove, and it only means anything when the
// neighbouring flags exist - hence --platform, --cache-from and --cache-to are
// all populated here too. --secret must land after the cache flags and before
// the first --tag, matching the documented command shape on BuildImageBuildx.
//
// The value carries an "=" and a "," because that is the shape every real
// buildx secret takes (id=npmrc,src=./.npmrc). If anyone ever "helpfully" adds
// the --platform style comma-splitting to --secret, this test fails: a split
// would turn one secret into the two meaningless fragments "id=npmrc" and
// "src=./.npmrc".
func TestAssembleBuildxArgsSecretFlagsPassedVerbatim(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
		Platform:       []string{"linux/amd64"},
		CacheFrom:      []string{"type=gha"},
		CacheTo:        []string{"type=gha,mode=max"},
		Secret:         []string{"id=npmrc,src=./.npmrc"},
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	want := []string{
		"buildx", "build",
		"--file", "Dockerfile",
		"--platform", "linux/amd64",
		"--cache-from", "type=gha",
		"--cache-to", "type=gha,mode=max",
		"--secret", "id=npmrc,src=./.npmrc",
		"--tag", "example.com/repo:hash123",
		"--push",
	}
	if wantProgressPlain() {
		want = append(want, "--progress", "plain")
	}
	want = append(want, "./context")

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assembleBuildxArgs secret flags not verbatim/positioned as expected:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// TestAssembleBuildxArgsMultipleSecretValuesEachGetOwnFlag confirms --secret is
// repeatable: every value gets its own flag, in the order given, and both of
// buildx's secret forms - src=<path> and env=<VAR> - survive intact. It also
// pins the ordering relative to its neighbours without depending on the exact
// contents of the rest of the slice.
func TestAssembleBuildxArgsMultipleSecretValuesEachGetOwnFlag(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
		CacheTo:        []string{"type=gha,mode=max"},
		Secret: []string{
			"id=npmrc,src=./.npmrc",
			"id=token,env=NPM_TOKEN",
			"id=bare",
		},
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	wantPairs := [][2]string{
		{"--secret", "id=npmrc,src=./.npmrc"},
		{"--secret", "id=token,env=NPM_TOKEN"},
		{"--secret", "id=bare"},
	}
	for _, pair := range wantPairs {
		found := false
		for i := 0; i+1 < len(got); i++ {
			if got[i] == pair[0] && got[i+1] == pair[1] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s %q to reach the argument list verbatim and adjacent, got %#v", pair[0], pair[1], got)
		}
	}

	if n := countArg(got, "--secret"); n != 3 {
		t.Errorf("expected exactly 3 --secret flags, got %d in %#v", n, got)
	}

	// --secret sits between the cache flags and the tags.
	if lastIndexOfArg(got, "--cache-to") > indexOfArg(got, "--secret") {
		t.Errorf("expected every --cache-to flag before the first --secret, got %#v", got)
	}
	if lastIndexOfArg(got, "--secret") > indexOfArg(got, "--tag") {
		t.Errorf("expected every --secret flag before the first --tag, got %#v", got)
	}
}

// TestAssembleBuildxArgsOmitsSecretFlagWhenUnset: like --platform and the cache
// flags, --secret must be entirely absent when not supplied, so a caller who
// never adopts it sees a byte-identical buildx invocation to before this change.
func TestAssembleBuildxArgsOmitsSecretFlagWhenUnset(t *testing.T) {
	params := BuildDockerImageParams{
		DockerfilePath: "Dockerfile",
		Directory:      "./context",
	}
	buildLog := &BuildLog{}

	got := assembleBuildxArgs(params, "example.com/repo:hash123", nil, buildLog)

	for _, a := range got {
		if a == "--secret" {
			t.Fatalf("assembleBuildxArgs must omit --secret entirely when unset, got %#v", got)
		}
	}
}

func indexOfArg(args []string, needle string) int {
	for i, a := range args {
		if a == needle {
			return i
		}
	}
	return -1
}

func lastIndexOfArg(args []string, needle string) int {
	idx := -1
	for i, a := range args {
		if a == needle {
			idx = i
		}
	}
	return idx
}

func countArg(args []string, needle string) int {
	n := 0
	for _, a := range args {
		if a == needle {
			n++
		}
	}
	return n
}
