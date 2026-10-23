/**
 * Ext UI host: owns the single non-overlay custom UI interaction for one
 * browser visit and switches between the persistent browser screen and
 * transient dialogs (launcher, creation dialog).
 *
 * The host is what Pi mounts; it renders and forwards input to whichever
 * component is active. Dialogs finish through their `finish` callback,
 * which disposes the dialog and returns to the retained browser without
 * resetting its selection or scroll position.
 */

import type { Theme } from "@earendil-works/pi-coding-agent";
import type { Component, TUI } from "@earendil-works/pi-tui";

// ---------------------------------------------------------------------------
// Ext UI interface (handed to browser action handlers)
// ---------------------------------------------------------------------------

/**
 * Open a dialog in place of the browser for the duration of the
 * interaction. The promise resolves with the dialog's result once it
 * finishes, or with null when the whole interaction was closed while the
 * dialog was still open.
 */
export interface ExtUi {
  showDialog<T>(
    create: (
      finish: (result: T) => void,
      tui: TUI,
      theme: Theme,
    ) => Component,
  ): Promise<T | null>;
}

// ---------------------------------------------------------------------------
// Host component
// ---------------------------------------------------------------------------

export class ExtUiHost implements Component, ExtUi {
  private tui: TUI;
  private theme: Theme;
  private done: (value: unknown) => void;

  /** Persistent home screen (the browser); kept mounted across dialogs. */
  private home: Component | null = null;
  /** Currently displayed component; the home while no dialog is open. */
  private active: Component | null = null;
  /** Promise resolvers of still-open dialogs, settled on host close. */
  private pendingResolves = new Set<(result: null) => void>();
  private isClosed = false;

  constructor(tui: TUI, theme: Theme, done: (value: unknown) => void) {
    this.tui = tui;
    this.theme = theme;
    this.done = done;
  }

  /** Register the persistent home screen returned by the Pi factory. */
  setHome(component: Component): void {
    this.home = component;
    this.active ??= component;
  }

  /** Finish the whole interaction; Pi restores its editor afterwards. */
  close(result: unknown): void {
    if (this.isClosed) return;
    this.isClosed = true;
    this.settlePendingDialogs();
    this.disposeComponents();
    this.done(result);
  }

  // -- ExtUi ----------------------------------------------------------------

  showDialog<T>(
    create: (
      finish: (result: T) => void,
      tui: TUI,
      theme: Theme,
    ) => Component,
  ): Promise<T | null> {
    return new Promise((resolve) => {
      if (this.isClosed) {
        resolve(null);
        return;
      }

      const settle = resolve as (result: T | null) => void;
      let dialog: Component;
      const finish = (result: T | null): void => {
        // Repeated or late finishes of an already-replaced dialog are ignored.
        if (this.active !== dialog) return;
        this.pendingResolves.delete(settle as (result: null) => void);
        this.active = this.home;
        disposeComponent(dialog);
        if (!this.isClosed) this.tui.requestRender();
        settle(result);
      };
      this.pendingResolves.add(settle as (result: null) => void);

      dialog = create(finish, this.tui, this.theme);
      if (this.isClosed) {
        // The host closed while the component was being created.
        this.pendingResolves.delete(settle as (result: null) => void);
        disposeComponent(dialog);
        settle(null);
        return;
      }

      this.active = dialog;
      this.tui.requestRender();
    });
  }

  // -- Component ------------------------------------------------------------

  handleInput(data: string): void {
    if (this.isClosed) return;
    this.active?.handleInput(data);
  }

  invalidate(): void {
    this.home?.invalidate();
    this.active?.invalidate();
  }

  render(width: number): string[] {
    return this.active?.render(width) ?? [];
  }

  dispose(): void {
    if (this.isClosed) return;
    this.isClosed = true;
    this.settlePendingDialogs();
    this.disposeComponents();
  }

  // -- Internal -------------------------------------------------------------

  /**
   * Settle still-open dialogs with null so their awaiting action handlers
   * observe the closure instead of hanging. The stateful finish wrappers
   * are bypassed: teardown owns the component lifetimes.
   */
  private settlePendingDialogs(): void {
    for (const settle of [...this.pendingResolves]) settle(null);
    this.pendingResolves.clear();
  }

  /** Dispose the active component and the home exactly once each. */
  private disposeComponents(): void {
    const active = this.active;
    this.active = null;
    const home = this.home;
    this.home = null;
    disposeComponent(active);
    if (home !== active) disposeComponent(home);
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Dispose a component that may optionally implement `dispose()`. */
function disposeComponent(component: Component | null): void {
  (component as { dispose?: () => void } | null)?.dispose?.();
}
