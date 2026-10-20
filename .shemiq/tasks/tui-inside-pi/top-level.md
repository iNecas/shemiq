# TUI inside Pi
:::shemiq
type: top-level
uuid: db5a0574-5fd2-4324-a8e7-14bba89dbe0a
status: refined
:::

## Description

- Provide a TUI inside the Pi coding agent for creating top-level tasks, refining top-level tasks and subtasks, implementing new or refined subtasks, and archiving top-level tasks.
- Expose most existing CLI workflow functionality through interactive task selection and keyboard actions.
- Ask for a description when creating a task and invoke `/shemiq-new`; select existing tasks through a task-first browser rather than an action-first menu.
- Keep the extension focused on UI and routing to existing prompts through standard Pi mechanisms; keep parsing and filesystem operations in the CLI.
- Preserve the existing standalone CLI commands and prompts without requiring the UI.
- Support keyboard invocation, navigation, and actions throughout.

## Context

- The completed CLI tree work is recorded in `.shemiq/archive/2026-10-12-cli-tree/top-level.md`; `shemiq tree --json` provides a validated, merged hierarchy for integrations.
- `extensions/shemiq/types.ts` contains the generated TypeScript tree contract; there is no extension implementation yet.
- `cmd/refine.go` and `cmd/implement.go` define the existing workflow eligibility and prompt-routing behavior to follow.
- Existing prompts cover task creation, top-level and subtask refinement, and implementation from either a refined subtask document or a new subtask's parent description.
- `shemiq archive <path>` already performs archiving without requiring an LLM interaction.
- CLI discovery finds the nearest project from the working directory; the returned tree can include externally referenced tasks.
- This document defines shared usage, behavior, and responsibility boundaries; component choices, internal structure, exact key combinations, and test tooling belong to subtask refinement.

## Interview

- Choose a task-first browser over an action-first menu or hybrid interface.
- Show the entire hierarchy expanded rather than using collapsible branches or drill-down lists.
- Use direct action keys and a visible legend, without an action menu.
- Follow existing CLI workflow eligibility rather than adding re-refinement or status overrides.
- Start refinement and implementation in fresh Pi sessions by default; provide alternate bindings to continue in the current session.
- Always create tasks in the current session after collecting a multiline description.
- Archive immediately without confirmation, keep the browser open, and refresh afterward.
- Keep the small amount of eligibility and prompt routing in the extension rather than adding a CLI routing API.
- Reload on open and support manual refresh; workflow refinement drops automatic pre-action reloads and additional task-validity checks in favor of launching from the displayed snapshot. Do not add background watchers.
- Keep a recovery screen on loading errors, with Refresh, New task, and Close; do not automatically repair metadata or install tools.
- Require Pi to be idle before opening the browser; do not interrupt or queue a delayed opening.
- Browser subtask refinement supersedes the original read-only policy: show external references with the same actions as local tasks, subject to normal eligibility and existing CLI restrictions; keep Pi's working directory unchanged.
- Use small automated workflow checks plus a manual Pi smoke test, without terminal end-to-end automation infrastructure or duplicate CLI validation tests.
- Keep this refinement usage-focused rather than prescribing specific UI components or implementation structure.
- Approve five subtasks, keeping creation and immediate archiving separate.

## Design

### Browser usage and responsibility boundaries

- Open the browser through `/shemiq` or a global keyboard shortcut only while Pi is idle; otherwise, notify the user without interrupting work or scheduling a delayed opening.
- Display the complete hierarchy expanded, including done tasks, with indentation, titles, status labels, and keyboard selection across top-level tasks and subtasks.
- Use direct keyboard actions for creation, refinement, implementation, archiving, refresh, and close; show available actions for the selected row in a visible legend.
- Make New task available independently of the selection; settle exact key combinations during subtask refinement.
- Discover from Pi's working directory using the CLI's nearest-project rules; externally referenced tasks are visible and actionable under the same eligibility rules as local tasks. Do not switch Pi's working directory or bypass existing CLI restrictions.
- Keep the extension responsible for UI, eligibility checks, and routing; use the existing CLI for parsing, validation, discovery, and archive filesystem operations.
- Do not add an action menu, document previews, search/filter controls, collapsible branches, background synchronization, a CLI routing API, or project switching.
- Preserve existing standalone CLI commands and prompt invocation.

