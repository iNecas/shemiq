# Pass title via command line
:::shemiq
type: task
parent: ./top-level.md
status: done
:::

## Context

`task new` currently reads a title from stdin after receiving the description as arguments. A future prompt/skill update will use the binary to create tasks non-interactively.

## Implementation plan

- Add an optional `--title` flag to `task new`; when provided, use its value instead of prompting, while retaining the existing prompt when omitted.
- Keep description handling and task creation unchanged so both title sources use the same validation, slug, collision, and document-writing behavior. An explicitly blank title must fail rather than fall back to a prompt.
- Add a focused command-level creation test for the non-interactive flag, and check that the existing interactive path still works. Run `go test ./...`.

## Implementation notes

- Added an optional Cobra `--title` flag to `task new`. When supplied, it skips the interactive prompt; when omitted, the existing title prompt remains. Both paths pass the title to the same task creation function.
- Explicit blank titles fail without falling back to stdin. Task creation also rejects line breaks in supplied titles to preserve the one-line document heading.
- Added a non-interactive command creation test and flag validation cases; the existing interactive tests still pass. `go test ./...`, `go vet ./...`, and `git diff --check` pass.

