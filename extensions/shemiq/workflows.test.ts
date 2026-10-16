/** Workflow handler checks: prompt routing and argument quoting. */

import { describe, it, expect } from "vitest";
import { builtPromptMessage } from "./workflows.ts";
import type { BrowserRow } from "./controller.ts";
import type { TreeNode } from "./types.ts";
import {
  TreeStatusNew,
  TreeStatusRefined,
  TreeTaskTypeTopLevel,
  TreeTaskTypeTask,
} from "./types.ts";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeNode(
  overrides: Partial<TreeNode> & { title: string },
): TreeNode {
  return {
    type: overrides.type ?? TreeTaskTypeTask,
    title: overrides.title,
    status: overrides.status ?? TreeStatusNew,
    projectRoot: "/project",
    subtasks: overrides.subtasks ?? [],
    ...overrides,
  };
}

function makeRow(
  node: TreeNode,
  depth = 0,
  ancestors: TreeNode[] = [],
): BrowserRow {
  return {
    id: node.uuid || node.path || node.title,
    node,
    depth,
    ancestors,
  };
}

// ---------------------------------------------------------------------------
// Prompt routing
// ---------------------------------------------------------------------------

describe("builtPromptMessage", () => {
  const cwd = "/project";
  const parent = makeNode({
    title: "Feature X",
    type: TreeTaskTypeTopLevel,
    status: TreeStatusRefined,
    path: "/project/.shemiq/tasks/feature-x/top-level.md",
  });

  it("routes top-level refinement", () => {
    const topLevel = makeNode({
      title: "Feature Y",
      type: TreeTaskTypeTopLevel,
      status: TreeStatusNew,
      path: "/project/.shemiq/tasks/feature-y/top-level.md",
    });
    const row = makeRow(topLevel);
    const result = builtPromptMessage(
      { kind: "refine", session: "fresh" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-refine-and-split");
    expect(result.message).toBe(
      "/shemiq-refine-and-split '.shemiq/tasks/feature-y/top-level.md'",
    );
  });

  it("routes subtask refinement with parent path and title", () => {
    const sub = makeNode({
      title: "Build widget",
      status: TreeStatusNew,
    });
    const row = makeRow(sub, 1, [parent]);
    const result = builtPromptMessage(
      { kind: "refine", session: "fresh" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-refine-sub-task");
    expect(result.message).toBe(
      "/shemiq-refine-sub-task '.shemiq/tasks/feature-x/top-level.md' 'Build widget'",
    );
  });

  it("routes refined subtask implementation", () => {
    const sub = makeNode({
      title: "Build widget",
      status: TreeStatusRefined,
      path: "/project/.shemiq/tasks/feature-x/build-widget.md",
    });
    const row = makeRow(sub, 1, [parent]);
    const result = builtPromptMessage(
      { kind: "implement", session: "fresh" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-implement-task");
    expect(result.message).toBe(
      "/shemiq-implement-task '.shemiq/tasks/feature-x/build-widget.md'",
    );
  });

  it("routes new subtask implementation through parent", () => {
    const sub = makeNode({
      title: "Build widget",
      status: TreeStatusNew,
    });
    const row = makeRow(sub, 1, [parent]);
    const result = builtPromptMessage(
      { kind: "implement", session: "fresh" },
      row,
      cwd,
    );
    expect(result.template).toBe("shemiq-implement-task-parent");
    expect(result.message).toBe(
      "/shemiq-implement-task-parent '.shemiq/tasks/feature-x/top-level.md' 'Build widget'",
    );
  });

  it("quotes titles with single quotes", () => {
    const sub = makeNode({
      title: "it's a test",
      status: TreeStatusNew,
    });
    const row = makeRow(sub, 1, [parent]);
    const result = builtPromptMessage(
      { kind: "refine", session: "fresh" },
      row,
      cwd,
    );
    expect(result.message).toContain(`'it'"'"'s a test'`);
  });

  it("uses relative paths including .. for external tasks", () => {
    const external = makeNode({
      title: "External",
      type: TreeTaskTypeTopLevel,
      status: TreeStatusNew,
      path: "/other-project/.shemiq/tasks/ext/top-level.md",
    });
    const row = makeRow(external);
    const result = builtPromptMessage(
      { kind: "refine", session: "fresh" },
      row,
      cwd,
    );
    expect(result.message).toContain("../other-project/");
  });
});
