package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DirName = ".devcontainer"
const JSONName = "devcontainer.json"

const (
	EnvTargetContainer = "containerEnv"
	EnvTargetRemote    = "remoteEnv"
)

var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func Dir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName)
}

func DefaultJSONPath(projectRoot string) string {
	return filepath.Join(Dir(projectRoot), JSONName)
}

func TaggedJSONPath(projectRoot, tag string) string {
	return filepath.Join(Dir(projectRoot), tag, JSONName)
}

func ValidateTag(tag string) error {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" {
		return fmt.Errorf("tag must not be empty")
	}
	if !tagPattern.MatchString(trimmed) {
		return fmt.Errorf("invalid tag %q, expected [a-zA-Z0-9._-]+", tag)
	}
	return nil
}

func ValidateDir(projectRoot string) error {
	path := Dir(projectRoot)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		if os.IsNotExist(err) || err == nil {
			return fmt.Errorf(".devcontainer/ directory not found in current project.")
		}
		return fmt.Errorf("stat .devcontainer/: %w", err)
	}
	return nil
}

func UpdateName(jsonPath string, projectName string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	payload["name"] = projectName
	if err := upsertNameRunArg(payload, projectName); err != nil {
		return err
	}
	return writeJSON(jsonPath, mode, payload)
}

func upsertNameRunArg(payload map[string]any, projectName string) error {
	nameArg := "--name=" + projectName

	runArgsValue, ok := payload["runArgs"]
	if !ok {
		payload["runArgs"] = []any{nameArg}
		return nil
	}

	runArgs, ok := runArgsValue.([]any)
	if !ok {
		return fmt.Errorf("parse devcontainer.json: runArgs must be an array")
	}

	updated := make([]any, 0, len(runArgs)+1)
	replaced := false
	for _, arg := range runArgs {
		text, ok := arg.(string)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: runArgs entries must be strings")
		}
		if strings.HasPrefix(text, "--name=") {
			if !replaced {
				updated = append(updated, nameArg)
				replaced = true
			}
			continue
		}
		updated = append(updated, text)
	}
	if !replaced {
		updated = append([]any{nameArg}, updated...)
	}
	payload["runArgs"] = updated
	return nil
}

func UpdateCodexHomeMountSources(jsonPath string, codexHome string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	mountsValue, ok := payload["mounts"]
	if !ok {
		return nil
	}
	mountsRaw, ok := mountsValue.([]any)
	if !ok {
		return fmt.Errorf("parse devcontainer.json: mounts must be an array")
	}

	prefix := "${env:HOME}/.codex/"
	base := strings.TrimSuffix(strings.TrimSpace(codexHome), "/")
	replacementPrefix := base + "/"
	updatedMounts := make([]any, 0, len(mountsRaw))
	for _, mount := range mountsRaw {
		mountText, ok := mount.(string)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: mounts entries must be strings")
		}
		if strings.Contains(mountText, "source=${env:HOME}/.codex,") {
			mountText = strings.Replace(mountText, "source=${env:HOME}/.codex,", "source="+base+",", 1)
		}
		if strings.Contains(mountText, "source="+prefix) {
			mountText = strings.Replace(mountText, "source="+prefix, "source="+replacementPrefix, 1)
		}
		updatedMounts = append(updatedMounts, mountText)
	}
	payload["mounts"] = updatedMounts

	return writeJSON(jsonPath, mode, payload)
}

func UpdateConfigHomeMounts(jsonPath, configHomeSource, targetBase string, entries []string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	mountsValue, ok := payload["mounts"]
	if !ok {
		mountsValue = []any{}
	}
	mountsRaw, ok := mountsValue.([]any)
	if !ok {
		return fmt.Errorf("parse devcontainer.json: mounts must be an array")
	}

	managedTargets := managedTargetPaths()

	var filtered []any
	insertAt := -1
	for i, mount := range mountsRaw {
		mountText, ok := mount.(string)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: mounts entries must be strings")
		}
		if shouldReplaceMountEntry(mountText, managedTargets) {
			if insertAt == -1 {
				insertAt = i
			}
			continue
		}
		filtered = append(filtered, mountText)
	}
	if insertAt == -1 {
		insertAt = len(filtered)
	}

	newMounts := make([]any, 0, len(entries))
	baseSource := strings.TrimSuffix(strings.TrimSpace(configHomeSource), "/")
	baseTarget := strings.TrimSuffix(strings.TrimSpace(targetBase), "/")
	for _, entry := range entries {
		mode := "bind"
		if entry == "version.json" {
			mode = "bind,readonly"
		}
		newMounts = append(newMounts, fmt.Sprintf("source=%s/%s,target=%s/%s,type=%s", baseSource, entry, baseTarget, entry, mode))
	}

	updatedMounts := make([]any, 0, len(filtered)+len(newMounts))
	updatedMounts = append(updatedMounts, filtered[:insertAt]...)
	updatedMounts = append(updatedMounts, newMounts...)
	updatedMounts = append(updatedMounts, filtered[insertAt:]...)
	payload["mounts"] = updatedMounts

	return writeJSON(jsonPath, mode, payload)
}

