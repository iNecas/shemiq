# Add shemiq refine CLI
:::shemiq
type: top-level
status: refined
uuid: be33cb2a-b82a-4659-ac91-f7152bed87e1
:::

## Description

Add a `shemiq refine` CLI that selects a top-level task and starts the appropriate collaborative refinement flow in Pi. Accept a task directory or its `top-level.md`:

```sh
shemiq refine .shemiq/tasks/my-task
shemiq refine .shemiq/tasks/my-task/top-level.md
```

For a new top-level task, start `/shemiq-refine-and-split`. For a refined top-level task, select a subtask on the CLI or interactively, then start `/shemiq-refine-sub-task`:

```sh
shemiq refine .shemiq/tasks/my-task --subtask "Subtask title"
shemiq refine .shemiq/tasks/my-task
```

Also support `shemiq refine` with no path: offer a list of top-level tasks to choose from. Choose a simple interactive selection mechanism and a way to launch Pi with the appropriate prompt. Target Pi now, but keep a well-defined boundary for other coding agent providers in the future.

## Context

The Go CLI uses Cobra commands registered in `cmd/root.go`; `cmd/new.go` and `internal/task/task.go` split command handling from project discovery and file operations. `internal/task/metadata.go` parses `:::shemiq` directives, and `internal/task/validate.go` currently accepts `new`, `progress`, and `done`. `shemiq new` creates `top-level.md` without `status:`, which must continue to work. Top-level documents list subtasks under `## Tasks` as headings with `type: task` directives; their separate `source:` files need not exist yet. Existing Pi templates are `prompts/shemiq-refine-and-split.md` and `prompts/shemiq-refine-sub-task.md`, installed as part of the Shemiq Pi package. `cmd/command_test.go` provides a CLI test helper; README and `skills/shemiq/SKILL.md` describe the workflow.

## Interview

- Replace `progress` with `refined` in the accepted status values: `new`, `refined`, `done`. A missing `status:` means `new`; existing documents need no migration.
- A new top-level task enters top-level refinement; a refined one enters subtask selection; a done one cannot be refined. Only new subtasks (including those without an explicit status) are eligible, even with `--subtask`.
- An explicit path may point to any existing `top-level.md` or its directory within a Shemiq project, including archived tasks. Locate the project from the target document, not just the caller's working directory; reject documents outside any `.shemiq/` project. Launch Pi from that project's root.
- Without a path, discover the nearest existing `.shemiq/` from the caller's working directory and offer a numbered picker of unfinished top-level tasks from its active `tasks/` directories; exclude archived and done tasks. With `--subtask` but no path, pick the top-level task first, then match its subtask title; reject `--subtask` when the chosen top-level task is still new.
- Use a numbered terminal list built with Go's standard library rather than a TUI dependency or external picker. Subtask titles and statuses come from the parent top-level document, not the separate subtask files.
- Launch interactive Pi using the installed Shemiq slash-command templates. The prompts, not the CLI process exit, set `status: refined` after the user accepts refinement; subtask refinement updates the parent-list directive and keeps the new subtask document consistent. Validation follows reciprocal `source:` / `parent:` links to detect status mismatches and `--fix` promotes the less advanced side (`new < refined < done`); missing `source:` targets remain allowed.
- Isolate agent invocation behind a small launcher interface carrying the resolved flow, document path, project root, and optional subtask title. Pi is the only implementation for now; no provider-selection flag is needed.

## Design

- Register `shemiq refine [path] [--subtask "Title"]` in the Cobra CLI. Resolve file/directory arguments from the caller's working directory, require a `top-level.md` in a project with an enclosing `.shemiq/`, and report missing or malformed documents. With no argument, enumerate direct child task directories of the nearest project's `.shemiq/tasks/` and show eligible top-level files in a numbered prompt. Do not create a project or change task files during selection.
- Use the top-level directive for top-level status and heading/directive pairs beneath `## Tasks` for subtask names and statuses. Treat missing status as new. Route new top-level tasks to the top-level flow, refined ones to eligible subtask selection or exact-title `--subtask` lookup, and reject done tasks. Reject no eligible tasks, ambiguous or ineligible titles, unsupported statuses, and cancelled/invalid selections before launching Pi.
- Keep selection independent of a minimal agent launcher. The Pi launcher starts an interactive child process at the selected project root with inherited terminal streams, passing an initial `/shemiq-refine-and-split <top-level-path>` or `/shemiq-refine-sub-task <top-level-path> "<title>"` prompt. Quote template arguments for Pi's command parser without using a shell; surface missing-executable and process failures. The installed Pi templates are a prerequisite, not bundled copies.
- Update both refinement prompts to set `refined` only after successful user-approved refinement; subtask refinement updates its authoritative parent-list status and the created subtask document. Update metadata validation and active user guidance for the new status lifecycle and CLI usage; leave historical archived documents unchanged. The CLI must never mark a task refined merely because Pi exits.
- Test the full CLI route through small temporary project fixtures and a fake Pi executable or injected launcher: no-argument selection, both explicit path forms, missing/explicit statuses, subtask selection, prompt arguments and working directory, and representative rejection/failure paths. Avoid live Pi or redundant parser-only test matrices.

## Current status

High-level design agreed. The refinement status lifecycle is implemented: validation follows reciprocal task links, repairs status mismatches with `--fix`, and accepts `new`, `refined`, and `done`. Both refinement prompts now record approved completion. Task resolution, terminal selection, Pi launch, and CLI usage documentation remain to be implemented.

## Tasks

### Establish the refinement status lifecycle
:::shemiq
type: task
source: ./establish-refinement-status-lifecycle.md
status: done
:::

Change accepted statuses to `new`, `refined`, and `done`, and update the refinement prompts to record completion.

### Resolve tasks and offer terminal selection
:::shemiq
type: task
source: ./resolve-tasks-and-offer-terminal-selection.md
status: new
:::

Add document discovery, status-based routing, and numbered top-level and subtask pickers.

Instead of launching pi, it will finish with "TODO: launch `pi ...`" with appropriate arguments

### Launch Pi from `shemiq refine`
:::shemiq
type: task
source: ./launch-pi-from-shemiq-refine.md
status: new
:::

Wire the CLI to a small launcher interface, invoke the installed Pi templates, add focused CLI coverage, and document usage.
