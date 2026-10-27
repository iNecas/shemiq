/**
 * Edit handler checks: result shape, error paths, and the
 * terminal-handoff boundary. The full happy path (argv construction,
 * spawn, refresh, selection) is covered end-to-end by the fake-editor
 * scenario in workflows.e2e.test.ts.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { spawn } from "child_process";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import {
  parseEditorCommand,
  registerEditHandler,
} from "./edit.ts";
import type { ActionHandler } from "./browser.ts";
import type { ExtUi } from "./ext-ui.ts";
import { flattenTree } from "./controller.ts";
import { makeTreeNode, type TreeNodeDefinition } from "./test/test-fixtures.ts";
import { captureActionHandlers, makeCommandContext } from "./test/test-helpers.ts";

vi.mock("child_process", () => ({
  spawn: vi.fn(),
}));

const definition: TreeNodeDefinition = {
  title: "Edit me",
  path: "/project/.shemiq/tasks/edit-me/top-level.md",
  subtasks: [],
};

/** Ext UI double recording terminal handoffs. */
function makeRecordingExtUi(): { extUi: ExtUi; handoffs: () => number } {
  let count = 0;
  return {
    extUi: {
      showDialog: () => Promise.resolve(null),
      runInTerminal: (op: () => Promise<unknown>) => {
        count++;
        return op();
      },
    },
    handoffs: () => count,
  };
}

/** Spawn double collecting event handlers for manual emission. */
function mockSpawnChild() {
  const handlers = new Map<string, (...args: unknown[]) => void>();
  vi.mocked(spawn).mockImplementation(
    (() => ({
      on: (event: string, cb: (...args: unknown[]) => void) =>
        handlers.set(event, cb),
    })) as unknown as typeof spawn,
  );
  return {
    emit: (event: string, ...args: unknown[]) =>
      handlers.get(event)?.(...args),
  };
}

/** Run the handler against a path-backed row and settle the editor run. */
async function runEdit(
  handler: ActionHandler,
  extUi: ExtUi,
  exit: { code: number | null; signal: NodeJS.Signals | null },
) {
  const child = mockSpawnChild();
  const [row] = flattenTree([makeTreeNode(definition)]);
  const promise = handler({ kind: "edit" }, row, makeCommandContext(), extUi);
  child.emit("close", exit.code, exit.signal);
  return promise;
}

describe("edit handler", () => {
  let handler: ActionHandler;

  beforeEach(() => {
    vi.mocked(spawn).mockReset();
    process.env.EDITOR = "editor";
    handler = captureActionHandlers(
      registerEditHandler,
      {} as ExtensionAPI,
    ).get("edit")!;
  });

  afterEach(() => {
    delete process.env.EDITOR;
  });

  it("refreshes without a notification after a successful exit", async () => {
    const { extUi, handoffs } = makeRecordingExtUi();
    await expect(
      runEdit(handler, extUi, { code: 0, signal: null }),
    ).resolves.toEqual({ refresh: true });
    expect(handoffs()).toBe(1);
  });

  it.each([
    ["nonzero exit", { code: 1, signal: null }, "Editor exited with status 1"],
    ["signal termination", { code: null, signal: "SIGTERM" }, "Editor terminated by SIGTERM"],
  ])(
    "refreshes with a notification after %s",
    async (_name, exit, notification) => {
      const { extUi } = makeRecordingExtUi();
      await expect(runEdit(handler, extUi, exit)).resolves.toEqual({
        refresh: true,
        notify: notification,
      });
    },
  );

  it("rejects on launch failures", async () => {
    const child = mockSpawnChild();
    const { extUi } = makeRecordingExtUi();
    const [row] = flattenTree([makeTreeNode(definition)]);

    const promise = handler({ kind: "edit" }, row, makeCommandContext(), extUi);
    child.emit(
      "error",
      Object.assign(new Error("spawn failed"), { code: "ENOENT" }),
    );
    await expect(promise).rejects.toThrow(
      "Failed to launch editor: executable not found: editor",
    );
  });

  it.each([
    ["missing path", "editor", "Cannot edit: task has no document path", false],
    ["missing $EDITOR", undefined, "Cannot edit: $EDITOR is not set", true],
    ["blank $EDITOR", "   ", "Cannot edit: $EDITOR is not set", true],
  ] as const)(
    "fails without terminal handoff on %s",
    async (_name, editor, message, withPath) => {
      if (editor === undefined) delete process.env.EDITOR;
      else process.env.EDITOR = editor;
      const { extUi, handoffs } = makeRecordingExtUi();
      const [row] = flattenTree([
        makeTreeNode(
          withPath
            ? definition
            : { title: "No path", subtasks: [] },
        ),
      ]);

      await expect(
        handler({ kind: "edit" }, row, makeCommandContext(), extUi),
      ).rejects.toThrow(message);
      expect(handoffs()).toBe(0);
    },
  );
});

// ---------------------------------------------------------------------------
// Command parsing
// ---------------------------------------------------------------------------

describe("parseEditorCommand", () => {
  it("groups quoted arguments and keeps inner spaces", () => {
    expect(parseEditorCommand('"/opt/my editor" -n "a file"')).toEqual([
      "/opt/my editor",
      "-n",
      "a file",
    ]);
    expect(parseEditorCommand("'/opt/space bin'")).toEqual(["/opt/space bin"]);
  });

  it("rejects unterminated quotes", () => {
    expect(() => parseEditorCommand('editor "oops')).toThrow(
      "$EDITOR has an unterminated quote",
    );
  });
});
