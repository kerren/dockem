package utils

import (
	"io"
	"os"
)

// This is the function that I'll use to open a SINGLE file

func osOpen(name string) (io.ReadCloser, error) {
	return os.Open(name)
}
