# Add the shemiq implement command
:::shemiq
type: task
parent: ./top-level.md
status: refined
uuid: 59fec9a0-4eae-45c6-a0aa-6f5a5b7cb35d
:::

## Context

Implement the complete `shemiq implement` workflow described in the parent:
CLI selection, direct-file routing, launcher flows, focused tests, and user
documentation. This remains one implementation task; implementation has not
started.

Relevant existing components:

- `cmd/refine.go` demonstrates Cobra command handling, numbered selection,
  exact-title matching, and the handoff to an injected launcher.
- `internal/task/store.go` loads scoped task snapshots without following
  metadata references. A file-scoped store cannot query another document.
- `internal/task/task.go` exposes parent-list titles, types, statuses, and
  resolved source paths through `Task.Path`. Its private `parentRef` already
  holds the resolved parent path.
- `internal/console/console.go` supplies numbered selection and cancellation.
- `internal/agent/agent.go` defines `Request`, `Flow`, `Launcher`, and original
  terminal streams. `internal/agent/pi.go` handles interactive Pi sessions and
  Pi-compatible argument quoting.
- Both implementation templates already exist under `prompts/`. Keep them
  unchanged. Pi and the installed, enabled templates remain prerequisites.

## Interview

- Preserve the approved parent design rather than introducing a shared
  refinement/implementation workflow framework.
- Do not add special validation or a dedicated test scenario for missing
  source files. Load selected documents with the existing store and retain
  its normal errors; do not preflight all references during selection.
- The command/selection design and the components/testing design were both
  approved. Parent-list status remains authoritative for every entry point.

## Implementation plan

### Command and selection

- Register `shemiq implement [path] [--subtask "Title"]` in `cmd/root.go`, with
  command handling and a focused implementation resolver in
  `cmd/implement.go`.
- Accept a top-level document, its directory, or a direct subtask-file path.
  Resolve arguments relative to the invocation directory. Reject `--subtask`
  when an explicit path identifies a subtask file.
- Without a path, discover the nearest existing `.shemiq` project and offer
  active, refined top-level tasks with eligible subtasks. Exclude archives
  from discovery, but allow explicit archived or other-project paths. Never
  create a project as part of selection.
- Only refined top-level parents qualify, including those reached through
  direct subtask paths. Parent-list entries qualify when they have a usable
  task type, a nonempty title, and status `new` or `refined`. Omitted status
  means `new`; exclude done or unusable entries.
- Show statuses in numbered subtask labels, for example `Add command [new]`.
  Keep explicit numbered selection even when only one subtask qualifies.
- Match `--subtask` against exact parent-list titles and reject missing or
  ambiguous matches. Preserve ambiguity checks for interactive title
  selection as well.
- For a direct subtask file, follow its resolved parent reference and find the
  unique parent-list entry whose resolved source path identifies that file.
  Use the entry's title and status, not the child's title or status.

### Routing and task API

- Add a read-only `Task.ParentPath() string` accessor over the existing
  resolved `parentRef`; do not introduce another independently stored value.
- Keep store queries scoped and non-traversing. Use file-scoped stores to
  load the selected parent or child rather than broadening discovery or
  automatically following all metadata references.
- Route `new` entries to
  `/shemiq-implement-task-parent <parent-path> <title>`. Top-level selection
  does not need to load the new entry's source file, which may not exist yet.
- Route `refined` entries to `/shemiq-implement-task <subtask-path>`. Load the
  selected source as a task document using the existing store. Do not add
  special missing-source checks, picker filtering, or metadata validation.
- Parent-list status controls routing even when the child disagrees. Do not
  reconcile statuses or require full reciprocal metadata validation.
- Preserve symlink-aware resolution and project ownership. Workflow documents
  must belong to a `.shemiq` project. The selected prompt document determines
  the launch project: the parent for the new flow, the child for the refined
  flow. Never implement a top-level document directly.

### Launcher integration

- Add `ImplementSubtask` and `ImplementSubtaskParent` to `agent.Flow` and map
  them to the existing `shemiq-implement-task` and
  `shemiq-implement-task-parent` templates respectively.
- Reuse `agent.Request`: flow, absolute document path, resolved project root,
  and optional subtask title. Keep the existing `agent.Launcher` boundary and
  command-local launcher injection.
- Preserve project-relative prompt arguments, Pi-compatible quoting,
  inherited environment, original terminal streams, and fresh interactive
  sessions. Do not add shell invocation or change launch mechanics.
- Generalize refinement-specific launcher comments and diagnostics so they
  also make sense for implementation. Preserve missing-Pi, redirected-stream,
  start, and process-failure behavior.
- Do not change either implementation prompt, add provider selection, bundle
  prompt copies, or infer task completion/status changes from Pi's exit.

### Verification and documentation

- Follow `cmd/refine_test.go` with small temporary-project command scenarios
  and an injected recording launcher. Cover both routes, numbered selection
  with status labels, exact-title selection, and direct-file selection.
- Include parent-status authority and document non-mutation in those
  scenarios where useful. Use a small set of routing rejections for
  ineligible parents/subtasks, ambiguous titles, incompatible flags, broken
  direct-file identification, and cancellation; avoid exhaustive matrices.
- Extend existing launcher-message coverage for both implementation flows
  and any changed diagnostics. Reuse existing quoting/process coverage
  rather than duplicating it. Do not require live Pi or add a dedicated
  missing-source test scenario.
- Preserve existing malformed-document errors and resolved project-containment
  checks. Unrelated semantic metadata findings remain the responsibility of
  `shemiq validate`; selection and launch never repair or edit documents.
- Update `README.md` and `skills/shemiq/SKILL.md` with command usage, routing,
  direct-file support, and the existing Pi/template prerequisites.
- Run `make check`. Keep functions ordered from high-level entry points to
  lower-level helpers, use `utils.Dedent` for multiline Go strings, and follow
  the project's line-length guidance.

## Implementation notes

Implemented the complete `shemiq implement` command:

- **`cmd/implement.go`**: Command registration, three entry points (discovery,
  top-level path, direct subtask file), subtask selection with status labels
  (`[new]`/`[refined]`), exact `--subtask` matching, and routing to the two
  implementation flows.
- **`internal/agent/agent.go`**: Added `ImplementSubtask` and
  `ImplementSubtaskParent` flows.
- **`internal/agent/pi.go`**: Extracted `flowTemplate()` to map all four flows
  to prompt template names. Generalized terminal and session-failure messages
  to be command-neutral.
- **`internal/task/task.go`**: Added `ParentPath()` read-only accessor over
  the existing resolved `parentRef`.
- **`cmd/implement_test.go`**: Tests cover interactive selection for both
  `new` and `refined` subtasks, exact-title selection, direct subtask file
  (both statuses), status labels in picker, and rejections (not-refined
  parent, done subtask, ambiguous titles, cancel, `--subtask` with direct
  file).
- **`internal/agent/pi_test.go`**: Added `TestPiImplementFlowMessages` for
  both implementation flow message formats.
- **`README.md`** and **`skills/shemiq/SKILL.md`**: Updated with command
  usage, routing description, and prerequisites.

Key design decision: when a direct subtask file is given, the store is scoped
to that single file, so loading the parent requires creating a separate
file-scoped store for the parent path. This keeps the existing non-traversing
store semantics intact.
