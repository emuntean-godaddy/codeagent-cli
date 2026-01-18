package project

import (
	"fmt"
	"os"
	"path/filepath"
)

var getwd = os.Getwd

func CurrentRoot() (string, error) {
	cwd, err := getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return abs, nil
}
