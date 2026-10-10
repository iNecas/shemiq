# Add shemiq implement command
:::shemiq
type: top-level
status: refined
uuid: f4febffd-4b70-4c20-b2ba-3d879f59c340
:::

## Description

Add `shemiq implement`, following the CLI workflow established in
`.shemiq/archive/2026-10-07-add-shemiq-refine-cli/top-level.md`, which added
`refine` and interactive task selection.

Implement subtasks in either `new` or `refined` state, displaying their statuses
in interactive selection for transparency. Refined subtasks use
`/shemiq-implement-task`; new subtasks use `/shemiq-implement-task-parent`.
Implementation always targets a subtask, even when its top-level task contains
only one subtask.

Support top-level document/directory paths with optional exact-title
`--subtask` selection, interactive selection without a path, and direct
subtask-file paths.

## Context

- The Go CLI uses Cobra commands registered in `cmd/root.go`. `cmd/refine.go`
  already resolves explicit tasks, offers numbered selection, matches exact
  subtask titles, and hands a resolved request to an agent launcher.
- `internal/task/store.go` provides scoped task discovery and document loading.
  Queries do not follow metadata references; a file-scoped store cannot query
  another document. Paths are resolved relative to the invocation directory,
  with symlink-aware ownership based on the resolved document.
- `internal/task/task.go` exposes parent-list subtask titles, types, statuses,
  and resolved `source:` paths through `Task.Path`. It already parses and
  resolves `parent:` references privately; implementation needs read access
  to that resolved parent path.
- Accepted statuses are `new`, `refined`, and `done`; omitted status means
  `new`. Parent-list entries live beneath `## Tasks` in a top-level document.
  Their `source:` targets may not exist yet. Full metadata validation and
  repairs are separate operations and must remain separate from selection.
- `internal/console/console.go` provides the existing numbered picker and
  cancellation behavior. `internal/agent/agent.go` defines the provider-neutral
  request and launcher; `internal/agent/pi.go` launches fresh interactive Pi
  sessions with original terminal streams and parser-compatible quoting.
- Both installed implementation templates already exist:
  `prompts/shemiq-implement-task.md` takes a subtask path, while
  `prompts/shemiq-implement-task-parent.md` takes a parent path and subtask
  title. Keep both prompts unchanged in this task.
- Follow the existing command/launcher test patterns and update usage in
  `README.md` and `skills/shemiq/SKILL.md`. The baseline `go test ./...` passed
  during refinement.

## Interview

- Support both top-level selection and direct subtask-file paths, rather than
  mirroring only `refine`'s top-level entry point.
- Direct new subtask files follow their `parent:` link and identify the
  corresponding parent-list entry before starting the parent-based prompt.
- Only `refined` top-level tasks are eligible, including parents reached
  through direct subtask paths. Their subtasks may be `new` or `refined`.
- Trust the parent-list status for every entry point, even when the child file
  disagrees. Prefer this simple rule over rejecting mismatches or running
  validation as part of selection.
- Keep the implementation prompts unchanged. Do not add completion-tracking
  instructions or infer status transitions from Pi's exit.
- Use a focused implementation resolver that reuses the existing store,
  console, and launcher. Do not first refactor `refine` into a shared generic
  workflow framework.
- The command/routing design and architecture/error/testing design were
  approved. Keep the implementation as one subtask rather than splitting the
  launcher and CLI work.
- Subtask refinement confirmed that missing sources need no special preflight
  validation or dedicated test scenario. Use normal store loading only for
  documents needed by the selected route.

## Design

### Command behavior and routing

- Register `shemiq implement [path] [--subtask "Title"]`. A path may name a
  top-level document, its directory, or a subtask file. `--subtask` applies
  only to top-level selection; reject it with a subtask-file path.
- Without a path, discover the nearest existing `.shemiq` project and offer
  active refined top-level tasks with eligible subtasks. Exclude archived
  tasks from discovery, but preserve `refine`'s support for explicit archived
  or other-project paths. Do not create a project during selection.
