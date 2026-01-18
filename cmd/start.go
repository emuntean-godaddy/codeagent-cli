package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start or attach to the project devcontainer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(command, extraEnv)
		},
	}
	cmd.Flags().StringVarP(&command, "command", "c", "codex --yolo", "Command to run inside the container")
	cmd.Flags().StringArrayVarP(&extraEnv, "env", "e", nil, "Environment variable to pass (KEY=VALUE)")
	return cmd
}

func runStart(command string, extraEnv []string) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeStartError(errutil.UserErrorf("resolve project root: %v", err))
	}

	if err := devcontainer.ValidateDir(projectRoot); err != nil {
		return writeStartError(errutil.UserError(err.Error()))
	}

	info, err := docker.ContainerByLocalFolder(context.Background(), startRunner, projectRoot)
	if err != nil {
		return writeStartError(errutil.UserErrorf("resolve container state: %v", err))
	}

	if info.State == docker.StateMissing || info.State == docker.StateStopped {
		if err := runDevcontainerUp(projectRoot); err != nil {
			return writeStartError(err)
		}
		info, err = docker.ContainerByLocalFolder(context.Background(), startRunner, projectRoot)
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

func runDevcontainerUp(projectRoot string) error {
	args := []string{"up", "--workspace-folder", projectRoot}
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
		containerName,
		shell,
		"-lc",
		command,
	}
	env := append(os.Environ(), extraEnv...)
	if err := startExecFunc(dockerPath, args, env); err != nil {
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
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			return nil, errutil.UserErrorf("invalid env %q, expected KEY=VALUE", entry)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, errutil.UserErrorf("invalid env %q, missing key", entry)
		}
		if strings.Contains(key, " ") {
			return nil, errutil.UserErrorf("invalid env %q, key contains space", entry)
		}
		value := strings.TrimSpace(parts[1])
		parsed = append(parsed, key+"="+value)
	}
	return parsed, nil
}
