package docker

import (
	"context"
	"fmt"
	"strings"
)

type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateMissing State = "missing"
)

func ContainerState(ctx context.Context, runner Runner, containerName string) (State, error) {
	if containerName == "" {
		return "", fmt.Errorf("container name is required")
	}

	args := []string{
		"ps",
		"-a",
		"--filter", "name=" + containerName,
		"--format", "{{.Names}}\t{{.State}}",
	}
	result, err := runner.Run(ctx, "docker", args...)
	if err != nil {
		return "", fmt.Errorf("docker ps failed (%s): %v; stderr: %s", formatCommand("docker", args), err, strings.TrimSpace(result.Stderr))
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return StateMissing, nil
	}

	var exactState string
	var exactCount int
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("unexpected docker output: %q", line)
		}
		name := strings.TrimSpace(parts[0])
		state := strings.TrimSpace(parts[1])
		if name == containerName {
			exactCount++
			exactState = state
		}
	}

	if exactCount == 0 {
		return StateMissing, nil
	}
	if exactCount > 1 {
		return "", fmt.Errorf("multiple containers matched exact name %q", containerName)
	}

	if strings.EqualFold(exactState, "running") {
		return StateRunning, nil
	}
	return StateStopped, nil
}
