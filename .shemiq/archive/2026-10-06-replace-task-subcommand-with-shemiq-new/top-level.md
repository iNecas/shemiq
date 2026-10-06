# Replace task subcommand with shemiq new
:::shemiq
type: top-level
uuid: 2fe7fc42-8c8c-4c0f-885d-12c5221e11b9
:::

## Description

remove the task subcommand in favour of `shemiq new`

## Context

The Go CLI uses Cobra. `cmd/root.go` registers the `task` group and `validate`, while `cmd/new.go` nests `new` under `task`. The `new` command collects description arguments and either prompts for a title or accepts `--title`; `internal/task` handles project discovery, slug generation, document creation, and collision protection. Command-level tests live in `cmd/new_test.go`. The active `/shemiq-new` prompt (`prompts/shemiq-new.md`), `skills/shemiq/SKILL.md`, and `DEV.md` refer to `shemiq task new`; completed task documents are historical records.

## Interview

- Remove the `task` subcommand entirely: `shemiq task new` must not remain as an alias or a special migration command.
- Preserve the existing creation behavior under `shemiq new`: description arguments, optional `--title`, and the interactive title prompt when the flag is omitted.
- Prefer promoting the existing Cobra command directly under the root over extracting a new handler or adding routing infrastructure.
- Update active command instructions, but leave historical completed task documents unchanged.

## Design

- Register `new` directly under the Cobra root alongside `validate`, remove the `task` wrapper, and update the bare `shemiq` error to point to `shemiq new`. Do not change the `validate` command.
- Keep the current input and data flow: join required description arguments, use `--title` or prompt for a one-line title, then call `internal/task.FindProjectDirectory` and `internal/task.CreateTopLevelTask`. Keep the existing `.shemiq/` discovery or creation, slug rules, Markdown structure, collision protection, and absolute path printed on stdout.
- Preserve validation and filesystem error handling for invalid descriptions or titles, slug failures, collisions, and write errors. Let Cobra reject the removed `task` route normally rather than providing an alias or special error.
- Update active CLI references in the prompt, skill, and development example. Adapt the existing command-level tests to exercise `shemiq new`, assert the old route fails and bare `shemiq` points to the new usage, then run `go test ./...` and smoke-check the CLI. Avoid duplicating task-creation tests or changing metadata behavior.

## Current status

Implementation complete: `new` is now a root command; `task new` is removed. Active instructions and focused command tests are updated. `go test ./...` and CLI smoke checks passed.

## Tasks

### Promote `new` to the root CLI
:::shemiq
type: task
source: ./promote-new-to-root.md
status: done
:::

Move the existing command out of the `task` group and update focused command tests while preserving creation behavior.

Change the prompt, skill, and development example to use `shemiq new`, then verify the documented invocation works.
