/**
 * Browser screen: renders the task hierarchy, handles keyboard navigation,
 * and dispatches actions.
 */

import type {
  ExtensionCommandContext,
  Theme,
} from "@earendil-works/pi-coding-agent";
import type { Component, TUI } from "@earendil-works/pi-tui";
import {
  matchesKey,
  truncateToWidth,
  visibleWidth,
} from "@earendil-works/pi-tui";
import {
  type ActionKind,
  type BrowserAction,
  type BrowserRow,
  type TreeLoadResult,
  actionBindings,
  getEligibleActions,
  getLegendEntries,
  loadTree,
  resolveSelection,
} from "./controller.ts";

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

export async function openBrowser(ctx: ExtensionCommandContext): Promise<void> {
  const cwd = ctx.cwd;
  const loadResult = await loadTree(cwd);

  await ctx.ui.custom<void>((tui, theme, _kb, done) => {
    return new BrowserComponent(tui, theme, cwd, loadResult, done, ctx);
  });
}

// ---------------------------------------------------------------------------
// Action handlers (placeholders for follow-up tasks)
// ---------------------------------------------------------------------------

export type ActionHandler = (
  action: BrowserAction,
  row: BrowserRow | null,
  ctx: ExtensionCommandContext,
) => Promise<{ close?: boolean; refresh?: boolean }>;

const actionHandlers: Map<ActionKind, ActionHandler> = new Map();

/**
 * Register an action handler. Used by follow-up tasks to connect workflows.
 */
export function registerActionHandler(
  kind: ActionKind,
  handler: ActionHandler,
): void {
  actionHandlers.set(kind, handler);
}

// ---------------------------------------------------------------------------
// Browser component
// ---------------------------------------------------------------------------

class BrowserComponent implements Component {
  private tui: TUI;
  private theme: Theme;
  private cwd: string;
  private done: (value: void) => void;
  private ctx: ExtensionCommandContext;

  private rows: BrowserRow[] = [];
  private selectedIndex = 0;
  private errorMessage: string | null = null;
  private isLoading = false;
  private isClosed = false;
  private notificationMessage: string | null = null;
  private notificationTimeout: ReturnType<typeof setTimeout> | null = null;

  // Rendering cache
  private cachedLines: string[] = [];
  private cachedWidth = 0;
  private version = 0;
  private cachedVersion = -1;

  // Scrolling
  private scrollOffset = 0;

  constructor(
    tui: TUI,
    theme: Theme,
    cwd: string,
    loadResult: TreeLoadResult,
    done: (value: void) => void,
    ctx: ExtensionCommandContext,
  ) {
    this.tui = tui;
    this.theme = theme;
    this.cwd = cwd;
    this.done = done;
    this.ctx = ctx;
    this.updateTreeState(loadResult);
  }

  // -- State management -----------------------------------------------------

  private updateTreeState(loadResult: TreeLoadResult): void {
    if (loadResult.ok) {
      const prevId = this.rows[this.selectedIndex]?.id ?? null;
      this.rows = loadResult.rows;
      this.selectedIndex = resolveSelection(
        this.rows,
        prevId,
        this.selectedIndex,
      );
      this.errorMessage = null;
    } else {
      this.rows = [];
      this.selectedIndex = -1;
      this.errorMessage = loadResult.message;
    }
    this.scrollOffset = 0;
    this.version++;
  }

  private async refresh(): Promise<void> {
    if (this.isLoading || this.isClosed) return;
    this.isLoading = true;
    this.version++;
    this.tui.requestRender();

    const loadResult = await loadTree(this.cwd);
    if (this.isClosed) return;

    this.isLoading = false;
    this.updateTreeState(loadResult);
    this.tui.requestRender();
  }

  private showNotification(msg: string): void {
    this.notificationMessage = msg;
    if (this.notificationTimeout) clearTimeout(this.notificationTimeout);
    this.notificationTimeout = setTimeout(() => {
      this.notificationMessage = null;
      this.version++;
      this.tui.requestRender();
    }, 3000);
    this.version++;
    this.tui.requestRender();
  }

  private close(): void {
    this.isClosed = true;
    if (this.notificationTimeout) clearTimeout(this.notificationTimeout);
    this.done(undefined);
  }

  // -- Input handling -------------------------------------------------------

