/**
 * Task creation: connects the browser's "new" action to the CLI and,
 * when the title is left blank, to the current-session prompt templates.
 */

import { execFile } from "child_process";
import type {
  ExtensionAPI,
  ExtensionCommandContext,
} from "@earendil-works/pi-coding-agent";
import type { ActionKind, BrowserRow } from "./controller.ts";
import type { ActionHandler } from "./browser.ts";
import type { ExtUi } from "./ext-ui.ts";
import { showCreationDialog } from "./creation-dialog.ts";
import type { PromptLaunch, PromptMessage } from "./workflows.ts";
import {
  quotePiArgument,
  relativePath,
  requireAvailableModel,
  requireTemplate,
} from "./workflows.ts";

// ---------------------------------------------------------------------------
// Public setup
// ---------------------------------------------------------------------------

/**
 * Register the "new" action handler on the browser.
 * Called once from the extension entry point.
 */
export function registerCreationHandler(
  pi: ExtensionAPI,
  register: (kind: ActionKind, handler: ActionHandler) => void,
): void {
  register("new", async (_action, row, ctx, extUi: ExtUi) => {
    const target = creationTargetFor(row);

    const input = await showCreationDialog(extUi, target);
    if (!input) {
      // Cancelled – return to the browser with selection preserved.
      return {};
    }

    const title = input.title.trim();
    const description = input.description.trim();
    if (!title && !description) {
      throw new Error("Title or description is required");
    }

    if (title) {
      const createdPath = await runShemiqCreate(
        ctx.cwd,
        buildCreationArgs(target, title, description),
      );
      return { refresh: true, notify: `Created ${createdPath}` };
    }

    // Blank title: close the browser and let the current session pick one.
    const route = builtCreationPromptMessage(target, description, ctx.cwd);
    requireTemplate(pi, route.template);
    const launch = await currentSessionLaunch(ctx, route);
    return { close: true, launch };
  });
}

// ---------------------------------------------------------------------------
// Target routing and command building (plain data, no side effects)
// ---------------------------------------------------------------------------

export type CreationTarget =
  | { kind: "top-level" }
  | {
      kind: "subtask";
      /** Absolute path of the parent top-level document. */
      parentPath: string;
      parentTitle: string;
      /** Exact title of the hovered sibling to insert after. */
      afterTitle: string;
    };

/**
 * Resolve what "n" creates from the hovered row (sibling semantics):
 * a subtask row targets a sibling in the same parent, anything else
 * (top-level row, no selection, empty hierarchy, error state) a new
 * top-level task.
 */
export function creationTargetFor(row: BrowserRow | null): CreationTarget {
  if (row && row.node.type === "task") {
    const parent = row.ancestors[row.ancestors.length - 1];
    if (!parent?.path) {
      throw new Error("Cannot create a subtask: parent has no document path");
    }
    return {
      kind: "subtask",
      parentPath: parent.path,
      parentTitle: parent.title,
      afterTitle: row.node.title,
    };
  }
  return { kind: "top-level" };
}

/** Build the non-interactive CLI arguments for a titled creation. */
export function buildCreationArgs(
  target: CreationTarget,
  title: string,
  description: string,
): string[] {
  const descriptionArgs = description ? [description] : [];
  if (target.kind === "top-level") {
    return ["new", "--title", title, ...descriptionArgs];
  }
  return [
    "new-subtask",
    target.parentPath,
    "--title",
    title,
    "--after",
    target.afterTitle,
    ...descriptionArgs,
  ];
}

/** Build the routed, quoted prompt message for a blank-title creation. */
export function builtCreationPromptMessage(
  target: CreationTarget,
  description: string,
  cwd: string,
): PromptMessage {
  if (target.kind === "top-level") {
    return {
      template: "shemiq-new",
      message: `/shemiq-new ${quotePiArgument(description)}`,
    };
  }
  return {
    template: "shemiq-new-subtask",
    message: [
      "/shemiq-new-subtask",
      quotePiArgument(relativePath(cwd, target.parentPath)),
      quotePiArgument(target.afterTitle),
      quotePiArgument(description),
    ].join(" "),
  };
}

// ---------------------------------------------------------------------------
// Blank-title dispatch through the current session
// ---------------------------------------------------------------------------

/** Build a current-session launch; model availability is checked up front. */
async function currentSessionLaunch(
  ctx: ExtensionCommandContext,
  route: PromptMessage,
): Promise<PromptLaunch> {
  const model = ctx.model;
  if (!model) {
    throw new Error(
      "No current model selected. Fill in the title to create directly.",
    );
  }
  const settings = { provider: model.provider, modelId: model.id };
  await requireAvailableModel(ctx, settings);
  return { ...route, ...settings, session: "current" };
}

// ---------------------------------------------------------------------------
// Direct CLI execution (same style as loadTree)
// ---------------------------------------------------------------------------

function runShemiqCreate(cwd: string, args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile("shemiq", args, { cwd }, (err, stdout, stderr) => {
      if (err) {
        if ((err as NodeJS.ErrnoException).code === "ENOENT") {
          reject(
            new Error(
              "shemiq CLI not found. Install it and make sure it is on PATH.",
            ),
          );
          return;
        }
        const detail = stderr?.trim() || err.message;
        reject(new Error(`shemiq ${args[0]} failed: ${detail}`));
      } else {
        resolve(stdout.trim());
      }
    });
  });
}
