/**
 * Prompt launcher dialog: model selection and session mode before
 * dispatching a refinement or implementation workflow.
 */

import type {
  ExtensionCommandContext,
  Theme,
} from "@earendil-works/pi-coding-agent";
import type { Api, Model } from "@earendil-works/pi-ai";
import type { Component, TUI } from "@earendil-works/pi-tui";
import {
  matchesKey,
  SelectList,
  visibleWidth,
  truncateToWidth,
} from "@earendil-works/pi-tui";
import type { SelectItem, SelectListTheme } from "@earendil-works/pi-tui";

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export interface LaunchSettings {
  provider: string;
  modelId: string;
  session: "fresh" | "current";
}

/**
 * Show the launcher dialog as an overlay. Returns selected settings or null
 * if the user cancels.
 */
export async function showLauncher(
  ctx: ExtensionCommandContext,
  actionLabel: string,
): Promise<LaunchSettings | null> {
  const models = await getAvailableChatModels(ctx);

  if (models.length === 0) {
    return ctx.ui.custom<LaunchSettings | null>(
      (tui, theme, _kb, done) =>
        new EmptyModelsComponent(tui, theme, actionLabel, done),
      { overlay: true },
    );
  }

  const currentModel = ctx.model;

  return ctx.ui.custom<LaunchSettings | null>(
    (tui, theme, _kb, done) =>
      new LauncherComponent(
        tui,
        theme,
        actionLabel,
        models,
        currentModel,
        done,
      ),
    { overlay: true },
  );
}

// ---------------------------------------------------------------------------
// Model discovery
// ---------------------------------------------------------------------------

async function getAvailableChatModels(
  ctx: ExtensionCommandContext,
): Promise<Model<Api>[]> {
  const models = await ctx.modelRegistry.getAvailable();
  if (ctx.scopedModels.length === 0) return [...models];

  return ctx.scopedModels.flatMap(({ model }) => {
    const available = models.find(
      (candidate) =>
        candidate.provider === model.provider && candidate.id === model.id,
    );
    return available ? [available] : [];
  });
}

/** Build a display label for a model: "provider / model-id". */
export function modelLabel(model: Model<Api>): string {
  return `${model.provider} / ${model.id}`;
}

/** Find the index of a model matching the current one. */
function findCurrentModelIndex(
  models: Model<Api>[],
  current: Model<Api> | undefined,
): number {
  if (!current) return 0;
  const idx = models.findIndex(
    (m) => m.provider === current.provider && m.id === current.id,
  );
  return idx >= 0 ? idx : 0;
}

// ---------------------------------------------------------------------------
// Launcher component
// ---------------------------------------------------------------------------

class LauncherComponent implements Component {
  private tui: TUI;
  private theme: Theme;
  private actionLabel: string;
  private models: Model<Api>[];
  private done: (result: LaunchSettings | null) => void;
  private selectList: SelectList;
  private useCurrentSession = false;
  private confirmed = false;

  constructor(
    tui: TUI,
    theme: Theme,
    actionLabel: string,
    models: Model<Api>[],
    currentModel: Model<Api> | undefined,
    done: (result: LaunchSettings | null) => void,
  ) {
    this.tui = tui;
    this.theme = theme;
    this.actionLabel = actionLabel;
    this.models = models;
    this.done = done;

    const items: SelectItem[] = models.map((m) => ({
      value: `${m.provider}:${m.id}`,
      label: modelLabel(m),
      description: m.name !== m.id ? m.name : undefined,
    }));

    const selectTheme: SelectListTheme = {
      selectedPrefix: (text) => theme.fg("accent", text),
      selectedText: (text) => theme.fg("accent", text),
      description: (text) => theme.fg("muted", text),
      scrollInfo: (text) => theme.fg("dim", text),
      noMatch: (text) => theme.fg("warning", text),
    };

    const maxVisible = Math.min(items.length, 15);
    this.selectList = new SelectList(items, maxVisible, selectTheme);

    const initialIndex = findCurrentModelIndex(models, currentModel);
    this.selectList.setSelectedIndex(initialIndex);

    this.selectList.onSelect = (item) => this.confirm(item);
    this.selectList.onCancel = () => this.cancel();
  }

