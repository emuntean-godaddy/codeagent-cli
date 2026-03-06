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
codeagent init --tag frontend
codeagent init --config-home='${env:HOME}/.gocodex'
codeagent init -c "codex --yolo"
codeagent init --tag codex-5.1 -c "codex --yolo -m gpt-5.1-codex"
codeagent init --tag claude -c "~/.local/bin/claude"
codeagent init --image-name ans-search-api:devcontainer-base
codeagent init --config-home '${env:HOME}/.gocodex' -e 'OPENAI_API_KEY=$GOCODE_API_TOKEN' -e 'OPENAI_BASE_URL=$GOCODE_BASE_URL' -t gocode -c "codex -m gpt-5.3-codex"
codeagent init --config-home '${env:HOME}/.claude' -e ANTHROPIC_MODEL=claude-sonnet-4-6 -e 'ANTHROPIC_AUTH_TOKEN=$GOCODE_API_TOKEN' -e 'ANTHROPIC_BASE_URL=$GOCODE_BASE_URL' -t claude -c "IS_SANDBOX=1 claude --allow-dangerously-skip-permissions"
codeagent init -e "OPENAI_API_KEY" -e "OPENAI_BASE_URL=$LOCAL_OPENAI_BASE_URL"
codeagent init --env-target remote -e "GITHUB_TOKEN"
```
Flags:
- `-o, --overwrite`: overwrite existing `.devcontainer/`
- `--config-home`: mount curated config entries from local config-home into container (`/root/.codex/*` for codex commands, `/root/.claude/*` for claude commands)
- `-e, --env`: add devcontainer env entries, repeatable. Supported forms:
  - `KEY` -> writes `${localEnv:KEY}`
  - `KEY=VALUE` -> writes literal value
  - `KEY=$LOCAL_ENV` or `KEY=${LOCAL_ENV}` -> writes `${localEnv:LOCAL_ENV}`
- `--env-target`: write env entries to `containerEnv` (default) or `remoteEnv` (short forms `container` and `remote` are also accepted)
- `-t, --tag`: initialize a tagged config at `.devcontainer/<tag>/devcontainer.json` (`[a-zA-Z0-9._-]+`)
- `-c, --command`: set `customizations.codeagent.startCommand` in generated `devcontainer.json` (fails if empty)
- `--image-name`: set `image` in generated `devcontainer.json` and remove `build` (image mode)

Config-home behavior:
- `--config-home` path must exist locally.
- Curated entries are profile-specific:
  - Codex (`startCommand` contains `codex`): `AGENTS.md`, `codex_guides`, `auth.json`, `history.jsonl`, `sessions`, `log`, `version.json`
  - Claude (`startCommand` contains `claude`): `CLAUDE.md`, `claude_guides`, `projects`, `history.jsonl`, `settings.json`, `todos`, `plugins`
- `version.json` is mounted readonly; other curated entries are bind mounts.
- If `-e OPENAI_API_KEY=$LOCAL_ENV` is provided with `--config-home`, `auth.json` must exist in config-home and is updated locally so `OPENAI_API_KEY` matches resolved local env value.

Choosing `--env-target`:
- Prefer `container` (`containerEnv`) when the variable must be visible to all processes in the container (shells, background services, tasks, scripts, and tools run outside VS Code terminals).
- Prefer `remote` (`remoteEnv`) when the variable is only needed by VS Code-driven processes (integrated terminal, debug sessions, tasks, extensions) and you do not want it injected into every container process.

Common cases:
- Use `containerEnv`: service configuration, build/runtime flags, values needed by entrypoint/startup scripts, values required by non-VS Code `docker exec` workflows.
- Use `remoteEnv`: editor/debug-specific tokens, CLI credentials used only during interactive development, per-developer IDE settings that should not affect container-wide behavior.

Tagged init behavior:
- No tag: templates are copied to `.devcontainer/Dockerfile` and `.devcontainer/devcontainer.json` (current behavior).
- With `--tag`: template JSON is copied to `.devcontainer/<tag>/devcontainer.json`, shared Dockerfile is `.devcontainer/Dockerfile`.
- Name behavior: default profile sets `name` to `<project>`; tagged profile sets `name` to `<project>-<tag>`.
- For tagged JSON with a `build` section, paths are rewritten to keep Dockerfile resolution correct:
  - `build.context` -> `..`
  - `build.dockerfile` -> `../Dockerfile`
- If `--image-name` is provided, generated JSON uses `image` mode and removes `build`.
- `--overwrite` with `--tag` only replaces `.devcontainer/<tag>/` and does not remove other tags.

### build-image
Build a reusable devcontainer image tag.
```bash
codeagent build-image --image-name ans-search-api:devcontainer-base
codeagent build-image --tag claude --image-name ans-search-api:devcontainer-base
codeagent build-image --tag claude --image-name ans-search-api:devcontainer-base --set-image
```
Behavior:
- Runs `devcontainer build --workspace-folder <projectRoot> --config <resolved-config> --image-name <name>`.
- Does not modify `devcontainer.json`; it only builds/tags the image.
- With `--set-image`, selected `devcontainer.json` is switched to image mode (`image` set, `build` removed) after successful build.
Flags:
- `-t, --tag`: use `.devcontainer/<tag>/devcontainer.json`
- `--image-name`: image name to build (required)
- `--set-image`: update selected config to use the built image

### user guide (3 tags)
Example setup for three profiles that share one image:
- `claude`: Anthropic-based profile
- `codex`: local codex profile
- `gocode`: codex profile with gateway env overrides

Use one image tag string consistently across all commands. Example:
```bash
export AGENT_IMAGE="harbor.muntean.online/homelab/agent-sandbox:030620260200"
```

1. Initialize `claude` profile from local Claude config:
```bash
codeagent init \
  --config-home '${env:HOME}/.claude' \
  -e ANTHROPIC_MODEL=claude-sonnet-4-6 \
  -e 'ANTHROPIC_AUTH_TOKEN=$GOCODE_API_TOKEN' \
  -e 'ANTHROPIC_BASE_URL=$GOCODE_BASE_URL' \
  -t claude \
  -c "IS_SANDBOX=1 claude --allow-dangerously-skip-permissions"
```

2. Build the shared image from the `claude` build config:
```bash
codeagent build-image --image-name "$AGENT_IMAGE" -t claude
```

3. Switch `claude` to image mode (so start uses the shared image):
```bash
codeagent build-image --image-name "$AGENT_IMAGE" -t claude --set-image
```

4. Initialize `codex` profile directly in image mode:
```bash
codeagent init --config-home '${env:HOME}/.codex' -t codex --image-name "$AGENT_IMAGE"
```

5. Initialize `gocode` profile directly in image mode:
```bash
codeagent init \
  --config-home '${env:HOME}/.gocodex' \
  -e 'OPENAI_API_KEY=$GOCODE_API_TOKEN' \
  -e 'OPENAI_BASE_URL=$GOCODE_BASE_URL' \
  -t gocode \
  -c "codex -m gpt-5.3-codex --yolo" \
  --image-name "$AGENT_IMAGE"
```

6. Start each profile:
```bash
codeagent start -t claude
codeagent start -t codex
codeagent start -t gocode
```

Notes:
- If you use `agent-sendbox` in one command and `agent-sandbox` in another, Docker treats them as different image names.
- `build-image` builds from the selected profile config; it does not change config unless `--set-image` is provided.
- Rebuild later with the same image tag, then re-run `build-image --set-image` only when you want to repin a profile config explicitly.

### start
Start or attach to the project devcontainer. Uses `devcontainer up --workspace-folder <projectRoot>` for missing/stopped containers, then attaches via `docker exec`.
```bash
codeagent start
codeagent start --tag frontend
codeagent start -c "codex resume abc -yolo"
codeagent start -e "OPENAI_API_KEY=xxx" -e "OPENAI_BASE_URL=https://api"
codeagent start -e "OPENAI_API_KEY" -e "OPENAI_BASE_URL=$LOCAL_OPENAI_BASE_URL"
```
Flags:
- `-c, --command`: command to run inside the container
- `-e, --env`: add environment variables, repeatable. Supported forms:
  - `KEY=VALUE`
  - `KEY=$LOCAL_ENV` or `KEY=${LOCAL_ENV}` (resolved from local shell env, fail-fast if missing)
  - `KEY` (resolved from local shell env using same name, fail-fast if missing)
- `-t, --tag`: use `.devcontainer/<tag>/devcontainer.json`

Start resolution rules:
- If `.devcontainer/devcontainer.json` exists and `--tag` is not provided, start uses the default config.
- If default config is missing and `--tag` is not provided, start fails fast and requires `--tag`.
- Container identity includes selected config (`devcontainer.config_file`), so different tags map to different containers for the same project.
- If `-c/--command` is not passed, start uses selected config `customizations.codeagent.startCommand` when present; if missing, it falls back to legacy `postStartCommand`, then to `codex --yolo`.

### status
Show project and container state.
```bash
codeagent status
codeagent status --tag frontend
codeagent status --all
```
Output:
```
Project: <project-name>
Path: <absolute-path>
Devcontainer: <default|tag>
Config: <.devcontainer/.../devcontainer.json>
Container: <container-id | missing>
State: running | stopped | missing
```
Flags:
- `-t, --tag`: use `.devcontainer/<tag>/devcontainer.json`
- `--all`: show status for all available profiles (default + tags)
- `--all` and `--tag` are mutually exclusive

### stop
Stop the project devcontainer if running.
```bash
codeagent stop
codeagent stop --tag frontend
```
Flags:
- `-t, --tag`: use `.devcontainer/<tag>/devcontainer.json`

### destroy
Remove the project devcontainer (`docker rm -f`).
```bash
codeagent destroy
codeagent destroy --tag frontend
```
Flags:
- `-t, --tag`: use `.devcontainer/<tag>/devcontainer.json`

The same default-vs-tag resolution rules used by `start` also apply to `status`, `stop`, and `destroy`.

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
