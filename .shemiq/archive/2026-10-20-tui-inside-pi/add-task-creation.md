# Add task creation
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 37418e1b-01eb-4a6b-aa42-ae387a7553e2
:::

## Context

- The browser (`browser.ts`, `controller.ts`) has a global `n` action with a placeholder handler, available regardless of selection including on empty and error/recovery screens.
- The parent design routes creation through `/shemiq-new` in the current session; this refinement makes the title optional and adds direct-CLI execution when the title is provided, superseding the always-`/shemiq-new` routing.
- `shemiq new --title <title> <description>` and `shemiq new-subtask <parent> --title <title> [--after <sibling>] <description>` are both fully non-interactive when `--title` is given; the CLI prints the created/edited path to stdout and rejects before writing on invalid input.
- Tree rows (`BrowserRow`) carry everything needed: the hovered subtask's title for `--after` and the parent top-level's path from `row.ancestors` / `node.path`, including external references.
- pi-tui provides single-line `Input` and multiline `Editor` components; the launcher shows the overlay-dialog pattern via `ctx.ui.custom(..., { overlay: true })`.
- The extension already executes the CLI directly via `execFile` from Pi's working directory (`loadTree`); archive will use the same pattern.
- The `shemiq` skill already documents `new-subtask` from the earlier CLI task.

## Interview

