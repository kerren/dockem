package utils

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestDirectoryDoesNotExist(t *testing.T) {

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := rand.Intn(100000000)
	dirName := "test_dir_" + fmt.Sprintf("%d", randomInt)
	exists, _ := DirectoryExists(dirName)
	if exists {
		t.Errorf("Directory %s should not exist", dirName)
	}

}

func TestDirectoryExists(t *testing.T) {

	dirName, _ := os.MkdirTemp("", "test_dir_")
	exists, _ := DirectoryExists(dirName)
	if !exists {
		t.Errorf("Directory %s should exist", dirName)
	}

	os.RemoveAll(dirName)
}
