/**
 * Task editing: connects the browser's "edit" action to `$EDITOR`,
 * handing the terminal to the external editor process for its duration.
 */

import { spawn } from "child_process";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { ActionKind } from "./controller.ts";
import type { ActionHandler } from "./browser.ts";

// ---------------------------------------------------------------------------
// Public setup
// ---------------------------------------------------------------------------

/**
 * Register the "edit" action handler on the browser.
 * Called once from the extension entry point.
 */
export function registerEditHandler(
  _pi: ExtensionAPI,
  register: (kind: ActionKind, handler: ActionHandler) => void,
): void {
  register("edit", async (_action, row, ctx, extUi) => {
    const nodePath = row?.node.path;
    if (!nodePath) {
      throw new Error("Cannot edit: task has no document path");
    }

    // Both checks happen before any terminal handoff.
    const editor = process.env.EDITOR?.trim();
    if (!editor) {
      throw new Error("Cannot edit: $EDITOR is not set");
    }

    const argv = [...parseEditorCommand(editor), nodePath];
    const outcome = await extUi.runInTerminal(() => runEditor(ctx.cwd, argv));

    // Refresh even after an unsuccessful exit: the editor may have
    // saved changes before failing.
    if (outcome.signal) {
      return {
        refresh: true,
        notify: `Editor terminated by ${outcome.signal}`,
      };
    }
    if (outcome.code !== 0) {
      return {
        refresh: true,
        notify: `Editor exited with status ${outcome.code}`,
      };
    }
    return { refresh: true };
  });
}

// ---------------------------------------------------------------------------
// Editor command handling
// ---------------------------------------------------------------------------

export interface EditorOutcome {
  code: number | null;
  signal: NodeJS.Signals | null;
}

/**
 * Run the editor with inherited terminal input/output and wait for it.
 * Launch failures reject; exits (including nonzero) resolve.
 */
function runEditor(cwd: string, argv: string[]): Promise<EditorOutcome> {
  return new Promise((resolve, reject) => {
    const child = spawn(argv[0], argv.slice(1), {
      cwd,
      stdio: "inherit",
    });
    let settled = false;
    child.on("error", (err: NodeJS.ErrnoException) => {
      if (settled) return;
      settled = true;
      const reason =
        err.code === "ENOENT"
          ? `executable not found: ${argv[0]}`
          : err.message;
      reject(new Error(`Failed to launch editor: ${reason}`));
    });
    child.on("close", (code, signal) => {
      if (settled) return;
      settled = true;
      resolve({ code, signal });
    });
  });
}

/**
 * Split `$EDITOR` into an executable plus arguments without shell
 * evaluation. Single- and double-quoted segments group words and may
 * contain spaces; quotes are removed.
 */
export function parseEditorCommand(editor: string): string[] {
  const words: string[] = [];
  let current = "";
  let hasWord = false;
  let quote: '"' | "'" | null = null;

  for (const ch of editor) {
    if (quote) {
      if (ch === quote) {
        quote = null;
      } else {
        current += ch;
      }
    } else if (ch === '"' || ch === "'") {
      quote = ch;
      hasWord = true;
    } else if (/\s/.test(ch)) {
      if (hasWord) {
        words.push(current);
        current = "";
        hasWord = false;
      }
    } else {
      current += ch;
      hasWord = true;
    }
  }

  if (quote) {
    throw new Error("$EDITOR has an unterminated quote");
  }
  if (hasWord) {
    words.push(current);
  }
  if (words.length === 0) {
    throw new Error("$EDITOR is empty");
  }
  return words;
}
