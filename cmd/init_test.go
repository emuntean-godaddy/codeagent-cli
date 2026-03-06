package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emuntean-godaddy/codeagent-cli/cmd"
	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
)

func TestInitMissingConfig(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != "Error: "+config.DisplayConfigDir+" configuration missing.\nExpected:\n  - "+config.DisplayConfigDir+"/Dockerfile\n  - "+config.DisplayConfigDir+"/devcontainer.json" {
		t.Fatalf("init error = %q, want missing config error", err.Error())
	}
}

func TestInitCreatesDevcontainer(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["name"] != filepath.Base(projectDir) {
		t.Fatalf("name = %v, want %v", payload["name"], filepath.Base(projectDir))
	}
	runArgs, ok := payload["runArgs"].([]any)
	if !ok {
		t.Fatalf("runArgs type = %T, want []any", payload["runArgs"])
	}
	if len(runArgs) != 1 || runArgs[0] != "--name="+filepath.Base(projectDir) {
		t.Fatalf("runArgs = %v, want [--name=%s]", runArgs, filepath.Base(projectDir))
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

func TestInitWithImageNameSetsImageMode(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"build":{"context":".","dockerfile":"Dockerfile"}}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--image-name", "ans-search-api:devcontainer-base"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["image"] != "ans-search-api:devcontainer-base" {
		t.Fatalf("image = %v, want ans-search-api:devcontainer-base", payload["image"])
	}
	if _, ok := payload["build"]; ok {
		t.Fatalf("build exists, want removed")
	}
}

