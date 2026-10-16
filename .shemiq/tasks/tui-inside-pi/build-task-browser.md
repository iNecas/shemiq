# Build the keyboard-driven task browser
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: f8957813-fb9b-4714-a2ed-c14bd696595e
:::

## Context

- Build the browser foundation for the in-Pi workflows described in `./top-level.md`; do not implement prompt execution, description entry, or archiving in this task.
- Add the extension entry point at `extensions/shemiq/index.ts`. Separate rendering, CLI loading, and action integration enough for the follow-up tasks to extend the browser without replacing it.
- Load `shemiq tree --json` from Pi's working directory. Reuse `extensions/shemiq/types.ts` unchanged; leave Markdown parsing, validation, project discovery, and archive filesystem operations in the CLI.
- External references remain visible and receive the same actions as local tasks, subject to ordinary eligibility and existing CLI restrictions. Do not add ownership-based read-only checks, switch Pi's working directory, or extend the JSON contract to discover the current project.
- Use Pi's extension and TUI APIs, theme, and terminal components; do not create a separate terminal renderer.

## Interview

- Choose a temporary custom screen as the primary presentation.
- Retain an overlay and a built-in selection dialog as fallback approaches if practical limitations emerge.
- Choose Alt+S for opening, with `/shemiq` as the reliable fallback. Neither conflicts with the inspected Pi defaults or current user keybindings; terminal, desktop, or other extension bindings can still interfere.
- Drop the parent's original external-task read-only policy; keep external references actionable without project switching or bypassing CLI restrictions.
- Show eligible but unconnected actions and report “not implemented yet” when pressed, rather than hiding them.
- Approve the screen/key map, typed action integration boundary, loading/recovery behavior, and lightweight verification plan below.
- Explicitly document why the shortcut dispatches the command rather than opening the browser with a shortcut-only context.

## Implementation plan

### Screen and keyboard interaction

- Use a temporary custom screen through `ctx.ui.custom()`.
- Open through `/shemiq` or Alt+S only in interactive TUI mode and while Pi is idle. If busy, notify immediately without interruption or delayed opening.
- Show the entire hierarchy expanded in CLI order, including done tasks. Include indentation, a selection marker, titles, and explicit status labels.
- Keep selection visible in a bounded scrolling area and the action legend outside that area. Fit long titles and narrow terminals without overflowing.
- Use these controls:
  - Alt+S: open the browser; `/shemiq` is the equivalent command.
  - Up/Down or k/j: move selection.
  - Page Up/Page Down and Home/End: navigate longer lists.
  - n: New task, independently of selection.
  - r / i: Refine / Implement in a fresh session.
  - Shift+R / Shift+I: Refine / Implement in the current session.
  - a: Archive the selected top-level task.
  - Ctrl+R: Refresh.
  - Esc or q: Close.
  - Enter: no default action.
- Omit ineligible task actions from the legend. Eligible actions without connected handlers remain visible; pressing them reports “not implemented yet” and leaves the browser open.
- Do not add search/filter controls, previews, action menus, collapsible branches, background watchers, or project switching.

### Command and shortcut entry: preserve session-changing capabilities

- Pi command handlers receive `ExtensionCommandContext`, which exposes session-changing operations such as `ctx.newSession()`.
- Shortcut handlers receive the more limited `ExtensionContext`, which can show UI but does not expose `newSession()`.
- Route both entry paths through the registered command:
  - `/shemiq` → command handler → browser.
  - Alt+S → programmatically dispatch `/shemiq` → command handler → browser.
- Use Pi's supported command dispatch mechanism, with command/template expansion enabled (`pi.sendUserMessage("/shemiq", { expandPromptTemplates: true })`). This dispatches the registered extension command; it does not send `/shemiq` to the LLM or require the user to submit editor text.
- Check the shortcut's idle/TUI prerequisites before dispatch and enforce them again in the command handler. Do not queue an opening while busy.
- Do not save a context from an earlier command invocation for later shortcut use: it could belong to a replaced session.
- Keep future session-changing work on the command-capable execution path. This task establishes that path; the workflow subtask implements session changes and prompt dispatch.

### Rows, selection, and refresh

