package main

import (
	"testing"

	"github.com/rubiin/projecto/helper"
)

func TestFileExistsFunc(t *testing.T) {

	fileExists := helper.ConfigFileExists("nonexistent-file.txt")

	if fileExists {
		t.Errorf("Should be false but got %v", fileExists)
	}

	fileExists = helper.ConfigFileExists("../README.md")

	if !fileExists {
		t.Errorf("Should be true but got %v", fileExists)
	}

}

func TestCurrentDirFunc(t *testing.T) {
	currentDir := helper.CurrentDir()

	if currentDir == nil {
		t.Errorf("Should be a string but got %v", currentDir)
	}

}
