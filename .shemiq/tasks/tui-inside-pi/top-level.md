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
- Provide alternate bindings for the same refinement or implementation workflow in the current session.
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
- `registerActionHandler()` in `browser.ts` is the integration point for the remaining four subtasks.
- The workflow connection subtask is implemented and done (`extensions/shemiq/workflows.ts`).
- Refine (r/R) and implement (i/I) browser actions now route to the four existing prompt templates with proper argument quoting.
- Fresh sessions (r/i) use `ctx.newSession()` + `withSession`; current sessions (R/I) use `pi.sendUserMessage()`.
- Missing template detection keeps the browser open with an actionable error.
- Creation, archiving, and verification/documentation remain new.

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

### Add task creation
:::shemiq
type: task
source: ./add-task-creation.md
status: new
:::

Collect a multiline description and start the existing creation prompt in the current session, returning to the browser if description entry is cancelled.

- Integration note: connect the browser's New task hook, replacing its placeholder, and reuse its return/close behavior; the workflow task's current-session dispatch boundary should be reusable.

### Add immediate archiving
:::shemiq
type: task
source: ./add-immediate-archiving.md
status: new
:::

Archive the selected top-level task without confirmation, keep the browser open, refresh the hierarchy, and report success or failure.

- Integration note: connect the browser's Archive hook and refresh/recovery foundation; existing CLI project restrictions still apply.

### Verify and document the in-Pi workflows
:::shemiq
type: task
source: ./verify-and-document-workflows.md
status: new
:::

Add lightweight automated workflow checks, a manual Pi smoke-test checklist, and usage and development documentation. Include common tasks in Makefile.

- Integration note: extend the browser task's small controller/recovery checks and manual checklist rather than duplicating them; include workflow launch paths, missing-template handling, and fresh-session cancellation without fallback or automatic reopening.

### Tests coverage for browser.ts

### Refactor hotkeys definitions

The hotkeys are quite hardcoded, would be better to extract them behind some layer,
so that it's more readable and can be configured/customized.
