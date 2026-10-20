# Add immediate archiving
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 05aef80d-161e-4cf0-974d-1fcf8e34825c
:::

## Context

- The browser already wires the `archive` action: `a` key binding, eligibility (top-level always archivable), and legend entry exist in `controller.ts`; dispatching `a` currently reports "not implemented yet" because no handler is registered.
- The browser's `dispatchAction` applies handler results in order: `notify` first, then `refresh`, keeping the browser open; thrown errors surface as `Action failed: <message>` without touching the displayed hierarchy.
- `shemiq archive <path>` accepts absolute or cwd-relative paths, applies project/ownership restrictions itself, and prints the archive directory path on success.
- The direct-CLI `execFile` pattern from `creation.ts` (including the `ENOENT` missing-binary check) is the established way to run non-interactive CLI commands.
- The real-runtime/virtual-terminal e2e harness from "More end to end tests" (`test/pi-runtime.ts`) is available for verification.

## Interview

- Missing `node.path` on the selected top-level row: guard in the handler with an actionable error (matching `creation.ts`), rather than passing a garbage path to the CLI.
- Success notification and CLI argument: `Archived <relative path of the original task dir>`, passing the task path to the CLI as a cwd-relative path (consistent with `workflows.ts`).
- Testing: focused unit tests plus one e2e test. Unit tests mock `execFile` following `creation.test.ts`; one e2e test uses the existing harness to cover the archive-and-refresh flow in the real component.

## Implementation plan

- New module `extensions/shemiq/archive.ts` exposing `registerArchiveHandler(pi, register)`, registered from `index.ts` into the runtime-local action-handler map (same pattern as `registerCreationHandler`).
- Handler for the selected top-level row:
  - Missing-path guard: throw `Cannot archive: task has no document path` when `row.node.path` is absent.
  - CLI invocation: `execFile("shemiq", ["archive", relativePath(cwd, node.path)], { cwd })` with the `creation.ts` helper style: `ENOENT` → "shemiq CLI not found. Install it and make sure it is on PATH."; other failures → `shemiq archive failed: <stderr detail>`.
  - Success: return `{ refresh: true, notify: "Archived <relative path of the task directory>" }` — the browser shows the notification, reloads the tree, and stays open.
  - Failure: throw; the browser's `Action failed: ...` notification surfaces the error without removing the task from the displayed hierarchy. A retry is preceded by manual refresh (Ctrl+R).
- No confirmation step and no LLM interaction. Eligibility (top-level archivable regardless of status, including external tasks) stays with `getEligibleActions`; CLI project/ownership restrictions remain the source of truth for rejection.
- Selection after refresh: existing `resolveSelection` clamps to a nearby row when the archived task is gone — no new logic.
- Tests:
  - `extensions/shemiq/archive.test.ts` with `vi.mock("child_process")`: success (correct CLI args and `{ refresh, notify }` result), CLI failure, `ENOENT`, and the missing-path guard (`execFile` never called).
  - One e2e test using `test/pi-runtime.ts` with the repo-built CLI and a fixture project: open the browser, select a top-level task, press `a`, assert the task disappears from the rendered hierarchy and the notification banner shows `Archived ...`.
  - No Go-side changes; CLI archive behavior is already covered there.

## Implementation notes

- Implemented in `extensions/shemiq/archive.ts` as `registerArchiveHandler(pi, register)`; registered from `index.ts` into the runtime-local action-handler map next to the creation handler (the `pi` argument is unused but kept for signature symmetry).
- The handler passes the task directory (derived via `path.dirname(node.path)`) as a cwd-relative path to `shemiq archive`, so the CLI itself remains the source of truth for project/ownership restrictions.
- Success returns `{ refresh: true, notify: "Archived <relative task dir>" }` — the browser applies `notify` before `refresh` and stays open; selection clamping after refresh reuses the existing `resolveSelection`.
- Missing-path guard, `ENOENT` missing-binary handling, and stderr-detail error wrapping follow the `creation.ts` `execFile` pattern exactly.
- No confirmation step, no LLM interaction, no new eligibility logic.
- Tests: `archive.test.ts` (4 focused tests: success/args, CLI failure, `ENOENT`, missing-path guard with no `execFile` call) and one new e2e case in `workflows.e2e.test.ts` (archive via `a` on the selected top-level row, notification shown, task gone from the rendered tree, remaining task still present, and the real CLI tree confirms the move).
- Full extension suite passes (43 tests).
