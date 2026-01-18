package identity_test

import (
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/identity"
)

func TestContainerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute path",
			path: "/tmp/codeagent-test",
			want: "codeagent-918e37d53e532daaaf4a3091ee8ce3f0a1be873e",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := identity.ContainerName(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ContainerName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Fatalf("ContainerName() = %q, want %q", got, tt.want)
			}
		})
	}
}
