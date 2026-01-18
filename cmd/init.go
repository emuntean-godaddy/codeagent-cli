package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a .devcontainer from templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(overwrite)
		},
	}
	cmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite existing .devcontainer")
	return cmd
}

func runInit(overwrite bool) error {
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

	devDir := devcontainer.Dir(projectRoot)
	if _, err := os.Stat(devDir); err == nil {
		if !overwrite {
			return errutil.UserError(".devcontainer/ already exists. Use --overwrite to regenerate.")
		}
		if err := os.RemoveAll(devDir); err != nil {
			return errutil.UserErrorf("remove existing .devcontainer/: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return errutil.UserErrorf("check .devcontainer/: %v", err)
	}

	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return errutil.UserErrorf("create .devcontainer/: %v", err)
	}

	for _, name := range config.RequiredFiles {
		src := filepath.Join(configDir, name)
		dst := filepath.Join(devDir, name)
		if err := config.CopyFile(src, dst); err != nil {
			return errutil.UserErrorf("copy %s: %v", name, err)
		}
	}

	projectName := filepath.Base(projectRoot)
	jsonPath := filepath.Join(devDir, config.DevcontainerJSONName)
	if err := devcontainer.UpdateName(jsonPath, projectName); err != nil {
		return errutil.UserErrorf("update devcontainer.json name: %v", err)
	}

	return nil
}
