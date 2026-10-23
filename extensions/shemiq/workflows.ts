/**
 * Workflow handlers: connect refinement and implementation actions
 * from the browser to Pi's prompt templates, with a launcher dialog
 * for model/session selection.
 */

import type {
  ExtensionAPI,
  ExtensionCommandContext,
} from "@earendil-works/pi-coding-agent";
import type { ActionKind, BrowserAction, BrowserRow } from "./controller.ts";
import type { ActionHandler } from "./browser.ts";
import type { ExtUi } from "./ext-ui.ts";
import type { LaunchSettings } from "./launcher.ts";
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
  register: (kind: ActionKind, handler: ActionHandler) => void,
): void {
  register("refine", createPromptDispatcher(pi));
  register("implement", createPromptDispatcher(pi));

  // Session replacement recreates this handler with the destination's pi API.
  pi.registerCommand("shemiq-launch", {
    description: "Internal Shemiq destination-session prompt dispatch",
    handler: async (args, ctx) => {
      try {
        const launch = parseLaunch(args);
        await applyModelAndPrompt(pi, ctx, launch);
      } catch (err) {
        ctx.ui.notify(`Shemiq launch failed: ${errorMessage(err)}`, "error");
      }
    },
  });
}

// ---------------------------------------------------------------------------
// Unified handler (refine + implement share structure)
// ---------------------------------------------------------------------------

function createPromptDispatcher(pi: ExtensionAPI) {
  return async (
    action: BrowserAction,
    row: BrowserRow | null,
    ctx: ExtensionCommandContext,
    extUi: ExtUi,
  ): Promise<{ launch?: PromptLaunch }> => {
    if (!row) {
      throw new Error("No task selected");
    }

    const route = builtPromptMessage(action, row, ctx.cwd);

    // Check template is loaded before showing the launcher.
    requireTemplate(pi, route.template);

    // Show the launcher dialog to select model and session mode.
    // Dynamic import avoids pulling pi-tui into pure-function test imports.
    const { showLauncher } = await import("./launcher.ts");
    const actionLabel = action.kind === "refine" ? "Refine" : "Implement";
    const settings = await showLauncher(
      ctx,
      extUi,
      `${actionLabel}: ${row.node.title}`,
    );

    if (!settings) {
      // User cancelled – return to browser with selection preserved.
      return {};
    }

    // Retain pre-launch failures in the browser. No session-bound side effects.
    requireTemplate(pi, route.template);
    await requireAvailableModel(ctx, settings);
    return { launch: { ...route, ...settings } };
  };
}

// ---------------------------------------------------------------------------
// Dispatch after the browser and dialog have been disposed
// ---------------------------------------------------------------------------

export interface PromptLaunch extends PromptMessage, LaunchSettings {}

/** Called only after openBrowser resolves; never from component input handlers. */
export async function dispatchLaunch(
  pi: ExtensionAPI,
  ctx: ExtensionCommandContext,
  launch: PromptLaunch,
): Promise<void> {
  if (launch.session === "current") {
    await applyModelAndPrompt(pi, ctx, launch);
    return;
  }

  // Only serialized launch data crosses replacement, never a captured pi API
  // or model object. The command resolves against the new extension runtime.
  const command = `/shemiq-launch ${JSON.stringify(launch)}`;
  const { cancelled } = await ctx.newSession({
    withSession: async (newCtx) => {
      await newCtx.sendUserMessage(command, { expandPromptTemplates: true });
    },
  });

  // Cancellation leaves the original context active. On success, do not touch it.
  // Replacement failures propagate to Pi's command error handling, not old UI.
  if (cancelled) ctx.ui.notify("Session creation was cancelled", "warning");
}

async function applyModelAndPrompt(
  pi: ExtensionAPI,
  ctx: ExtensionCommandContext,
  launch: PromptLaunch,
): Promise<void> {
  requireTemplate(pi, launch.template);
  const model = await requireAvailableModel(ctx, launch);
  if (!(await pi.setModel(model))) {
    throw new Error(
      `Failed to set model ${launch.provider}/${launch.modelId}. Check provider authentication.`,
    );
  }
  pi.sendUserMessage(launch.message, { expandPromptTemplates: true });
}

function parseLaunch(args: string): PromptLaunch {
  const data = JSON.parse(args);
  if (
    !data ||
    typeof data.provider !== "string" ||
    typeof data.modelId !== "string" ||
    typeof data.template !== "string" ||
    typeof data.message !== "string" ||
    (data.session !== "fresh" && data.session !== "current") ||
    !data.template.startsWith("shemiq-") ||
    !data.message.startsWith(`/${data.template} `)
  ) {
    throw new Error("Invalid Shemiq launch data. Open /shemiq and try again.");
  }
  return data;
}

export async function requireAvailableModel(
  ctx: ExtensionCommandContext,
  settings: Pick<LaunchSettings, "provider" | "modelId">,
) {
  const models = await ctx.modelRegistry.getAvailable();
  const model = models.find(
    (candidate) =>
      candidate.provider === settings.provider && candidate.id === settings.modelId,
  );
  if (!model) {
    throw new Error(
      `Model ${settings.provider}/${settings.modelId} is unavailable. Check provider authentication.`,
    );
  }
  return model;
}

/** Check a prompt template is loaded; throws an actionable error otherwise. */
export function requireTemplate(pi: ExtensionAPI, template: string): void {
  if (!isTemplateLoaded(pi, template)) {
    throw new Error(
      `Prompt template "/${template}" is not loaded. ` +
        `Load the Shemiq prompt templates and run /reload.`,
    );
  }
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
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
export function quotePiArgument(value: string): string {
  return "'" + value.replace(/'/g, `'"'"'`) + "'";
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function relativePath(cwd: string, absolutePath: string): string {
  return path.relative(cwd, absolutePath);
}

function isTemplateLoaded(pi: ExtensionAPI, templateName: string): boolean {
  const commands = pi.getCommands();
  return commands.some((cmd) => cmd.name === templateName);
}
