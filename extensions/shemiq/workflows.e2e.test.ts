/**
 * End-to-end workflow suite: drives the real Pi interactive/session runtime,
 * the Shemiq extension, browser, dialogs, and the repository CLI together
 * through virtual-terminal keyboard input, with only scripted faux models.
 *
 * Kept small and representative on purpose: component edge cases and
 * mocked context handling stay in the focused browser/launcher tests.
 */

import { beforeAll, describe, expect, it } from "vitest";
import { fauxAssistantMessage } from "@earendil-works/pi-ai";
import { buildShemiqBinary, createHarness, type E2EHarness } from "./test/pi-runtime.ts";

const TEST_TIMEOUT = 60_000;

let harness: E2EHarness;

beforeAll(async () => {
	await buildShemiqBinary();
}, 240_000);

/** Open the browser through the /shemiq command and wait for the tree. */
async function openBrowser(title: string): Promise<void> {
	harness.typeCommand("/shemiq");
	await harness.waitFor(`browser showing ${title}`, () => {
		const screen = harness.screen();
		return screen.includes("Shemiq Tasks") && screen.includes(title);
	});
}

/** Open the launcher for the given action and wait for the model dialog. */
async function openLauncher(key: string): Promise<void> {
	harness.send(key);
	await harness.waitFor("launcher dialog", () =>
		harness.screen().includes("Select Model"),
	);
}

/** Wait until the nth prompt has been dispatched and the session is idle. */
async function waitForPromptDispatched(count: number): Promise<void> {
	await harness.waitFor(`prompt ${count} dispatched`, () => {
		const session = harness.runtime.session;
		return harness.prompts.length >= count && !session.isStreaming;
	});
}

