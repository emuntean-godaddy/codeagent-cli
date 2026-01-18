package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	destroyRunner docker.Runner = docker.ExecRunner{}
	destroyOut    io.Writer     = os.Stdout
)

func SetDestroyRunner(runner docker.Runner) func() {
	previous := destroyRunner
	if runner == nil {
		destroyRunner = docker.ExecRunner{}
	} else {
		destroyRunner = runner
	}
	return func() {
		destroyRunner = previous
	}
}

func SetDestroyWriter(writer io.Writer) func() {
	previous := destroyOut
	if writer == nil {
		destroyOut = os.Stdout
	} else {
		destroyOut = writer
	}
	return func() {
		destroyOut = previous
	}
}

func newDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Delete the project devcontainer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDestroy()
		},
	}
}

func runDestroy() error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeDestroyError(errutil.UserErrorf("resolve project root: %v", err))
	}

	info, err := docker.ContainerByLocalFolder(context.Background(), destroyRunner, projectRoot)
	if err != nil {
		return writeDestroyError(errutil.UserErrorf("resolve container state: %v", err))
	}
	if info.State == docker.StateMissing || info.ID == "" {
		return writeDestroyError(errutil.UserError("container not found for project"))
	}

	result, err := destroyRunner.Run(context.Background(), "docker", "rm", "-f", info.ID)
	if err != nil {
		message := fmt.Sprintf("remove container: %v; stderr: %s", err, strings.TrimSpace(result.Stderr))
		return writeDestroyError(errutil.UserError(message))
	}

	return nil
}

func writeDestroyError(err error) error {
	if err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(destroyOut, err.Error())
	return err
}
