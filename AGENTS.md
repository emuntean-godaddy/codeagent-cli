# Repository Guidelines

## Project Structure and Module Organization
This repository is in the early bootstrap phase. Current structure:
- `docs/`: product and functional requirements.

Planned code layout (Go):
- `cmd/`: service entrypoints (one folder per binary, e.g., `cmd/api`).
- `internal/`: application and domain packages (not exported outside the repo).
- `internal/.../testdata`: fixture files when needed.

## Build, Test, and Development Commands
The codebase is not implemented yet. For Go once code exists:
- `go test ./...`: run all unit tests across modules.
- `go test ./internal/...`: run core package tests only.
- `go test ./cmd/...`: validate entrypoint packages.
- `make test`: run all tests with race detection.
- `make coverage`: run tests with race detection and generate `coverage.out`.

Add build/run commands when binaries are introduced (e.g., `go build ./cmd/api`).
Linting and formatting:
- `gofmt -w .`: format all Go files.
- `golangci-lint run`: run lint checks across the repo.

## Coding Style and Naming Conventions
- Go formatting: `gofmt` on all `.go` files.
- Linting: `golangci-lint` must pass on all changes.
- Package naming: lowercase, short, no underscores.
- Files: `snake_case.go` only when multiple words are required.
- Errors: wrap with `%w`; prefer sentinel errors for stable checks.

## Testing Guidelines
- Use Go’s standard `testing` package.
- Maintain at least 80% statement coverage per package.
- Test files end with `_test.go`.
- Table-driven tests for multiple input/output cases.
- Keep unit tests in the same package unless black-box testing is required (`package foo_test`).

## Commit and Pull Request Guidelines
No commit-message or PR template is defined yet. Until a convention is set:
- Use clear, imperative commit summaries (e.g., “Add Tetris game loop”).
- PRs should describe changes, include test commands run, and link any tracking issue if one exists.

## Critical Documentation Index
- **Repository Guidelines**: `AGENTS.md` - Contributor guide and conventions.
- **Project README**: `README.md` - Usage, setup, and command reference.
- **Product Requirements**: `docs/prd.md` - Scope and goals.
- **Functional Requirements**: `docs/frd.md` - MVP behavior and flows.
- **Implementation Plan (Jira tasks)**: `Jira/` - Ordered implementation tasks and acceptance criteria.
