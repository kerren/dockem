package utils

import "os"

// FileExists Checks if the path given is a file that exists.
// It will throw an error if it does not exist or if it is not a file.
func FileExists(path string) (bool, error) {
	exists, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	is_file := !exists.IsDir()
	if err == nil && is_file {
		return true, nil
	}
	return false, err
}
