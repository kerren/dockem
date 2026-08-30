package utils

import "testing"

// These are pure unit tests: no filesystem, no registry, no credentials.

// TestHashStringKnownVector pins HashString against an independently verified
// SHA256 digest (computed with `printf 'dockem' | sha256sum`), so a change to
// the hashing primitive itself - not just its callers - would be caught here.
func TestHashStringKnownVector(t *testing.T) {
	const want = "1147bbc8f02eda032e2e169e6dd8b140884a6d7d04dcc4d6fe879842aa8868aa"
	if got := HashString("dockem"); got != want {
		t.Fatalf("HashString(\"dockem\") = %q, want %q", got, want)
	}
}

// TestHashStringEmptyInput pins the digest of the empty string (the standard
// SHA256 empty-input constant, verified with `printf '' | sha256sum`).
func TestHashStringEmptyInput(t *testing.T) {
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := HashString(""); got != want {
		t.Fatalf("HashString(\"\") = %q, want %q", got, want)
	}
}

// TestHashStringDeterministic guards the property overallHash relies on: the
// same input must always produce the same output across calls, since it is
// what makes the hash usable as a cache key at all.
func TestHashStringDeterministic(t *testing.T) {
	const input = "some concatenated hash inputs"
	first := HashString(input)
	for i := 0; i < 5; i++ {
		if got := HashString(input); got != first {
			t.Fatalf("HashString(%q) = %q on call %d, want %q (same as first call)", input, got, i, first)
		}
	}
}
