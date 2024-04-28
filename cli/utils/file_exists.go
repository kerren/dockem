package utils

import "os"

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