func TestInitOverwrite(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DockerfileName), []byte("old\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(devDir, config.DevcontainerJSONName), []byte(`{"name":"old"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--overwrite"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(devDir, config.DockerfileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "FROM scratch\n" {
		t.Fatalf("Dockerfile = %q, want %q", string(data), "FROM scratch\n")
	}
}

func TestInitExistingDevcontainerNoOverwrite(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	devDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != "Error: .devcontainer/ already exists. Use --overwrite to regenerate." {
		t.Fatalf("init error = %q, want overwrite message", err.Error())
	}
}

func TestInitConfigHomeBuildsCuratedMounts(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	configHome := filepath.Join(home, ".gocodex")
	if err := os.MkdirAll(filepath.Join(configHome, "codex_guides"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, "sessions"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, "log"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "AGENTS.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "auth.json"), []byte(`{"OPENAI_API_KEY":"old"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "history.jsonl"), []byte("[]"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "version.json"), []byte(`{"v":"1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	devcontainerJSON := `{
  "customizations":{"codeagent":{"startCommand":"codex --yolo"}},
  "mounts": [
    "source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
    "source=${env:HOME}/.codex/AGENTS.md,target=/root/.codex/AGENTS.md,type=bind",
    "source=${env:HOME}/.codex/sessions,target=/root/.codex/sessions,type=bind"
  ]
}`
	if err := setupConfig(home, "FROM scratch\n", devcontainerJSON, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--config-home", "${env:HOME}/.gocodex"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
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
			t.Fatalf("mount type = %T, want string", mount)
		}
		got = append(got, text)
	}
	want := []string{
		"source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
		"source=${env:HOME}/.gocodex/AGENTS.md,target=/root/.codex/AGENTS.md,type=bind",
		"source=${env:HOME}/.gocodex/codex_guides,target=/root/.codex/codex_guides,type=bind",
		"source=${env:HOME}/.gocodex/auth.json,target=/root/.codex/auth.json,type=bind",
		"source=${env:HOME}/.gocodex/history.jsonl,target=/root/.codex/history.jsonl,type=bind",
		"source=${env:HOME}/.gocodex/sessions,target=/root/.codex/sessions,type=bind",
		"source=${env:HOME}/.gocodex/log,target=/root/.codex/log,type=bind",
		"source=${env:HOME}/.gocodex/version.json,target=/root/.codex/version.json,type=bind,readonly",
	}
	for _, expected := range want {
		found := false
		for _, mount := range got {
			if mount == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mount missing: %q in %v", expected, got)
		}
	}
}

func TestInitConfigHomeSyncsAuthJSON(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOCODE_API_TOKEN", "new-token")
	configHome := filepath.Join(home, ".gocodex")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "auth.json"), []byte(`{"OPENAI_API_KEY":"old-token"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := setupConfig(home, "FROM scratch\n", `{"customizations":{"codeagent":{"startCommand":"codex --yolo"}}}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "init", "--config-home", "${env:HOME}/.gocodex", "-e", "OPENAI_API_KEY=$GOCODE_API_TOKEN"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configHome, "auth.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["OPENAI_API_KEY"] != "new-token" {
		t.Fatalf("OPENAI_API_KEY = %v, want new-token", payload["OPENAI_API_KEY"])
	}
}

func TestInitConfigHomeAuthJSONMissingFails(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOCODE_API_TOKEN", "new-token")
	configHome := filepath.Join(home, ".gocodex")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := setupConfig(home, "FROM scratch\n", `{"customizations":{"codeagent":{"startCommand":"codex --yolo"}}}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "init", "--config-home", "${env:HOME}/.gocodex", "-e", "OPENAI_API_KEY=$GOCODE_API_TOKEN"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if !strings.Contains(err.Error(), "auth.json not found") {
		t.Fatalf("init error = %q, want missing auth.json error", err.Error())
	}
}

func TestInitConfigHomeClaudeCuratedMounts(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	configHome := filepath.Join(home, ".claude")
	for _, dir := range []string{"claude_guides", "projects", "todos", "plugins"} {
		if err := os.MkdirAll(filepath.Join(configHome, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	for _, file := range []string{"CLAUDE.md", "history.jsonl", "settings.json"} {
		if err := os.WriteFile(filepath.Join(configHome, file), []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(configHome, "ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	devcontainerJSON := `{
  "customizations":{"codeagent":{"startCommand":"claude"}},
  "mounts": [
    "source=${env:HOME}/.gitconfig,target=/root/.gitconfig,type=bind,readonly",
    "source=${env:HOME}/.claude/CLAUDE.md,target=/root/.claude/CLAUDE.md,type=bind"
  ]
}`
	if err := setupConfig(home, "FROM scratch\n", devcontainerJSON, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"codeagent", "init", "--config-home", "${env:HOME}/.claude"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
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

	want := []string{
		"source=${env:HOME}/.claude/CLAUDE.md,target=/root/.claude/CLAUDE.md,type=bind",
		"source=${env:HOME}/.claude/claude_guides,target=/root/.claude/claude_guides,type=bind",
		"source=${env:HOME}/.claude/projects,target=/root/.claude/projects,type=bind",
		"source=${env:HOME}/.claude/history.jsonl,target=/root/.claude/history.jsonl,type=bind",
		"source=${env:HOME}/.claude/settings.json,target=/root/.claude/settings.json,type=bind",
		"source=${env:HOME}/.claude/todos,target=/root/.claude/todos,type=bind",
		"source=${env:HOME}/.claude/plugins,target=/root/.claude/plugins,type=bind",
	}
	for _, expected := range want {
		found := false
		for _, mount := range got {
			if mount == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mount missing: %q in %v", expected, got)
		}
	}
}

func TestInitEnvDefaultContainerTarget(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "-e", "OPENAI_API_KEY", "-e", "OPENAI_BASE_URL=$LOCAL_OPENAI_BASE_URL", "-e", "PLAIN=value"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	containerEnv, ok := payload["containerEnv"].(map[string]any)
	if !ok {
		t.Fatalf("containerEnv type = %T, want map[string]any", payload["containerEnv"])
	}
	if containerEnv["OPENAI_API_KEY"] != "${localEnv:OPENAI_API_KEY}" {
		t.Fatalf("OPENAI_API_KEY = %v, want ${localEnv:OPENAI_API_KEY}", containerEnv["OPENAI_API_KEY"])
	}
	if containerEnv["OPENAI_BASE_URL"] != "${localEnv:LOCAL_OPENAI_BASE_URL}" {
		t.Fatalf("OPENAI_BASE_URL = %v, want ${localEnv:LOCAL_OPENAI_BASE_URL}", containerEnv["OPENAI_BASE_URL"])
	}
	if containerEnv["PLAIN"] != "value" {
		t.Fatalf("PLAIN = %v, want value", containerEnv["PLAIN"])
	}
}

func TestInitEnvRemoteTarget(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--env-target", "remote", "-e", "GITHUB_TOKEN"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	remoteEnv, ok := payload["remoteEnv"].(map[string]any)
	if !ok {
		t.Fatalf("remoteEnv type = %T, want map[string]any", payload["remoteEnv"])
	}
	if remoteEnv["GITHUB_TOKEN"] != "${localEnv:GITHUB_TOKEN}" {
		t.Fatalf("GITHUB_TOKEN = %v, want ${localEnv:GITHUB_TOKEN}", remoteEnv["GITHUB_TOKEN"])
	}
}

func TestInitInvalidEnvTarget(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--env-target", "bad", "-e", "A"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != `Error: invalid env target "bad", expected containerEnv or remoteEnv` {
		t.Fatalf("init error = %q, want invalid env target error", err.Error())
	}
}

func TestInitWithTagCreatesTaggedConfigAndSharedDockerfile(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	devcontainerJSON := `{
  "build": {
    "context": ".",
    "dockerfile": "Dockerfile"
  }
}`
	if err := setupConfig(home, "FROM scratch\n", devcontainerJSON, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--tag", "gocode"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".devcontainer", config.DockerfileName)); err != nil {
		t.Fatalf("shared Dockerfile stat error = %v", err)
	}
	jsonPath := filepath.Join(projectDir, ".devcontainer", "gocode", config.DevcontainerJSONName)
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
	if payload["name"] != filepath.Base(projectDir)+"-gocode" {
		t.Fatalf("name = %v, want %v", payload["name"], filepath.Base(projectDir)+"-gocode")
	}
	if build["context"] != ".." {
		t.Fatalf("build.context = %v, want ..", build["context"])
	}
	if build["dockerfile"] != "../Dockerfile" {
		t.Fatalf("build.dockerfile = %v, want ../Dockerfile", build["dockerfile"])
	}
}

func TestInitWithTagAndImageNameUsesImageMode(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	devcontainerJSON := `{
  "build": {
    "context": ".",
    "dockerfile": "Dockerfile"
  }
}`
	if err := setupConfig(home, "FROM scratch\n", devcontainerJSON, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--tag", "claude", "--image-name", "ans-search-api:devcontainer-base"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", "claude", config.DevcontainerJSONName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["image"] != "ans-search-api:devcontainer-base" {
		t.Fatalf("image = %v, want ans-search-api:devcontainer-base", payload["image"])
	}
	if _, ok := payload["build"]; ok {
		t.Fatalf("build exists, want removed")
	}
}

func TestInitWithTagOverwriteOnlyTaggedDir(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	rootDev := filepath.Join(projectDir, ".devcontainer")
	frontendDir := filepath.Join(rootDev, "frontend")
	backendDir := filepath.Join(rootDev, "backend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(frontend) error = %v", err)
	}
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(backend) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, config.DevcontainerJSONName), []byte(`{"name":"old-front"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(frontend) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, config.DevcontainerJSONName), []byte(`{"name":"keep-me"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(backend) error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--tag", "frontend", "--overwrite"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	backendData, err := os.ReadFile(filepath.Join(backendDir, config.DevcontainerJSONName))
	if err != nil {
		t.Fatalf("ReadFile(backend) error = %v", err)
	}
	if string(backendData) != `{"name":"keep-me"}` {
		t.Fatalf("backend config changed = %q, want untouched", string(backendData))
	}
}

func TestInitWithInvalidTag(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--tag", "bad/tag"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != `Error: invalid tag "bad/tag", expected [a-zA-Z0-9._-]+` {
		t.Fatalf("init error = %q, want invalid tag error", err.Error())
	}
}

func TestInitCommandSetsPostStartCommand(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"customizations":{"codeagent":{"startCommand":"old"}}}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "-c", "codex --yolo"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", config.DevcontainerJSONName)
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
		t.Fatalf("customizations.codeagent type = %T, want map[string]any", customizations["codeagent"])
	}
	if codeagent["startCommand"] != "codex --yolo" {
		t.Fatalf("customizations.codeagent.startCommand = %v, want codex --yolo", codeagent["startCommand"])
	}
}

func TestInitCommandSetsPostStartCommandForTag(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"customizations":{"codeagent":{"startCommand":"old-tag"}}}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "--tag", "claude", "-c", "~/.local/bin/claude"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := runCommand(); err != nil {
		t.Fatalf("init error = %v", err)
	}

	jsonPath := filepath.Join(projectDir, ".devcontainer", "claude", config.DevcontainerJSONName)
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
		t.Fatalf("customizations.codeagent type = %T, want map[string]any", customizations["codeagent"])
	}
	if codeagent["startCommand"] != "~/.local/bin/claude" {
		t.Fatalf("customizations.codeagent.startCommand = %v, want ~/.local/bin/claude", codeagent["startCommand"])
	}
}

func TestInitCommandEmptyFails(t *testing.T) {
	projectDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setupConfig(home, "FROM scratch\n", `{"image":"golang:1.22"}`, 0o640); err != nil {
		t.Fatalf("setupConfig() error = %v", err)
	}

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"codeagent", "init", "-c", "   "}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = runCommand()
	if err == nil {
		t.Fatalf("init error = nil, want error")
	}
	if err.Error() != "Error: command must not be empty" {
		t.Fatalf("init error = %q, want empty command error", err.Error())
	}
}

func runCommand() error {
	return cmd.Execute()
}

func setupConfig(home, dockerfile, jsonContent string, mode os.FileMode) error {
	dir := filepath.Join(home, config.ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, config.DockerfileName), []byte(dockerfile), mode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, config.DevcontainerJSONName), []byte(jsonContent), mode); err != nil {
		return err
	}
	return nil
}
