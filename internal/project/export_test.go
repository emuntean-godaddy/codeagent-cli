package project

import "os"

func SetGetwd(fn func() (string, error)) func() {
	previous := getwd
	if fn == nil {
		getwd = os.Getwd
	} else {
		getwd = fn
	}
	return func() {
		getwd = previous
	}
}
