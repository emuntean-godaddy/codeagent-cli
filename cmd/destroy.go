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
	var tag string
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete the project devcontainer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDestroy(tag)
		},
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Tagged devcontainer to use from .devcontainer/<tag>/devcontainer.json")
	return cmd
}

func runDestroy(tag string) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeDestroyError(errutil.UserErrorf("resolve project root: %v", err))
	}
	configPath, selector, err := resolveDevcontainerConfig(projectRoot, tag)
	if err != nil {
		return writeDestroyError(err)
	}
	projectName := filepath.Base(projectRoot)

	info, err := docker.ContainerByLocalFolderAndConfig(context.Background(), destroyRunner, projectRoot, configPath)
	if err != nil {
		return writeDestroyError(errutil.UserErrorf("resolve container state: %v", err))
	}
	if info.State == docker.StateMissing || info.ID == "" {
		return writeDestroyError(errutil.UserErrorf("container not found for project (%s)", selector))
	}

	result, err := destroyRunner.Run(context.Background(), "docker", "rm", "-f", info.ID)
	if err != nil {
		message := fmt.Sprintf("remove container: %v; stderr: %s", err, strings.TrimSpace(result.Stderr))
		return writeDestroyError(errutil.UserError(message))
	}

	if _, err := fmt.Fprintf(destroyOut, "Destroyed container for %s (%s): %s\n", projectName, selector, info.ID); err != nil {
		return writeDestroyError(errutil.UserErrorf("write destroy output: %v", err))
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
