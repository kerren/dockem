package utils

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestFileDoesNotExist(t *testing.T) {

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomInt := rand.Intn(100000000)
	fileName := "test_file_" + fmt.Sprintf("%d", randomInt)
	exists, _ := FileExists(fileName)
	if exists {
		t.Errorf("File %s should not exist", fileName)
	}

}

func TestFileDoesExist(t *testing.T) {

	file, _ := os.CreateTemp("", "test_file")
	exists, _ := FileExists(file.Name())
	if !exists {
		t.Errorf("File %s should exist", file.Name())
	}

	os.RemoveAll(file.Name())
}
