package cmd_test

import (
	"path/filepath"
	"reflect"
)

func labelArgsFor(path string) []string {
	return labelArgsForConfig(path, filepath.Join(path, ".devcontainer", "devcontainer.json"))
}

func labelArgsForConfig(path string, configPath string) []string {
	return []string{
		"ps",
		"-a",
		"--filter", "label=devcontainer.config_file=" + configPath,
		"--filter", "label=devcontainer.local_folder=" + path,
		"--format", "{{.ID}}\t{{.State}}",
	}
}

func argsMatchAny(got []string, candidates ...[]string) bool {
	for _, candidate := range candidates {
		if reflect.DeepEqual(got, candidate) {
			return true
		}
	}
	return false
}
