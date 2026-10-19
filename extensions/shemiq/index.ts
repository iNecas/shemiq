/**
 * Shemiq task browser extension for Pi.
 *
 * Provides `/shemiq` command and Alt+S shortcut to open a keyboard-driven
 * task browser. Loads the hierarchy from `shemiq tree --json`, displays it
 * with selection, status labels, and an action legend.
 *
 * This file is the entry point; the browser component and controller logic
 * are in separate modules for follow-up task extensibility.
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Key } from "@earendil-works/pi-tui";
import { openBrowser, type ActionHandler } from "./browser.ts";
import type { ActionKind } from "./controller.ts";
import { registerCreationHandler } from "./creation.ts";
import { dispatchLaunch, registerWorkflowHandlers } from "./workflows.ts";

export default function shemiqExtension(pi: ExtensionAPI) {
  // Runtime-local handlers must not retain a pi API across reload/replacement.
  const actionHandlers = new Map<ActionKind, ActionHandler>();
  registerWorkflowHandlers(pi, (kind, handler) => actionHandlers.set(kind, handler));
  registerCreationHandler(pi, (kind, handler) => actionHandlers.set(kind, handler));
  pi.registerCommand("shemiq", {
    description: "Open the Shemiq task browser",
    handler: async (_args, ctx) => {
      if (ctx.mode !== "tui") {
        ctx.ui.notify("Shemiq browser requires interactive mode", "error");
        return;
      }
      if (!ctx.isIdle()) {
        ctx.ui.notify("Cannot open browser while Pi is busy", "warning");
        return;
      }
      const launch = await openBrowser(ctx, actionHandlers);
      if (launch) await dispatchLaunch(pi, ctx, launch);
    },
  });

  pi.registerShortcut(Key.alt("s"), {
    description: "Open Shemiq task browser",
    handler: async () => {
      pi.sendUserMessage("/shemiq", { expandPromptTemplates: true });
    },
  });
}
