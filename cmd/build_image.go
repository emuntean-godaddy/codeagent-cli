package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var buildImageRunner docker.Runner = docker.ExecRunner{}
var buildImageOut io.Writer = os.Stderr

func SetBuildImageRunner(runner docker.Runner) func() {
	previous := buildImageRunner
	if runner == nil {
		buildImageRunner = docker.ExecRunner{}
	} else {
		buildImageRunner = runner
	}
	return func() {
		buildImageRunner = previous
	}
}

func SetBuildImageWriter(writer io.Writer) func() {
	previous := buildImageOut
	if writer == nil {
		buildImageOut = os.Stderr
	} else {
		buildImageOut = writer
	}
	return func() {
		buildImageOut = previous
	}
}

func newBuildImageCmd() *cobra.Command {
	var tag string
	var imageName string
	var setImage bool

	cmd := &cobra.Command{
		Use:   "build-image",
		Short: "Build a devcontainer image tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeBuildImageError(runBuildImage(tag, imageName, setImage))
		},
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Tagged devcontainer to use from .devcontainer/<tag>/devcontainer.json")
	cmd.Flags().StringVar(&imageName, "image-name", "", "Image name to build (for example: repo/name:tag)")
	cmd.Flags().BoolVar(&setImage, "set-image", false, "Update selected devcontainer.json to image mode after successful build")
	return cmd
}

func runBuildImage(tag, imageName string, setImage bool) error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return errutil.UserErrorf("resolve project root: %v", err)
	}

	if err := devcontainer.ValidateDir(projectRoot); err != nil {
		return errutil.UserError(err.Error())
	}
	configPath, _, err := resolveDevcontainerConfig(projectRoot, tag)
	if err != nil {
		return err
	}

	builtImage := strings.TrimSpace(imageName)
	if builtImage == "" {
		return errutil.UserError("image name required: set --image-name")
	}

	args := []string{
		"build",
		"--workspace-folder", projectRoot,
		"--config", configPath,
		"--image-name", builtImage,
	}
	result, err := buildImageRunner.Run(context.Background(), "devcontainer", args...)
	if err != nil {
		return errutil.UserError(fmt.Sprintf("devcontainer build failed: %v; stderr: %s", err, strings.TrimSpace(result.Stderr)))
	}
	if setImage {
		if err := devcontainer.SetImage(configPath, builtImage); err != nil {
			return errutil.UserErrorf("update devcontainer.json image: %v", err)
		}
	}
	return nil
}

func writeBuildImageError(err error) error {
	if err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(buildImageOut, err.Error())
	return err
}
