# More end to end tests
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 2ebc4c22-9e8c-43d8-bb1a-532e498c4fa1
:::

## Context

- Existing extension tests cover controller policy, workflow routing/transport, and creation handlers; they do not drive the real interactive browser and dialogs together.
- Add a small representative workflow suite using Pi's real interactive/session runtime and a virtual terminal, without subprocess/pty terminal automation or real model services.
- Pi has an xterm-backed `VirtualTerminal` in `packages/tui/test/virtual-terminal.ts`. It is exported from that source module, but is not re-exported by the package entry point or included in the normal npm publishing file list.
- A disposable experiment against installed Pi 1.0.0 verified `/shemiq`, real CLI tree loading, launcher cancellation, fresh/current-session launches, prompt expansion, and launch → `/reload` → Alt+S → launch across real runtime replacement.
- The experiment used scripted faux models and made no fetch calls or real model requests. Its files were moved to `tmp/` for inspection; they are reference material, not dependencies of the implementation.

## Interview

- Choose representative real-runtime workflows over lifecycle-only coverage or exhaustive UI coverage.
- Copy only the small terminal helper needed for testing, rather than requiring a Pi checkout or copying Pi's larger session-testing harness.
- Add a local package definition under `extensions/shemiq` so development and testing do not depend on the system Pi installation.
- Approve npm with a lockfile, Vitest, explicit local development dependencies, and initially aligned Pi 1.0.0 versions validated by the experiment.
- Keep detailed component edge cases in the browser/launcher coverage task and archiving behavior in the archive task.
- Move minimal package/test-runner setup into this task; keep Makefile integration, documentation, and the manual checklist in the verification/documentation task.

## Implementation plan

### Standalone test setup

- Add a private ESM `extensions/shemiq/package.json` and npm lockfile.
- Declare development dependencies on Pi's coding-agent, AI, and TUI packages, Vitest, and `@xterm/headless`.
- Pin compatible Pi versions together, initially 1.0.0. Resolve dependencies locally, without `/nix/store`, global package installations, or sibling Pi checkouts.
- Provide `npm test` from `extensions/shemiq` to run the existing focused tests and the new workflow suite.
- Copy the minimum useful `VirtualTerminal` implementation into test support:
  - Implement Pi's `Terminal` interface and support keyboard injection, viewport inspection, and flushing pending writes.
  - Preserve upstream attribution, license information, and the source-version reference.
  - Do not copy Pi's larger session-testing harness; construct the runtime through public SDK APIs.

### Runtime and isolation

- Exercise the real `InteractiveMode`, session runtime, Shemiq entry point, browser, and dialogs through virtual-terminal keyboard input.
- Use isolated temporary working directories and Pi configuration; do not load personal credentials or configuration.
- Register only scripted faux models and disable network-dependent startup activity. No real provider requests or paid tokens.
- Build the repository's Shemiq CLI once for the suite and put that binary on the test process's PATH, rather than relying on a system-installed `shemiq`.
- Exercise real CLI/filesystem behavior against temporary fixture projects.
- Use bounded waits for observable state, not arbitrary sleeps or whole-screen snapshots.

### Representative workflows

- Launch and runtime replacement:
  - Open through `/shemiq`, cancel a launcher, and then launch with a selected faux model.
  - Cover fresh-session and current-session modes.
  - After launch, run `/reload`, reopen through Alt+S, and launch again in both session modes.
  - Assert session identity, selected model, and the expanded prompt received by the faux provider, rather than only checking a dispatch mock.
- Creation:
  - Submit a title and description to create a sibling subtask through the real CLI; verify insertion position and refreshed browser contents.
  - Submit a blank title to exercise prompt-based creation in the current session; verify the description reaches the expanded prompt.
- Loading-error recovery:
  - Cause a real tree-loading failure and assert the recovery screen.
  - Correct the fixture and use Refresh to recover.

### Boundaries and related tasks

- Keep the suite small and focused on the representative workflows above; do not duplicate Go parsing/validation coverage or exhaustively test every UI edge case.
- Reuse existing focused tests rather than duplicating their routing and transport assertions in the new suite.
- The browser/launcher coverage task retains detailed component and mocked context-handling edge cases; it can reuse this harness instead of adding another real-runtime smoke test for launch/reload.
- Archiving behavior remains in the archive task, which can reuse the harness once its handler exists.
- The verification/documentation task retains Makefile integration, usage/development documentation, the manual smoke checklist, and remaining focused checks; reuse this task's npm runner and representative workflows.

## Implementation notes

- Standalone npm setup in `extensions/shemiq`: private ESM `package.json` plus `package-lock.json`; dev deps on `@earendil-works/pi-coding-agent`, `pi-ai`, `pi-tui` (all pinned to the experiment-validated 1.0.0), `@xterm/headless` 5.5.0, and Vitest 3; `npm test` runs the focused tests plus the e2e suite (38 tests, ~3 s).
- Minimal `VirtualTerminal` copied to `extensions/shemiq/test/virtual-terminal.ts` from pi-tui 1.0.0 `test/virtual-terminal.ts` (MIT), with an attribution header; imports use normal package specifiers, no Pi checkout required.
- Reusable harness `extensions/shemiq/test/pi-runtime.ts` (`createHarness`):
  - builds the repository CLI once per suite (`go build`, memoized) and puts it first on PATH;
  - isolated temporary HOME/cache/agent dirs and `PI_OFFLINE=1`/`PI_SKIP_VERSION_CHECK=1`/`PI_TELEMETRY=0`, restored on dispose;
  - only a scripted faux provider registered; `globalThis.fetch` replaced with a throwing counter (asserted zero);
  - real runtime through public SDK APIs (`createAgentSessionRuntime` + `createAgentSessionServices` with `additionalExtensionPaths` for the extension and `additionalPromptTemplatePaths` for the repository `prompts/`); real `InteractiveMode` over the virtual terminal;
  - exposes `waitFor` (bounded 8 s waits on rendered screen/prompt state), `typeCommand`/`send`/`typeText` (bracketed-paste typing for dialog fields), `runShemiq`, and fixture file helpers.
- `workflows.e2e.test.ts` covers the planned representative workflows in 5 tests:
  - launcher cancel returns to the browser; fresh-session launch asserts session replacement, selected model (`model-b` via arrow keys), expanded `# Refine:` prompt, and faux call count; current-session launch (`c` toggle) keeps session identity;
  - `/reload` → Alt+S reopen → launch again in both session modes against the replaced runtime;
  - direct sibling-subtask creation through the real CLI: dialog on a hovered subtask row, title + description submit, browser refresh with notification, insertion position asserted via `shemiq tree --json`;
  - blank-title creation closes the browser and dispatches the expanded `/shemiq-new` prompt in the current session;
  - loading-error recovery: corrupted UUID shows the recovery screen with non-actionable rows (`r` → "No task selected"), fixing the fixture plus Ctrl+R recovers the tree.
- Follow-ups can reuse the harness: the archive task for archive flows, the verification task for the manual checklist/Makefile wiring.
