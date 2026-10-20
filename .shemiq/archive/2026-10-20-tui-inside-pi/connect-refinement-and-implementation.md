# Connect refinement and implementation workflows
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: d0d9ffeb-81fc-4fb6-9f4c-6455ec0a47ce
:::

## Context

- Connect refinement and implementation to the existing task browser in `extensions/shemiq/index.ts`, `browser.ts`, and `controller.ts`.
- Reuse `registerActionHandler()`, the command-capable browser entry path, `BrowserAction`, `BrowserRow`, and `getEligibleActions()`.
- Follow the existing prompt routing in `cmd/refine.go`, `cmd/implement.go`, and `internal/agent/pi.go`; keep standalone CLI commands and prompts unchanged.
- Keep parsing, validation, and discovery in the CLI. Use the existing generated tree contract in `extensions/shemiq/types.ts` unchanged.
- The browser task's planned reload-and-resolve helper was not implemented. This refinement explicitly drops that requirement rather than adding it here.
- Do not implement task creation, archiving, hotkey customization, or consolidated documentation in this task.

## Interview

- Use Pi's loaded prompt templates only, preserving configured overrides; do not fall back to bundled files or manually expand prompt files.
- Keep the existing shared eligibility rules unchanged. Do not add another CLI-policy layer or tighten parent-status checks in this task.
- Launch from the displayed selection without automatic pre-action reloads or additional task-validity checks. Retain reload-on-open, manual refresh, and refresh after archiving.
- Preserve fresh-session defaults and alternate current-session bindings; submit automatically without a confirmation or editor draft.
- Leave the browser closed and report cancellation or failure when fresh-session creation does not succeed. Never fall back to the old session or reopen automatically.
- Approve the routing/integration design and revised launch lifecycle and verification plan below.

## Implementation plan

### Actions and prompt routing

- Register handlers for refinement and implementation through the existing browser action hooks.
- Keep r / i for fresh-session refinement / implementation and Shift+R / Shift+I for their current-session equivalents.
- Use the selected row from the displayed snapshot and the existing `getEligibleActions()` policy; do not reload the tree before dispatch.
- Route workflows as follows:
  - Top-level refinement: `/shemiq-refine-and-split <task-path>`.
  - Subtask refinement: `/shemiq-refine-sub-task <parent-path> <exact-title>`.
  - Refined-subtask implementation: `/shemiq-implement-task <subtask-path>`.
  - New-subtask implementation: `/shemiq-implement-task-parent <parent-path> <exact-title>`.
- Use paths relative to Pi's working directory, including `..` for external tasks. Do not change Pi's working directory.
- Preserve spaces, quotes, and backslashes through Pi-compatible argument quoting, following the existing Go launcher's behavior.

### Preparation and browser integration

- Represent the prepared launch as plain data: template name, arguments, and fresh/current-session choice.
- Separate preparation from dispatch so the browser's custom screen finishes and disposes before session replacement or prompt submission.
- Check that the required template is loaded before closing the browser. If unavailable, keep the browser open and explain how to load the Shemiq prompts and run `/reload`.
- Use Pi's loaded templates and standard template expansion; do not introduce bundled-file fallbacks or a second prompt loader.
- Ensure one action cannot trigger duplicate launches.
- Keep the launch boundary small enough for the creation task to reuse current-session dispatch later; do not build a general workflow framework.

### Session lifecycle

- Successful launches close the browser and automatically submit the corresponding prompt, without an editor draft or confirmation.
- For fresh sessions, use `ctx.newSession()` and submit through the replacement context supplied to `withSession`, with template expansion enabled.
- Carry only the prepared prompt inputs across the switch. Do not copy conversation history or generate a handoff.
- Do not reuse the old session context after replacement; use the replacement context for session-bound work.
- For current-session launches, use Pi's standard user-message dispatch with template expansion enabled.
- If fresh-session creation is cancelled or fails, leave the browser closed and report it. Never submit into the old session as a fallback or automatically reopen the browser.

### Minimal verification and follow-up boundaries

- Add focused automated checks for the four prompt routes and argument preservation, fresh/current-session dispatch, missing-template handling, and cancelled fresh-session creation.
- Do not add stale-selection checks for an automatic recheck path: that behavior is explicitly out of scope.
- Do not duplicate Go parsing/validation tests or introduce terminal end-to-end automation infrastructure.
- Leave consolidated documentation and the full manual smoke checklist to the verification task; note these launch paths and cancellation behavior for that follow-up.
- Creation remains a separate task; it may reuse current-session dispatch. Archiving remains a separate task using the existing browser refresh/recovery foundation, without requiring a new reload-and-resolve helper.

## Implementation notes

- Added `extensions/shemiq/workflows.ts` with all four prompt routes matching Go launcher's `piMessage()` in `internal/agent/pi.go`.
- `buildPromptMessage()` is a pure function producing template name + quoted message string — exported for testing.
- `registerWorkflowHandlers()` takes a `register` callback (injected from `index.ts`) to avoid importing browser.ts at test time.
- Handler checks template availability via `pi.getCommands()` before closing the browser; throws with actionable message if missing.
- Fresh sessions use `ctx.newSession({ withSession })` — the replacement context's `sendUserMessage` submits the prompt with template expansion.
- Current sessions use `pi.sendUserMessage()` with `expandPromptTemplates: true`.
- Cancelled fresh-session creation closes browser and shows a warning notification (no fallback to old session).
- Pi-compatible argument quoting mirrors Go's `quotePiArgument`: single-quote wrapping with `'"'"'` for embedded quotes.
- Paths are made relative to Pi's cwd via `path.relative()`, naturally producing `..` prefixes for external tasks.
- Added `workflows.test.ts` with 6 tests covering all four routing paths, quote escaping, and external path handling.
- All 16 extension tests pass (10 controller + 6 workflow).
