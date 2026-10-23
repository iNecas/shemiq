# unify tui components
:::shemiq
type: top-level
uuid: ad7e00b4-cde6-4b6c-be18-3db860d41283
status: refined
:::

## Description

- The browser currently uses a non-overlay custom interaction
  (`extensions/shemiq/browser.ts`), while the launcher uses an overlay
  (`extensions/shemiq/launcher.ts`). The creation dialog also uses an overlay
  (`extensions/shemiq/creation-dialog.ts`).
- Overlays keep the browser mounted when a child screen opens, so cancelling
  returns to the browser in its original state. Simply changing a nested child
  interaction to non-overlay would replace the browser and restore Pi's editor
  on cancellation instead.
- Use non-overlay navigation for all three screens while preserving the browser's
  original selection and scroll position when a child screen is cancelled.

## Context

### Pi custom UI lifecycle (findings from exploration)

- `ctx.ui.custom()` is wired to `InteractiveMode.showExtensionCustom()` by
  `createExtensionUIContext()` in Pi's
  `packages/coding-agent/src/modes/interactive/interactive-mode.ts`. The local
  checkout inspected is `/home/inecas/projects/ai/pi/pi`; useful entry points
  are `createExtensionUIContext()` around line 2633 and `showExtensionCustom()`
  around line 2966. The public API contract is in
  `packages/coding-agent/src/core/extensions/types.ts` (`ExtensionUIContext.custom`).
- The core chooses the mounting strategy with
  `const isOverlay = options?.overlay ?? false`. Without an overlay, it clears
  `editorContainer`, adds the extension component, and gives it keyboard focus.
  This replaces only the editor area, not the entire chat UI. With an overlay,
  it leaves the existing component mounted and calls `tui.showOverlay()` instead.
- Clearing a container only sets its `children` to an empty array; it does not
  destroy or dispose those components. Pi retains the original editor separately
  in `this.editor`. Before opening a custom UI, it also captures the editor text.
- The factory's `done(result)` callback is Pi's `close()` closure. For a
  non-overlay interaction, it calls `restoreEditor()`: clear the custom component,
  re-add `this.editor`, restore the captured text, focus the editor, and request a
  render. It then resolves the custom UI promise and calls the custom component's
  optional `dispose()`. Crucially, it restores the main editor, **not whichever
  custom component previously occupied the editor container**. Non-overlay
  `ui.custom()` calls do not provide an automatic stack/back-navigation mechanism.
- Consequently, changing the launcher to a nested non-overlay `ui.custom()` call
  alone would detach the browser, and cancelling the launcher would restore Pi's
  editor rather than the browser. Detachment is distinct from disposal, but Pi
  does not automatically remount the detached browser or resolve its outstanding
  custom UI promise. Refinement needs to address component ownership, return
  navigation, and preservation of browser state (selection and scroll position).
- The current overlay launcher avoids that replacement: the browser remains
  mounted underneath, and completing the launcher removes only its own overlay
  and restores focus. The browser instance and its state remain available.

### Overlay implementation references

- In `packages/tui/src/tui.ts`, `showOverlay()` (around line 729) maintains the
  overlay stack and previous focus; `resolveOverlayLayout()` (around line 1215)
  determines sizing/position; `compositeOverlays()` (around line 1350) renders
  overlay lines over the underlying UI. Defaults are a centered overlay up to
  80 columns wide, constrained by available terminal width.
- Components use the same `render(width)`, `handleInput(data)`, and `invalidate()`
  interface in either mounting mode. The launcher component itself does not
  decide whether it replaces the editor area or appears as an overlay.
- Finish an interaction created by `ctx.ui.custom()` using its supplied `done()`
  callback, not by directly hiding its overlay handle; the callback owns promise
  resolution and disposal. `OverlayHandle.setHidden()` is for temporary visibility
  changes while keeping an interaction active.
- Pi's `docs/tui.md` and `examples/extensions/overlay-qa-tests.ts` describe and
  demonstrate focus, stacking, sizing, and visibility behavior.

## Interview

- Scope: browser, launcher, and creation dialog; no visual standardization or Pi
  core changes.
