/** Workflow checks: prompt routing, quoting, and destination-command transport. */

import { describe, it, expect, vi } from "vitest";
import type { ExtensionAPI, ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import {
  builtPromptMessage,
  dispatchLaunch,
  registerWorkflowHandlers,
  type PromptLaunch,
} from "./workflows.ts";
import { flattenTree } from "./controller.ts";
import { makeTreeNode, type TreeNodeDefinition } from "./test/test-fixtures.ts";
import { TreeStatusRefined } from "./types.ts";

// ---------------------------------------------------------------------------
// Prompt routing
// ---------------------------------------------------------------------------

describe("builtPromptMessage", () => {
  const cwd = "/project";
  const parent: TreeNodeDefinition = {
    title: "Feature X",
    status: TreeStatusRefined,
    path: "/project/.shemiq/tasks/feature-x/top-level.md",
  };

  it("routes top-level refinement", () => {
    const [row] = flattenTree([
      makeTreeNode({
        title: "Feature Y",
        path: "/project/.shemiq/tasks/feature-y/top-level.md",
      }),
    ]);
    const result = builtPromptMessage(
      { kind: "refine" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-refine-and-split");
    expect(result.message).toBe(
      "/shemiq-refine-and-split '.shemiq/tasks/feature-y/top-level.md'",
    );
  });

  it("routes subtask refinement with parent path and title", () => {
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "Build widget" }] }),
    ]);
    const result = builtPromptMessage(
      { kind: "refine" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-refine-sub-task");
    expect(result.message).toBe(
      "/shemiq-refine-sub-task '.shemiq/tasks/feature-x/top-level.md' 'Build widget'",
    );
  });

  it("routes refined subtask implementation", () => {
    const [, row] = flattenTree([
      makeTreeNode({
        ...parent,
        subtasks: [
          {
            title: "Build widget",
            status: TreeStatusRefined,
            path: "/project/.shemiq/tasks/feature-x/build-widget.md",
          },
        ],
      }),
    ]);
    const result = builtPromptMessage(
      { kind: "implement" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-implement-task");
    expect(result.message).toBe(
      "/shemiq-implement-task '.shemiq/tasks/feature-x/build-widget.md'",
    );
  });

  it("routes new subtask implementation through parent", () => {
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "Build widget" }] }),
    ]);
    const result = builtPromptMessage(
      { kind: "implement" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-implement-task-parent");
    expect(result.message).toBe(
      "/shemiq-implement-task-parent '.shemiq/tasks/feature-x/top-level.md' 'Build widget'",
    );
  });

  it("quotes titles with single quotes", () => {
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "it's a test" }] }),
    ]);
    const result = builtPromptMessage(
      { kind: "refine" },
      row,
      cwd,
    );
    expect(result.message).toContain(`'it'"'"'s a test'`);
  });

  it("uses relative paths including .. for external tasks", () => {
    const [row] = flattenTree([
      makeTreeNode({
        title: "External",
        projectRoot: "/other-project",
        path: "/other-project/.shemiq/tasks/ext/top-level.md",
      }),
    ]);
    const result = builtPromptMessage(
      { kind: "refine" },
      row,
      cwd,
    );
    expect(result.message).toContain("../other-project/");
  });
});

// ---------------------------------------------------------------------------
// JSON transport into the registered destination-session command
// ---------------------------------------------------------------------------

describe("shemiq-launch", () => {
  const [, row] = flattenTree([
    makeTreeNode({
      title: "Parent",
      status: TreeStatusRefined,
      path: "/project/.shemiq/tasks/parent with spaces/top-level.md",
      subtasks: [{ title: "It's \"quoted\" \\ Ω\nnext\tline" }],
    }),
  ]);
  const launch: PromptLaunch = {
    ...builtPromptMessage({ kind: "refine" }, row, "/project"),
    provider: "test",
    modelId: "chat:latest",
    session: "fresh",
  };

  it("round-trips JSON.stringify(launch) through the command before sending the original prompt", async () => {
    const destination = makeLaunchDestination(launch);
    const source = { setModel: vi.fn(), sendUserMessage: vi.fn() };
    const sendUserMessage = vi.fn(async (text: string) => {
      // Pi splits extension commands at the first space and passes raw args.
      const separator = text.indexOf(" ");
      expect(text.slice(0, separator)).toBe("/shemiq-launch");
      await destination.command.handler(text.slice(separator + 1), destination.ctx);
    });
    const newSession = vi.fn<ExtensionCommandContext["newSession"]>(async (options) => {
      await options!.withSession!(Object.assign(destination.ctx, {
        sendUserMessage,
        sendMessage: vi.fn(async () => {}),
      }));
      return { cancelled: false };
    });

    await dispatchLaunch(
      source as unknown as ExtensionAPI,
      { newSession } as unknown as ExtensionCommandContext,
      launch,
    );

    expect(sendUserMessage).toHaveBeenCalledWith(
      `/shemiq-launch ${JSON.stringify(launch)}`,
      { expandPromptTemplates: true },
    );
    expect(destination.api.setModel).toHaveBeenCalledExactlyOnceWith(destination.model);
    expect(destination.api.sendUserMessage).toHaveBeenCalledExactlyOnceWith(
      launch.message, { expandPromptTemplates: true },
    );
    expect(destination.api.setModel.mock.invocationCallOrder[0]).toBeLessThan(
      destination.api.sendUserMessage.mock.invocationCallOrder[0],
    );
    expect(destination.notify).not.toHaveBeenCalled();
    expect(source.setModel).not.toHaveBeenCalled();
    expect(source.sendUserMessage).not.toHaveBeenCalled();
  });

  it.each(["{not-json", "{}"])("rejects invalid payload %s without changing the model or sending a prompt", async (args) => {
    const destination = makeLaunchDestination(launch);
    await destination.command.handler(args, destination.ctx);
    expect(destination.notify).toHaveBeenCalledWith(
      expect.stringContaining("Shemiq launch failed:"), "error",
    );
    expect(destination.api.setModel).not.toHaveBeenCalled();
    expect(destination.api.sendUserMessage).not.toHaveBeenCalled();
  });
});

function makeLaunchDestination(launch: PromptLaunch) {
  const model = { provider: launch.provider, id: launch.modelId };
  const commands = new Map<string, Parameters<ExtensionAPI["registerCommand"]>[1]>();
  const api = {
    registerCommand: (name: string, command: Parameters<ExtensionAPI["registerCommand"]>[1]) => commands.set(name, command),
    getCommands: () => [{ name: launch.template }],
    setModel: vi.fn(async () => true),
    sendUserMessage: vi.fn(),
  };
  const notify = vi.fn();
  const ctx = {
    modelRegistry: { getAvailable: () => [model] },
    ui: { notify },
  } as unknown as ExtensionCommandContext;
  registerWorkflowHandlers(api as unknown as ExtensionAPI, () => {});
  return { api, ctx, model, notify, command: commands.get("shemiq-launch")! };
}
