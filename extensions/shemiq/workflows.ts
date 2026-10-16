/**
 * Workflow handlers: connect refinement and implementation actions
 * from the browser to Pi's prompt templates.
 */

import type {
  ExtensionAPI,
  ExtensionCommandContext,
} from "@earendil-works/pi-coding-agent";
import type { BrowserAction, BrowserRow } from "./controller.ts";
import type { ActionHandler } from "./browser.ts";
import * as path from "node:path";

// ---------------------------------------------------------------------------
// Public setup
// ---------------------------------------------------------------------------

/**
 * Register refine and implement action handlers on the browser.
 * Called once from the extension entry point.
 */
export function registerWorkflowHandlers(
  pi: ExtensionAPI,
  register: (kind: string, handler: ActionHandler) => void,
): void {
  register("refine", createPromptDispatcher(pi));
  register("implement", createPromptDispatcher(pi));
}

// ---------------------------------------------------------------------------
// Unified handler (refine + implement share structure)
// ---------------------------------------------------------------------------

function createPromptDispatcher(pi: ExtensionAPI) {
  return async (
    action: BrowserAction,
    row: BrowserRow | null,
    ctx: ExtensionCommandContext,
  ): Promise<{ close?: boolean; refresh?: boolean }> => {
    if (!row) {
      throw new Error("No task selected");
    }

    const route = builtPromptMessage(action, row, ctx.cwd);
    const session = (action as { session: "fresh" | "current" }).session;

    // Check template is loaded before closing the browser.
    if (!isTemplateLoaded(pi, route.template)) {
      throw new Error(
        `Prompt template "/${route.template}" is not loaded. ` +
          `Load the Shemiq prompt templates and run /reload.`,
      );
    }

    const message = route.message;

    if (session === "current") {
      pi.sendUserMessage(message, { expandPromptTemplates: true });
      return { close: true };
    }

    // Fresh session: close browser first (done by returning close: true),
    // then create new session and submit there.
    // We need to do it the other way: first prepare, then close, then create session.
    // But the browser dispatch already handles close for us when we return { close: true }.
    // The trick: we start newSession which replaces context. The browser's done()
    // will be called after we return. So we fire newSession and return close.

    const { cancelled } = await ctx.newSession({
      withSession: async (newCtx) => {
        await newCtx.sendUserMessage(message, { expandPromptTemplates: true });
      },
    });

    if (cancelled) {
      // Per spec: leave browser closed, report cancellation.
      // We still return close: true so the browser disposes.
      ctx.ui.notify("Session creation was cancelled", "warning");
      return { close: true };
    }

    return { close: true };
  };
}

// ---------------------------------------------------------------------------
// Launch preparation (plain data, no side effects)
// ---------------------------------------------------------------------------

export interface PromptMessage {
  template: string;
  message: string;
}

/** Build the routed, quoted prompt message for a given action and row. */
export function builtPromptMessage(
  action: BrowserAction,
  row: BrowserRow,
  cwd: string,
): PromptMessage {
  const { node } = row;
  let template: string;
  let args: string[];

  if (action.kind === "refine" && node.type === "top-level") {
    template = "shemiq-refine-and-split";
    args = [relativePath(cwd, node.path!)];
  } else if (
    action.kind === "implement" &&
    node.status === "refined" &&
    node.path
  ) {
    template = "shemiq-implement-task";
    args = [relativePath(cwd, node.path)];
  } else if (action.kind === "refine" || action.kind === "implement") {
    const parent = row.ancestors[row.ancestors.length - 1];
    if (!parent?.path) {
      throw new Error("Subtask has no parent with a path");
    }
    template =
      action.kind === "refine"
        ? "shemiq-refine-sub-task"
        : "shemiq-implement-task-parent";
    args = [relativePath(cwd, parent.path), node.title];
  } else {
    throw new Error(`Unexpected action kind: ${action.kind}`);
  }

  return {
    template,
    message: [`/${template}`, ...args.map(quotePiArgument)].join(" "),
  };
}

// ---------------------------------------------------------------------------
// Message building with Pi-compatible quoting
// ---------------------------------------------------------------------------

/**
 * Quote a single argument for Pi's template parser.
 * Mirrors the Go quotePiArgument in internal/agent/pi.go:
 * Pi joins adjacent quoted segments but does not interpret backslash escapes.
 */
function quotePiArgument(value: string): string {
  return "'" + value.replace(/'/g, `'"'"'`) + "'";
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function relativePath(cwd: string, absolutePath: string): string {
  return path.relative(cwd, absolutePath);
}

function isTemplateLoaded(pi: ExtensionAPI, templateName: string): boolean {
  const commands = pi.getCommands();
  return commands.some((cmd) => cmd.name === templateName);
}
