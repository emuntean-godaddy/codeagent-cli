package errutil_test

import (
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
)

func TestUserError(t *testing.T) {
	t.Parallel()

	err := errutil.UserError("missing file")
	if err == nil {
		t.Fatalf("UserError() error = nil, want error")
	}
	if err.Error() != "Error: missing file" {
		t.Fatalf("UserError() = %q, want %q", err.Error(), "Error: missing file")
	}
}

func TestUserErrorf(t *testing.T) {
	t.Parallel()

	err := errutil.UserErrorf("missing %s", "config")
	if err == nil {
		t.Fatalf("UserErrorf() error = nil, want error")
	}
	if err.Error() != "Error: missing config" {
		t.Fatalf("UserErrorf() = %q, want %q", err.Error(), "Error: missing config")
	}
}
