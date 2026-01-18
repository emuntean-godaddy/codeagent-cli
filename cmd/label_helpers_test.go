package cmd_test

import "reflect"

func labelArgsFor(path string) []string {
	return []string{
		"ps",
		"-a",
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