  handleInput(data: string): void {
    if (this.isClosed) return;

    // Close
    if (matchesKey(data, "escape") || data === "q") {
      this.close();
      return;
    }

    // Refresh
    if (matchesKey(data, "ctrl+r")) {
      this.refresh();
      return;
    }

    // Navigation
    if (matchesKey(data, "up") || data === "k") {
      this.moveSelection(-1);
      return;
    }
    if (matchesKey(data, "down") || data === "j") {
      this.moveSelection(1);
      return;
    }
    if (matchesKey(data, "pageUp")) {
      this.moveSelection(-10);
      return;
    }
    if (matchesKey(data, "pageDown")) {
      this.moveSelection(10);
      return;
    }
    if (matchesKey(data, "home")) {
      this.moveSelectionTo(0);
      return;
    }
    if (matchesKey(data, "end")) {
      this.moveSelectionTo(this.rows.length - 1);
      return;
    }

    // Actions share bindings with the legend.
    const binding = actionBindings.find(({ key }) => key === data);
    if (binding) {
      void this.dispatchAction(binding.action);
    }
  }

  private moveSelection(delta: number): void {
    if (this.rows.length === 0) return;
    const newIdx = Math.max(
      0,
      Math.min(this.rows.length - 1, this.selectedIndex + delta),
    );
    if (newIdx !== this.selectedIndex) {
      this.selectedIndex = newIdx;
      this.version++;
      this.tui.requestRender();
    }
  }

  private moveSelectionTo(index: number): void {
    if (this.rows.length === 0) return;
    const newIdx = Math.max(0, Math.min(this.rows.length - 1, index));
    if (newIdx !== this.selectedIndex) {
      this.selectedIndex = newIdx;
      this.version++;
      this.tui.requestRender();
    }
  }

  private async dispatchAction(action: BrowserAction): Promise<void> {
    const { kind } = action;
    const row = this.rows[this.selectedIndex] ?? null;

    // "new" is global, doesn't need a selected row
    if (kind !== "new" && !row) {
      this.showNotification("No task selected");
      return;
    }

    // Check eligibility for row-specific actions
    if (kind !== "new" && row) {
      const eligible = getEligibleActions(row);
      if (!eligible.has(kind)) {
        this.showNotification(`Action "${kind}" is not available for this task`);
        return;
      }
    }

    // Check for connected handler
    const handler = actionHandlers.get(kind);
    if (!handler) {
      this.showNotification(`"${kind}" is not implemented yet`);
      return;
    }

    try {
      const result = await handler(action, row, this.ctx);
      if (this.isClosed) return;
      if (result.close) {
        this.close();
        return;
      }
      if (result.refresh) {
        await this.refresh();
      }
    } catch (err: unknown) {
      if (this.isClosed) return;
      const msg = err instanceof Error ? err.message : String(err);
      this.showNotification(`Action failed: ${msg}`);
    }
  }

  // -- Rendering ------------------------------------------------------------

  invalidate(): void {
    this.cachedWidth = 0;
    this.cachedVersion = -1;
  }

  render(width: number): string[] {
    if (width === this.cachedWidth && this.cachedVersion === this.version) {
      return this.cachedLines;
    }

    const t = this.theme;
    const dim = (s: string) => t.fg("dim", s);
    const bold = (s: string) => t.bold(s);
    const accent = (s: string) => t.fg("accent", s);
    const warning = (s: string) => t.fg("warning", s);
    const error = (s: string) => t.fg("error", s);
    const muted = (s: string) => t.fg("muted", s);

    const lines: string[] = [];

    // Header
    const titleText = " Shemiq Tasks ";
    const titleLen = visibleWidth(titleText);
    const borderLen = Math.max(0, width - titleLen);
    const leftBorder = Math.floor(borderLen / 2);
    const rightBorder = borderLen - leftBorder;
    lines.push(
      dim("─".repeat(leftBorder)) +
        bold(accent(titleText)) +
        dim("─".repeat(rightBorder)),
    );
    lines.push("");

    // Loading indicator
    if (this.isLoading) {
      lines.push(centerPad(muted("Loading..."), width));
      lines.push("");
    }

    // Error state
    if (this.errorMessage) {
      lines.push(centerPad(error("⚠ Error loading tasks"), width));
      lines.push("");
      // Word-wrap error message
      const errLines = wrapText(this.errorMessage, Math.min(width - 4, 72));
      for (const el of errLines) {
        lines.push(centerPad(muted(el), width));
      }
      lines.push("");
    } else if (this.rows.length === 0 && !this.isLoading) {
      // Empty state
      lines.push(centerPad(muted("No tasks found"), width));
      lines.push(
        centerPad(
          muted('Press "n" to create a new task, Ctrl+R to refresh'),
          width,
        ),
      );
      lines.push("");
    } else if (!this.isLoading) {
      // Task list
      const availableHeight = Math.max(5, (this.tui.terminal?.rows ?? 24) - 10);
      this.adjustScroll(availableHeight);

      const visibleStart = this.scrollOffset;
      const visibleEnd = Math.min(
        this.rows.length,
        this.scrollOffset + availableHeight,
      );

      // Scroll indicator top
      if (visibleStart > 0) {
        lines.push(centerPad(dim(`▲ ${visibleStart} more`), width));
      }

      for (let i = visibleStart; i < visibleEnd; i++) {
        const row = this.rows[i];
        const isSelected = i === this.selectedIndex;
        lines.push(this.renderRow(row, isSelected, width));
      }

      // Scroll indicator bottom
      const remaining = this.rows.length - visibleEnd;
      if (remaining > 0) {
        lines.push(centerPad(dim(`▼ ${remaining} more`), width));
      }

      lines.push("");
    }

    // Notification
    if (this.notificationMessage) {
      lines.push(centerPad(warning(this.notificationMessage), width));
      lines.push("");
    }

    // Legend
    const selectedRow = this.rows[this.selectedIndex] ?? null;
    const legend = this.renderLegend(selectedRow, width);
    lines.push(legend);

    // Footer with navigation help
    const navHelp = `${bold("↑↓")}/${bold("jk")} navigate  ${dim("│")}  ${bold("Ctrl+R")} refresh  ${dim("│")}  ${bold("Esc")}/${bold("q")} close`;
    lines.push(centerPad(navHelp, width));

    lines.push("");
    lines.push(dim("─".repeat(width)));

    this.cachedLines = lines;
    this.cachedWidth = width;
    this.cachedVersion = this.version;
    return lines;
  }

