package errutil

import "fmt"

func UserError(msg string) error {
	return fmt.Errorf("Error: %s", msg)
}

func UserErrorf(format string, args ...any) error {
	return fmt.Errorf("Error: %s", fmt.Sprintf(format, args...))
}
