package utils

import "os"

// DirectoryExists Checks if the path given is a directory that exists.
// It will throw an error if it does not exist or if it is not a directory.
func DirectoryExists(path string) (bool, error) {
	exists, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	is_directory := exists.IsDir()
	if err == nil && is_directory {
		return true, nil
	}
	return false, err
}
