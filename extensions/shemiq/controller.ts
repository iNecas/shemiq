/**
 * Browser controller: loads the tree, flattens rows, manages selection,
 * computes eligibility, and provides action hooks.
 */

import type { TreeDocument, TreeNode, TreeStatus } from "./types.ts";

// ---------------------------------------------------------------------------
// Row model
// ---------------------------------------------------------------------------

export interface BrowserRow {
  /** Stable identity: uuid if available, else path, else generated id. */
  id: string;
  node: TreeNode;
  depth: number;
  /** Ancestor chain from root to this node's parent. */
  ancestors: TreeNode[];
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

export type BrowserAction =
  | { kind: "new" | "archive" }
  | { kind: "refine" | "implement"; session: "fresh" | "current" };

export type ActionKind = BrowserAction["kind"];

export interface ActionBinding {
  key: string;
  label: string;
  action: BrowserAction;
}

export const actionBindings: ActionBinding[] = [
  { key: "n", label: "New task", action: { kind: "new" } },
  { key: "r", label: "Refine", action: { kind: "refine", session: "fresh" } },
  { key: "i", label: "Implement", action: { kind: "implement", session: "fresh" } },
  {
    key: "R",
    label: "Refine (current)",
    action: { kind: "refine", session: "current" },
  },
  {
    key: "I",
    label: "Implement (current)",
    action: { kind: "implement", session: "current" },
  },
  { key: "a", label: "Archive", action: { kind: "archive" } },
];

// ---------------------------------------------------------------------------
// Eligibility
// ---------------------------------------------------------------------------

/**
 * Return the set of eligible action kinds for the given row.
 * Follows the centralized eligibility rules from the top-level design.
 */
export function getEligibleActions(row: BrowserRow): Set<ActionKind> {
  const actions = new Set<ActionKind>();
  const { node, ancestors } = row;
  const isTopLevel = node.type === "top-level";
  const status = node.status as TreeStatus;

  if (isTopLevel) {
    // Top-level tasks can always be archived
    actions.add("archive");

    if (status === "new") {
      // New top-level: refine
      actions.add("refine");
    }
    // Refined or done top-level: no refine/implement (select subtasks instead)
  } else {
    // Subtask
    const parent = ancestors[ancestors.length - 1];
    const parentRefined =
      parent && (parent.status === "refined" || parent.status === "done");

    if (status === "done") {
      // Done subtask: no actions
    } else if (parentRefined) {
      if (status === "new") {
        actions.add("refine");
        actions.add("implement");
      } else if (status === "refined") {
        actions.add("implement");
      }
    }
  }

  return actions;
}

/**
 * Get the legend entries for the given row (eligible actions with labels).
 */
export function getLegendEntries(row: BrowserRow | null): ActionBinding[] {
  const eligible = row ? getEligibleActions(row) : new Set<ActionKind>();
  return actionBindings.filter(
    ({ action }) => action.kind === "new" || eligible.has(action.kind),
  );
}

// ---------------------------------------------------------------------------
// Tree loading
// ---------------------------------------------------------------------------

import { execFile } from "child_process";

export type TreeLoadResult =
  | { ok: true; rows: BrowserRow[] }
  | { ok: false; message: string };

export async function loadTree(cwd: string): Promise<TreeLoadResult> {
  try {
    const json = await runShemiqTree(cwd);
    const doc: TreeDocument = JSON.parse(json);
    const rows = flattenTree(doc.tasks);
    return { ok: true, rows };
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    return { ok: false, message: msg };
  }
}

function runShemiqTree(cwd: string): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile("shemiq", ["tree", "--json"], { cwd }, (err, stdout, stderr) => {
      if (err) {
        const detail = stderr?.trim() || err.message;
        reject(new Error(`shemiq tree failed: ${detail}`));
      } else {
        resolve(stdout);
      }
    });
  });
}

export function flattenTree(
  nodes: TreeNode[],
  depth = 0,
  ancestors: TreeNode[] = [],
): BrowserRow[] {
  const rows: BrowserRow[] = [];
  let counter = 0;
  for (const node of nodes) {
    const id = node.uuid || node.path || `__generated_${depth}_${counter++}`;
    rows.push({ id, node, depth, ancestors: [...ancestors] });
    if (node.subtasks && node.subtasks.length > 0) {
      rows.push(...flattenTree(node.subtasks, depth + 1, [...ancestors, node]));
    }
  }
  return rows;
}

// ---------------------------------------------------------------------------
// Selection management
// ---------------------------------------------------------------------------

/**
 * Find the best selection index after a refresh.
 * If previousId still exists, select it. Otherwise, find a nearby row.
 */
export function resolveSelection(
  rows: BrowserRow[],
  previousId: string | null,
  previousIndex: number,
): number {
  if (rows.length === 0) return -1;
  if (previousId) {
    const idx = rows.findIndex((r) => r.id === previousId);
    if (idx >= 0) return idx;
  }
  // Clamp to valid range
  return Math.min(previousIndex, rows.length - 1);
}
