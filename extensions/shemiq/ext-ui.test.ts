/** Ext UI host lifecycle: switching, completion idempotency, and closure. */

import { describe, expect, it, vi } from "vitest";
import type { Theme } from "@earendil-works/pi-coding-agent";
import type { Component, TUI } from "@earendil-works/pi-tui";
import { ExtUiHost } from "./ext-ui.ts";

// ---------------------------------------------------------------------------
// Doubles
// ---------------------------------------------------------------------------

class TestDialog implements Component {
	renderCount = 0;
	inputs: string[] = [];
	disposeCount = 0;
	finish: ((result: string | null) => void) | undefined;

	constructor(public lines = ["dialog"]) {}

	handleInput(data: string): void {
		this.inputs.push(data);
	}
	invalidate(): void {}
	render(_width: number): string[] {
		this.renderCount++;
		return this.lines;
	}
	dispose(): void {
		this.disposeCount++;
	}
}

interface HostFixture {
	host: ExtUiHost;
	home: TestDialog;
	tui: TUI;
	doneResults: unknown[];
}

function makeHost(): HostFixture {
	const tui = {
		requestRender: vi.fn(),
		start: vi.fn(),
		stop: vi.fn(),
	} as unknown as TUI;
	const doneResults: unknown[] = [];
	const host = new ExtUiHost(tui, {} as Theme, (result) =>
		doneResults.push(result),
	);
	const home = new TestDialog(["home"]);
	host.setHome(home);
	return { host, home, tui, doneResults };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ExtUiHost", () => {
	it("hands the terminal to an operation and restarts with a forced redraw", async () => {
		const { host, tui } = makeHost();
		let stoppedDuringOp = false;

		const result = await host.runInTerminal(async () => {
			stoppedDuringOp = vi.mocked(tui.stop).mock.calls.length === 1;
			return "edited";
		});

		expect(result).toBe("edited");
		expect(stoppedDuringOp).toBe(true);
		expect(tui.stop).toHaveBeenCalledExactlyOnceWith({ preserveScreen: true });
		expect(tui.start).toHaveBeenCalledOnce();
		// The forced redraw comes after the restart.
		expect(tui.requestRender).toHaveBeenLastCalledWith(true);
	});

	it("restarts the terminal even when the operation fails", async () => {
		const { host, tui } = makeHost();

		await expect(
			host.runInTerminal(async () => {
				throw new Error("editor crashed");
			}),
		).rejects.toThrow("editor crashed");

		expect(tui.stop).toHaveBeenCalledOnce();
		expect(tui.start).toHaveBeenCalledOnce();
		expect(tui.requestRender).toHaveBeenLastCalledWith(true);
	});

	it("shows a dialog and returns to the retained home on finish", async () => {
		const { host, home } = makeHost();
		expect(host.render(80)).toEqual(["home"]);

		let dialog!: TestDialog;
		const promise = host.showDialog((finish) => {
			dialog = new TestDialog(["dialog"]);
			dialog.finish = finish;
			return dialog;
		});
		expect(host.render(80)).toEqual(["dialog"]);

		host.handleInput("x");
		// The dialog receives input while it is active; the home does not.
		expect(home.inputs).toEqual([]);
		expect(dialog.inputs).toEqual(["x"]);

		dialog.finish?.("done");
		await expect(promise).resolves.toBe("done");
		expect(host.render(80)).toEqual(["home"]);
	});

	it("resolves the dialog result exactly once and disposes it once", async () => {
		const { host } = makeHost();
		let finish!: (result: string | null) => void;
		const dialog = new TestDialog(["dialog"]);
		const promise = host.showDialog((f) => {
			finish = f;
			return dialog;
		});

		finish("done");
		// Repeated or late finishes of the replaced dialog are ignored.
		finish("other");
		finish(null);

		await expect(promise).resolves.toBe("done");
		expect(dialog.disposeCount).toBe(1);
		expect(host.render(80)).toEqual(["home"]);
	});

	it("forwards input only to the active component", () => {
		const { host, home } = makeHost();
		let dialog!: TestDialog;
		void host.showDialog((finish) => {
			dialog = new TestDialog(["dialog"]);
			dialog.finish = finish;
			return dialog;
		});
		host.handleInput("a");
		expect(dialog.inputs).toEqual(["a"]);
		expect(home.inputs).toEqual([]);
		dialog.finish?.(null);
		host.handleInput("b");
		expect(home.inputs).toEqual(["b"]);
		expect(dialog.inputs).toEqual(["a"]);
	});

	it("close resolves pending dialogs with null and cleans everything up", async () => {
		const { host, home, doneResults } = makeHost();
		let dialog!: TestDialog;
		const promise = host.showDialog((finish) => {
			dialog = new TestDialog(["dialog"]);
			dialog.finish = finish;
			return dialog;
		});

		host.close("final");
		expect(doneResults).toEqual(["final"]);
		await expect(promise).resolves.toBeNull();
		expect(dialog.disposeCount).toBe(1);
		expect(home.disposeCount).toBe(1);

		// Closed host: no rendering, no input, dialogs resolve to null.
		expect(host.render(80)).toEqual([]);
		host.handleInput("x");
		expect(home.inputs).toEqual([]);
		await expect(
			host.showDialog(() => new TestDialog(["late"])),
		).resolves.toBeNull();

		// Repeated close and dispose are tolerated.
		host.close("again");
		expect(doneResults).toEqual(["final"]);
		host.dispose();
		expect(home.disposeCount).toBe(1);
	});
});
