# Promote `new` to the root CLI
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: f01a2ea9-79a4-4d72-a20c-391eeb0f517b
:::

## Context

Move the existing `task new` command to `shemiq new` without changing task creation behavior. Update active usage instructions and focused CLI coverage; leave completed historical task documents alone.

## Implementation plan

- Register the existing `new` Cobra command at the root, remove the `task` wrapper, and point bare-command guidance to `shemiq new`.
- Adapt command tests to exercise the new route, confirm the old route fails, and check bare-command guidance.
- Update the active prompt, skill, and development example; run Go tests and smoke-check the CLI.

## Implementation notes

- Registered the existing `new` command directly under the Cobra root, removed the `task` wrapper, and updated bare-command guidance. Task creation and validation internals were unchanged.
- Adapted command tests for the new invocation and added coverage for the removed route and bare-command guidance. Updated the active prompt, skill, and development example; left historical completed tasks unchanged.
- `go test ./...` passed. Built the CLI in a temporary directory and smoke-checked `new --title`, validation of its document, rejection of `task new`, and bare-command guidance.
