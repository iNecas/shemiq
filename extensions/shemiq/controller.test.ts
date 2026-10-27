/**
 * Controller checks: action contracts, eligible legends, and selection preservation.
 */

import { describe, it, expect } from "vitest";
import {
  flattenTree,
  getEligibleActions,
  getLegendEntries,
  resolveSelection,
} from "./controller.ts";
import { makeTreeNode } from "./test/test-fixtures.ts";
import { TreeStatusRefined, TreeStatusDone } from "./types.ts";

// ---------------------------------------------------------------------------
// Eligibility contract
// ---------------------------------------------------------------------------

describe("getEligibleActions", () => {
  const [parent, child] = flattenTree([
    makeTreeNode({
      title: "Parent",
      status: TreeStatusRefined,
      path: "/project/.shemiq/tasks/parent/top-level.md",
      subtasks: [{ title: "Inline task" }],
    }),
  ]);

  it.each([
    {
      name: "document-backed top-level task",
      row: parent,
      expected: new Set(["archive", "edit"]),
    },
    {
      name: "inline subtask with ancestors but no document path or UUID",
      row: child,
      expected: new Set(["refine", "implement"]),
    },
  ])("returns action kinds for a $name", ({ row, expected }) => {
    const actions = getEligibleActions(row);
    expect(actions).toBeInstanceOf(Set);
    expect(actions).toEqual(expected);
  });

  it("offers edit for any row with a path, independent of status", () => {
    const [parent, doneChild] = flattenTree([
      makeTreeNode({
        title: "Parent",
        status: TreeStatusDone,
        path: "/project/.shemiq/tasks/parent/top-level.md",
        subtasks: [
          {
            title: "Done child",
            status: TreeStatusDone,
            path: "/project/.shemiq/tasks/parent/done-child.md",
          },
        ],
      }),
    ]);

    expect(getEligibleActions(parent)).toContain("edit");
    expect(getEligibleActions(doneChild)).toEqual(new Set(["edit"]));
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

  it("labels creation as New subtask on subtask rows", () => {
    const [, child] = flattenTree([
      makeTreeNode({
        title: "Parent",
        status: TreeStatusRefined,
        subtasks: [{ title: "Child" }],
      }),
    ]);
    const entries = getLegendEntries(child);
    expect(entries.find(({ key }) => key === "n")?.label).toBe(
      "New subtask",
    );
  });

  it("includes eligible actions for a row", () => {
    const [row] = flattenTree([makeTreeNode({ title: "Task" })]);
    const entries = getLegendEntries(row);
    expect(entries.map(({ key, action }) => ({ key, action }))).toEqual([
      { key: "n", action: { kind: "new" } },
      { key: "r", action: { kind: "refine" } },
      { key: "a", action: { kind: "archive" } },
    ]);
  });

  it("shows the edit legend entry only for rows with a path", () => {
    const [withPath, pathless] = flattenTree([
      makeTreeNode({
        title: "With path",
        path: "/project/.shemiq/tasks/with-path/top-level.md",
        subtasks: [
          { title: "Pathless child" },
          {
            title: "Pathed child",
            path: "/project/.shemiq/tasks/with-path/child.md",
          },
        ],
      }),
    ]);

    expect(getLegendEntries(withPath).map(({ key }) => key)).toContain("e");
    expect(getLegendEntries(pathless).map(({ key }) => key)).not.toContain("e");
  });

  it("keeps only global creation available for completed subtasks", () => {
    const [, child] = flattenTree([
      makeTreeNode({
        title: "Parent",
        status: TreeStatusRefined,
        subtasks: [{ title: "Child", status: TreeStatusDone }],
      }),
    ]);
    const entries = getLegendEntries(child);
    expect(entries.map(({ action }) => action)).toEqual([{ kind: "new" }]);
  });

  it("offers implement for an eligible refined subtask", () => {
    const [, child] = flattenTree([
      makeTreeNode({
        title: "Parent",
        status: TreeStatusRefined,
        subtasks: [{ title: "Child", status: TreeStatusRefined }],
      }),
    ]);
    const entries = getLegendEntries(child);
    expect(entries.map(({ key, action }) => ({ key, action }))).toEqual([
      { key: "n", action: { kind: "new" } },
      { key: "i", action: { kind: "implement" } },
    ]);
  });
});

// ---------------------------------------------------------------------------
// Selection resolution
// ---------------------------------------------------------------------------

describe("resolveSelection", () => {
  const rows = flattenTree([
    makeTreeNode({ title: "A", uuid: "aaa" }),
    makeTreeNode({ title: "B", uuid: "bbb" }),
    makeTreeNode({ title: "C", uuid: "ccc" }),
  ]);

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