### Action eligibility

- New top-level task: allow refinement and archiving.
- Refined top-level task: allow archiving; select individual subtasks for refinement or implementation.
- New subtask under a refined parent: allow refinement or immediate implementation.
- Refined subtask under a refined parent: allow implementation.
- Done tasks and subtasks: disallow refinement and implementation; top-level tasks remain archivable regardless of status.
- New task remains available everywhere; external ownership does not suppress task-specific actions.
- Omit unavailable actions from the selected row's legend.

### Starting workflows

- Refinement and implementation close the browser and automatically start the corresponding existing prompt in a fresh Pi session by default.
- Launcher-settings refinement supersedes alternate current-session bindings: `r/i` open a model/session dialog, with `c` toggling current-session mode and fresh sessions selected by default. Creation does not use this dialog.
- Supply normal task prompt inputs to fresh sessions; do not generate a conversation handoff or copy previous conversation history.
- Route top-level refinement to its existing prompt and subtask refinement to the existing parent-and-subtask prompt.
- Implement refined subtasks from their own documents; implement new subtasks through the existing parent-based prompt.
- For New task, collect a description in a multiline editor and pass the complete text to `/shemiq-new` in the current session.
- Cancelling description entry returns to the browser without creating anything; submitting closes the browser and starts the prompt.
- Archive the selected top-level task immediately through the CLI, without confirmation or an LLM interaction; keep the browser open, reload the hierarchy, and report success.
- Execute prompts through Pi's standard mechanisms rather than replacing their existing behavior.

### Data flow and refresh

- Load the hierarchy through `shemiq tree --json` whenever the browser opens.
- Provide manual refresh; launch actions from the displayed selection using the existing shared eligibility policy, without automatic pre-action reloads or additional task-validity checks.
- Never silently substitute another row for the selected task. Changes made outside the browser become visible on manual refresh or reopening.
- Refresh after archiving so the displayed hierarchy reflects the operation.

### Error handling

- On loading failure, display available diagnostics and retain a recovery screen with Refresh, New task, and Close; stale rows must not remain actionable.
- Treat an absent `.shemiq` project as a recoverable starting state where New task still uses the existing creation prompt.
- Report missing CLI, invalid metadata, and unreadable files with actionable messages rather than presenting failures as a successfully loaded empty hierarchy.
- The extension does not automatically install tools, repair metadata, or modify task documents itself; existing prompts retain their established responsibilities.
- Check launch prerequisites before replacing the current session where possible.
- If fresh-session creation is cancelled or fails, leave the browser closed and report it; do not automatically reopen or fall back to dispatching the prompt into the old session.
- Report archive failures without claiming success or removing the task from the displayed hierarchy; refresh before another attempt, using the recovery screen if reloading fails.

### Testing and documentation

- Keep automated checks small and focused on representative eligibility and prompt routing, fresh/current-session dispatch, and archive refresh and failure handling.
- Do not duplicate Go parsing or CLI validation coverage or introduce full terminal end-to-end automation infrastructure.
- Add a manual Pi smoke-test checklist covering command and shortcut invocation, fully expanded keyboard navigation, the action legend, multiline creation, cancellation, session behavior, immediate archiving, and error recovery.
- Retain existing Go checks and document browser usage and the lightweight verification procedure.
- Leave test tooling and internal test structure to subtask refinement.

## Current status

