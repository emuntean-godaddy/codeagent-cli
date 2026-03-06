package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	startRunner   docker.Runner                          = docker.ExecRunner{}
	startOut      io.Writer                              = os.Stdout
	startExecFunc func(string, []string, []string) error = syscall.Exec
	startLookPath func(string) (string, error)           = exec.LookPath
)

func SetStartRunner(runner docker.Runner) func() {
	previous := startRunner
	if runner == nil {
		startRunner = docker.ExecRunner{}
	} else {
		startRunner = runner
	}
	return func() {
		startRunner = previous
	}
}

func SetStartWriter(writer io.Writer) func() {
	previous := startOut
	if writer == nil {
		startOut = os.Stdout
	} else {
		startOut = writer
	}
	return func() {
		startOut = previous
	}
}

func SetStartExec(fn func(string, []string, []string) error) func() {
	previous := startExecFunc
	if fn == nil {
		startExecFunc = syscall.Exec
	} else {
		startExecFunc = fn
	}
	return func() {
		startExecFunc = previous
	}
}

func SetStartLookPath(fn func(string) (string, error)) func() {
	previous := startLookPath
	if fn == nil {
		startLookPath = exec.LookPath
	} else {
		startLookPath = fn
	}
	return func() {
		startLookPath = previous
	}
}

func newStartCmd() *cobra.Command {
	var command string
	var extraEnv []string
	var tag string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start or attach to the project devcontainer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(command, extraEnv, tag, cmd.Flags().Changed("command"))
		},
	}
	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to run inside the container")
	cmd.Flags().StringArrayVarP(&extraEnv, "env", "e", nil, "Environment variable to pass (KEY, KEY=VALUE, KEY=$LOCAL_ENV)")
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Tagged devcontainer to use from .devcontainer/<tag>/devcontainer.json")
	return cmd
}

func runStart(command string, extraEnv []string, tag string, commandSet bool) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeStartError(errutil.UserErrorf("resolve project root: %v", err))
	}

	if err := devcontainer.ValidateDir(projectRoot); err != nil {
		return writeStartError(errutil.UserError(err.Error()))
	}
	configPath, _, err := resolveDevcontainerConfig(projectRoot, tag)
	if err != nil {
		return writeStartError(err)
	}
	command, err = resolveStartCommand(configPath, command, commandSet)
	if err != nil {
		return writeStartError(err)
	}

	info, err := docker.ContainerByLocalFolderAndConfig(context.Background(), startRunner, projectRoot, configPath)
	if err != nil {
		return writeStartError(errutil.UserErrorf("resolve container state: %v", err))
	}

	if info.State == docker.StateMissing || info.State == docker.StateStopped {
		if err := runDevcontainerUp(projectRoot, configPath); err != nil {
			return writeStartError(err)
		}
		info, err = docker.ContainerByLocalFolderAndConfig(context.Background(), startRunner, projectRoot, configPath)
		if err != nil {
			return writeStartError(errutil.UserErrorf("resolve container state: %v", err))
		}
		if info.ID == "" {
			return writeStartError(errutil.UserError("container not found after devcontainer up"))
		}
	}

	if err := attachToContainer(info.ID, command, extraEnv); err != nil {
		return writeStartError(err)
	}
	return nil
}

func resolveStartCommand(configPath, command string, commandSet bool) (string, error) {
	if commandSet {
		if strings.TrimSpace(command) == "" {
			return "", errutil.UserError("command must not be empty")
		}
		return command, nil
	}

	fromConfig, ok, err := devcontainer.ReadStartCommand(configPath)
	if err != nil {
		return "", errutil.UserErrorf("read devcontainer.json customizations.codeagent.startCommand: %v", err)
	}
	if ok {
		return fromConfig, nil
	}
	return "codex --yolo", nil
}

func runDevcontainerUp(projectRoot, configPath string) error {
	args := []string{"up", "--workspace-folder", projectRoot}
	if strings.TrimSpace(configPath) != "" {
		args = append(args, "--config", configPath)
	}
	result, err := startRunner.Run(context.Background(), "devcontainer", args...)
	if err != nil {
		message := fmt.Sprintf("devcontainer up failed: %v; stderr: %s", err, strings.TrimSpace(result.Stderr))
		return errutil.UserError(message)
	}
	return nil
}

func attachToContainer(containerName string, command string, extraEnv []string) error {
	if strings.TrimSpace(command) == "" {
		return errutil.UserError("command must not be empty")
	}
	parsedEnv, err := parseExtraEnv(extraEnv)
	if err != nil {
		return err
	}

	dockerPath, err := startLookPath("docker")
	if err != nil {
		return errutil.UserError("docker not found in PATH")
	}

	if err := checkShell(containerName, "bash"); err != nil {
		if err := checkShell(containerName, "sh"); err != nil {
			return errutil.UserError("no supported shell found in container")
		}
		return execShell(dockerPath, containerName, "sh", command, parsedEnv)
	}
	return execShell(dockerPath, containerName, "bash", command, parsedEnv)
}

func checkShell(containerName, shell string) error {
	_, err := startRunner.Run(context.Background(), "docker", "exec", containerName, shell, "-lc", "exit 0")
	return err
}

func execShell(dockerPath, containerName, shell string, command string, extraEnv []string) error {
	args := []string{
		"docker",
		"exec",
		"-it",
	}
	for _, entry := range extraEnv {
		args = append(args, "-e", entry)
	}
	args = append(args, containerName, shell, "-lc", command)
	if err := startExecFunc(dockerPath, args, os.Environ()); err != nil {
		return errutil.UserErrorf("exec docker: %v", err)
	}
	return nil
}

func writeStartError(err error) error {
	if err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(startOut, err.Error())
	return err
}

func parseExtraEnv(entries []string) ([]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	var parsed []string
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			return nil, errutil.UserError("env must not be empty")
		}

		if !strings.Contains(trimmed, "=") {
			key := strings.TrimSpace(trimmed)
			if err := validateEnvKey(entry, key); err != nil {
				return nil, err
			}
			value, ok := os.LookupEnv(key)
			if !ok {
				return nil, errutil.UserErrorf("local env %q is not set", key)
			}
			parsed = append(parsed, key+"="+value)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		if err := validateEnvKey(entry, key); err != nil {
			return nil, err
		}

		value, err := resolveEnvValue(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, key+"="+value)
	}
	return parsed, nil
}

func validateEnvKey(raw, key string) error {
	if key == "" {
		return errutil.UserErrorf("invalid env %q, missing key", raw)
	}
	if !envKeyPattern.MatchString(key) {
		return errutil.UserErrorf("invalid env %q, key must match [A-Za-z_][A-Za-z0-9_]*", raw)
	}
	return nil
}

var envRefPattern = regexp.MustCompile(`^\$(\{[A-Za-z_][A-Za-z0-9_]*\}|[A-Za-z_][A-Za-z0-9_]*)$`)
var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func resolveEnvValue(raw string) (string, error) {
	matches := envRefPattern.FindStringSubmatch(raw)
	if len(matches) != 2 {
		return raw, nil
	}
	ref := matches[1]
	if strings.HasPrefix(ref, "{") && strings.HasSuffix(ref, "}") {
		ref = strings.TrimSuffix(strings.TrimPrefix(ref, "{"), "}")
	}
	value, ok := os.LookupEnv(ref)
	if !ok {
		return "", errutil.UserErrorf("local env %q is not set", ref)
	}
	return value, nil
}
