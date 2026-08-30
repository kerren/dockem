package utils

import (
	"os"
	"strings"
	"testing"
)

// TestSecretsExcludedFromImageHash is the --secret counterpart of
// TestCacheFromCacheToExcludedFromImageHash, and the reasoning it enforces is
// deliberately different from the cache flags' rather than a copy of it.
//
// A build secret is a credential the build *uses* (an npm token, an SSH key),
// not an input that defines what the build *produces*. Two runs with identical
// inputs and a rotated token want the same image, and folding a secret into
// overallHash would do the opposite: a CI token that changes on every run would
// change the hash on every run, so every image would rebuild every time and the
// entire point of this tool would be gone.
//
// As with the cache flags there is therefore no hashSecrets-style function whose
// output a test could pin, because the guarantee under test is precisely that no
// such term exists. So this reads dockem's own source and fails if the region
// between "overallHash := hashVersion" and "imageHash := HashString(overallHash)"
// ever comes to mention Secret. Note that BuildDockerImage's ResolveBuilder call
// legitimately reads params.Secret and sits just ABOVE the seed line for exactly
// this reason - see the comment on that seed.
func TestSecretsExcludedFromImageHash(t *testing.T) {
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

	if strings.Contains(hashRegion, "Secret") {
		t.Errorf("build_docker_image.go's overallHash accumulation must never reference params.Secret - --secret is buildx-only passthrough carrying a credential the build consumes, not an input that defines the image, and hashing it would invalidate every tag on every credential rotation:\n%s", hashRegion)
	}
}

// TestSecretsDoNotAlterAssembledHashFormula documents the same guarantee from
// the other direction, using this package's established pattern of pinning the
// exact formula BuildDockerImage runs against a known-good literal (see
// build_docker_image_platform_hash_test.go). Populating Secret on a
// BuildDockerImageParams has no way to reach that formula at all, so the hash is
// identical whether or not secrets were supplied - which is what lets a pipeline
// add --secret to an existing build without invalidating a single published tag.
func TestSecretsDoNotAlterAssembledHashFormula(t *testing.T) {
	withoutSecrets := BuildDockerImageParams{}
	withSecrets := BuildDockerImageParams{
		Secret: []string{"id=npmrc,src=./.npmrc", "id=token,env=NPM_TOKEN"},
	}

	formula := func(p BuildDockerImageParams) string {
		return HashString(baseHashBeforePlatforms + hashPlatforms(p.Platform))
	}

	got := formula(withoutSecrets)
	if got != preRung8ImageHash {
		t.Fatalf("baseline formula regressed: got %q, want the pinned literal %q", got, preRung8ImageHash)
	}
	if withSecret := formula(withSecrets); withSecret != got {
		t.Fatalf("populating Secret changed the assembled hash: %q (with secrets) != %q (without) - build secrets must never feed the image hash", withSecret, got)
	}
}
