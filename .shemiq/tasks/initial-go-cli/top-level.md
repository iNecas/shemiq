# Initial Go CLI
:::shemiq
type: top-level
:::

## Description

Build the initial Go CLI for Shemiq, starting with `shemiq task new` to create a top-level task.

## Context

Shemiq tracks work in Markdown under `.shemiq/tasks/`, using `:::shemiq` directives for task metadata. The README currently describes creating a top-level task by hand or through a Pi prompt; the CLI is listed as future work. The repository has a Go module but no Go implementation yet. The existing `.shemiq/tasks/initial-go-cli/top-level.md` illustrates the document structure to generate.

## Interview

- The first CLI feature is `shemiq task new`, creating only top-level tasks; subtask creation is out of scope.
- The description is required on the command line as the remaining arguments after `shemiq task new`. The command prompts interactively for a one-line title. Missing description is a usage error.
- Derive the task directory slug from the title; do not prompt for a slug or silently add a numeric suffix.
- Look upward from the working directory for the nearest `.shemiq/`. If none exists, create `.shemiq/` in the working directory.
- If the derived task directory already exists, fail without overwriting it.
- Keep title collection separate from task creation so a future agent call can suggest the title from the supplied description. Agent integration is not part of this task.

## Design

- Implement `shemiq task new <description...>` using Cobra for command routing, without a provider framework or extra extensibility scaffolding. Join the description arguments, prompt for a one-line title, validate the inputs, then pass the resulting data to task creation independently of how the title was obtained.
- Find the project directory by walking up to the nearest `.shemiq/`; when absent, create one in the current directory. Create `.shemiq/tasks/<slug>/top-level.md` and print its absolute path on success. Derive the slug using lowercase ASCII letters and digits separated by hyphens.
- Generate a top-level Markdown document with the title, `type: top-level` directive, supplied `## Description`, and `[TBD]` in the Context, Interview, Design, Current status, and Tasks sections. Do not create subtasks or other project files.
- Reject blank descriptions, blank titles or ended input, and titles that do not yield a nonempty filesystem-safe slug. Report collisions and filesystem errors clearly; avoid overwriting and best-effort clean up a newly created task directory if its document write fails. Crash-safe atomicity is out of scope.
- Keep tests focused on the command's end-to-end creation behavior: existing project discovery from a nested directory, new `.shemiq/` creation, required input, and collision protection. No agent integration tests are needed.

## Current status

The initial Go CLI implements `shemiq task new <description...>` with a prompted title, ASCII slug generation, upward project discovery, top-level document creation, collision protection, and focused command tests. The executable remains at the module root; the Cobra commands and tests now live in `cmd`, while reusable project discovery and task creation live in `internal/task`. Title collection remains separate from file creation for future agent integration. `task new` now also accepts `--title` to create a task without prompting; omitting the flag retains the interactive behavior. Flag titles must be nonblank and one line.

## Tasks

### Implement `shemiq task new`
:::shemiq
type: task
source: ./implement-task-new.md
status: done
:::

Build the interactive Go command to create a top-level task from a command-line description and prompted title, with project discovery, validation, and focused tests.

### Improve the code structure
:::shemiq
type: task
source: ./improve-code-structure.md
status: done
:::

While the initial implementation works, reorganize the CLI and task operations to support future task commands without changing behavior.

### Pass title via command line
:::shemiq
type: task
source: ./pass-title-via-command-line.md
status: done
:::

Currently, only interactive method of passing title in `task new` is supported.

Let's add `task new --title "The title" descriptino in the rest of args` to provide
the info on command line.

### Update the prompts and skills to start using shemiq binary

prompts/shemiq-new.md is the counterpart of the `shemiq task new`. Let's make
it more deterministic by using the binary for actual task creation.

`task new --title` is now available for a non-interactive prompt workflow; this task still needs to update the prompt and skill.

The skill in skills/shemiq/SKILL.md should be expanded for the awareness
of the binary and to use `make install` in the repo dir in case the binary is
not available.