describe("Shemiq e2e workflows", () => {
	it(
		"launches refinement in both session modes through the launcher",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq([
						"new",
						"--title",
						"Launch feature",
						"Browser launch fixture",
					]);
				},
			});
			try {
				await openBrowser("Launch feature");

				// Cancelling the launcher returns to the browser.
				await openLauncher("r");
				expect(harness.screen()).toContain("Use current session");
				harness.send("\x1b");
				await harness.waitFor("launcher cancelled", () => {
					const screen = harness.screen();
					return (
						!screen.includes("Select Model") &&
						screen.includes("Shemiq Tasks")
					);
				});

				// Fresh session: select the other model and launch.
				const oldSession = harness.runtime.session;
				harness.faux.setResponses([
					fauxAssistantMessage("Refined via fresh session"),
				]);
				await openLauncher("r");
				harness.send("\x1b[A"); // up: select the non-default model
				harness.send("\r");
				await waitForPromptDispatched(1);

				expect(harness.runtime.session).not.toBe(oldSession);
				expect(harness.prompts[0].model).toBe("model-b");
				expect(harness.prompts[0].prompt).toContain(
					"# Refine: .shemiq/tasks/launch-feature/top-level.md",
				);
				expect(harness.faux.state.callCount).toBe(1);

				// Current session: toggle and launch without replacement.
				const currentSession = harness.runtime.session;
				harness.faux.appendResponses([
					fauxAssistantMessage("Refined via current session"),
				]);
				await openBrowser("Launch feature");
				await openLauncher("r");
				harness.send("c");
				await harness.waitFor("current session checked", () =>
					harness.screen().includes("[✓] Use current session"),
				);
				harness.send("\r");
				await waitForPromptDispatched(2);

				expect(harness.runtime.session).toBe(currentSession);
				expect(harness.prompts[1].prompt).toContain("# Refine:");
				expect(harness.faux.state.callCount).toBe(2);
				expect(harness.networkAttempts()).toBe(0);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"preserves browser selection and scroll when cancelling dialogs",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					for (let i = 1; i <= 24; i++) {
						await runShemiq([
							"new",
							"--title",
							`Scroll task ${String(i).padStart(2, "0")}`,
						]);
					}
				},
			});
			try {
				await openBrowser("Scroll task 01");

				// Move deep enough to scroll the viewport. Plain pageUp/pageDown/
				// home/end keys are consumed by Pi's viewport scrolling, so use "j".
				for (let i = 0; i < 25; i++) harness.send("j");
				await harness.waitFor("scrolled viewport", () => {
					const screen = harness.screen();
					return screen.includes("▲ 2 more") && screen.includes("Scroll task 24");
				});
				const before = harness.screen();

				// Cancelling the launcher returns the exact same browser viewport.
				await openLauncher("r");
				harness.send("\x1b");
				await harness.waitFor("launcher cancelled", () => {
					const screen = harness.screen();
					return (
						!screen.includes("Select Model") &&
						screen.includes("Shemiq Tasks")
					);
				});
				expect(harness.screen()).toBe(before);

				// Cancelling the creation dialog discards the draft and keeps the
				// browser selection and scroll position.
				harness.send("n");
				await harness.waitFor("creation dialog", () =>
					harness.screen().includes("Title:"),
				);
				harness.typeText("Draft title");
				await harness.waitFor("draft typed", () =>
					harness.screen().includes("Draft title"),
				);
				harness.send("\x1b");
				await harness.waitFor("creation cancelled", () => {
					const screen = harness.screen();
					return (
						!screen.includes("Title:") &&
						screen.includes("Shemiq Tasks")
					);
				});
				expect(harness.screen()).toBe(before);

				// Reopening starts from fresh child state, not the cancelled draft.
				harness.send("n");
				await harness.waitFor("creation dialog reopened", () =>
					harness.screen().includes("Title:"),
				);
				const reopened = harness.screen();
				expect(reopened).toContain("(optional – leave blank");
				expect(reopened).not.toContain("Draft title");
				harness.send("\x1b");
				await harness.waitFor("creation cancelled again", () =>
					!harness.screen().includes("Title:"),
				);

				// Closing the browser restores keyboard control to Pi's editor.
				harness.send("\x1b");
				await harness.waitFor("browser closed", () =>
					!harness.screen().includes("Shemiq Tasks"),
				);
				harness.typeText("editor control");
				await harness.waitFor("editor focus restored", () =>
					harness.screen().includes("editor control"),
				);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"relaunches after /reload through Alt+S in both session modes",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq([
						"new",
						"--title",
						"Reload feature",
						"Reload workflow fixture",
					]);
				},
			});
			try {
				// Initial fresh-session launch.
				harness.faux.setResponses([fauxAssistantMessage("First pass")]);
				await openBrowser("Reload feature");
				await openLauncher("r");
				harness.send("\r");
				await waitForPromptDispatched(1);
				const sessionsBeforeReload = harness.sessionStartCount();

				// Reload replaces the runtime; Alt+S reopens through the
				// renewed extension runtime.
				harness.typeCommand("/reload");
				await harness.waitFor("runtime reloaded", () => {
					const session = harness.runtime.session;
					return (
						harness.sessionStartCount() > sessionsBeforeReload &&
						!session.isStreaming
					);
				});
				harness.send("\x1bs");
				await openBrowser("Reload feature");

				// Current-session launch in the replaced runtime.
				harness.faux.appendResponses([
					fauxAssistantMessage("Current after reload"),
				]);
				const sessionAfterReload = harness.runtime.session;
				await openLauncher("r");
				harness.send("c");
				await harness.waitFor("current session checked", () =>
					harness.screen().includes("[✓] Use current session"),
				);
				harness.send("\r");
				await waitForPromptDispatched(2);
				expect(harness.runtime.session).toBe(sessionAfterReload);

				// Fresh-session launch in the replaced runtime as well.
				harness.faux.appendResponses([
					fauxAssistantMessage("Fresh after reload"),
				]);
				harness.send("\x1bs");
				await openBrowser("Reload feature");
				await openLauncher("r");
				harness.send("\r");
				await waitForPromptDispatched(3);

				expect(harness.prompts[2].prompt).toContain(
					"# Refine: .shemiq/tasks/reload-feature/top-level.md",
				);
				expect(harness.faux.state.callCount).toBe(3);
				expect(harness.networkAttempts()).toBe(0);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"creates a title-only top-level task through the dialog",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq(["new", "--title", "Existing feature"]);
				},
			});
			try {
				await openBrowser("Existing feature");
				harness.send("n");
				await harness.waitFor("creation dialog", () =>
					harness.screen().includes("New task"),
				);

				// Both fields blank: stay in the dialog with an error.
				harness.send("\x13");
				await harness.waitFor("empty input error", () =>
					harness.screen().includes("Title or description is required"),
				);

				// A title alone is sufficient to create directly.
				harness.typeText("Title only feature");
				harness.send("\x13");
				await harness.waitFor("browser refreshed with new task", () => {
					const screen = harness.screen();
					return screen.includes("Shemiq Tasks") &&
						screen.includes("Title only feature") && screen.includes("Created");
				});
				const tree = JSON.parse(await harness.runShemiq(["tree", "--json"]));
				expect(tree.tasks.map((task: { title: string }) => task.title)).toContain("Title only feature");
				expect(harness.prompts).toHaveLength(0);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it.each(["", "Second subtask description from e2e"])(
		"creates a sibling subtask directly through the CLI with description %j",
		async (description) => {
			harness = await createHarness({
				setupProject: async ({ runShemiq, projectDir }) => {
					await runShemiq([
						"new",
						"--title",
						"Parent feature",
						"Parent description",
					]);
					await runShemiq([
						"new-subtask",
						`${projectDir}/.shemiq/tasks/parent-feature/top-level.md`,
						"--title",
						"First subtask",
						"First description",
					]);
				},
			});
			try {
				await openBrowser("Parent feature");

				// Hover the subtask row: "n" offers a sibling subtask.
				harness.send("j");
				await harness.waitFor("subtask legend", () =>
					harness.screen().includes("New subtask"),
				);
				harness.send("n");
				await harness.waitFor("creation dialog", () =>
					harness
						.screen()
						.includes('under "Parent feature" after "First subtask"'),
				);

				// Title filled: direct non-interactive CLI creation.
				harness.typeText("Second subtask");
				if (description) {
					harness.send("\t");
					harness.typeText(description);
				}
				harness.send("\x13"); // Ctrl+S submits

				await harness.waitFor("browser refreshed with new task", () => {
					const screen = harness.screen();
					return (
						screen.includes("Shemiq Tasks") &&
						screen.includes("Second subtask") &&
						screen.includes("Created")
					);
				});

				// Real CLI/filesystem result: inserted right after the hovered sibling.
				const tree = JSON.parse(await harness.runShemiq(["tree", "--json"]));
				const titles = tree.tasks[0].subtasks.map(
					(subtask: { title: string }) => subtask.title,
				);
				expect(titles).toEqual(["First subtask", "Second subtask"]);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"dispatches prompt-based creation with a blank title in the current session",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq([
						"new",
						"--title",
						"Existing feature",
						"Existing fixture",
					]);
				},
			});
			try {
				await openBrowser("Existing feature");

				// Blank title: browser closes and /shemiq-new runs in place.
				harness.faux.setResponses([fauxAssistantMessage("Created")]);
				const session = harness.runtime.session;
				harness.send("n");
				await harness.waitFor("creation dialog", () =>
					harness.screen().includes("New task"),
				);
				harness.send("\t"); // skip the title, focus the description
				harness.typeText("A brand new task description from e2e");
				harness.send("\x13"); // Ctrl+S submits

				await waitForPromptDispatched(1);
				expect(harness.runtime.session).toBe(session);
				expect(harness.prompts[0].prompt).toContain("New shemiq task");
				expect(harness.prompts[0].prompt).toContain(
					"A brand new task description from e2e",
				);
				expect(harness.screen()).not.toContain("Shemiq Tasks");
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"archives a top-level task immediately and refreshes the browser",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq([
						"new",
						"--title",
						"Archive feature",
						"Archive workflow fixture",
					]);
					await runShemiq([
						"new",
						"--title",
						"Remaining feature",
						"Stays after archiving",
					]);
				},
			});
			try {
				await openBrowser("Archive feature");

				// The top-level row is selected by default; "a" archives it.
				harness.send("a");
				await harness.waitFor("task archived and tree refreshed", () => {
					const screen = harness.screen();
					return (
						screen.includes("Shemiq Tasks") &&
						screen.includes("Archived .shemiq/tasks/archive-feature") &&
						!screen.includes("Archive feature") &&
						screen.includes("Remaining feature")
					);
				});

				// Real CLI/filesystem result: the task moved to the archive.
				const tree = JSON.parse(await harness.runShemiq(["tree", "--json"]));
				const titles = tree.tasks.map(
					(task: { title: string }) => task.title,
				);
				expect(titles).toEqual(["Remaining feature"]);
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);

	it(
		"recovers from tree loading failures through Refresh",
		async () => {
			harness = await createHarness({
				setupProject: async ({ runShemiq }) => {
					await runShemiq([
						"new",
						"--title",
						"Recovery feature",
						"Recovery fixture",
					]);
				},
			});
			const taskFile = ".shemiq/tasks/recovery-feature/top-level.md";
			try {
				// Corrupt the metadata so the real CLI tree load fails.
				const original = await harness.readProjectFile(taskFile);
				await harness.writeProjectFile(
					taskFile,
					original.replace(/uuid: [^\n]+/, "uuid: not-a-uuid"),
				);

				harness.typeCommand("/shemiq");
				await harness.waitFor("recovery screen", () =>
					harness.screen().includes("Error loading tasks"),
				);

				// Stale state is not actionable while the error is displayed.
				harness.send("r");
				await harness.waitFor("no actionable rows", () =>
					harness.screen().includes("No task selected"),
				);

				// Correct the fixture and recover through Refresh.
				await harness.writeProjectFile(taskFile, original);
				harness.send("\x12"); // Ctrl+R
				await harness.waitFor("tree recovered", () => {
					const screen = harness.screen();
					return (
						screen.includes("Shemiq Tasks") &&
						screen.includes("Recovery feature") &&
						!screen.includes("Error loading tasks")
					);
				});
			} finally {
				await harness.dispose();
			}
		},
		TEST_TIMEOUT,
	);
});
