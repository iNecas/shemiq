# Improve the code structure
:::shemiq
type: task
parent: ./top-level.md
status: new
:::

## Context

The initial `shemiq task new` command works, but its Cobra command and task creation code both live in root `package main` (`main.go` and `task.go`). The parent task and `implement-task-new.md` define the behavior to preserve. This refactor should make room for more task commands without designing a general framework or adding features. Keep the completed implementation task and its behavior contract intact.

## Interview

- Prefer a small CLI/task package boundary over either reorganizing only the root package or introducing separate domain, storage, and interaction layers.
- Structure for additional task commands, not for agent integration or new user-facing features.
- Keep the behavior of `task new` unchanged.
- Retain the root executable so `go install github.com/iNecas/shemiq@latest` installs a binary named `shemiq`. Put the importable Cobra command code in `./cmd` (`package cmd`, **not** `package main`); no `internal/cli` or `cmd/shemiq` is needed.

## Implementation plan

- Make root `main.go` a thin executable entry point that delegates to `cmd` and exits nonzero on command errors. Move the Cobra tree, argument handling, title prompt, and CLI output into `./cmd`. Expose only the small command entry point needed by root `main.go`; retain a way for command tests to supply stdin/stdout/stderr without launching a subprocess.
- Move task operations into `internal/task`: nearest `.shemiq/` discovery, validation and ASCII slug generation, top-level document rendering, filesystem creation, collision protection, and best-effort cleanup. Provide a small API for the command to find the project directory and create a top-level task from a project directory, description, and title. Task operations must not depend on Cobra, stdin, or CLI output. Other task commands should be able to reuse project discovery; do not add a general task model, storage interface, or agent/provider abstractions in anticipation of future work.
- Preserve the existing command contract: description arguments, stderr title prompt, error and usage output, absolute result path on stdout, Markdown structure, project discovery/creation, slug rules, input rejection, and collision behavior. This is a structural change only.
- Move/adapt existing Cobra end-to-end tests to the command package, retaining temporary-directory checks for existing/new projects, invalid input, output/document content, and non-overwrite behavior. Add tests only for meaningful gaps exposed by the move. Verify `go test ./...`, `go vet ./...`, and a root build. The existing root `make build`/`make run` and `go install` entry points should keep working; update development documentation only if a documented instruction changes.