- Offer parent-list entries with usable task type, nonempty title, and status
  `new` or `refined`. Display statuses in numbered subtask labels, for example
  `Add command [new]`. Exclude done or unusable entries. Keep explicit
  selection even when only one eligible subtask exists.
- Match `--subtask` by exact parent-list title and reject missing or ambiguous
  matches. Preserve ambiguity checks for titles selected interactively.
- For a direct subtask file, follow its resolved `parent:` reference and find
  the unique parent-list entry whose resolved `source:` points to that file.
  Use the parent entry's title and status, not the child's title or status,
  for routing. The parent must be a refined top-level task.
- Route `new` entries to
  `/shemiq-implement-task-parent <parent-path> <title>`. Their separate source
  file need not exist; top-level selection does not require loading it.
- Route `refined` entries to `/shemiq-implement-task <subtask-path>`. Load the
  selected source as a task document through the existing store, preserving
  normal loading errors without special missing-source validation.
- Parent-list status is authoritative even when an existing child disagrees.
  Do not perform status reconciliation or full metadata validation. The CLI
  never implements a top-level document directly or changes task status
  based on session exit.

### Components and data flow

- Add a focused resolver in `cmd/implement.go`, registered in `cmd/root.go`.
  Keep eligibility, selection, and routing in this workflow, leaving
  `refine`'s selection rules unchanged. Reuse small existing components rather
  than introducing configurable workflow infrastructure.
- Expose read access to the task model's already-parsed, resolved parent
  path. Preserve scoped, non-traversing store queries. Load selected
  parent/source documents through narrowly scoped stores instead of widening
  discovery or implicitly following all references.
- Resolve arguments or numbered selections into one parent-list entry, then
  into an implementation flow, document path, and project root for the
  existing `agent.Request` and `agent.Launcher` boundary.
- Preserve symlink-aware path resolution and require workflow documents to
  belong to a `.shemiq` project. Launch from the resolved project root of the
  document handed to the prompt: the parent for the new flow, the subtask
  for the refined flow.
- Add two implementation flows to `internal/agent` and map them to the
  existing installed templates. Retain project-relative prompt paths,
  Pi-compatible argument quoting, inherited environment, original terminal
  streams, and fresh interactive sessions. Make shared launcher diagnostics
  appropriate to both commands without changing launch mechanics.
- Pi and the installed, enabled Shemiq templates remain prerequisites. No
  provider-selection flag, new picker dependency, or bundled prompt copies
  are needed. Keep both implementation prompt files unchanged.

### Errors and verification

- Fail before launch for ineligible parents/subtasks, missing or ambiguous
  titles, broken direct-file parent/source identification, wrong task types,
  or cancelled selection. Do not preflight all referenced source files.
- Preserve existing malformed-document handling and project-containment
  checks, while keeping unrelated semantic metadata issues outside this
  workflow. Selection and launch must not repair or edit task documents.
- Surface missing Pi, redirected-terminal, start, and process failures using
  the existing launcher behavior and workflow-appropriate diagnostics.
- Keep tests small and workflow-focused: temporary-project command scenarios
  for numbered selection, both statuses, exact-title selection, and direct
  subtask paths, plus representative rejection cases and launcher-message
  coverage. Include parent-status authority and document non-mutation where
  they contribute to those scenarios.
- Reuse existing launcher/process coverage instead of duplicating its matrix;
  do not require live Pi. Update README and the Shemiq skill with command
  usage and prerequisites, then run `make check`.

## Current status

Implementation is complete. The `shemiq implement` command supports
interactive discovery, explicit top-level paths, `--subtask` exact-title
selection, and direct subtask-file paths. Subtask labels show statuses;
routing uses the parent-list status to choose between the `new` and `refined`
implementation flows. All tests pass (`make check`).

## Tasks

### Add the shemiq implement command
:::shemiq
type: task
source: ./add-shemiq-implement-command.md
status: refined
:::

Implement the CLI, task selection and direct-file routing, launcher flows,
focused workflow tests, and user documentation according to the agreed design.
