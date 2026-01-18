package docker

import "path/filepath"

func SetEvalSymlinks(fn func(string) (string, error)) func() {
	previous := evalSymlinks
	if fn == nil {
		evalSymlinks = filepath.EvalSymlinks
	} else {
		evalSymlinks = fn
	}
	return func() {
		evalSymlinks = previous
	}
}