- Use a single context-dependent `n` key: sibling semantics — top-level row (or no selection/empty/error state) targets a new top-level task; subtask row targets a new subtask in the same parent with the hovered task as `--after`. No separate second binding.
- Execute subtask creation as a direct CLI call, not through an LLM prompt, when the title is provided.
- Same for top-level creation: when the title is provided, call `shemiq new --title ...` directly, superseding the parent design's always-`/shemiq-new` routing.
- Allow a blank title for both targets; when blank, close the browser and dispatch an LLM prompt in the current session that derives the title: existing `/shemiq-new` for top-level, a new separate `shemiq-new-subtask` template for subtasks (over extending `/shemiq-new` with conditional context).
- Use a single overlay dialog with an optional one-line title `Input` above a required multiline description `Editor`, with `Tab`/`Shift+Tab` focus switching, `Enter` inserting a newline, a distinct submit binding (`Ctrl+S` or `Alt+Enter`, implementation's choice), and `Esc` cancelling back to the browser with selection preserved.

## Implementation plan

### Targeting and dialog

- Make `n` context-dependent: on a top-level row, no selection, empty hierarchy, or the error/recovery screen → new top-level task; on a subtask row → new subtask in the same parent inserted after the hovered subtask (`--after` with the hovered row's exact title).
- Make the legend label context-dependent: "New task" vs "New subtask".
- Open a creation overlay dialog (launcher-style `ctx.ui.custom` with `{ overlay: true }`):
  - One-line title `Input`, optional; blank defers title selection to the LLM fallback.
  - Multiline description `Editor`, required (nonblank after trimming).
  - `Tab`/`Shift+Tab` (or arrows at field boundaries) switch focus; `Enter` inserts a newline in the description; `Ctrl+S` or `Alt+Enter` submits; `Esc` cancels.
  - Header shows the target context (e.g. new subtask under `<parent title>` after `<sibling>`).
- Cancelling returns to the browser with selection preserved; submitting with a blank description keeps the dialog open with an inline error.

### Execution paths

- Title filled → direct CLI via `execFile` from Pi's working directory (same style as `loadTree`), browser stays open:
  - Top-level: `shemiq new --title <title> <description>`.
  - Subtask: `shemiq new-subtask <parent-path> --title <title> --after <hovered title> <description>` with the parent path from the tree snapshot.
  - On success: refresh the hierarchy and show a success notification with the path printed by the CLI.
  - On failure: error notification; no refresh needed since the CLI rejects before writing.
- Title blank → close the browser and dispatch in the **current session** (no launcher dialog, no `shemiq-launch` machinery):
  - Top-level: existing `/shemiq-new <description>`.
  - Subtask: new `/shemiq-new-subtask <parent-path> '<after title>' <description>`.
- Follow the existing prepare → close/dispose → dispatch boundary: run template-loaded checks and build the quoted message (reusing `quotePiArgument`) before the browser closes.

### New prompt template

- Add `prompts/shemiq-new-subtask.md` mirroring `/shemiq-new`: instruct the agent to pick a concise one-line title, load the `shemiq` skill, run `shemiq new-subtask --title <title> --after <after> <parent> <description>` with safely quoted arguments and the entire original description as one argument, and report the printed path (or the error) without creating files by hand.
- Mention the template in the README usage section.

### Error handling

- Surface missing `shemiq` binary, CLI failures, and template-not-loaded errors as actionable notifications with the browser kept open; the template check happens before closing.
- Report CLI rejection text (duplicate sibling title, ambiguous structure, parent outside a `.shemiq` project, etc.) verbatim; the document is left unchanged.
- External subtasks: use the parent's absolute path from the tree row; the CLI's project restrictions apply and rejections surface as notifications.

### Testing

- Small focused unit tests reusing `makeTreeNode` and `flattenTree`:
  - Target routing: top-level row / subtask row / no selection → correct command and arguments (including `--after` and quoting) and correct template choice, for both title-filled and title-blank paths.
  - The new template's argument contract (message starts with `/shemiq-new-subtask `).
- Cover direct-CLI execution and dialog behavior (title provided/blank, cancel, blank description, insertion position) in the manual smoke-test checklist from the verify task.

## Implementation notes

- `extensions/shemiq/creation.ts` owns the `new` handler: target routing, CLI argument/prompt building, direct execution, and current-session launch preparation; registered from `index.ts` next to the workflow handlers.
- `creationTargetFor()` implements sibling semantics: subtask row → subtask under the immediate parent with the hovered title as `--after`; top-level row, no selection, empty, or error state → new top-level task. A subtask row whose parent lacks a document path fails with an actionable notification.
- `extensions/shemiq/creation-dialog.ts` renders the overlay dialog: optional one-line title `Input`, required multiline `Editor` (`disableSubmit`), `Tab`/`Shift+Tab` focus switching (Enter in the title moves to the description, Enter in the editor inserts a newline via `insertTextAtCursor`), `Ctrl+S`/`Alt+Enter` submit, `Esc` cancel, inline error on blank description, header shows the subtask context (`under "…" after "…"`).
- Title filled → direct `execFile` of `shemiq new` / `shemiq new-subtask` from Pi's working directory; on success the handler returns `{refresh, notify}` and the browser refreshes with a success notification carrying the printed path; failures (including `ENOENT` for a missing binary) surface as browser notifications without refreshing.
- Title blank → template check, model-availability pre-check, and quoted message building all happen before the browser closes; the handler returns `{close, launch}` with `session: "current"` and the current model, dispatched through the existing `dispatchLaunch()` boundary (no launcher dialog, no `shemiq-launch` machinery).
- `browser.ts` `ActionHandlerResult` gained a `notify` field applied before refresh; `workflows.ts` now exports `quotePiArgument`, `relativePath`, `requireTemplate`, and `requireAvailableModel` for reuse.
- The legend label is context-dependent: `getLegendEntries()` reports "New subtask" on subtask rows, "New task" otherwise (`controller.ts`).
- Added `prompts/shemiq-new-subtask.md` (arguments: parent path, after title, description as `$1`/`$2`/`$3`) and documented it in the README usage section.
- Tests: `creation.test.ts` covers target routing, CLI args (including `--after`), prompt routing/quoting/template choice, and the three handler paths (direct CLI, blank-title launch, cancel) plus CLI rejection, with the dialog and `child_process` mocked; `controller.test.ts` gained the subtask legend-label check. 33 extension tests pass; strict `tsc` type-check clean.
