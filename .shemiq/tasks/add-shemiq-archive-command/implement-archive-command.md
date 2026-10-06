# Implement `shemiq archive`
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 12604e57-0070-47e9-ad48-cdc7e4551e41
:::

## Context

Archive exactly one immediate child directory of the nearest existing project's `.shemiq/tasks/`, without editing its contents. Accept the directory path or an existing `top-level.md` file within it. Fail on collisions and invalid paths.

## Implementation plan

- Add an `archive <path>` Cobra command and an `internal/task` operation that discovers an existing project, checks the selected task, and renames it to a UTC-dated directory under `.shemiq/archive/`.
- Keep the path and filesystem safety checks independent of metadata validation; print the absolute destination only after a successful move.
- Add focused temporary-project CLI coverage for both path forms, multi-file preservation, collisions and invalid scope; document command usage in the active skill.

## Implementation notes

- Registered `archive <path>` in `cmd/root.go`; `cmd/archive.go` resolves the working directory, delegates the move, and prints the destination only on success.
- `internal/task/archive.go` requires an existing project, accepts an immediate child task directory or its existing `top-level.md`, rejects symlinked task entries and invalid paths, and renames the complete directory into a UTC-dated archive destination after checking for collisions. No metadata is read or rewritten; errors from filesystem operations are returned.
- Documented the command in `skills/shemiq/SKILL.md`. Focused CLI tests cover both argument forms, multi-file byte preservation, destination/output, collision and invalid paths, and the missing-project case. `go test ./...` passes.
