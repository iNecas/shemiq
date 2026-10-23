# Unify Shemiq screen navigation
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 63683116-257e-49d2-8a8c-95ed8ec4c536
:::

## Context

The browser opens as a non-overlay `ctx.ui.custom()` interaction, while the
launcher and creation dialog open as nested overlay interactions. The goal is
a single non-overlay interaction per browser visit that hosts all three
screens, with the browser instance retained (selection + scroll preserved)
while a child screen is open. See the parent top-level document for the full
Pi custom-UI lifecycle analysis and design decisions.

## Implementation plan

- New `extensions/shemiq/screens.ts`:
  - `ScreenOpener` interface: `showScreen(create)` where `create` receives a
    `finish` callback plus `tui`/`theme`; returns a promise resolving when the
    child finishes (or `null` if the host closed meanwhile).
  - `ScreenHost` component: mounted once per browser visit (non-overlay),
    owns the persistent browser ("home") and the active child screen;
    forwards `render`/`handleInput`/`invalidate`, disposes finished children,
    and finishes the outer interaction through Pi's `done` callback.
    Idempotent close/dispose; late or repeated child finishes are ignored.
- `browser.ts`: `openBrowser` constructs the host instead of mounting the
  browser directly; `ActionHandler` gains an explicit 4th `screens`
  parameter; browser `done` becomes the host close.
- `launcher.ts` / `creation-dialog.ts`: `show*` helpers take the
  `ScreenOpener` and create their components through it instead of nested
  overlay `ctx.ui.custom()` calls; component internals unchanged.
- `workflows.ts` / `creation.ts`: handlers forward `screens` to the helpers;
  everything else (validation, notifications, launch semantics) unchanged.
- Tests: update focused tests for the new handler signature; extend
  `workflows.e2e.test.ts` with a navigation regression: scroll the browser,
  cancel launcher and creation dialog, verify selection/viewport unchanged and
  fresh child state on reopen; closing the browser restores the editor.

## Implementation notes

- New `extensions/shemiq/ext-ui.ts`:
  - `ExtUiHost` implements both `Component` and `ExtUi`; it is the single
    non-overlay `ctx.ui.custom()` interaction per browser visit.
  - The browser is a persistent "home" screen; `extUi.showDialog(create)`
    swaps in a dialog, and the dialog's `finish` callback disposes it and
    returns to the retained home without touching its selection or scroll
    state.
  - `close()` ends the whole interaction through Pi's `done`; teardown
    settles still-pending dialog promises with `null` (bypassing the stateful
    finish wrappers) and disposes active component + home exactly once each.
    Repeated close/dispose and late dialog finishes are ignored.
- `browser.ts`: `openBrowser` constructs the host, registers the browser as
  home, and passes `host.close` as the browser's `done`. `ActionHandler`
  gained an explicit 4th `extUi: ExtUi` parameter.
- `launcher.ts` / `creation-dialog.ts`: `showLauncher` / `showCreationDialog`
  take the `ExtUi` and build their components through
  `extUi.showDialog(...)` instead of nested overlay `ctx.ui.custom()`
  calls. Component internals are unchanged; `showCreationDialog` no longer
  needs `ctx`.
- `workflows.ts` / `creation.ts`: handlers forward `extUi`; validation,
  notifications, refresh, and launch/session dispatch semantics unchanged.
- Testing:
  - `ext-ui.test.ts`: focused lifecycle coverage (switching, single
    completion/disposal, input forwarding, close-while-pending, repeated
    close/dispose).
  - `workflows.e2e.test.ts`: new regression scrolls the browser (24 tasks),
    cancels launcher and creation dialog, asserts byte-identical viewport
    (selection + scroll preserved), fresh child state on reopen (draft
    discarded), and editor control restored after closing the browser.
  - `makeExtUi()` double added to `test/test-helpers.ts`.
- Finding (Pi behavior, out of scope): Pi's alt-screen viewport input
  handler consumes plain `pageUp`/`pageDown`/`home`/`end` (bound to
  `tui.altScreen.*`) before any focused custom component sees them, so the
  browser's `pageUp`/`pageDown`/`home`/`end` bindings are dead in the real
  terminal (`j`/`k` and arrows work). Candidate follow-up: rebind or drop
  those keys; the e2e test scrolls with `j` accordingly.
- Dev infra: `npm run test:fresh` clears vite's transform cache
  (`node_modules/.vite`) and runs the suite — useful when instrumenting
  `node_modules` during debugging, so stale cached transforms don't mask
  the patches.
