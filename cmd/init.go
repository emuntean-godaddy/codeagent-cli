package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var overwrite bool
	var configHome string
	var extraEnv []string
	var envTarget string
	var tag string
	var command string
	var imageName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a .devcontainer from templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(overwrite, configHome, extraEnv, envTarget, tag, command, imageName, cmd.Flags().Changed("command"))
		},
	}
	cmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite existing .devcontainer")
	cmd.Flags().StringVar(&configHome, "config-home", "", "Mount curated local config files into container home for selected agent")
	cmd.Flags().StringArrayVarP(&extraEnv, "env", "e", nil, "Environment variable for devcontainer.json (KEY, KEY=VALUE, KEY=$LOCAL_ENV)")
	cmd.Flags().StringVar(&envTarget, "env-target", "containerEnv", "devcontainer env target: containerEnv or remoteEnv")
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Initialize tagged devcontainer at .devcontainer/<tag>/devcontainer.json")
	cmd.Flags().StringVarP(&command, "command", "c", "", "Set customizations.codeagent.startCommand in devcontainer.json")
	cmd.Flags().StringVar(&imageName, "image-name", "", "Set devcontainer image and remove build configuration")
	return cmd
}

func runInit(overwrite bool, configHome string, extraEnv []string, envTarget string, tag string, command string, imageName string, commandSet bool) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return errutil.UserErrorf("resolve project root: %v", err)
	}

	configDir, err := config.Dir()
	if err != nil {
		return errutil.UserErrorf("resolve config directory: %v", err)
	}

	if err := config.Validate(configDir); err != nil {
		var missing config.MissingConfigError
		if errors.As(err, &missing) {
			return errutil.UserError(missing.Error())
		}
		return errutil.UserErrorf("validate config: %v", err)
	}

	jsonPath, err := prepareDevcontainerLayout(projectRoot, configDir, overwrite, strings.TrimSpace(tag))
	if err != nil {
		return err
	}
	projectName := filepath.Base(projectRoot)
	devcontainerName := projectName
	if strings.TrimSpace(tag) != "" {
		devcontainerName = projectName + "-" + strings.TrimSpace(tag)
	}
	if err := devcontainer.UpdateName(jsonPath, devcontainerName); err != nil {
		return errutil.UserErrorf("update devcontainer.json name: %v", err)
	}
	if strings.TrimSpace(imageName) != "" {
		if err := devcontainer.SetImage(jsonPath, imageName); err != nil {
			return errutil.UserErrorf("update devcontainer.json image: %v", err)
		}
	} else if strings.TrimSpace(tag) != "" {
		if err := devcontainer.UpdateBuildForTaggedConfig(jsonPath); err != nil {
			return errutil.UserErrorf("update devcontainer.json build: %v", err)
		}
	}
	if len(extraEnv) > 0 {
		target, err := parseInitEnvTarget(envTarget)
		if err != nil {
			return err
		}
		values, err := parseInitEnv(extraEnv)
		if err != nil {
			return err
		}
		if err := devcontainer.UpsertEnv(jsonPath, target, values); err != nil {
			return errutil.UserErrorf("update devcontainer.json env: %v", err)
		}
		if err := maybeSyncAuthJSON(configHome, values); err != nil {
			return err
		}
	}
	if commandSet && strings.TrimSpace(command) == "" {
		return errutil.UserError("command must not be empty")
	}
	if strings.TrimSpace(command) != "" {
		if err := devcontainer.UpdateStartCommand(jsonPath, strings.TrimSpace(command)); err != nil {
			return errutil.UserErrorf("update devcontainer.json customizations.codeagent.startCommand: %v", err)
		}
	}
	if strings.TrimSpace(configHome) != "" {
		localConfigHome, err := resolveConfigHomeLocalPath(configHome)
		if err != nil {
			return err
		}
		entries, err := readConfigHomeEntries(localConfigHome)
		if err != nil {
			return err
		}
		startCommand, ok, err := devcontainer.ReadStartCommand(jsonPath)
		if err != nil {
			return errutil.UserErrorf("read devcontainer.json start command: %v", err)
		}
		if !ok {
			return errutil.UserError("config-home requires codeagent start command to include codex or claude")
		}
		targetBase, profile, err := targetBaseForStartCommand(startCommand)
		if err != nil {
			return err
		}
		curated := devcontainer.CuratedConfigEntries(profile, entries)
		if err := devcontainer.UpdateConfigHomeMounts(jsonPath, strings.TrimSpace(configHome), targetBase, curated); err != nil {
			return errutil.UserErrorf("update devcontainer.json mounts: %v", err)
		}
	}

	return nil
}

func resolveConfigHomeLocalPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errutil.UserError("config-home must not be empty")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errutil.UserErrorf("resolve home directory: %v", err)
	}
	replaced := trimmed
	replaced = strings.Replace(replaced, "${env:HOME}", home, 1)
	replaced = strings.Replace(replaced, "$HOME", home, 1)
	replaced = strings.Replace(replaced, "~", home, 1)
	path := filepath.Clean(replaced)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errutil.UserErrorf("config-home path not found: %s", path)
		}
		return "", errutil.UserErrorf("check config-home path: %v", err)
	}
	if !info.IsDir() {
		return "", errutil.UserErrorf("config-home is not a directory: %s", path)
	}
	return path, nil
}

func readConfigHomeEntries(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errutil.UserErrorf("read config-home: %v", err)
	}
	out := make(map[string]bool, len(entries))
	for _, entry := range entries {
		out[entry.Name()] = true
	}
	return out, nil
}

func targetBaseForStartCommand(command string) (string, string, error) {
	lower := strings.ToLower(command)
	hasCodex := strings.Contains(lower, "codex")
	hasClaude := strings.Contains(lower, "claude")
	if hasCodex && hasClaude {
		return "", "", errutil.UserError("cannot infer config mount target: start command matches both codex and claude")
	}
	if hasCodex {
		return "/root/.codex", "codex", nil
	}
	if hasClaude {
		return "/root/.claude", "claude", nil
	}
	return "", "", errutil.UserError("config-home requires codeagent start command to include codex or claude")
}

var localEnvRefPattern = regexp.MustCompile(`^\$\{localEnv:([A-Za-z_][A-Za-z0-9_]*)\}$`)

