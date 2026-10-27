# Implement the task edit action
:::shemiq
type: task
parent: ./top-level.md
uuid: f61ca2cc-c602-4dee-9db2-a4e9997774a7
status: done
:::

## Context

The browser already has a shared action-binding/eligibility layer
(`controller.ts`), an action-handler dispatch loop with refresh/notify
results (`browser.ts`), a host-owned UI abstraction (`ext-ui.ts`), and a
per-action module registration pattern (`archive.ts`, `creation.ts`,
`workflows.ts`) wired in `index.ts`. The tree rows already expose an
optional `path`; subtasks with a `source:` target that does not exist yet
still get a path.

## Implementation plan

- `controller.ts`: add `edit` action kind with key `e` / label `Edit`;
  eligibility is solely "row has a path", independent of type/status, so
  pathless rows hide the legend entry and the browser reports the action
  as unavailable.
- `ext-ui.ts`: extend `ExtUi` with `runInTerminal(op)`: stop the TUI with
  `preserveScreen`, run the operation, always restart and force a redraw.
- `edit.ts` (new): `$EDITOR` handler following the archive pattern:
  - read/validate `$EDITOR` (no `VISUAL` fallback) and the task path
    before any terminal handoff;
  - parse `$EDITOR` into executable + arguments with shell-like quoting
    (no shell evaluation), append the task path as one argument;
  - spawn with inherited stdio from Pi's cwd and wait for exit;
  - refresh after any finished editor run, notify on nonzero exit/signal,
    throw on launch failure (terminal restored by the host).
- `index.ts`: register the handler.
- Tests: focused checks for command parsing, eligibility, terminal
  restoration, and error paths; one e2e case with a fake editor creating
  a previously missing task document and returning to the refreshed,
  still-selected browser.

## Implementation notes

- `controller.ts`: `edit` kind with key `e` / label `Edit`; eligibility is
  purely `node.path` presence, added before the type/status branching so it
  is independent of them (done/pathless rows behave as designed).
- `ext-ui.ts`: `ExtUi.runInTerminal(op)` stops the TUI with
  `{ preserveScreen: true }` (leaves the renderer output for the editor),
  then always restarts and forces a redraw in a `finally`; a closed host
  runs the operation without handoff.
- `edit.ts`: handler validates path and `$EDITOR` (trimmed, no `VISUAL`
  fallback) before any handoff; `parseEditorCommand` splits executable +
  arguments with single/double-quote grouping (no shell evaluation);
  spawns with inherited stdio from `ctx.cwd` and waits.
- Exit handling: code 0 → refresh only; nonzero/signal → refresh + notify
  (the editor may have saved changes); launch failures (e.g. ENOENT)
  reject and surface through the browser's existing "Action failed"
  notification, with the terminal restored by the host.
- Registered in `index.ts` next to the archive handler; no CLI, Go, or
  tree-contract changes.
- Tests: focused `edit.test.ts` (unsuccessful-exit/launch-error paths,
  no-handoff validation, quote grouping — the happy path is covered by
  the e2e), `controller.test.ts` (path-based eligibility/legend),
  `ext-ui.test.ts` (stop/restart/forced-redraw, restore on failure); e2e
  in `workflows.e2e.test.ts` uses a fake `sh` editor (quoted path with a
  space) creating a previously missing `source:` target and returns to
  the refreshed browser with the row still selected.
- Manual smoke test with a real terminal editor (keyboard ownership and
  redraw) still pending — not automatable in the harness.