- The browser subtask is implemented and done (`extensions/shemiq/index.ts`, `browser.ts`, `controller.ts`).
- `/shemiq` command and `Alt+S` shortcut open a full custom screen with hierarchy, keyboard navigation, action legend, and refresh.
- Eligibility rules and action handler hooks are in place; unconnected actions report "not implemented yet".
- `index.ts` owns a runtime-local action-handler map passed into `openBrowser()`; remaining actions register in that map, not module-global state.
- The workflow connection subtask is implemented and done (`extensions/shemiq/workflows.ts`).
- Refine (`r`) and implement (`i`) route to the four existing prompt templates with proper argument quoting.
- Browser/dialog disposal precedes dispatch; fresh sessions use `withSession` to call `/shemiq-launch` in the destination runtime, while current-session mode applies the model directly.
- Missing template detection keeps the browser open with an actionable error.
- Fixture refactoring is implemented and done: shared recursive `makeTreeNode()` and exported production `flattenTree()` now drive controller/workflow fixtures.
- All 19 extension tests pass, including `shemiq-launch` JSON transport/validation and destination model-before-prompt ordering; broader browser coverage and runner/Makefile setup remain follow-ups.
- Prompt launcher settings is implemented and done: `r/i` now open a model/session overlay dialog (`launcher.ts`) instead of directly dispatching; `R/I` current-session bindings removed.
- Launcher model choices respect Pi's scoped models and their order when configured, falling back to the full authenticated list when no scope is set.
- Task creation is implemented and done (`extensions/shemiq/creation.ts`, `creation-dialog.ts`).
- `n` is context-dependent (sibling semantics): subtask row → new subtask under the same parent inserted after the hovered task; top-level row, no selection, empty, or error state → new top-level task; the legend label follows ("New task"/"New subtask").
- A single overlay dialog collects an optional title and a required multiline description (Tab/Shift+Tab focus, Enter newline, Ctrl+S/Alt+Enter submit, Esc cancel); blank description keeps the dialog open with an inline error.
- Title filled → direct non-interactive CLI call (`shemiq new` / `shemiq new-subtask --after`), browser stays open, refreshes, and notifies with the printed path; CLI rejections and a missing binary surface as notifications.
- Title blank → browser closes and `/shemiq-new` or the new `/shemiq-new-subtask` prompt is dispatched in the current session through the existing `dispatchLaunch()` boundary (template and model checked before closing).
- More end to end tests is implemented and done: standalone npm/Vitest setup under `extensions/shemiq` (`npm test`, lockfile, Pi 1.0.0 dev deps).
- A reusable real-runtime harness (`test/pi-runtime.ts` + copied minimal `VirtualTerminal`) drives real `InteractiveMode`/session runtime, extension, dialogs, and the repo-built CLI with only scripted faux models, isolated env, and forbidden network.
- The e2e suite (`workflows.e2e.test.ts`, 5 tests) covers launcher cancel and fresh/current launches, `/reload` → Alt+S relaunch, direct and prompt-based creation, and loading-error recovery with Refresh.
- `make pi-test` runs the extension suite (focused + e2e), installing npm dependencies on first use; the rest of the Makefile/documentation work stays with the verification task.
- Archiving and verification/documentation remain new.

## Tasks

### Build the keyboard-driven task browser
:::shemiq
type: task
source: ./build-task-browser.md
status: done
:::

Provide idle-only entry, the fully expanded hierarchy, keyboard navigation and action legend, refresh, and loading-error recovery.

### Connect refinement and implementation workflows
:::shemiq
type: task
source: ./connect-refinement-and-implementation.md
status: done
:::

Apply existing workflow eligibility and prompt routing from the displayed snapshot, with fresh sessions by default and alternate current-session bindings.

- Integration note: reuse the browser task's shared eligibility, action hooks, and command-capable entry path; automatic pre-action reloads and a reload-and-resolve helper are no longer required.

### Refactor fixtures definitions
:::shemiq
type: task
source: ./refactor-fixtures-definitions.md
status: done
:::

Unify test fixtures with a shared recursive `makeTreeNode()` helper, express hierarchies as nested definitions, and reuse production `flattenTree()` for browser rows.


### Prompt launcher settings
:::shemiq
type: task
source: ./prompt-launcher-settings.md
status: done
:::

Before refinement or implementation, show a model/session dialog with available authenticated chat models, Pi selection movement bindings, `c` to toggle current-session mode, Enter to launch, and Esc to return to the browser. Default to the current model and a fresh session; replace the separate `R/I` actions.

### Add task creation
:::shemiq
type: task
source: ./add-task-creation.md
status: done
:::

Collect an optional title and a multiline description in a single dialog; when the title is filled, create the sibling task directly through the CLI (top-level or subtask with the hovered task as `--after`); when blank, close the browser and dispatch `/shemiq-new` or the new `/shemiq-new-subtask` in the current session. Cancelling returns to the browser.