- Flatten the returned hierarchy into rows while retaining node data, ancestors, depth, and stable identity for selection and action handling.
- Do not identify a task solely by title or list index.
- Preserve the selected task across refresh when it survives; otherwise select a nearby surviving row. Never transfer a pending action to that replacement selection.
- Support inline subtasks and uncreated source documents. An absent subtask document does not itself make refinement or parent-based implementation unavailable.
- Load on every opening and Ctrl+R. Rebuild rows, selection, and the legend from the successful snapshot.
- Show loading state, prevent duplicate operations, keep Close responsive, and ignore late results after closing.
- Treat a successful empty hierarchy as an empty state with New task, Refresh, and Close.

### Action integration and follow-up boundaries

- Establish a small typed action interface carrying the action, fresh/current-session choice where relevant, and selected-task context.
- Provide handler hooks for refinement/implementation, task creation, and archiving; do not build a general-purpose plugin system.
- Centralize the parent's eligibility rules so the legend and eventual execution share one policy:
  - New top-level task: Refine and Archive.
  - Refined or done top-level task: Archive only.
  - New subtask under a refined parent: Refine and Implement.
  - Refined subtask under a refined parent: Implement.
  - Done subtasks: neither Refine nor Implement.
  - New task is global; ownership does not suppress actions on external references.
- Provide a reload-and-resolve helper for selected-task actions. Connected handlers must recheck task identity and eligibility before dispatching.
- If the selected task disappears or becomes ineligible, refresh and explain why the attempt was cancelled. Never silently target another row or use stale metadata.
- Let handlers close the browser, return to it, or refresh it. This supports returning after creation cancellation and keeping the browser open after archiving.
- The workflow subtask connects prompts, fresh/current-session behavior, and execution-time rechecks to this foundation.
- The creation subtask supplies multiline entry and its cancellation/submission behavior.
- The archive subtask supplies the CLI operation and success/failure handling, reusing browser refresh.
- The verification subtask retains full workflow smoke coverage and consolidated documentation.

### Recovery

- Treat a missing project, missing CLI, invalid metadata, unreadable files, malformed JSON, or command failure as recovery states, not successfully loaded empty hierarchies.
- Show useful diagnostics and retain New task, Refresh, and Close. New task uses the placeholder until creation is connected.
- Remove or disable stale rows after loading failure so they cannot launch task-specific actions.
- Do not automatically install tools, repair metadata, or add background synchronization.

### Presentation alternatives

- Primary: temporary custom screen; gives the hierarchy, legend, and diagnostics room within Pi's interactive area.
- Fallback: overlay above the conversation; preserves visible conversation context but introduces sizing/focus handling and less usable space.
- Fallback: adapt Pi's built-in selection dialog; minimizes custom UI code but is less suited to hierarchy, selection-specific direct actions, and recovery screens.

### Minimal verification

- Add a representative browser/controller check covering hierarchy navigation, eligibility-driven legends, placeholder actions, and selection preservation.
- Add a recovery check: failed refresh disables stale task actions; a later successful refresh restores browsing.
- Do not duplicate CLI parsing/validation tests or introduce terminal end-to-end automation infrastructure.
- Add a short manual checklist covering `/shemiq` and Alt+S, busy-state refusal, scrolling, narrow terminals, refresh, placeholder notifications, and closing.
- Leave full workflow smoke coverage and consolidated usage/development documentation to the verification subtask.

## Implementation notes

- Extension entry point: `extensions/shemiq/index.ts` with `/shemiq` command and `Alt+S` shortcut
- Browser component: `extensions/shemiq/browser.ts` — custom screen via `ctx.ui.custom()`, full keyboard navigation
- Controller logic: `extensions/shemiq/controller.ts` — tree loading, row flattening, eligibility rules, selection management
- Existing types: `extensions/shemiq/types.ts` — unchanged, generated from Go
- Tests: `extensions/shemiq/controller.test.ts` — 12 tests covering eligibility, legend, and selection resolution
- Alt+S dispatches `/shemiq` via `pi.sendUserMessage("/shemiq", { expandPromptTemplates: true })` to get `ExtensionCommandContext`
- Follow-up simplification: TUI/idle checks centralized in the command handler; Alt+S only dispatches `/shemiq`, which Pi handles before model execution or queuing.
- Action handler registry (`registerActionHandler`) provides typed hooks for follow-up tasks
- Unconnected actions show "not implemented yet" notification
- Recovery: error state shows diagnostics, retains New/Refresh/Close; stale rows cleared
- Scrolling: bounded viewport based on terminal height, scroll indicators for overflow
- Selection preserved across refresh by stable id (uuid > path > generated)
