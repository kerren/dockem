package utils

import (
	"os"
	"strings"
	"testing"
)

// TestCacheFromCacheToExcludedFromImageHash is the Phase 4.7 guarantee: --cache-from
// and --cache-to affect build *speed* only (which buildx layers get reused) and
// must never affect the image hash that decides whether a build is skipped -
// see the overallHash / hashVersion comments in build_docker_image.go, and
// contrast with Platform, which Phase 4.3 deliberately DOES fold into the hash
// via hashPlatforms (see build_docker_image_platform_hash_test.go).
//
// Because CacheFrom/CacheTo must have NO term in the overallHash concatenation
// at all, there is deliberately no hashCacheFlags-style function whose output
// this test could pin against a literal the way the platform-hash tests do -
// the guarantee under test is precisely that BuildDockerImage's hash
// computation never reads params.CacheFrom or params.CacheTo. The only direct
// way to assert that without actually running a full build against a real
// registry (which this package's e2e tests do, and which - per this rung's
// constraints - must never run without credentials) is to check dockem's own
// source: this reads build_docker_image.go and fails loudly if the region
// between "overallHash := hashVersion" and "imageHash := HashString(overallHash)"
// - the exact span that produces the image hash - ever comes to mention
// CacheFrom or CacheTo. Code added outside that span (eg. the "cache flags are
// buildx-only" warning, which intentionally lives just after imageHash is
// computed) does not trip this test.
func TestCacheFromCacheToExcludedFromImageHash(t *testing.T) {
	src, err := os.ReadFile("build_docker_image.go")
	if err != nil {
		t.Fatalf("could not read build_docker_image.go to verify the hash-exclusion guarantee: %s", err)
	}
	source := string(src)

	const startMarker = "overallHash := hashVersion"
	const endMarker = "imageHash := HashString(overallHash)"
	start := strings.Index(source, startMarker)
	end := strings.Index(source, endMarker)
	if start == -1 || end == -1 || end < start {
		t.Fatalf("could not locate the overallHash accumulation region in build_docker_image.go (markers %q / %q) - BuildDockerImage may have been restructured; update this test's markers to match", startMarker, endMarker)
	}
	hashRegion := source[start:end]

	if strings.Contains(hashRegion, "CacheFrom") {
		t.Errorf("build_docker_image.go's overallHash accumulation must never reference params.CacheFrom - --cache-from is buildx-only passthrough that affects build speed only and must be excluded from the image hash:\n%s", hashRegion)
	}
	if strings.Contains(hashRegion, "CacheTo") {
		t.Errorf("build_docker_image.go's overallHash accumulation must never reference params.CacheTo - --cache-to is buildx-only passthrough that affects build speed only and must be excluded from the image hash:\n%s", hashRegion)
	}
}

// TestCacheFlagsDoNotAlterAssembledHashFormula documents the same guarantee
// from the other direction, using this package's established pattern (see
// build_docker_image_platform_hash_test.go) of pinning the exact formula
// BuildDockerImage runs against a known-good literal. hashPlatforms is the
// last real term the formula has - baseHashBeforePlatforms + hashPlatforms(...)
// - and populating CacheFrom/CacheTo on a BuildDockerImageParams has no way to
// reach that formula at all, so the hash it produces is unaffected whether or
// not they are set.
func TestCacheFlagsDoNotAlterAssembledHashFormula(t *testing.T) {
	withoutCacheFlags := BuildDockerImageParams{}
	withCacheFlags := BuildDockerImageParams{
		CacheFrom: []string{"type=gha"},
		CacheTo:   []string{"type=gha,mode=max"},
	}

	formula := func(p BuildDockerImageParams) string {
		return HashString(baseHashBeforePlatforms + hashPlatforms(p.Platform))
	}

	got := formula(withoutCacheFlags)
	if got != preRung8ImageHash {
		t.Fatalf("baseline formula regressed: got %q, want the pinned literal %q", got, preRung8ImageHash)
	}
	if withCache := formula(withCacheFlags); withCache != got {
		t.Fatalf("populating CacheFrom/CacheTo changed the assembled hash: %q (with cache flags) != %q (without) - cache flags must never feed the image hash", withCache, got)
	}
}
