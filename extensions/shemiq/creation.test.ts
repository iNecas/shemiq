/** Creation checks: target routing, CLI arguments, and prompt routing. */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { execFile } from "child_process";
import {
  buildCreationArgs,
  builtCreationPromptMessage,
  creationTargetFor,
  registerCreationHandler,
  type CreationTarget,
} from "./creation.ts";
import { showCreationDialog } from "./creation-dialog.ts";
import type { ActionHandler } from "./browser.ts";
import type { ActionKind } from "./controller.ts";
import { flattenTree } from "./controller.ts";
import { makeTreeNode, type TreeNodeDefinition } from "./test-fixtures.ts";
import { TreeStatusRefined } from "./types.ts";

vi.mock("./creation-dialog.ts", () => ({
  showCreationDialog: vi.fn(),
}));
vi.mock("child_process", () => ({
  execFile: vi.fn(),
}));

const parent: TreeNodeDefinition = {
  title: "Feature X",
  status: TreeStatusRefined,
  path: "/project/.shemiq/tasks/feature-x/top-level.md",
  subtasks: [{ title: "Build widget" }],
};

// ---------------------------------------------------------------------------
// Target routing (sibling semantics)
// ---------------------------------------------------------------------------

describe("creationTargetFor", () => {
  it("routes a subtask row to a sibling in the same parent", () => {
    const [, row] = flattenTree([makeTreeNode(parent)]);
    expect(creationTargetFor(row)).toEqual({
      kind: "subtask",
      parentPath: "/project/.shemiq/tasks/feature-x/top-level.md",
      parentTitle: "Feature X",
      afterTitle: "Build widget",
    });
  });

  it("routes a top-level row to a new top-level task", () => {
    const [row] = flattenTree([makeTreeNode(parent)]);
    expect(creationTargetFor(row)).toEqual({ kind: "top-level" });
  });

  it("routes no selection to a new top-level task", () => {
    expect(creationTargetFor(null)).toEqual({ kind: "top-level" });
  });
});

// ---------------------------------------------------------------------------
// Direct CLI arguments (title filled)
// ---------------------------------------------------------------------------

describe("buildCreationArgs", () => {
  it("builds the top-level creation command", () => {
    expect(
      buildCreationArgs({ kind: "top-level" }, "My task", "Do things"),
    ).toEqual(["new", "--title", "My task", "Do things"]);
  });

  it("builds the subtask insertion command with the hovered sibling", () => {
    const target: CreationTarget = {
      kind: "subtask",
      parentPath: "/project/.shemiq/tasks/feature-x/top-level.md",
      parentTitle: "Feature X",
      afterTitle: "Build widget",
    };
    expect(buildCreationArgs(target, "My task", "Do things")).toEqual([
      "new-subtask",
      "/project/.shemiq/tasks/feature-x/top-level.md",
      "--title",
      "My task",
      "--after",
      "Build widget",
      "Do things",
    ]);
  });
});

// ---------------------------------------------------------------------------
// Prompt routing (title blank)
// ---------------------------------------------------------------------------

describe("builtCreationPromptMessage", () => {
  const target: CreationTarget = {
    kind: "subtask",
    parentPath: "/project/.shemiq/tasks/feature-x/top-level.md",
    parentTitle: "Feature X",
    afterTitle: "Build widget",
  };

  it("routes blank-title top-level creation to /shemiq-new", () => {
    const result = builtCreationPromptMessage(
      { kind: "top-level" },
      "Do things",
      "/project",
    );
    expect(result.template).toBe("shemiq-new");
    expect(result.message).toBe("/shemiq-new 'Do things'");
  });

  it("routes blank-title subtask creation to /shemiq-new-subtask", () => {
    const result = builtCreationPromptMessage(target, "Do things", "/project");
    expect(result.template).toBe("shemiq-new-subtask");
    expect(result.message).toBe(
      "/shemiq-new-subtask '.shemiq/tasks/feature-x/top-level.md' " +
        "'Build widget' 'Do things'",
    );
  });

  it("quotes arguments with special characters", () => {
    const result = builtCreationPromptMessage(
      target,
      'it\'s "quoted" \\ Ω\nnext line',
      "/project",
    );
    expect(result.message).toContain(`'it'"'"'s "quoted" \\ Ω\nnext line'`);
  });

  it("uses relative paths including .. for external parents", () => {
    const external: CreationTarget = {
      kind: "subtask",
      parentPath: "/other-project/.shemiq/tasks/ext/top-level.md",
      parentTitle: "External",
      afterTitle: "Sibling",
    };
    const result = builtCreationPromptMessage(external, "Do", "/project");
    expect(result.message).toContain("../other-project/");
  });
});

