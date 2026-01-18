package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

var evalSymlinks = filepath.EvalSymlinks

type ContainerInfo struct {
	ID    string
	State State
}

func ContainerByLabel(ctx context.Context, runner Runner, labelKey, labelValue string) (ContainerInfo, error) {
	if labelKey == "" || labelValue == "" {
		return ContainerInfo{}, fmt.Errorf("label key and value are required")
	}

	filter := fmt.Sprintf("label=%s=%s", labelKey, labelValue)
	args := []string{
		"ps",
		"-a",
		"--filter", filter,
		"--format", "{{.ID}}\t{{.State}}",
	}
	result, err := runner.Run(ctx, "docker", args...)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("docker ps failed (%s): %v; stderr: %s", formatCommand("docker", args), err, strings.TrimSpace(result.Stderr))
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return ContainerInfo{State: StateMissing}, nil
	}

	var info ContainerInfo
	var count int
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return ContainerInfo{}, fmt.Errorf("unexpected docker output: %q", line)
		}
		count++
		info = ContainerInfo{
			ID:    strings.TrimSpace(parts[0]),
			State: mapState(strings.TrimSpace(parts[1])),
		}
	}

	if count == 0 {
		return ContainerInfo{State: StateMissing}, nil
	}
	if count > 1 {
		return ContainerInfo{}, fmt.Errorf("multiple containers matched label %s=%s", labelKey, labelValue)
	}
	return info, nil
}

func mapState(raw string) State {
	if strings.EqualFold(raw, "running") {
		return StateRunning
	}
	return StateStopped
}

func ContainerByLocalFolder(ctx context.Context, runner Runner, folder string) (ContainerInfo, error) {
	info, err := ContainerByLabel(ctx, runner, "devcontainer.local_folder", folder)
	if err != nil {
		return info, err
	}
	if info.State != StateMissing {
		return info, nil
	}

	resolved, err := evalSymlinks(folder)
	if err != nil || resolved == folder {
		return info, nil
	}

	infoResolved, err := ContainerByLabel(ctx, runner, "devcontainer.local_folder", resolved)
	if err != nil {
		return infoResolved, err
	}
	if infoResolved.State == StateMissing {
		return info, nil
	}
	return infoResolved, nil
}