- Integration note: connect the browser's New task hook, replacing its placeholder, and reuse its return/close behavior; the workflow task's current-session dispatch boundary should be reusable.
- Launcher integration note: creation stays current-session only without this dialog; follow the prepare → close/dispose → dispatch boundary used by `openBrowser()` and `dispatchLaunch()`.

Take into account new feature of adding a subtask as well .shemiq/tasks/new-subtask-command/top-level.md.
It should be creating sibling tasks - on top-level, it should target new top-level task, when on top of a sub task,
it should create a sub-task with the hovered task as "--after".

### More end to end tests
:::shemiq
type: task
source: ./more-end-to-end-tests.md
status: done
:::

Add a standalone local npm/Vitest setup and a small keyboard-driven workflow suite using real Pi interactive/session runtimes, a minimal copied virtual-terminal helper, faux models, and the repository-built CLI. Cover launch/reload in both session modes, direct and prompt-based creation, and loading-error recovery without real model services or pty automation.

### Add immediate archiving
:::shemiq
type: task
source: ./add-immediate-archiving.md
status: new
:::

Archive the selected top-level task without confirmation, keep the browser open, refresh the hierarchy, and report success or failure.

- Integration note: connect the browser's Archive hook and refresh/recovery foundation; existing CLI project restrictions still apply.
- Creation integration note: the handler-result `notify` field (applied before refresh) can carry the archive success message; the direct-CLI `execFile` pattern from `creation.ts` (including the `ENOENT` missing-binary check) is reusable for `shemiq archive`.
- E2E integration note: archive behavior remains here; reuse the real-runtime/virtual-terminal harness from More end to end tests once the handler exists.

### Verify and document the in-Pi workflows
:::shemiq
type: task
source: ./verify-and-document-workflows.md
status: new
:::

Add lightweight automated workflow checks, a manual Pi smoke-test checklist, and usage and development documentation. Include common tasks in Makefile.

- Integration note: extend the browser task's small controller/recovery checks and manual checklist rather than duplicating them; include workflow launch paths, missing-template handling, and fresh-session cancellation without fallback or automatic reopening.
- Fixture integration note: reuse the fixture task's shared `makeTreeNode()` and exported `flattenTree()` for tree-based scenarios; test-runner setup and Makefile integration remain here.
- Launcher integration note: cover model selection, the `c` current-session toggle, Enter/Esc, both session modes, selected-model behavior, cancellation, and launch failures rather than the superseded `R/I` bindings.
- Creation integration note: cover direct-CLI execution and dialog behavior (title provided/blank, cancel, blank description, insertion position) as planned in the creation task; the dialog is `creation-dialog.ts`, not the launcher.
- E2E integration note: More end to end tests now owns local package/test-runner setup and representative real-runtime workflows, superseding the earlier runner assignment. Reuse those checks; Makefile integration, documentation, the manual checklist, and remaining focused checks stay here.

### Tests coverage for browser.ts and launcher.ts

- Existing focused coverage in `workflows.test.ts` exercises `dispatchLaunch()` → serialized command → registered `shemiq-launch` handler, exact prompt preservation, model-before-prompt ordering, and invalid payload rejection; reuse rather than duplicate it.
- Context-handling regression: launch → `/reload` → reopen browser → Enter to launch again, in both session modes; include real Pi replacement-runtime smoke coverage.
- Invalidate source `pi` and command-context getters in mocks; verify model selection uses the destination runtime, browser/dialog disposal precedes dispatch, and runtime-local handlers are renewed on reload.
- Cover duplicate confirmation, model-before-prompt ordering, cancellation, and replacement/model failures without old-session fallback or reopening. Context-handling tests remain in this subtask, not the launcher fix.
- E2E integration note: More end to end tests now owns real-runtime launch/reload coverage in both session modes. Reuse that harness and coverage; detailed component and mocked context-handling edge cases remain here.

### Refactor hotkeys definitions

The hotkeys are quite hardcoded, would be better to extract them behind some layer,
so that it's more readable and can be configured/customized.

- Launcher integration note: the settings task removes `R/I` and adds a dialog-local `c` current-session toggle using Pi selection movement bindings; general hotkey customization remains here.
