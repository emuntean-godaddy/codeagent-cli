package identity

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

func ContainerName(absPath string) (string, error) {
	if absPath == "" {
		return "", fmt.Errorf("project path is empty")
	}
	if !filepath.IsAbs(absPath) {
		return "", fmt.Errorf("project path must be absolute")
	}
	sum := sha1.Sum([]byte(absPath))
	return "codeagent-" + hex.EncodeToString(sum[:]), nil
}
