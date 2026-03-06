package devcontainer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
)

func TestValidateDirMissing(t *testing.T) {
	t.Parallel()

	if err := devcontainer.ValidateDir(t.TempDir()); err == nil {
		t.Fatalf("ValidateDir() error = nil, want error")
	}
}

func TestValidateDirNotDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, devcontainer.DirName)
	if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.ValidateDir(dir); err == nil {
		t.Fatalf("ValidateDir() error = nil, want error")
	}
}

func TestUpdateNameAddsField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	contents := `{"image":"golang:1.22"}`
	if err := os.WriteFile(jsonPath, []byte(contents), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateName(jsonPath, "project-a"); err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if payload["name"] != "project-a" {
		t.Fatalf("name = %v, want %v", payload["name"], "project-a")
	}
	if payload["image"] != "golang:1.22" {
		t.Fatalf("image = %v, want %v", payload["image"], "golang:1.22")
	}

	info, err := os.Stat(jsonPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), 0o640)
	}
}

func TestUpdateNameInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateName(jsonPath, "project-b"); err == nil {
		t.Fatalf("UpdateName() error = nil, want error")
	}
}

func TestUpdateNameMissingFile(t *testing.T) {
	t.Parallel()

	if err := devcontainer.UpdateName(filepath.Join(t.TempDir(), "devcontainer.json"), "project-c"); err == nil {
		t.Fatalf("UpdateName() error = nil, want error")
	}
}

