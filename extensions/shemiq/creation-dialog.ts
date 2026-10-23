/**
 * Creation dialog: overlay with an optional one-line title input above
 * a multiline description editor (required only without a title).
 * Returns the entered values or null when cancelled.
 */

import type {
  ExtensionCommandContext,
  Theme,
} from "@earendil-works/pi-coding-agent";
import type { Component, TUI } from "@earendil-works/pi-tui";
import {
  Editor,
  Input,
  Key,
  matchesKey,
  truncateToWidth,
  visibleWidth,
} from "@earendil-works/pi-tui";
import type { EditorTheme } from "@earendil-works/pi-tui";
import type { SelectListTheme } from "@earendil-works/pi-tui";
import type { CreationTarget } from "./creation.ts";

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export interface CreationInput {
  /** Optional one-line title; blank defers title selection to the LLM. */
  title: string;
  /** Multiline description; required only when the title is blank. */
  description: string;
}

/**
 * Show the creation dialog as an overlay. Returns the entered title and
 * description, or null if the user cancels.
 */
export async function showCreationDialog(
  ctx: ExtensionCommandContext,
  target: CreationTarget,
): Promise<CreationInput | null> {
  return ctx.ui.custom<CreationInput | null>(
    (tui, theme, _kb, done) =>
      new CreationDialogComponent(tui, theme, target, done),
    { overlay: true },
  );
}

// ---------------------------------------------------------------------------
// Dialog component
// ---------------------------------------------------------------------------

type Focus = "title" | "description";

class CreationDialogComponent implements Component {
  private tui: TUI;
  private theme: Theme;
  private target: CreationTarget;
  private done: (result: CreationInput | null) => void;
  private isDone = false;

  private titleInput: Input;
  private editor: Editor;
  private focus: Focus = "title";
  private errorMessage: string | null = null;

  constructor(
    tui: TUI,
    theme: Theme,
    target: CreationTarget,
    done: (result: CreationInput | null) => void,
  ) {
    this.tui = tui;
    this.theme = theme;
    this.target = target;
    this.done = done;

    this.titleInput = new Input({
      prompt: "Title: ",
      placeholder: "(optional – leave blank to let the agent pick)",
      placeholderStyle: (text) => theme.fg("muted", text),
    });

    const editorTheme: EditorTheme = {
      borderColor: (text) => theme.fg("dim", text),
      selectList: selectListTheme(theme),
    };
    this.editor = new Editor(tui, editorTheme, { paddingX: 1 });
    // Enter inserts a newline; submission uses a distinct binding.
    this.editor.disableSubmit = true;

    this.applyFocus();
  }

  // -- Input handling -------------------------------------------------------

  handleInput(data: string): void {
    if (this.isDone) return;

    // Submit and cancel take precedence over field-local bindings.
    if (matchesKey(data, Key.ctrl("s")) || matchesKey(data, Key.alt("enter"))) {
      this.submit();
      return;
    }
    if (matchesKey(data, "escape")) {
      this.finish(null);
      return;
    }

    // Focus switching between the two fields.
    if (matchesKey(data, "tab") || matchesKey(data, Key.shift("tab"))) {
      this.setFocus(this.focus === "title" ? "description" : "title");
      return;
    }

    if (this.focus === "title") {
      if (matchesKey(data, "enter")) {
        this.setFocus("description");
        return;
      }
      this.titleInput.handleInput(data);
    } else {
      if (matchesKey(data, "enter")) {
        this.editor.insertTextAtCursor("\n");
      } else {
        this.editor.handleInput(data);
      }
    }
    this.tui.requestRender();
  }

  // -- Submission -----------------------------------------------------------

  private submit(): void {
    const title = this.titleInput.getValue().trim();
    const description = this.editor.getExpandedText().trim();
    if (!title && !description) {
      this.errorMessage = "Title or description is required";
      this.tui.requestRender();
      return;
    }
    this.errorMessage = null;
    this.finish({ title, description });
  }

  private finish(result: CreationInput | null): void {
    if (this.isDone) return;
    this.isDone = true;
    this.done(result);
  }

  // -- Focus ------------------------------------------------------------------

  private setFocus(focus: Focus): void {
    if (this.focus === focus) return;
    this.focus = focus;
    this.applyFocus();
    this.tui.requestRender();
  }

  private applyFocus(): void {
    this.titleInput.focused = this.focus === "title";
    this.editor.focused = this.focus === "description";
    this.editor.borderColor = this.editor.focused
      ? (text) => this.theme.fg("accent", text)
      : (text) => this.theme.fg("dim", text);
  }

  // -- Rendering --------------------------------------------------------------

  invalidate(): void {
    this.titleInput.invalidate();
    this.editor.invalidate();
  }

  render(width: number): string[] {
    const t = this.theme;
    const lines: string[] = [];
    const innerWidth = Math.max(20, width - 4);

    // Header with target context
    lines.push("");
    lines.push(
      padCenter(
        t.bold(t.fg("accent", ` ${creationHeader(this.target)} `)),
        innerWidth,
      ),
    );
    if (this.target.kind === "subtask") {
      const context = `under "${this.target.parentTitle}" after "${this.target.afterTitle}"`;
      lines.push(padCenter(t.fg("muted", truncateToWidth(context, innerWidth)), innerWidth));
    }
    lines.push(
      padCenter(t.fg("dim", "─".repeat(Math.min(innerWidth, 60))), innerWidth),
    );
    lines.push("");

    // Title input
    lines.push(...this.titleInput.render(innerWidth));
    lines.push("");

    // Description editor
    lines.push(...this.editor.render(innerWidth));

    // Inline error
    if (this.errorMessage) {
      lines.push("");
      lines.push(padCenter(t.fg("error", this.errorMessage), innerWidth));
    }

    // Legend
    lines.push("");
    const legend = [
      `${t.bold("Tab")} switch field`,
      `${t.bold("Ctrl+S")}/${t.bold("Alt+Enter")} create`,
      `${t.bold("Esc")} cancel`,
    ].join(t.fg("dim", "  │  "));
    lines.push(padCenter(legend, innerWidth));
    lines.push("");

    return lines;
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function creationHeader(target: CreationTarget): string {
  return target.kind === "top-level" ? "New task" : "New subtask";
}

function selectListTheme(theme: Theme): SelectListTheme {
  return {
    selectedPrefix: (text) => theme.fg("accent", text),
    selectedText: (text) => theme.fg("accent", text),
    description: (text) => theme.fg("muted", text),
    scrollInfo: (text) => theme.fg("dim", text),
    noMatch: (text) => theme.fg("warning", text),
  };
}

function padCenter(content: string, width: number): string {
  const len = visibleWidth(content);
  if (len >= width) return truncateToWidth(content, width);
  const pad = width - len;
  const left = Math.floor(pad / 2);
  return " ".repeat(left) + content + " ".repeat(pad - left);
}
