package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigDirName        = ".codeagent"
	DisplayConfigDir     = "~/.codeagent"
	DockerfileName       = "Dockerfile"
	DevcontainerJSONName = "devcontainer.json"
)

var RequiredFiles = []string{
	DockerfileName,
	DevcontainerJSONName,
}

type MissingConfigError struct {
	DisplayDir string
}

func (e MissingConfigError) Error() string {
	return fmt.Sprintf("%s configuration missing.\nExpected:\n  - %s/%s\n  - %s/%s",
		e.DisplayDir, e.DisplayDir, DockerfileName, e.DisplayDir, DevcontainerJSONName)
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName), nil
}

func Validate(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return MissingConfigError{DisplayDir: DisplayConfigDir}
	}

	for _, name := range RequiredFiles {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return MissingConfigError{DisplayDir: DisplayConfigDir}
		}
	}

	return nil
}
