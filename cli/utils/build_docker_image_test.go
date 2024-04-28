package utils

import (
	"testing"
)

func TestStandardBuildWhereHashExists(t *testing.T) {
	// In this test, I'm going to test a build where the hash is the same.
	// In that case, it should not trigger a build but should rather copy
	// the tag from the existing image to the new image.

}
