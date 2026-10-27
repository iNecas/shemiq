/** Shared doubles for focused extension tests. */

import { vi } from "vitest";
import { execFile } from "child_process";
import type {
  ExtensionAPI,
  ExtensionCommandContext,
} from "@earendil-works/pi-coding-agent";
import type { ActionHandler } from "../browser.ts";
import type { ActionKind } from "../controller.ts";
import type { ExtUi } from "../ext-ui.ts";

// ---------------------------------------------------------------------------
// Context double
// ---------------------------------------------------------------------------

/** Command-context double with a working directory and a selected model. */
export function makeCommandContext(): ExtensionCommandContext {
  return {
    cwd: "/project",
    model: { provider: "test", id: "chat:latest" },
    modelRegistry: {
      getAvailable: () => [{ provider: "test", id: "chat:latest" }],
    },
  } as unknown as ExtensionCommandContext;
}

// ---------------------------------------------------------------------------
// Handler registration
// ---------------------------------------------------------------------------

export type RegisterActionHandler = (
  kind: ActionKind,
  handler: ActionHandler,
) => void;

/**
 * Run a `register*Handler(pi, register)` setup against a fresh map
 * and return the captured handlers.
 */
export function captureActionHandlers(
  setup: (pi: ExtensionAPI, register: RegisterActionHandler) => void,
  pi: ExtensionAPI,
): ReadonlyMap<ActionKind, ActionHandler> {
  const handlers = new Map<ActionKind, ActionHandler>();
  setup(pi, (kind, handler) => handlers.set(kind, handler));
  return handlers;
}

// ---------------------------------------------------------------------------
// Ext UI double
// ---------------------------------------------------------------------------

/** Ext UI double; dialogs resolve to cancellation, terminal ops run inline. */
export function makeExtUi(): ExtUi {
  return {
    showDialog: () => Promise.resolve(null),
    runInTerminal: (op: () => Promise<unknown>) => op(),
  };
}

// ---------------------------------------------------------------------------
// CLI mocking
// ---------------------------------------------------------------------------

type ExecFileCallback = (
  err: Error | null,
  stdout: string,
  stderr: string,
) => void;

/** The execFile shape the handlers use; narrower than the real overloads. */
type ExecFileFn = (
  command: string,
  args: string[],
  options: { cwd: string },
  callback: ExecFileCallback,
) => void;

/** Resolve the next `execFile` result for a mocked CLI run. */
export function mockExecFileResult(
  err: Error | null,
  stdout: string,
  stderr: string,
): void {
  vi.mocked(execFile as unknown as ExecFileFn).mockImplementation(
    (_cmd, _args, _opts, cb) => cb(err, stdout, stderr),
  );
}
