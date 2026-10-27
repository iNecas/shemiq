# Edit action
:::shemiq
type: top-level
uuid: 9ca9f8fd-ceee-4457-8f44-3146bf6f1034
status: done
:::

## Description

- Support `e` in the Pi task browser to run `$EDITOR` with the selected task file.
- Fail when `$EDITOR` is not set or the task has no document path.
- Offer editing whenever a path is provided, even if the file does not exist yet.

## Context

- This feature belongs to the Pi extension; no new CLI command, Go changes, or generated tree-contract changes are expected.
- Task rows already expose an optional document path; some subtasks exist only in their parent document and have no path.
- `extensions/shemiq/controller.ts` owns shared action bindings, eligibility, legends, and selection resolution.
- `extensions/shemiq/browser.ts` dispatches actions and handles refreshes, notifications, and tree-loading errors.
- `extensions/shemiq/index.ts` registers action handlers; follow the existing archive/creation handler pattern.
- `extensions/shemiq/ext-ui.ts` owns the mounted browser and has access to Pi's TUI.
- The existing real Pi/CLI end-to-end harness is in `extensions/shemiq/test/pi-runtime.ts`.

## Interview

- Return to the same browser after editing, rather than closing it or launching an independent editor.
- Temporarily stop Pi's terminal rendering and keyboard handling so the external editor owns the terminal; do not suspend Pi's process.
- Show `e Edit` only when the selected task has a path, independent of task and parent status, editor availability, or file existence.
- For pathless tasks, hide the legend entry and report editing as unavailable when `e` is pressed; do not open the parent document instead.
- Require `$EDITOR`, with no fallback to `$VISUAL` or a default editor.
- Support an executable plus arguments without shell evaluation, including quoted arguments and paths containing spaces.
- Restore the terminal and report failures without closing the browser.
- Refresh after an editor process finishes, even unsuccessfully, because it may have saved changes.
- Do not automatically validate, repair, or rewrite task metadata.
- Keep implementation in one subtask; the feature is small enough not to split further.

## Design

### Architecture and components

- Add `e Edit` to the shared action bindings and make a provided path the sole edit-eligibility condition.
- Add a dedicated edit-action module and register its handler in the extension entry point.
- Extend `ExtUi` with a small terminal-handoff operation owned by the UI host, rather than exposing raw TUI access to the edit handler.
- Stop terminal handling for the supplied operation and guarantee restart and forced redraw afterward.
- Retain the browser instance throughout editing; do not open a launcher, create a session, or send an agent prompt.

### Data flow

- Read `$EDITOR` at invocation time and check the selected task path before handing over terminal control.
- Interpret `$EDITOR` as an executable plus arguments without shell evaluation and append the task filename as one argument.
- Launch from Pi's current working directory with inherited terminal input/output and wait for the editor process to finish.
- GUI editors must use their own waiting option, such as `code --wait`.
- Do not require the task file to exist or create/populate it beforehand; pass the configured path to the editor.
- Restore terminal handling and redraw, then reload the task tree after the editor finishes, including unsuccessful exits.
- Preserve selection using the existing UUID/path/index fallback.

### Error handling

- Missing or blank `$EDITOR`, or a missing task path, must fail without handing over terminal control.
- Report editor launch errors, nonzero exits, and signal termination through browser notifications.
- Always restore terminal handling after handoff, including when launch or execution fails.
- Use the existing tree-loading error display if the edited document cannot be loaded.
- Do not automatically validate, repair, or rewrite metadata.

### Testing

- Keep tests minimal and reuse the existing real Pi/CLI end-to-end harness.
- Cover one end-to-end scenario with a small fake editor process: press `e`, create a previously missing task file, return to the browser, and observe refreshed content with the task still selected.
- Add only focused checks needed for path-based eligibility, command arguments, and terminal restoration/error reporting.
- Avoid a broad editor/platform test matrix.
- Manually smoke-test a real terminal editor to verify keyboard ownership and redraw.

## Current status

- Implemented: path-based `e Edit` eligibility and legend, host-owned
  terminal handoff (`ExtUi.runInTerminal`), and the `$EDITOR` handler with
  refresh-after-exit and notification-based error reporting; see
  `edit-action.md` implementation notes.
- All extension tests pass (focused + e2e with a fake editor creating a
  previously missing task document).
- Manual smoke test with a real terminal editor still pending.

## Tasks

### Implement the task edit action
:::shemiq
type: task
source: ./edit-action.md
status: done
:::

Add path-based `e Edit` eligibility, host-owned terminal handoff, and the `$EDITOR` handler with the agreed refresh/error behavior and minimal tests.
