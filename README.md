# CodeAgent

CodeAgent is a Go CLI that manages VS Code devcontainers for the current project directory. It standardizes devcontainer setup and provides start/stop/status/doctor workflows without manual Docker commands.

## Requirements
- macOS or Linux
- Docker CLI + running Docker daemon
- Dev Containers CLI (`devcontainer`)

## Install
```bash
make install
```
This installs `codeagent` into your Go bin directory (`$GOBIN` or `$GOPATH/bin`).

## Templates and Configuration
CodeAgent expects template files in `~/.codeagent`:
- `~/.codeagent/Dockerfile`
- `~/.codeagent/devcontainer.json`

`codeagent init` copies these into the current project as `.devcontainer/` and sets `devcontainer.json` `name` to the project folder basename.

## Commands

### init
Initialize `.devcontainer/` from templates.
```bash
codeagent init
codeagent init --overwrite
```

### start
Start or attach to the project devcontainer. Uses `devcontainer up --workspace-folder <projectRoot>` for missing/stopped containers, then attaches via `docker exec`.
```bash
codeagent start
codeagent start -c "codex resume abc -yolo"
codeagent start -e "OPENAI_API_KEY=xxx" -e "OPENAI_BASE_URL=https://api"
```
Flags:
- `-c, --command`: command to run inside the container (default `codex --yolo`)
- `-e, --env`: add environment variables (`KEY=VALUE`), repeatable

### status
Show project and container state.
```bash
codeagent status
```
Output:
```
Project: <project-name>
Path: <absolute-path>
Container: <container-id | missing>
State: running | stopped | missing
```

### stop
Stop the project devcontainer if running.
```bash
codeagent stop
```

### destroy
Remove the project devcontainer (`docker rm -f`).
```bash
codeagent destroy
```

### doctor
Validate local environment and configuration.
```bash
codeagent doctor
```
Checks:
- Docker CLI availability
- Docker daemon reachable
- `~/.codeagent` config presence
- `.devcontainer/` presence in the project

## How CodeAgent Finds Containers
CodeAgent looks up containers by label:
```
devcontainer.local_folder=<absolute-project-path>
```
It handles macOS path aliasing (`/var` vs `/private/var`) by retrying with a symlink-resolved path.

## Troubleshooting
- `.devcontainer/ directory not found`: run `codeagent init`.
- `docker not found in PATH`: install Docker CLI.
- `no supported shell found in container`: ensure `bash` or `sh` exists in the image.
- `container not found for project`: run `codeagent start` to create it.

## Development
```bash
make test
make coverage
```
