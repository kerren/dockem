package utils

import (
	"reflect"
	"testing"
)

// These are pure unit tests: no filesystem, no registry, no credentials.

func TestRemoveEmptyStringsFromArrayOrderPreserved(t *testing.T) {
	got := RemoveEmptyStringsFromArray([]string{"a", "", "b", "", "c"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveEmptyStringsFromArray(...) = %#v, want %#v", got, want)
	}
}

// TestRemoveEmptyStringsFromArrayAllEmptyReturnsNil pins that an input made
// entirely of empty strings returns a true nil slice, not an empty non-nil
// one - the function declares its accumulator with `var newArray []string`
// and only ever appends, so it is never assigned an empty literal. A caller
// that checks `result == nil` to detect "no tags survived" depends on this.
func TestRemoveEmptyStringsFromArrayAllEmptyReturnsNil(t *testing.T) {
	got := RemoveEmptyStringsFromArray([]string{"", "", ""})
	if got != nil {
		t.Fatalf("RemoveEmptyStringsFromArray(all-empty) = %#v, want nil", got)
	}
}

func TestRemoveEmptyStringsFromArrayEmptyInputReturnsNil(t *testing.T) {
	got := RemoveEmptyStringsFromArray([]string{})
	if got != nil {
		t.Fatalf("RemoveEmptyStringsFromArray([]string{}) = %#v, want nil", got)
	}
}

// TestRemoveEmptyStringsFromArrayWhitespaceOnlyIsKept pins a currently-silent
// edge from docs/testing-plan.md Phase T2.5: the function only drops strings
// that are exactly "", so a whitespace-only value like " " (e.g. from
// `--tag " "`) is not empty by this check and survives into the result. This
// documents existing behaviour rather than endorsing it.
func TestRemoveEmptyStringsFromArrayWhitespaceOnlyIsKept(t *testing.T) {
	got := RemoveEmptyStringsFromArray([]string{"a", " ", "b"})
	want := []string{"a", " ", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveEmptyStringsFromArray with whitespace-only entry = %#v, want %#v (pinned known edge)", got, want)
	}
}
