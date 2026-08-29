package utils

import "testing"

// These are pure unit tests for the Phase 4.3 requirement that the --platform
// list feeds the image hash. They need no registry and no credentials.

// baseHashBeforePlatforms stands in for the overallHash string that
// BuildDockerImage has accumulated by the time it reaches the platform step:
// the hashVersion prefix followed by the watch-file, watch-directory,
// build-directory and Dockerfile component hashes. The exact component strings
// are irrelevant - what matters is that HashString of this string is the image
// hash a pre-rung-8 dockem would have produced, because pre-rung-8 nothing was
// appended to overallHash after the Dockerfile hash.
const baseHashBeforePlatforms = "dockem-hash-v2" + "watch-files-hash" + "watch-dirs-hash" + "build-dir-hash" + "dockerfile-hash"

// preRung8ImageHash is that baseline captured as a literal: the sha256 (i.e.
// HashString) of baseHashBeforePlatforms as computed on the parent commit
// (feature/buildx-detect), before the platform list ever fed the hash. An unset
// --platform must reproduce exactly this value, byte for byte.
const preRung8ImageHash = "ae0c49a87d327c70f53ebc35cc9b24c1a18a76dd0ee95101bd51c4ab56fb48a6"

// TestHashPlatformsUnsetReproducesPreRung8Hash is the load-bearing guarantee of
// this rung: a build with no --platform hashes exactly as a pre-platform-support
// dockem did, so existing single-platform users see no cache invalidation from
// this change.
func TestHashPlatformsUnsetReproducesPreRung8Hash(t *testing.T) {
	// Guard against the literal drifting from HashString's algorithm: if this
	// fails, either HashString changed or the captured literal is stale.
	if got := HashString(baseHashBeforePlatforms); got != preRung8ImageHash {
		t.Fatalf("captured baseline is stale: HashString(base) = %q, want the literal %q", got, preRung8ImageHash)
	}

	for _, tc := range []struct {
		name      string
		platforms []string
	}{
		{"nil", nil},
		{"empty", []string{}},
	} {
		overall := baseHashBeforePlatforms + hashPlatforms(tc.platforms)
		if got := HashString(overall); got != preRung8ImageHash {
			t.Errorf("unset platform list (%s) changed the image hash: got %q, want the pre-rung-8 literal %q", tc.name, got, preRung8ImageHash)
		}
	}
}

// TestHashPlatformsDifferentListsYieldDifferentHashes: a different set of target
// platforms is a different required output and so must produce a different hash,
// or dockem would copy a single-arch image forward forever.
func TestHashPlatformsDifferentListsYieldDifferentHashes(t *testing.T) {
	single := HashString(baseHashBeforePlatforms + hashPlatforms([]string{"linux/amd64"}))
	multi := HashString(baseHashBeforePlatforms + hashPlatforms([]string{"linux/amd64", "linux/arm64"}))

	if single == preRung8ImageHash {
		t.Errorf("a single-platform build must not share the hash of the unset (host-platform) build")
	}
	if single == multi {
		t.Errorf("a single-platform and a two-platform build must not share a hash; both were %q", single)
	}
}

// TestHashPlatformsIsSortInvariant: the cache identity depends on the set of
// platforms, not the order they were passed, so the list is sorted before it is
// joined.
func TestHashPlatformsIsSortInvariant(t *testing.T) {
	forward := hashPlatforms([]string{"linux/amd64", "linux/arm64"})
	reversed := hashPlatforms([]string{"linux/arm64", "linux/amd64"})
	if forward != reversed {
		t.Errorf("hashPlatforms must sort before joining: %q != %q", forward, reversed)
	}
	if forward != "linux/amd64,linux/arm64" {
		t.Errorf("unexpected joined form: got %q, want %q", forward, "linux/amd64,linux/arm64")
	}
	if hashPlatforms(nil) != "" {
		t.Errorf("hashPlatforms(nil) must be the empty string, got %q", hashPlatforms(nil))
	}
}
