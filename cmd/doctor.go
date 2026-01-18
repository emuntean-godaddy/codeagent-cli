package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/emuntean-godaddy/codeagent-cli/internal/config"
	"github.com/emuntean-godaddy/codeagent-cli/internal/devcontainer"
	"github.com/emuntean-godaddy/codeagent-cli/internal/docker"
	"github.com/emuntean-godaddy/codeagent-cli/internal/errutil"
	"github.com/emuntean-godaddy/codeagent-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	doctorRunner docker.Runner = docker.ExecRunner{}
	doctorOut    io.Writer     = os.Stdout
)

func SetDoctorRunner(runner docker.Runner) func() {
	previous := doctorRunner
	if runner == nil {
		doctorRunner = docker.ExecRunner{}
	} else {
		doctorRunner = runner
	}
	return func() {
		doctorRunner = previous
	}
}

func SetDoctorWriter(writer io.Writer) func() {
	previous := doctorOut
	if writer == nil {
		doctorOut = os.Stdout
	} else {
		doctorOut = writer
	}
	return func() {
		doctorOut = previous
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate local environment and configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	var errs []error

	if err := checkDockerCLI(); err != nil {
		errs = append(errs, err)
	}
	if err := checkDockerDaemon(); err != nil {
		errs = append(errs, err)
	}
	if err := checkConfig(); err != nil {
		errs = append(errs, err)
	}
	if err := checkDevcontainer(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errutil.UserError("doctor found errors")
	}
	return nil
}

func checkDockerCLI() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return writeDoctorError("Docker CLI", fmt.Errorf("docker not found in PATH"))
	}
	return writeDoctorOK("Docker CLI")
}

func checkDockerDaemon() error {
	result, err := doctorRunner.Run(context.Background(), "docker", "info")
	if err != nil {
		message := fmt.Sprintf("docker info failed: %v; stderr: %s", err, strings.TrimSpace(result.Stderr))
		return writeDoctorError("Docker Daemon", fmt.Errorf(message))
	}
	return writeDoctorOK("Docker Daemon")
}

func checkConfig() error {
	dir, err := config.Dir()
	if err != nil {
		return writeDoctorError("Config", err)
	}
	if err := config.Validate(dir); err != nil {
		return writeDoctorError("Config", err)
	}
	return writeDoctorOK("Config")
}

func checkDevcontainer() error {
	projectRoot, err := project.CurrentRoot()
	if err != nil {
		return writeDoctorError("Devcontainer", err)
	}
	if err := devcontainer.ValidateDir(projectRoot); err != nil {
		return writeDoctorError("Devcontainer", err)
	}
	return writeDoctorOK("Devcontainer")
}

func writeDoctorOK(name string) error {
	_, err := fmt.Fprintf(doctorOut, "%s: ok\n", name)
	return err
}

func writeDoctorError(name string, err error) error {
	if _, writeErr := fmt.Fprintf(doctorOut, "%s: %v\n", name, err); writeErr != nil {
		return writeErr
	}
	return err
}
