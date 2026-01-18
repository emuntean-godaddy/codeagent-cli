package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	statusRunner docker.Runner = docker.ExecRunner{}
	statusOut    io.Writer     = os.Stdout
)

func SetStatusRunner(runner docker.Runner) func() {
	previous := statusRunner
	if runner == nil {
		statusRunner = docker.ExecRunner{}
	} else {
		statusRunner = runner
	}
	return func() {
		statusRunner = previous
	}
}

func SetStatusWriter(writer io.Writer) func() {
	previous := statusOut
	if writer == nil {
		statusOut = os.Stdout
	} else {
		statusOut = writer
	}
	return func() {
		statusOut = previous
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show container state for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return errutil.UserErrorf("resolve project root: %v", err)
	}

	info, err := docker.ContainerByLocalFolder(context.Background(), statusRunner, projectRoot)
	if err != nil {
		return errutil.UserErrorf("resolve container state: %v", err)
	}

	projectName := filepath.Base(projectRoot)
	containerDisplay := info.ID
	if containerDisplay == "" {
		containerDisplay = "missing"
	}
	_, err = fmt.Fprintf(statusOut, "Project: %s\nPath: %s\nContainer: %s\nState: %s\n",
		projectName, projectRoot, containerDisplay, info.State)
	if err != nil {
		return errutil.UserErrorf("write status output: %v", err)
	}

	return nil
}
