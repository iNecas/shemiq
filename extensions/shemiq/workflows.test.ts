/** Workflow handler checks: prompt routing and argument quoting. */

import { describe, it, expect } from "vitest";
import { builtPromptMessage } from "./workflows.ts";
import { flattenTree } from "./controller.ts";
import { makeTreeNode, type TreeNodeDefinition } from "./test-fixtures.ts";
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
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "Build widget" }] }),
    ]);
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
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "Build widget" }] }),
    ]);
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
    const [, row] = flattenTree([
      makeTreeNode({ ...parent, subtasks: [{ title: "it's a test" }] }),
    ]);
    const result = builtPromptMessage(
      { kind: "refine", session: "fresh" },
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
      { kind: "refine", session: "fresh" },
      row,
      cwd,
    );
    expect(result.message).toContain("../other-project/");
  });
});
