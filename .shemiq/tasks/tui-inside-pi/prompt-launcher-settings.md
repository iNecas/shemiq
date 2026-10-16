# Prompt launcher settings
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 1ed06419-8071-4423-b370-c280b9f47a44
:::

## Context

- Add a pre-launch model/session dialog to the existing refinement and implementation workflows in `extensions/shemiq/workflows.ts`.
- Reuse the browser action hooks and launch integration in `browser.ts`, `controller.ts`, and `index.ts`.
- This task supersedes the separate `R/I` current-session actions described in the parent and completed workflow task; fresh sessions remain the default.
- Preserve eligibility, the four existing prompt routes, argument quoting, loaded-template checks, and standalone CLI behavior.
- Task creation remains current-session only without this dialog; archiving remains immediate without a prompt.

## Interview

- Apply settings to refinement and implementation only, not task creation.
- Follow-up: show only scoped models when Pi has a model scope; otherwise fall back to the full authenticated chat model list.
- On every opening, preselect the current Pi model and leave “Use current session” unchecked; do not remember launcher preferences.
- Use a single screen with a model list and checkbox; `c` toggles current-session mode directly without changing focus.
- Confirm with Enter; cancel with Esc and preserve the browser's selected task.
- Let the chosen model remain active in the destination session, like a normal Pi model change; do not restore the previous model or change saved startup defaults.
- Approve the dialog, launch-boundary, failure-handling, and minimal verification design below.

## Implementation plan

### Dialog and browser actions

- Keep `r` for Refine and `i` for Implement; remove `R/I` current-session actions and their legend entries.
- Identify the selected task and action in the dialog.
- Show a scrollable model list from Pi's available authenticated chat models, with provider and model identifiers to distinguish similarly named models.
- Initialize the selection from the current Pi model and the “Use current session” checkbox to unchecked each time.
- Use Pi's configured selection movement bindings, including normal Up/Down and page-navigation defaults.
- Toggle the checkbox with `c` while focus stays on the model list; show a compact keyboard legend.
- Enter confirms and launches; Esc cancels without changing the model or session and returns to the browser with the same task selected.
- Do not add a model-search field, effort control, saved per-action settings, or general hotkey-configuration system.

### Preparation and dispatch boundary

- Launch from the displayed task snapshot using existing eligibility and prompt routing; do not add tree reloads or change CLI behavior.
- Keep dialog state separate from dispatch. Confirmation returns plain launch data containing the routed prompt, selected provider/model ID, and fresh/current-session choice; cancellation produces no launch.
- Finish and dispose the browser/dialog before changing the active model, replacing the session, or submitting the prompt.
- Block duplicate confirmation while launching.
- In the current session, apply the selected model through Pi's normal model-switch API, then submit the existing prompt with template expansion enabled.
- In a fresh session, create the session, apply the selected model in the replacement runtime, then submit the existing prompt with template expansion enabled.
- Pi recreates its extension runtime on session replacement. Use a small destination-session command/handler boundary to apply the model against the active runtime; do not call a captured old `pi.setModel()` from `withSession`.
- Carry only plain launch data across replacement; retain existing fresh-session behavior without conversation copying or a handoff.
- Leave the selected model active in the destination session; do not restore the previous model or write Pi's saved startup default or launcher preferences.

### Errors and cancellation

- Check the required loaded template and selected model's availability before leaving the browser; retain actionable pre-launch errors in the browser.
- If no models are available, explain the problem and allow cancellation, but disable confirmation.
- If applying the selected model fails, report the error and do not submit the prompt with another model.
- After launch begins, session cancellation or failure leaves the browser closed; report it without old-session fallback or automatic reopening.

### Minimal verification and follow-up boundaries

- Add minimal automated checks for initial model/checkbox state, `c`, confirmation/cancellation, and model-before-prompt ordering in fresh and current sessions, plus one representative failure.
- Manually smoke-test the real Pi replacement-runtime path; mocked ordering alone does not establish that runtime replacement works.
- Retain existing routing coverage; do not add terminal end-to-end automation infrastructure.
- Keep task creation, immediate archiving, saved action settings, hotkey refactoring, and consolidated documentation as separate tasks.
- Creation may reuse the small dispatch boundary without adopting this dialog or changing its current-session behavior.
- The hotkey follow-up should account for removal of `R/I` and the dialog's `c` toggle; broad hotkey customization remains out of scope.
- The verification/documentation follow-up should cover the dialog controls, both session modes, selected-model behavior, cancellation, and launch failures.

## Implementation notes

- Created `launcher.ts` with `showLauncher()` that opens an overlay dialog on top of the browser.
- `LauncherComponent` uses Pi's `SelectList` for model selection, `c` toggles current-session checkbox, Enter confirms, Esc cancels.
- `EmptyModelsComponent` handles the no-models case with an explanation and Esc-only exit.
- Models fetched via `ctx.modelRegistry.getAvailable()`, restricted by provider/model ID to `ctx.scopedModels` in scope order when nonempty; no scope falls back to the full authenticated chat model list.
- Removed `R/I` action bindings from `controller.ts`; `r/i` now open the launcher dialog instead of directly dispatching.
- `BrowserAction` type simplified: `refine`/`implement` no longer carry `session` — session choice lives in the launcher dialog.
- Confirmation returns plain routed prompt/provider/model/session data; browser/dialog close and dispose before `index.ts` calls `dispatchLaunch()`.
- Fixed stale-context crash: `withSession` sends `/shemiq-launch` with serialized data; its newly registered destination handler resolves and applies the model using the active runtime's `pi`, then submits the prompt.
- Current-session path applies the model directly, then sends the message; selected-model availability and loaded templates are checked before closing and again at dispatch.
- Action-handler maps are factory-local in `index.ts`, preventing captured APIs from leaking across reloads; browser blocks concurrent actions and clears notification timers on disposal.
- Destination model failures notify without prompt submission or fallback; cancellation leaves the browser closed, and replacement errors propagate without accessing old UI/context.
- Context-handling regression coverage is deferred to the extension-testing subtask, including launch → `/reload` → launch and real Pi replacement-runtime smoke verification.
- Dynamic import of `launcher.ts` in `workflows.ts` keeps pure-function tests (prompt routing) working without pi-tui dependency.
- All 19 extension tests pass; focused `shemiq-launch` coverage round-trips `JSON.stringify(launch)` through the registered destination handler with quotes, backslashes, Unicode and whitespace, checks model-before-prompt ordering, and rejects malformed/incomplete payloads without dispatch.
