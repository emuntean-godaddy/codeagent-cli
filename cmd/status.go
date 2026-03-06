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
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show container state for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && tag != "" {
				return writeStatusError(errutil.UserError("cannot combine --all with --tag"))
			}
			return runStatus(tag, all)
		},
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Tagged devcontainer to use from .devcontainer/<tag>/devcontainer.json")
	cmd.Flags().BoolVar(&all, "all", false, "Show status for all available devcontainer profiles")
	return cmd
}

func runStatus(tag string, all bool) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeStatusError(errutil.UserErrorf("resolve project root: %v", err))
	}

	if all {
		profiles, err := discoverDevcontainerProfiles(projectRoot)
		if err != nil {
			return writeStatusError(err)
		}
		projectName := filepath.Base(projectRoot)
		for i, profile := range profiles {
			info, err := docker.ContainerByLocalFolderAndConfig(context.Background(), statusRunner, projectRoot, profile.ConfigPath)
			if err != nil {
				return writeStatusError(errutil.UserErrorf("resolve container state: %v", err))
			}
			containerDisplay := info.ID
			if containerDisplay == "" {
				containerDisplay = "missing"
			}
			if i > 0 {
				if _, err := fmt.Fprintln(statusOut); err != nil {
					return writeStatusError(errutil.UserErrorf("write status output: %v", err))
				}
			}
			_, err = fmt.Fprintf(statusOut, "Project: %s\nPath: %s\nDevcontainer: %s\nConfig: %s\nContainer: %s\nState: %s\n",
				projectName, projectRoot, profile.Selector, displayConfigPath(projectRoot, profile.ConfigPath), containerDisplay, info.State)
			if err != nil {
				return writeStatusError(errutil.UserErrorf("write status output: %v", err))
			}
		}
		return nil
	}

	configPath, selector, err := resolveDevcontainerConfig(projectRoot, tag)
	if err != nil {
		return writeStatusError(err)
	}
	info, err := docker.ContainerByLocalFolderAndConfig(context.Background(), statusRunner, projectRoot, configPath)
	if err != nil {
		return writeStatusError(errutil.UserErrorf("resolve container state: %v", err))
	}

	projectName := filepath.Base(projectRoot)
	containerDisplay := info.ID
	if containerDisplay == "" {
		containerDisplay = "missing"
	}
	_, err = fmt.Fprintf(statusOut, "Project: %s\nPath: %s\nDevcontainer: %s\nConfig: %s\nContainer: %s\nState: %s\n",
		projectName, projectRoot, selector, displayConfigPath(projectRoot, configPath), containerDisplay, info.State)
	if err != nil {
		return writeStatusError(errutil.UserErrorf("write status output: %v", err))
	}

	return nil
}

func writeStatusError(err error) error {
	if err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(statusOut, err.Error())
	return err
}
