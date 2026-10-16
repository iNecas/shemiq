import {
  type TreeNode,
  TreeStatusNew,
  TreeTaskTypeTopLevel,
  TreeTaskTypeTask,
} from "./types.ts";

export type TreeNodeDefinition = Pick<TreeNode, "title"> &
  Partial<Omit<TreeNode, "title" | "subtasks">> & {
    subtasks?: TreeNodeDefinition[];
  };

/** Build a fresh fixture tree, leaving document identities absent unless supplied. */
export function makeTreeNode(definition: TreeNodeDefinition): TreeNode {
  return buildTreeNode(definition, true, "/project");
}

function buildTreeNode(
  definition: TreeNodeDefinition,
  isRoot: boolean,
  inheritedProjectRoot: string,
): TreeNode {
  const projectRoot = definition.projectRoot ?? inheritedProjectRoot;
  return {
    ...definition,
    type: definition.type ?? (isRoot ? TreeTaskTypeTopLevel : TreeTaskTypeTask),
    status: definition.status ?? TreeStatusNew,
    projectRoot,
    subtasks: (definition.subtasks ?? []).map((child) =>
      buildTreeNode(child, false, projectRoot),
    ),
  };
}