// ---------------------------------------------------------------------------
// Handler execution paths
// ---------------------------------------------------------------------------

describe("creation handler", () => {
  const [, subtaskRow] = flattenTree([makeTreeNode(parent)]);
  let handler: ActionHandler;

  beforeEach(() => {
    vi.mocked(showCreationDialog).mockReset();
    vi.mocked(execFile).mockReset();
    const handlers = new Map<ActionKind, ActionHandler>();
    registerCreationHandler(makePi(), (kind, h) => handlers.set(kind, h));
    handler = handlers.get("new")!;
  });

  it("creates directly through the CLI when the title is filled", async () => {
    vi.mocked(showCreationDialog).mockResolvedValue({
      title: "My task",
      description: "Do things",
    });
    mockExecFileResult(
      null,
      "/project/.shemiq/tasks/feature-x/top-level.md\n",
      "",
    );

    const result = await handler({ kind: "new" }, null, makeCtx());

    expect(execFile).toHaveBeenCalledExactlyOnceWith(
      "shemiq",
      ["new", "--title", "My task", "Do things"],
      { cwd: "/project" },
      expect.any(Function),
    );
    expect(result).toEqual({
      refresh: true,
      notify: "Created /project/.shemiq/tasks/feature-x/top-level.md",
    });
  });

  it("closes the browser with a current-session launch when the title is blank", async () => {
    vi.mocked(showCreationDialog).mockResolvedValue({
      title: "",
      description: "Do things",
    });

    const result = await handler({ kind: "new" }, subtaskRow, makeCtx());

    expect(result).toEqual({
      close: true,
      launch: {
        template: "shemiq-new-subtask",
        message:
          "/shemiq-new-subtask '.shemiq/tasks/feature-x/top-level.md' " +
          "'Build widget' 'Do things'",
        provider: "test",
        modelId: "chat:latest",
        session: "current",
      },
    });
  });

  it("returns to the browser when the dialog is cancelled", async () => {
    vi.mocked(showCreationDialog).mockResolvedValue(null);
    const result = await handler({ kind: "new" }, subtaskRow, makeCtx());
    expect(result).toEqual({});
    expect(execFile).not.toHaveBeenCalled();
  });

  it("keeps failures in the browser when the CLI rejects", async () => {
    vi.mocked(showCreationDialog).mockResolvedValue({
      title: "My task",
      description: "Do things",
    });
    mockExecFileResult(
      new Error("exit 1"),
      "",
      "duplicate subtask title: My task",
    );

    await expect(handler({ kind: "new" }, null, makeCtx())).rejects.toThrow(
      "shemiq new failed: duplicate subtask title: My task",
    );
  });
});

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

function makePi() {
  return {
    getCommands: () => [{ name: "shemiq-new" }, { name: "shemiq-new-subtask" }],
  } as unknown as Parameters<typeof registerCreationHandler>[0];
}

function makeCtx() {
  return {
    cwd: "/project",
    model: { provider: "test", id: "chat:latest" },
    modelRegistry: {
      getAvailable: () => [{ provider: "test", id: "chat:latest" }],
    },
  } as unknown as ExtensionCommandContext;
}

type ExecFileCallback = (
  err: Error | null,
  stdout: string,
  stderr: string,
) => void;

/** The execFile shape creation.ts uses; narrower than the real overloads. */
type ExecFileFn = (
  command: string,
  args: string[],
  options: { cwd: string },
  callback: ExecFileCallback,
) => void;

function mockExecFileResult(
  err: Error | null,
  stdout: string,
  stderr: string,
) {
  vi.mocked(execFile as unknown as ExecFileFn).mockImplementation(
    (_cmd, _args, _opts, cb) => cb(err, stdout, stderr),
  );
}
