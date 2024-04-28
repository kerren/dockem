package utils

import "os"

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
