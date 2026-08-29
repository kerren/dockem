package utils

import (
	"crypto/sha256"
	"fmt"
)

// HashString Hashes the given string using SHA256
func HashString(s string) string {
	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(s))
	return fmt.Sprintf("%x", sha256Hash.Sum(nil))
}
