package utils

import "testing"

// These are pure unit tests: no filesystem, no registry, no credentials.

func TestParseVersionFileJsonValid(t *testing.T) {
	got, err := ParseVersionFileJson([]byte(`{"version": "1.2.3"}`))
	if err != nil {
		t.Fatalf("ParseVersionFileJson returned an unexpected error: %v", err)
	}
	if got.Version != "1.2.3" {
		t.Fatalf("ParseVersionFileJson().Version = %q, want %q", got.Version, "1.2.3")
	}
}

func TestParseVersionFileJsonMalformed(t *testing.T) {
	_, err := ParseVersionFileJson([]byte(`{"version": "1.2.3"`))
	if err == nil {
		t.Fatal("ParseVersionFileJson with malformed JSON returned a nil error, want an error")
	}
}

// TestParseVersionFileJsonNonStringVersion pins that a non-string `version`
// value (json.Unmarshal cannot coerce a number into the string field) is
// rejected as an error rather than silently stringified.
func TestParseVersionFileJsonNonStringVersion(t *testing.T) {
	_, err := ParseVersionFileJson([]byte(`{"version": 123}`))
	if err == nil {
		t.Fatal("ParseVersionFileJson with a numeric version value returned a nil error, want an error")
	}
}

// TestParseVersionFileJsonEmptyObject pins a currently-silent edge: a version
// file with no `version` key at all parses successfully and yields an empty
// Version string. This is a deliberate documentation of existing behaviour,
// not an endorsement of it - see ExtractVersion's "{} yields v" test for the
// user-visible consequence.
func TestParseVersionFileJsonEmptyObject(t *testing.T) {
	got, err := ParseVersionFileJson([]byte(`{}`))
	if err != nil {
		t.Fatalf("ParseVersionFileJson(\"{}\") returned an unexpected error: %v", err)
	}
	if got.Version != "" {
		t.Fatalf("ParseVersionFileJson(\"{}\").Version = %q, want empty string", got.Version)
	}
}
