# archive interactive
:::shemiq
type: top-level
uuid: 47d2c7ea-0e1f-47e4-8f37-2d94f6cfdb5b
status: done
:::

## Description

- `shemiq archive` currently requires an explicit task path, while other commands already support interactive selection.
- Allow choosing a top-level task interactively when no path is provided, while preserving explicit-path archiving.

## Context

- `cmd/archive.go` currently requires one argument and owns the existing archive operation and filesystem safety checks.
- `cmd/refine.go` and `cmd/implement.go` provide the existing discovery and numbered-selection patterns.
- Reuse `internal/task.Store` for discovery and `internal/console.Console.Pick` for selection.
- Existing archiving moves the whole task directory unchanged into `.shemiq/archive/YYYY-MM-DD-<task-directory-name>/`, using the UTC date.
- Explicit-path archiving ignores metadata and status, accepts a task directory or its `top-level.md`, and supports directories without a top-level document.
- Prompts and diagnostics belong on stderr; successful output is only the absolute archive destination path on stdout.

## Interview

- Include all task statuses (`new`, `refined`, and `done`); unfinished work may be archived.
- Choose existing store-based discovery over direct directory listing or hybrid title/fallback discovery.
- Limit interactive candidates to recognized `top-level.md` documents in immediate child directories of `.shemiq/tasks/`.
- Display `Title [status]`; identify selections by numbered entry, allowing duplicate titles.
- Archive immediately after selection without another confirmation.
- Always prompt, even for one candidate; retain existing picker input and cancellation behavior.
- Propagate discovery/parsing failures rather than silently skipping damaged documents.
- Add only one successful interactive CLI test and rely on existing coverage otherwise.
- Update command help, `README.md`, and `skills/shemiq/SKILL.md`.
- Architecture, data flow, error handling, testing, and the single-subtask split were approved.

## Design

### Architecture and components

- Accept zero or one path argument; keep orchestration in `cmd/archive.go`.
- With an explicit path, retain the existing archive flow without metadata loading or an interactive menu.
- Without a path, discover tasks in the nearest existing project's `.shemiq/tasks/` through `task.Store`.
- Include only recognized `.shemiq/tasks/<task>/top-level.md` documents, across all statuses.
- Exclude nonstandard filenames, nested top-level documents, and directories without a qualifying document from interactive discovery; explicit-path support remains unchanged.
- Reuse `Console.Pick` with numbered `Title [status]` entries, including when only one candidate exists.
- Duplicate titles do not make selection ambiguous because each numbered entry identifies its candidate.

### Data flow

- Discover candidates → filter to the eligible layout → display labels → resolve the selected task directory → invoke the existing archive operation.
- Archive immediately after selection; do not add automatic selection or a confirmation step.
- Preserve the entire directory and its contents without rewriting metadata.
- Retain the UTC-dated archive destination and print its absolute path followed by a newline on success.

### Error handling

- Report clear errors for no eligible tasks or no existing project; never create a project as an archive side effect.
- Propagate store loading and parsing errors; malformed Markdown in the discovery scope can prevent selection.
- Blank or invalid input repeats the prompt; EOF cancels selection.
- Discovery failure or cancellation must not move a task or alter its contents.
- Retain immediate-child containment, symlink rejection, collision checks, and filesystem error reporting from the existing archive operation.
- Keep menus and diagnostics on stderr and only the successful destination path on stdout.

### Testing and documentation

- Add one happy-path command-level test in `cmd/archive_test.go` using the existing CLI harness.
- Exercise interactive selection through the directory move, checking the menu, stdout/stderr separation, and preserved contents.
- Keep existing explicit-path and collision tests; rely on existing coverage otherwise, without a new error-case matrix or separate picker/store suite.
- Update command help, `README.md`, and `skills/shemiq/SKILL.md` to explain the optional path, candidate scope, status labels, and immediate archive action.

## Current status

- Interactive archive selection is complete: optional path, all-status numbered picker, standard-layout filtering, and immediate archiving.
- Explicit-path behavior and existing archive safety checks remain unchanged; help, README, and skill documentation are updated.
- One happy-path CLI test was added; `go test ./...` and `go vet ./...` pass.
- Implementation details are recorded in [the task file](./interactive-archive-selection.md).

## Tasks

### Add interactive archive selection
:::shemiq
type: task
source: ./interactive-archive-selection.md
status: done
:::

Add optional-path archive selection using the existing task store and console, preserve explicit-path behavior, and include one happy-path CLI test plus help and documentation updates.