func TestUpdateCodexHomeMountSources(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	contents := `{
  "mounts": [
    "source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
    "source=${env:HOME}/.codex/AGENTS.md,target=/root/.codex/AGENTS.md,type=bind",
    "source=${env:HOME}/.codex/sessions,target=/root/.codex/sessions,type=bind"
  ]
}`
	if err := os.WriteFile(jsonPath, []byte(contents), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateCodexHomeMountSources(jsonPath, "${env:HOME}/.gocodex"); err != nil {
		t.Fatalf("UpdateCodexHomeMountSources() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	mounts, ok := payload["mounts"].([]any)
	if !ok {
		t.Fatalf("mounts type = %T, want []any", payload["mounts"])
	}
	got := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		text, ok := mount.(string)
		if !ok {
			t.Fatalf("mount entry type = %T, want string", mount)
		}
		got = append(got, text)
	}

	want := []string{
		"source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
		"source=${env:HOME}/.gocodex/AGENTS.md,target=/root/.codex/AGENTS.md,type=bind",
		"source=${env:HOME}/.gocodex/sessions,target=/root/.codex/sessions,type=bind",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mount[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUpsertEnvContainerTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	contents := `{
  "containerEnv": {
    "EXISTING": "1"
  }
}`
	if err := os.WriteFile(jsonPath, []byte(contents), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpsertEnv(jsonPath, devcontainer.EnvTargetContainer, map[string]string{
		"OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}",
	}); err != nil {
		t.Fatalf("UpsertEnv() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	env, ok := payload["containerEnv"].(map[string]any)
	if !ok {
		t.Fatalf("containerEnv type = %T, want map[string]any", payload["containerEnv"])
	}
	if env["EXISTING"] != "1" {
		t.Fatalf("EXISTING = %v, want 1", env["EXISTING"])
	}
	if env["OPENAI_API_KEY"] != "${localEnv:OPENAI_API_KEY}" {
		t.Fatalf("OPENAI_API_KEY = %v, want ${localEnv:OPENAI_API_KEY}", env["OPENAI_API_KEY"])
	}
}

func TestUpsertEnvRemoteTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(`{"name":"x"}`), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpsertEnv(jsonPath, devcontainer.EnvTargetRemote, map[string]string{
		"TOKEN": "${localEnv:TOKEN}",
	}); err != nil {
		t.Fatalf("UpsertEnv() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	env, ok := payload["remoteEnv"].(map[string]any)
	if !ok {
		t.Fatalf("remoteEnv type = %T, want map[string]any", payload["remoteEnv"])
	}
	if env["TOKEN"] != "${localEnv:TOKEN}" {
		t.Fatalf("TOKEN = %v, want ${localEnv:TOKEN}", env["TOKEN"])
	}
}

func TestUpdateBuildForTaggedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(`{"build":{"context":".","dockerfile":"Dockerfile"}}`), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateBuildForTaggedConfig(jsonPath); err != nil {
		t.Fatalf("UpdateBuildForTaggedConfig() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	build, ok := payload["build"].(map[string]any)
	if !ok {
		t.Fatalf("build type = %T, want map[string]any", payload["build"])
	}
	if build["context"] != ".." {
		t.Fatalf("build.context = %v, want ..", build["context"])
	}
	if build["dockerfile"] != "../Dockerfile" {
		t.Fatalf("build.dockerfile = %v, want ../Dockerfile", build["dockerfile"])
	}
}

func TestValidateTag(t *testing.T) {
	t.Parallel()

	if err := devcontainer.ValidateTag("frontend.v2-1"); err != nil {
		t.Fatalf("ValidateTag(valid) error = %v", err)
	}
	if err := devcontainer.ValidateTag("bad/tag"); err == nil {
		t.Fatalf("ValidateTag(invalid) error = nil, want error")
	}
}

func TestUpdateStartCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(`{"customizations":{"codeagent":{"startCommand":"old"}}}`), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateStartCommand(jsonPath, "codex --yolo"); err != nil {
		t.Fatalf("UpdateStartCommand() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	customizations, ok := payload["customizations"].(map[string]any)
	if !ok {
		t.Fatalf("customizations type = %T, want map[string]any", payload["customizations"])
	}
	codeagent, ok := customizations["codeagent"].(map[string]any)
	if !ok {
		t.Fatalf("codeagent type = %T, want map[string]any", customizations["codeagent"])
	}
	if codeagent["startCommand"] != "codex --yolo" {
		t.Fatalf("startCommand = %v, want codex --yolo", codeagent["startCommand"])
	}
}

func TestReadStartCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(`{"customizations":{"codeagent":{"startCommand":"claude"}}}`), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, ok, err := devcontainer.ReadStartCommand(jsonPath)
	if err != nil {
		t.Fatalf("ReadStartCommand() error = %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != "claude" {
		t.Fatalf("command = %q, want %q", got, "claude")
	}
}

func TestReadStartCommandFallsBackToPostStartCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(`{"postStartCommand":"legacy-cmd"}`), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, ok, err := devcontainer.ReadStartCommand(jsonPath)
	if err != nil {
		t.Fatalf("ReadStartCommand() error = %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != "legacy-cmd" {
		t.Fatalf("command = %q, want %q", got, "legacy-cmd")
	}
}

func TestUpdateConfigHomeMountsReplacesManagedMountsAcrossProfiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "devcontainer.json")
	contents := `{
  "mounts": [
    "source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
    "source=${env:HOME}/.codex/AGENTS.md,target=/root/.codex/AGENTS.md,type=bind",
    "source=${env:HOME}/.codex/sessions,target=/root/.codex/sessions,type=bind"
  ]
}`
	if err := os.WriteFile(jsonPath, []byte(contents), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := devcontainer.UpdateConfigHomeMounts(jsonPath, "${env:HOME}/.claude", "/root/.claude", []string{"CLAUDE.md", "projects"}); err != nil {
		t.Fatalf("UpdateConfigHomeMounts() error = %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	mounts, ok := payload["mounts"].([]any)
	if !ok {
		t.Fatalf("mounts type = %T, want []any", payload["mounts"])
	}
	var got []string
	for _, mount := range mounts {
		text, ok := mount.(string)
		if !ok {
			t.Fatalf("mount type = %T, want string", mount)
		}
		got = append(got, text)
	}

	for _, mount := range got {
		if strings.Contains(mount, "target=/root/.codex/") {
			t.Fatalf("unexpected codex mount left after claude update: %q", mount)
		}
	}
	required := []string{
		"source=${env:HOME}/.claude/CLAUDE.md,target=/root/.claude/CLAUDE.md,type=bind",
		"source=${env:HOME}/.claude/projects,target=/root/.claude/projects,type=bind",
	}
	for _, expected := range required {
		found := false
		for _, mount := range got {
			if mount == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing mount %q in %v", expected, got)
		}
	}
}
