/**
 * Immediate archiving: connects the browser's "archive" action to the CLI.
 */

import { execFile } from "child_process";
import * as path from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { ActionKind } from "./controller.ts";
import type { ActionHandler } from "./browser.ts";
import { relativePath } from "./workflows.ts";

// ---------------------------------------------------------------------------
// Public setup
// ---------------------------------------------------------------------------

/**
 * Register the "archive" action handler on the browser.
 * Called once from the extension entry point.
 */
export function registerArchiveHandler(
  _pi: ExtensionAPI,
  register: (kind: ActionKind, handler: ActionHandler) => void,
): void {
  register("archive", async (_action, row, ctx) => {
    const nodePath = row?.node.path;
    if (!nodePath) {
      throw new Error("Cannot archive: task has no document path");
    }

    const target = relativePath(ctx.cwd, path.dirname(nodePath));
    await runShemiqArchive(ctx.cwd, target);
    return { refresh: true, notify: `Archived ${target}` };
  });
}

// ---------------------------------------------------------------------------
// Direct CLI execution (same style as creation.ts)
// ---------------------------------------------------------------------------

function runShemiqArchive(cwd: string, target: string): Promise<void> {
  return new Promise((resolve, reject) => {
    execFile("shemiq", ["archive", target], { cwd }, (err, _stdout, stderr) => {
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
        reject(new Error(`shemiq archive failed: ${detail}`));
      } else {
        resolve();
      }
    });
  });
}
