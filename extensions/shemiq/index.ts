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
import { openBrowser, registerActionHandler } from "./browser.ts";
import { registerWorkflowHandlers } from "./workflows.ts";

export default function shemiqExtension(pi: ExtensionAPI) {
  registerWorkflowHandlers(pi, registerActionHandler);
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
      await openBrowser(ctx);
    },
  });

  pi.registerShortcut(Key.alt("s"), {
    description: "Open Shemiq task browser",
    handler: async () => {
      pi.sendUserMessage("/shemiq", { expandPromptTemplates: true });
    },
  });
}