func maybeSyncAuthJSON(configHome string, env map[string]string) error {
	if strings.TrimSpace(configHome) == "" {
		return nil
	}
	value, ok := env["OPENAI_API_KEY"]
	if !ok {
		return nil
	}
	matches := localEnvRefPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) != 2 {
		return nil
	}
	localConfigHome, err := resolveConfigHomeLocalPath(configHome)
	if err != nil {
		return err
	}

	localEnvVar := matches[1]
	token, ok := os.LookupEnv(localEnvVar)
	if !ok || strings.TrimSpace(token) == "" {
		return errutil.UserErrorf("local env %q is not set", localEnvVar)
	}

	authPath := filepath.Join(localConfigHome, "auth.json")
	info, err := os.Stat(authPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errutil.UserErrorf("auth.json not found at config-home: %s", authPath)
		}
		return errutil.UserErrorf("check auth.json: %v", err)
	}
	if info.IsDir() {
		return errutil.UserErrorf("auth.json path is a directory: %s", authPath)
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		return errutil.UserErrorf("read auth.json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return errutil.UserErrorf("parse auth.json: %v", err)
	}
	payload["OPENAI_API_KEY"] = token
	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return errutil.UserErrorf("marshal auth.json: %v", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(authPath, updated, info.Mode().Perm()); err != nil {
		return errutil.UserErrorf("write auth.json: %v", err)
	}
	return nil
}

func prepareDevcontainerLayout(projectRoot, configDir string, overwrite bool, tag string) (string, error) {
	rootDevDir := devcontainer.Dir(projectRoot)
	if strings.TrimSpace(tag) == "" {
		if _, err := os.Stat(rootDevDir); err == nil {
			if !overwrite {
				return "", errutil.UserError(".devcontainer/ already exists. Use --overwrite to regenerate.")
			}
			if err := os.RemoveAll(rootDevDir); err != nil {
				return "", errutil.UserErrorf("remove existing .devcontainer/: %v", err)
			}
		} else if !os.IsNotExist(err) {
			return "", errutil.UserErrorf("check .devcontainer/: %v", err)
		}
		if err := os.MkdirAll(rootDevDir, 0o755); err != nil {
			return "", errutil.UserErrorf("create .devcontainer/: %v", err)
		}
		for _, name := range config.RequiredFiles {
			src := filepath.Join(configDir, name)
			dst := filepath.Join(rootDevDir, name)
			if err := config.CopyFile(src, dst); err != nil {
				return "", errutil.UserErrorf("copy %s: %v", name, err)
			}
		}
		return devcontainer.DefaultJSONPath(projectRoot), nil
	}

	if err := devcontainer.ValidateTag(tag); err != nil {
		return "", errutil.UserError(err.Error())
	}
	tagDir := filepath.Join(rootDevDir, tag)
	if _, err := os.Stat(tagDir); err == nil {
		if !overwrite {
			return "", errutil.UserErrorf(".devcontainer/%s/ already exists. Use --overwrite to regenerate.", tag)
		}
		if err := os.RemoveAll(tagDir); err != nil {
			return "", errutil.UserErrorf("remove existing .devcontainer/%s/: %v", tag, err)
		}
	} else if !os.IsNotExist(err) {
		return "", errutil.UserErrorf("check .devcontainer/%s/: %v", tag, err)
	}

	if err := os.MkdirAll(tagDir, 0o755); err != nil {
		return "", errutil.UserErrorf("create .devcontainer/%s/: %v", tag, err)
	}
	if err := os.MkdirAll(rootDevDir, 0o755); err != nil {
		return "", errutil.UserErrorf("create .devcontainer/: %v", err)
	}
	dockerfileSrc := filepath.Join(configDir, config.DockerfileName)
	dockerfileDst := filepath.Join(rootDevDir, config.DockerfileName)
	if _, err := os.Stat(dockerfileDst); err != nil {
		if !os.IsNotExist(err) {
			return "", errutil.UserErrorf("check .devcontainer/%s: %v", config.DockerfileName, err)
		}
		if err := config.CopyFile(dockerfileSrc, dockerfileDst); err != nil {
			return "", errutil.UserErrorf("copy %s: %v", config.DockerfileName, err)
		}
	}
	jsonSrc := filepath.Join(configDir, config.DevcontainerJSONName)
	jsonDst := devcontainer.TaggedJSONPath(projectRoot, tag)
	if err := config.CopyFile(jsonSrc, jsonDst); err != nil {
		return "", errutil.UserErrorf("copy %s: %v", config.DevcontainerJSONName, err)
	}
	return jsonDst, nil
}

func parseInitEnvTarget(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "container", "containerenv":
		return devcontainer.EnvTargetContainer, nil
	case "remote", "remoteenv":
		return devcontainer.EnvTargetRemote, nil
	default:
		return "", errutil.UserErrorf("invalid env target %q, expected containerEnv or remoteEnv", raw)
	}
}

func parseInitEnv(entries []string) (map[string]string, error) {
	parsed := make(map[string]string, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			return nil, errutil.UserError("env must not be empty")
		}

		if !strings.Contains(trimmed, "=") {
			key := trimmed
			if err := validateEnvKey(entry, key); err != nil {
				return nil, err
			}
			parsed[key] = fmt.Sprintf("${localEnv:%s}", key)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		if err := validateEnvKey(entry, key); err != nil {
			return nil, err
		}
		parsed[key] = initEnvValue(strings.TrimSpace(parts[1]))
	}
	return parsed, nil
}

func initEnvValue(raw string) string {
	matches := envRefPattern.FindStringSubmatch(raw)
	if len(matches) != 2 {
		return raw
	}
	ref := matches[1]
	if strings.HasPrefix(ref, "{") && strings.HasSuffix(ref, "}") {
		ref = strings.TrimSuffix(strings.TrimPrefix(ref, "{"), "}")
	}
	return fmt.Sprintf("${localEnv:%s}", ref)
}
