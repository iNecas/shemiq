/**
 * Controller checks: action contracts, eligible legends, and selection preservation.
 */

import { describe, it, expect } from "vitest";
import {
  type BrowserRow,
  getEligibleActions,
  getLegendEntries,
  resolveSelection,
} from "./controller.ts";
import {
  type TreeNode,
  TreeStatusNew,
  TreeStatusRefined,
  TreeStatusDone,
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
    projectRoot: "/test",
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
// Eligibility contract
// ---------------------------------------------------------------------------

describe("getEligibleActions", () => {
  const parent = makeNode({
    title: "Parent",
    type: TreeTaskTypeTopLevel,
    status: TreeStatusRefined,
    path: "/test/.shemiq/tasks/parent/top-level.md",
  });

  it.each([
    {
      name: "document-backed top-level task",
      row: makeRow(parent),
      expected: new Set(["archive"]),
    },
    {
      name: "inline subtask with ancestors but no document path or UUID",
      row: makeRow(makeNode({ title: "Inline task" }), 1, [parent]),
      expected: new Set(["refine", "implement"]),
    },
  ])("returns action kinds for a $name", ({ row, expected }) => {
    const actions = getEligibleActions(row);
    expect(actions).toBeInstanceOf(Set);
    expect(actions).toEqual(expected);
  });
});

// ---------------------------------------------------------------------------
// Legend
// ---------------------------------------------------------------------------

describe("getLegendEntries", () => {
  it("always includes New task", () => {
    const entries = getLegendEntries(null);
    expect(entries).toEqual([
      { key: "n", label: "New task", action: { kind: "new" } },
    ]);
  });

  it("includes eligible actions for a row", () => {
    const node = makeNode({
      title: "Task",
      type: TreeTaskTypeTopLevel,
      status: TreeStatusNew,
    });
    const row = makeRow(node);
    const entries = getLegendEntries(row);
    expect(entries.map(({ key, action }) => ({ key, action }))).toEqual([
      { key: "n", action: { kind: "new" } },
      { key: "r", action: { kind: "refine", session: "fresh" } },
      { key: "R", action: { kind: "refine", session: "current" } },
      { key: "a", action: { kind: "archive" } },
    ]);
  });

  it("keeps only global creation available for completed subtasks", () => {
    const parent = makeNode({
      title: "Parent",
      type: TreeTaskTypeTopLevel,
      status: TreeStatusRefined,
    });
    const child = makeNode({ title: "Child", status: TreeStatusDone });
    const entries = getLegendEntries(makeRow(child, 1, [parent]));
    expect(entries.map(({ action }) => action)).toEqual([{ kind: "new" }]);
  });

  it("offers both session choices for an eligible implementation", () => {
    const parent = makeNode({
      title: "Parent",
      type: TreeTaskTypeTopLevel,
      status: TreeStatusRefined,
    });
    const child = makeNode({ title: "Child", status: TreeStatusRefined });
    const entries = getLegendEntries(makeRow(child, 1, [parent]));
    expect(entries.map(({ key, action }) => ({ key, action }))).toEqual([
      { key: "n", action: { kind: "new" } },
      { key: "i", action: { kind: "implement", session: "fresh" } },
      { key: "I", action: { kind: "implement", session: "current" } },
    ]);
  });
});

// ---------------------------------------------------------------------------
// Selection resolution
// ---------------------------------------------------------------------------

describe("resolveSelection", () => {
  const rows: BrowserRow[] = [
    makeRow(makeNode({ title: "A", uuid: "aaa" })),
    makeRow(makeNode({ title: "B", uuid: "bbb" })),
    makeRow(makeNode({ title: "C", uuid: "ccc" })),
  ];

  it("preserves selection by id", () => {
    expect(resolveSelection(rows, "bbb", 1)).toBe(1);
  });

  it("falls back to clamped index when id disappears", () => {
    expect(resolveSelection(rows, "zzz", 5)).toBe(2);
  });

  it("returns -1 for empty rows", () => {
    expect(resolveSelection([], "aaa", 0)).toBe(-1);
  });

  it("selects nearby row after removal", () => {
    // If row at index 1 was removed and id gone, clamp to last
    const shortened = [rows[0], rows[2]];
    expect(resolveSelection(shortened, "bbb", 1)).toBe(1);
  });
});