func shouldReplaceMountEntry(mountText string, managedTargets map[string]struct{}) bool {
	if !strings.Contains(mountText, "target=") {
		return false
	}
	parts := strings.Split(mountText, ",")
	var target string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "target=") {
			target = strings.TrimPrefix(part, "target=")
			break
		}
	}
	if target == "" {
		return false
	}

	_, ok := managedTargets[target]
	return ok
}

func UpsertEnv(jsonPath, target string, env map[string]string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	if target != EnvTargetContainer && target != EnvTargetRemote {
		return fmt.Errorf("invalid env target %q", target)
	}

	targetValue, ok := payload[target]
	envMap := map[string]any{}
	if ok {
		asMap, ok := targetValue.(map[string]any)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: %s must be an object", target)
		}
		envMap = asMap
	}

	for key, value := range env {
		envMap[key] = value
	}
	payload[target] = envMap

	return writeJSON(jsonPath, mode, payload)
}

func UpdateStartCommand(jsonPath, command string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	customizations := map[string]any{}
	if current, ok := payload["customizations"]; ok {
		parsed, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: customizations must be an object")
		}
		customizations = parsed
	}
	codeagent := map[string]any{}
	if current, ok := customizations["codeagent"]; ok {
		parsed, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("parse devcontainer.json: customizations.codeagent must be an object")
		}
		codeagent = parsed
	}
	codeagent["startCommand"] = command
	customizations["codeagent"] = codeagent
	payload["customizations"] = customizations

	return writeJSON(jsonPath, mode, payload)
}

func ReadStartCommand(jsonPath string) (string, bool, error) {
	payload, _, err := readJSON(jsonPath)
	if err != nil {
		return "", false, err
	}

	customizations, ok := payload["customizations"]
	if ok {
		customizationsMap, ok := customizations.(map[string]any)
		if !ok {
			return "", false, fmt.Errorf("parse devcontainer.json: customizations must be an object")
		}
		if codeagent, ok := customizationsMap["codeagent"]; ok {
			codeagentMap, ok := codeagent.(map[string]any)
			if !ok {
				return "", false, fmt.Errorf("parse devcontainer.json: customizations.codeagent must be an object")
			}
			if startCommand, ok := codeagentMap["startCommand"]; ok {
				text, ok := startCommand.(string)
				if !ok {
					return "", false, fmt.Errorf("parse devcontainer.json: customizations.codeagent.startCommand must be a string")
				}
				trimmed := strings.TrimSpace(text)
				if trimmed != "" {
					return trimmed, true, nil
				}
			}
		}
	}

	value, ok := payload["postStartCommand"]
	if !ok {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("parse devcontainer.json: postStartCommand must be a string")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false, nil
	}
	return trimmed, true, nil
}

func CuratedConfigEntries(profile string, entries map[string]bool) []string {
	allowed := codexCuratedEntries()
	if profile == "claude" {
		allowed = claudeCuratedEntries()
	}
	var out []string
	for _, name := range allowed {
		if entries[name] {
			out = append(out, name)
		}
	}
	return out
}

func codexCuratedEntries() []string {
	return []string{
		"AGENTS.md",
		"codex_guides",
		"auth.json",
		"history.jsonl",
		"sessions",
		"log",
		"version.json",
	}
}

func claudeCuratedEntries() []string {
	return []string{
		"CLAUDE.md",
		"claude_guides",
		"projects",
		"history.jsonl",
		"settings.json",
		"todos",
		"plugins",
	}
}

func managedTargetPaths() map[string]struct{} {
	out := map[string]struct{}{}
	for _, entry := range codexCuratedEntries() {
		out["/root/.codex/"+entry] = struct{}{}
	}
	for _, entry := range claudeCuratedEntries() {
		out["/root/.claude/"+entry] = struct{}{}
	}
	return out
}

func UpdateBuildForTaggedConfig(jsonPath string) error {
	payload, mode, err := readJSON(jsonPath)
	if err != nil {
		return err
	}

	build, ok := payload["build"]
	if !ok {
		return nil
	}
	buildMap, ok := build.(map[string]any)
	if !ok {
		return fmt.Errorf("parse devcontainer.json: build must be an object")
	}
	buildMap["context"] = ".."
	buildMap["dockerfile"] = "../Dockerfile"
	payload["build"] = buildMap

	return writeJSON(jsonPath, mode, payload)
}

func writeJSON(path string, mode os.FileMode, payload map[string]any) error {
	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(path, updated, mode); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}
	return nil
}

func readJSON(path string) (map[string]any, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("stat devcontainer.json: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read devcontainer.json: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, 0, fmt.Errorf("parse devcontainer.json: %w", err)
	}
	return payload, info.Mode().Perm(), nil
}