- Considered retaining one host, closing/reopening screens with saved state, and
  adding custom-screen stacking to Pi; chose one non-overlay host retaining the
  browser instance.
- Preserve browser selection and scrolling on cancellation without reloading the
  tree; discard cancelled creation drafts and launcher choices.
- Keep navigation explicit through a small screen-opening interface passed to
  action handlers, rather than moving workflow orchestration into the host.
- Preserve existing validation, error notifications, refresh behavior, and
  prompt/session dispatch semantics.
- Prefer minimal real-runtime E2E regression coverage, with focused lifecycle
  tests only where they add useful coverage.
- Implement as one subtask intended to fit one session, rather than separate host,
  migration, and testing tasks.

## Design

### Architecture and components

- Open one non-overlay `ctx.ui.custom()` interaction per browser visit; keep a
  Shemiq host mounted for the interaction and display only its active child.
- The host owns screen switching, rendering/input/focus forwarding, and component
  lifetime. Support the current browser-to-child-and-back flows, not an arbitrary
  navigation stack or general-purpose router.
- The browser retains ownership of task rows, selection, scrolling, refreshes,
  and notifications.
- Pass an explicit, small screen-opening interface to action handlers. Launcher
  and creation helpers use it to await child results instead of opening nested
  Pi custom interactions.
- Keep workflow routing and CLI business logic in existing handlers; the host
  does not orchestrate business workflows.
- Relevant integration points are `extensions/shemiq/browser.ts`, `launcher.ts`,
  `creation-dialog.ts`, `workflows.ts`, `creation.ts`, and the extension entry
  point `index.ts`.
- Keep existing screen controls and workflow behavior; visual standardization
  and changes to Pi itself are out of scope.

### Data flow and state

- Opening a child retains the browser instance without rendering it.
- Cancelling ends and disposes the child interaction and returns to the same
  browser, without reloading the tree or resetting selection and scrolling.
- Reopening a cancelled child starts from existing defaults; do not retain
  creation drafts or launcher choices.
- Successful titled creation returns to the browser and refreshes the tree as
  today.
- Prompt-based creation and confirmed workflow launches close the UI before
  prompt dispatch or session changes, preserving the existing separation in
  `workflows.ts`.

### Error handling and lifecycle

- Preserve existing behavior: creation-field validation remains inline; no
  available models produces a dismissible launcher screen that returns to the
  retained browser.
- Missing templates, unavailable selected models, and CLI failures continue to
  surface through the browser's existing action-error notifications.
- Launch/session failures after UI closure remain with Pi's existing command
  handling; do not reopen the browser to recover from them.
- Returning from a child ends that child's lifetime, not the retained browser's.
- Ending the outer interaction cleans up all owned components and timers.
  Completion and cleanup tolerate repeated calls; late asynchronous results
  must not revive a closed interaction.

### Testing

- Extend `extensions/shemiq/workflows.e2e.test.ts` using the existing real Pi
  runtime and virtual-terminal harness.
- Cancel both launcher and creation screens after navigating far enough to
  scroll; verify the selected task and viewport remain unchanged.
- Verify reopening uses fresh child state and closing the browser restores
  keyboard control to Pi's editor.
- Reuse existing successful creation and fresh/current-session launch scenarios.
- Add focused lifecycle tests only where terminal-driven scenarios cannot
  easily establish useful coverage, such as repeated completion/disposal or late
  async results; avoid broad rendering snapshots and a large mock-based suite.

## Current status

- Context, design, and the single-task split approved during refinement.
- "Unify Shemiq screen navigation" implemented: a single non-overlay
  `ScreenHost` owns the browser plus child screens; the launcher and
  creation dialog were migrated off overlays onto the screen-opening
  interface, with navigation/lifecycle regression coverage added.
- Finding for potential follow-up: Pi's viewport scrolling consumes plain
  pageUp/pageDown/home/end before focused custom components, so the
  browser's bindings for those keys never fire in the real terminal.

## Tasks

### Unify Shemiq screen navigation
:::shemiq
type: task
source: ./unify-screen-navigation.md
status: done
:::
Introduce the single non-overlay host, migrate launcher and creation screens while preserving existing behavior, and add minimal navigation/lifecycle regression coverage.
