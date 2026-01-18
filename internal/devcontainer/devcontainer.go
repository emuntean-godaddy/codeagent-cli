package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DirName = ".devcontainer"

func Dir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName)
}

func ValidateDir(projectRoot string) error {
	path := Dir(projectRoot)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		if os.IsNotExist(err) || err == nil {
			return fmt.Errorf(".devcontainer/ directory not found in current project.")
		}
		return fmt.Errorf("stat .devcontainer/: %w", err)
	}
	return nil
}

func UpdateName(jsonPath string, projectName string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	payload["name"] = projectName
	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(jsonPath, updated, mode); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}
	return nil
}

func readJSON(path string) (map[string]any, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("stat devcontainer.json: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read devcontainer.json: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, 0, fmt.Errorf("parse devcontainer.json: %w", err)
	}
	return payload, info.Mode().Perm(), nil
}
