package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	stopRunner docker.Runner = docker.ExecRunner{}
	stopOut    io.Writer     = os.Stdout
)

func SetStopRunner(runner docker.Runner) func() {
	previous := stopRunner
	if runner == nil {
		stopRunner = docker.ExecRunner{}
	} else {
		stopRunner = runner
	}
	return func() {
		stopRunner = previous
	}
}

func SetStopWriter(writer io.Writer) func() {
	previous := stopOut
	if writer == nil {
		stopOut = os.Stdout
	} else {
		stopOut = writer
	}
	return func() {
		stopOut = previous
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the project devcontainer if running",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop()
		},
	}
}

func runStop() error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeStopError(errutil.UserErrorf("resolve project root: %v", err))
	}

	info, err := docker.ContainerByLocalFolder(context.Background(), stopRunner, projectRoot)
	if err != nil {
		return writeStopError(errutil.UserErrorf("resolve container state: %v", err))
	}

	projectName := filepath.Base(projectRoot)
	containerDisplay := info.ID
	if containerDisplay == "" {
		containerDisplay = "missing"
	}

	switch info.State {
	case docker.StateRunning:
		result, err := stopRunner.Run(context.Background(), "docker", "stop", info.ID)
		if err != nil {
			message := fmt.Sprintf("stop container: %v; stderr: %s", err, strings.TrimSpace(result.Stderr))
			return writeStopError(errutil.UserError(message))
		}
		return writeStopOutput(fmt.Sprintf("Stopped container for %s: %s\n", projectName, containerDisplay))
	case docker.StateStopped:
		return writeStopOutput(fmt.Sprintf("Container already stopped for %s: %s\n", projectName, containerDisplay))
	case docker.StateMissing:
		return writeStopOutput(fmt.Sprintf("Container missing for %s: %s\n", projectName, containerDisplay))
	default:
		return writeStopError(errutil.UserErrorf("unknown container state: %s", info.State))
	}
}

func writeStopOutput(message string) error {
	if _, err := fmt.Fprint(stopOut, message); err != nil {
		return errutil.UserErrorf("write stop output: %v", err)
	}
	return nil
}

func writeStopError(err error) error {
	if err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(stopOut, err.Error())
	return err
}
