package utils

import (
	"io"
	"os"
)

// osOpen This is the function that I'll use to open a SINGLE file

func osOpen(name string) (io.ReadCloser, error) {
	return os.Open(name)
}
