/** Archive handler checks: CLI arguments, result shape, and error paths. */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { execFile } from "child_process";
import { registerArchiveHandler } from "./archive.ts";
import type { ActionHandler } from "./browser.ts";
import { flattenTree } from "./controller.ts";
import { makeTreeNode, type TreeNodeDefinition } from "./test/test-fixtures.ts";
import {
  captureActionHandlers,
  makeCommandContext,
  mockExecFileResult,
} from "./test/test-helpers.ts";

vi.mock("child_process", () => ({
  execFile: vi.fn(),
}));

const definition: TreeNodeDefinition = {
  title: "Archive me",
  path: "/project/.shemiq/tasks/archive-me/top-level.md",
  subtasks: [],
};

describe("archive handler", () => {
  let handler: ActionHandler;

  beforeEach(() => {
    vi.mocked(execFile).mockReset();
    handler = captureActionHandlers(
      registerArchiveHandler,
      {} as ExtensionAPI,
    ).get("archive")!;
  });

  it("archives the selected top-level task and requests a refresh", async () => {
    mockExecFileResult(null, "", "");
    const [row] = flattenTree([makeTreeNode(definition)]);

    const result = await handler({ kind: "archive" }, row, makeCommandContext());

    expect(execFile).toHaveBeenCalledExactlyOnceWith(
      "shemiq",
      ["archive", ".shemiq/tasks/archive-me"],
      { cwd: "/project" },
      expect.any(Function),
    );
    expect(result).toEqual({
      refresh: true,
      notify: "Archived .shemiq/tasks/archive-me",
    });
  });

  it("keeps failures in the browser when the CLI rejects", async () => {
    mockExecFileResult(new Error("exit 1"), "", "task not found");
    const [row] = flattenTree([makeTreeNode(definition)]);

    await expect(
      handler({ kind: "archive" }, row, makeCommandContext()),
    ).rejects.toThrow("shemiq archive failed: task not found");
  });

  it("reports a missing CLI binary", async () => {
    const err = new Error("spawn failed") as NodeJS.ErrnoException;
    err.code = "ENOENT";
    mockExecFileResult(err, "", "");
    const [row] = flattenTree([makeTreeNode(definition)]);

    await expect(
      handler({ kind: "archive" }, row, makeCommandContext()),
    ).rejects.toThrow(
      "shemiq CLI not found. Install it and make sure it is on PATH.",
    );
  });

  it("throws without touching the CLI when the row has no path", async () => {
    const [row] = flattenTree([
      makeTreeNode({ title: "No path", subtasks: [] }),
    ]);

    await expect(
      handler({ kind: "archive" }, row, makeCommandContext()),
    ).rejects.toThrow("Cannot archive: task has no document path");
    await expect(
      handler({ kind: "archive" }, null, makeCommandContext()),
    ).rejects.toThrow("Cannot archive: task has no document path");
    expect(execFile).not.toHaveBeenCalled();
  });
});
