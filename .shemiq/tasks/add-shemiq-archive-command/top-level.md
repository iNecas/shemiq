# Add shemiq archive command
:::shemiq
type: top-level
uuid: 41607b7f-3020-4b75-9746-a3f9ca6c17fd
:::

## Description

Add `shemiq archive <path>` to move one top-level task directory from `.shemiq/tasks/` into a dated `.shemiq/archive/` directory.

## Context

The Go CLI registers root-level Cobra commands in `cmd/root.go`. `cmd/new.go` and `internal/task/task.go` provide a useful split between CLI handling, project discovery, and filesystem operations; `cmd/command_test.go` supports temporary-project CLI tests. `shemiq validate` scans task metadata, but archiving should not depend on validation. Active command guidance lives in `skills/shemiq/SKILL.md`. There is no existing archive convention or command.

## Interview

- Archive one task directory at a time, including all of its contents, to `.shemiq/archive/`; do not archive individual files or bulk-select completed tasks.
- Accept a task directory path or its `top-level.md` path. Resolve relative paths from the working directory and require an immediate child of the nearest existing project's `.shemiq/tasks/`. A directory need not contain `top-level.md`; a supplied file path must exist.
- Do not require a completed status, validate metadata, or change the contents of task documents.
- Prefix the destination directory name with the archive date in UTC: `YYYY-MM-DD-<original-task-directory-name>`. Fail rather than overwrite if that destination already exists.
- Move by renaming the directory, without copy-and-delete fallback, and print the absolute destination path on success.

## Design

- Register a Cobra `archive <path>` command beside `new` and `validate`; keep argument handling and success output in `cmd`, with project/path checks and the filesystem move in `internal/task`.
- Reuse upward `.shemiq/` discovery but require an existing project, without creating one. Resolve the argument to a task directory immediately under that project's `tasks/`; allow an existing `top-level.md` file as shorthand for its parent. Reject nonexistent, out-of-scope, non-directory, or symlinked task entries. Do not parse Markdown or inspect task statuses.
- Form `.shemiq/archive/YYYY-MM-DD-<name>/` using the current UTC date. Check for a destination collision, create the archive parent if necessary, and rename the complete directory. Leave file bytes and references untouched. Surface move and filesystem errors; emit no success path on failure.
- Keep verification minimal: a temporary-project CLI scenario covering an unchanged multi-file move, destination path and output, both argument forms (including a directory without `top-level.md`), plus collision and out-of-scope errors. Document usage and layout in the active skill; do not rewrite historical task documents.

## Current status

Implementation complete. `shemiq archive <path>` moves a task directory into a UTC-dated `.shemiq/archive/` destination without editing its contents; CLI coverage and active skill guidance are in place. `go test ./...` passes.

## Tasks

### Implement `shemiq archive`
:::shemiq
type: task
source: ./implement-archive-command.md
status: done
:::

Add the path-based archive command, safe directory move, focused CLI tests, and active skill documentation.
