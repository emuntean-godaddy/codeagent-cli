package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
)

func resolveDevcontainerConfig(projectRoot, tag string) (string, string, error) {
	trimmedTag := strings.TrimSpace(tag)
	if trimmedTag != "" {
		if err := devcontainer.ValidateTag(trimmedTag); err != nil {
			return "", "", errutil.UserError(err.Error())
		}
		tagged := devcontainer.TaggedJSONPath(projectRoot, trimmedTag)
		if _, err := os.Stat(tagged); err != nil {
			if os.IsNotExist(err) {
				return "", "", errutil.UserErrorf("tagged devcontainer config not found: .devcontainer/%s/devcontainer.json", trimmedTag)
			}
			return "", "", errutil.UserErrorf("check tagged devcontainer config: %v", err)
		}
		return tagged, trimmedTag, nil
	}

	defaultPath := devcontainer.DefaultJSONPath(projectRoot)
	if _, err := os.Stat(defaultPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", errutil.UserError("default .devcontainer/devcontainer.json not found. Use --tag to select a tagged devcontainer.")
		}
		return "", "", errutil.UserErrorf("check default devcontainer config: %v", err)
	}
	return defaultPath, "default", nil
}

func displayConfigPath(projectRoot, configPath string) string {
	if rel, err := filepath.Rel(projectRoot, configPath); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return configPath
}

type devcontainerProfile struct {
	Selector   string
	ConfigPath string
}

func discoverDevcontainerProfiles(projectRoot string) ([]devcontainerProfile, error) {
	root := devcontainer.Dir(projectRoot)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errutil.UserError(".devcontainer/ directory not found in current project.")
		}
		return nil, errutil.UserErrorf("read .devcontainer/: %v", err)
	}

	var profiles []devcontainerProfile
	defaultPath := devcontainer.DefaultJSONPath(projectRoot)
	if info, err := os.Stat(defaultPath); err == nil && !info.IsDir() {
		profiles = append(profiles, devcontainerProfile{
			Selector:   "default",
			ConfigPath: defaultPath,
		})
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errutil.UserErrorf("check default devcontainer config: %v", err)
	}

	var tags []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tag := entry.Name()
		tagPath := devcontainer.TaggedJSONPath(projectRoot, tag)
		info, err := os.Stat(tagPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errutil.UserErrorf("check tagged devcontainer config: %v", err)
		}
		if info.IsDir() {
			continue
		}
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		profiles = append(profiles, devcontainerProfile{
			Selector:   tag,
			ConfigPath: devcontainer.TaggedJSONPath(projectRoot, tag),
		})
	}

	if len(profiles) == 0 {
		return nil, errutil.UserError("no devcontainer configuration found. Add .devcontainer/devcontainer.json or .devcontainer/<tag>/devcontainer.json")
	}
	return profiles, nil
}