  private confirm(item: SelectItem): void {
    if (this.confirmed) return;
    this.confirmed = true;
    const model = this.models.find(
      (m) => `${m.provider}:${m.id}` === item.value,
    );
    if (!model) {
      this.done(null);
      return;
    }
    this.done({
      provider: model.provider,
      modelId: model.id,
      session: this.useCurrentSession ? "current" : "fresh",
    });
  }

  private cancel(): void {
    this.done(null);
  }

  handleInput(data: string): void {
    if (this.confirmed) return;

    // Toggle current-session with 'c' without changing focus
    if (data === "c") {
      this.useCurrentSession = !this.useCurrentSession;
      this.tui.requestRender();
      return;
    }

    // SelectList handles Enter, Esc, and navigation
    this.selectList.handleInput(data);
    this.tui.requestRender();
  }

  invalidate(): void {
    this.selectList.invalidate();
  }

  render(width: number): string[] {
    const t = this.theme;
    const lines: string[] = [];
    const innerWidth = Math.max(20, width - 4);

    // Header
    lines.push("");
    lines.push(
      padCenter(
        t.bold(t.fg("accent", ` ${this.actionLabel} – Select Model `)),
        innerWidth,
      ),
    );
    lines.push(padCenter(t.fg("dim", "─".repeat(Math.min(innerWidth, 50))), innerWidth));
    lines.push("");

    // Session mode checkbox
    const checkmark = this.useCurrentSession ? "✓" : " ";
    const sessionLine = `[${checkmark}] Use current session`;
    lines.push(
      padCenter(
        this.useCurrentSession
          ? t.fg("accent", sessionLine)
          : t.fg("muted", sessionLine),
        innerWidth,
      ),
    );
    lines.push("");

    // Model list
    const listLines = this.selectList.render(innerWidth);
    lines.push(...listLines);
    lines.push("");

    // Legend
    const legend = [
      `${t.bold("↑↓")} navigate`,
      `${t.bold("c")} toggle session`,
      `${t.bold("Enter")} launch`,
      `${t.bold("Esc")} cancel`,
    ].join(t.fg("dim", "  │  "));
    lines.push(padCenter(legend, innerWidth));
    lines.push("");

    return lines;
  }
}

// ---------------------------------------------------------------------------
// Empty models component
// ---------------------------------------------------------------------------

class EmptyModelsComponent implements Component {
  private tui: TUI;
  private theme: Theme;
  private actionLabel: string;
  private done: (result: null) => void;

  constructor(
    tui: TUI,
    theme: Theme,
    actionLabel: string,
    done: (result: null) => void,
  ) {
    this.tui = tui;
    this.theme = theme;
    this.actionLabel = actionLabel;
    this.done = done;
  }

  handleInput(data: string): void {
    if (matchesKey(data, "escape") || matchesKey(data, "enter")) {
      this.done(null);
    }
  }

  invalidate(): void {}

  render(width: number): string[] {
    const t = this.theme;
    const lines: string[] = [];
    lines.push("");
    lines.push(
      padCenter(t.bold(t.fg("accent", ` ${this.actionLabel} `)), width),
    );
    lines.push("");
    lines.push(
      padCenter(
        t.fg("error", "No authenticated chat models available."),
        width,
      ),
    );
    lines.push(
      padCenter(
        t.fg("muted", "Configure API keys or authenticate a provider, then try again."),
        width,
      ),
    );
    lines.push("");
    lines.push(
      padCenter(t.fg("dim", `Press ${t.bold("Esc")} to return`), width),
    );
    lines.push("");
    return lines;
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function padCenter(content: string, width: number): string {
  const len = visibleWidth(content);
  if (len >= width) return truncateToWidth(content, width);
  const pad = width - len;
  const left = Math.floor(pad / 2);
  return " ".repeat(left) + content + " ".repeat(pad - left);
}
