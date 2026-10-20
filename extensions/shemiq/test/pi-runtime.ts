/**
 * Real-runtime harness for the Shemiq extension e2e tests.
 *
 * Boots the real Pi interactive mode and session runtime against a
 * virtual terminal, with an isolated temporary environment:
 * - temporary HOME/cache/agent directories, no personal configuration;
 * - only a scripted faux model registered, network fetches forbidden;
 * - the repository-built `shemiq` CLI on PATH for real CLI behavior;
 * - the extension and the repository prompt templates loaded as resources.
 */

import { execFile } from "node:child_process";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import type {
  AgentSessionRuntime,
  CreateAgentSessionRuntimeFactory,
} from "@earendil-works/pi-coding-agent";
import { VirtualTerminal } from "./virtual-terminal.ts";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../..",
);
const extensionPath = path.join(repoRoot, "extensions/shemiq/index.ts");
const promptsPath = path.join(repoRoot, "prompts");

// ---------------------------------------------------------------------------
// CLI build
// ---------------------------------------------------------------------------

let shemiqBinDirPromise: Promise<string> | null = null;

/** Build the repository's shemiq CLI once per suite; returns its directory. */
export function buildShemiqBinary(): Promise<string> {
  shemiqBinDirPromise ??= (async () => {
    const dir = await mkdtemp(path.join(tmpdir(), "shemiq-e2e-bin-"));
    const bin = path.join(dir, "shemiq");
    await execFileAsync("go", ["build", "-o", bin, "."], { cwd: repoRoot });
    return dir;
  })();
  return shemiqBinDirPromise;
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

/** A prompt dispatched to the faux model, as seen by the provider. */
export interface CapturedPrompt {
  prompt: string;
  model: string | undefined;
}

export interface E2EHarness {
  /** Directory of the fixture project Pi runs in. */
  projectDir: string;
  /** The faux provider; script responses with setResponses/appendResponses. */
  faux: ReturnType<(typeof import("@earendil-works/pi-ai"))["fauxProvider"]>;
  /** Prompts dispatched so far, with the model selected at dispatch time. */
  prompts: CapturedPrompt[];
  /** The active runtime (its `session` changes on replacement). */
  runtime: AgentSessionRuntime;
  /** Network fetch attempts made during the test (must stay zero). */
  networkAttempts(): number;
  /** Number of session start events seen (initial, replacements, reloads). */
  sessionStartCount(): number;
  /** Full viewport as a single string. */
  screen(): string;
  /** Bounded wait until the predicate holds against the rendered screen. */
  waitFor(label: string, predicate: () => boolean): Promise<void>;
  /** Type a slash command into the main input and submit it. */
  typeCommand(command: string): void;
  /** Send a single key sequence to the focused component. */
  send(key: string): void;
  /** Type literal text into the focused input/editor field. */
  typeText(text: string): void;
  /** Run the repository CLI in the fixture project; returns stdout. */
  runShemiq(args: string[]): Promise<string>;
  /** Read a file inside the fixture project. */
  readProjectFile(relativePath: string): Promise<string>;
  /** Overwrite a file inside the fixture project. */
  writeProjectFile(relativePath: string, content: string): Promise<void>;
  dispose(): Promise<void>;
}

export interface HarnessOptions {
  /** Populate the fixture project; the CLI is already on PATH. */
  setupProject?: (fixture: {
    projectDir: string;
    /** Run the repository CLI in the fixture project; returns stdout. */
    runShemiq: (args: string[]) => Promise<string>;
  }) => Promise<void> | void;
}

export async function createHarness(
  options: HarnessOptions = {},
): Promise<E2EHarness> {
  const sdk = await import("@earendil-works/pi-coding-agent");
  const { fauxProvider } = await import("@earendil-works/pi-ai");

  const binDir = await buildShemiqBinary();
  const root = await mkdtemp(path.join(tmpdir(), "shemiq-e2e-"));
  const projectDir = path.join(root, "project");
  const agentDir = path.join(root, "agent");
  for (const dir of [
    projectDir,
    agentDir,
    path.join(root, "home"),
    path.join(root, "cache"),
    path.join(root, "sessions"),
  ]) {
    await mkdir(dir, { recursive: true });
  }

  // Isolated environment: no personal config, no network, repo CLI first on PATH.
  const savedEnv = pickEnv(
    "HOME",
    "XDG_CACHE_HOME",
    "PI_CODING_AGENT_DIR",
    "PI_OFFLINE",
    "PI_SKIP_VERSION_CHECK",
    "PI_TELEMETRY",
    "PATH",
  );
  process.env.HOME = path.join(root, "home");
  process.env.XDG_CACHE_HOME = path.join(root, "cache");
  process.env.PI_CODING_AGENT_DIR = agentDir;
  process.env.PI_OFFLINE = "1";
  process.env.PI_SKIP_VERSION_CHECK = "1";
  process.env.PI_TELEMETRY = "0";
  process.env.PATH = `${binDir}:${process.env.PATH}`;

  let networkAttempts = 0;
  const savedFetch = globalThis.fetch;
  globalThis.fetch = (async () => {
    networkAttempts++;
    throw new Error("Network access is forbidden in e2e tests");
  }) as typeof fetch;

  const runWithEnv = (args: string[], cwd: string) =>
    execFileAsync(path.join(binDir, "shemiq"), args, { cwd });

  if (options.setupProject) {
    await options.setupProject({
      projectDir,
      runShemiq: (args: string[]) => runWithEnv(args, projectDir),
    });
  }

  const faux = fauxProvider({
    provider: "shemiq-e2e-faux",
    models: [{ id: "model-a" }, { id: "model-b" }],
  });
  const prompts: CapturedPrompt[] = [];
  const sessionStarts: unknown[] = [];

  const terminal = new VirtualTerminal(100, 32);

  const createRuntime: CreateAgentSessionRuntimeFactory = async ({
    cwd,
    sessionManager,
    sessionStartEvent,
  }) => {
    const modelRuntime = await sdk.ModelRuntime.create({
      authPath: path.join(agentDir, "auth.json"),
      modelsPath: null,
      refreshOnCreate: false,
      allowModelNetwork: false,
    });
    const services = await sdk.createAgentSessionServices({
      cwd,
      agentDir,
      modelRuntime,
      settingsManager: sdk.SettingsManager.inMemory({
        quietStartup: true,
        theme: "dark",
        retry: { enabled: false },
        compaction: { enabled: false },
      }),
      resourceLoaderOptions: {
        noExtensions: true,
        noSkills: true,
        noThemes: true,
        noContextFiles: true,
        noPromptTemplates: true,
        additionalExtensionPaths: [extensionPath],
        additionalPromptTemplatePaths: [promptsPath],
        extensionFactories: [
          (pi) => {
            pi.registerProvider(faux.provider);
            pi.on(
              "before_agent_start",
              (event: { prompt: string }, ctx: { model?: { id: string } }) => {
                prompts.push({ prompt: event.prompt, model: ctx.model?.id });
              },
            );
            pi.on("session_start", () => {
              sessionStarts.push({});
            });
          },
        ],
      },
    });
    await modelRuntime.setRuntimeApiKey(faux.getModel().provider, "faux-key");
    const created = await sdk.createAgentSessionFromServices({
      services,
      sessionManager,
      sessionStartEvent,
      model: faux.getModel(),
      noTools: true,
    });
    return { ...created, services, diagnostics: services.diagnostics };
  };

  const runtime = await sdk.createAgentSessionRuntime(createRuntime, {
    cwd: projectDir,
    agentDir,
    sessionManager: sdk.SessionManager.create(
      projectDir,
      path.join(root, "sessions"),
    ),
  });

  const mode = new sdk.InteractiveMode(runtime, {
    terminal,
    initialThemeSetting: "dark",
  });
  await mode.init();
  void mode.run().catch((error: unknown) => {
    console.error("Interactive loop:", error);
  });
  await terminal.waitForRender();

  const screen = () => terminal.getViewport().join("\n");

  async function waitFor(
    label: string,
    predicate: () => boolean,
  ): Promise<void> {
    const deadline = Date.now() + 8000;
    while (Date.now() < deadline) {
      await terminal.waitForRender();
      if (predicate()) return;
    }
    throw new Error(
      `Timed out: ${label}\n${screen()}\nPrompts: ${JSON.stringify(prompts)}`,
    );
  }

  function typeCommand(command: string): void {
    terminal.sendInput(`\x1b[200~${command}\x1b[201~`);
    terminal.sendInput("\r");
  }

  function typeText(text: string): void {
    terminal.sendInput(`\x1b[200~${text}\x1b[201~`);
  }

  return {
    projectDir,
    faux,
    prompts,
    runtime,
    networkAttempts: () => networkAttempts,
    sessionStartCount: () => sessionStarts.length,
    screen,
    waitFor,
    typeCommand,
    send: (key: string) => terminal.sendInput(key),
    typeText,
    runShemiq: (args: string[]) => runWithEnv(args, projectDir),
    readProjectFile: (relativePath: string) =>
      readFile(path.join(projectDir, relativePath), "utf8"),
    writeProjectFile: (relativePath: string, content: string) =>
      writeFile(path.join(projectDir, relativePath), content),
    dispose: async () => {
      await mode.stop();
      await runtime.dispose();
      globalThis.fetch = savedFetch;
      restoreEnv(savedEnv);
    },
  };
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function execFileAsync(
  command: string,
  args: string[],
  options: { cwd: string },
): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile(
      command,
      args,
      options,
      (
        err: (Error & { code?: string }) | null,
        stdout: string,
        stderr: string,
      ) => {
        if (err) {
          const detail = stderr?.trim() || err.message;
          reject(new Error(`${command} failed: ${detail}`));
        } else {
          resolve(stdout);
        }
      },
    );
  });
}

function pickEnv(...keys: string[]): Record<string, string | undefined> {
  const saved: Record<string, string | undefined> = {};
  for (const key of keys) saved[key] = process.env[key];
  return saved;
}

function restoreEnv(saved: Record<string, string | undefined>): void {
  for (const [key, value] of Object.entries(saved)) {
    if (value === undefined) delete process.env[key];
    else process.env[key] = value;
  }
}