  private renderRow(row: BrowserRow, isSelected: boolean, width: number): string {
    const t = this.theme;
    const indent = "  ".repeat(row.depth);
    const marker = isSelected ? t.fg("accent", "❯ ") : "  ";
    const statusLabel = this.formatStatus(row.node.status);
    const typeIndicator =
      row.node.type === "top-level"
        ? t.bold("◆ ")
        : "  ";

    const title = row.node.title;
    // Reserve space for indent + marker + type + status + padding
    const fixedLen =
      visibleWidth(indent) +
      visibleWidth(marker) +
      visibleWidth(typeIndicator) +
      visibleWidth(statusLabel) +
      3;
    const availableTitleWidth = Math.max(10, width - fixedLen);
    const truncTitle = truncateToWidth(title, availableTitleWidth);

    return `${indent}${marker}${typeIndicator}${truncTitle} ${statusLabel}`;
  }

  private formatStatus(status: string): string {
    const t = this.theme;
    switch (status) {
      case "new":
        return t.fg("muted", "[new]");
      case "refined":
        return t.fg("accent", "[refined]");
      case "done":
        return t.fg("success", "[done]");
      default:
        return t.fg("dim", `[${status}]`);
    }
  }

  private renderLegend(selectedRow: BrowserRow | null, width: number): string {
    const t = this.theme;
    const entries = getLegendEntries(selectedRow);
    const parts = entries.map(
      (e) => `${t.bold(e.key)} ${t.fg("muted", e.label)}`,
    );
    const legend = parts.join(t.fg("dim", "  │  "));
    return centerPad(legend, width);
  }

  private adjustScroll(viewportHeight: number): void {
    if (this.selectedIndex < this.scrollOffset) {
      this.scrollOffset = this.selectedIndex;
    } else if (this.selectedIndex >= this.scrollOffset + viewportHeight) {
      this.scrollOffset = this.selectedIndex - viewportHeight + 1;
    }
    // Clamp
    this.scrollOffset = Math.max(
      0,
      Math.min(this.scrollOffset, Math.max(0, this.rows.length - viewportHeight)),
    );
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function centerPad(content: string, width: number): string {
  const contentLen = visibleWidth(content);
  if (contentLen >= width) return truncateToWidth(content, width);
  const pad = width - contentLen;
  const left = Math.floor(pad / 2);
  return " ".repeat(left) + content + " ".repeat(pad - left);
}

function wrapText(text: string, maxWidth: number): string[] {
  const words = text.split(/\s+/);
  const lines: string[] = [];
  let current = "";
  for (const word of words) {
    if (current.length + word.length + 1 > maxWidth && current.length > 0) {
      lines.push(current);
      current = word;
    } else {
      current = current ? `${current} ${word}` : word;
    }
  }
  if (current) lines.push(current);
  return lines;
}
