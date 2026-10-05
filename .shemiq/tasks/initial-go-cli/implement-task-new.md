# Implement `shemiq task new`
:::shemiq
type: task
parent: ./top-level.md
status: done
:::

## Context

The Go module exists, but there is no CLI implementation yet. The parent task defines the top-level document format and the first command's scope. This command creates only top-level tasks; the README and `prompts/shemiq-new.md` describe the existing manual/prompt workflow. Keep title collection independent of file creation so a future agent can propose titles without changing how tasks are written.

## Interview

- Use Cobra for the CLI command tree, rather than hand-parsing commands; do not build a separate provider or extensibility framework.
- Slugs use ASCII only: lowercase `A–Z`, retain `a–z` and `0–9`, replace each run of other characters with `-`, and trim leading/trailing hyphens. Reject an empty result; do not append a collision suffix.
- Print an absolute path on success, even when the project was discovered above the invocation directory.
- Prefer a direct document write with best-effort cleanup on error over temporary-file staging or crash-safe atomicity.
- Keep tests small and centered on exercising the command with real temporary directories.

## Implementation plan

- Add a Cobra root command with `task new <description...>`. Require at least one nonblank description argument; join the description arguments with spaces. Prompt on stderr for one title line; trim surrounding whitespace and reject a blank title or EOF without a title. Keep command errors and usage output on stderr, and reserve stdout for the resulting absolute file path on success. Cobra's normal `--` separator can be used if description text begins with a flag-like word. No other command or flags are required.
- Separate the Cobra/prompt layer from a small task-creation function that takes the project location, description, and title as data and returns the created path or an error. Avoid tying creation to stdin or to how the title was obtained. Derive the slug using the agreed ASCII rule and validate inputs before making directories.
- From the invocation directory, walk upward to the nearest existing `.shemiq/` directory; if none is found, create `.shemiq/` in that directory. Create `.shemiq/tasks/<slug>/` exclusively and fail if it already exists, without touching its contents. Write `top-level.md` directly inside the newly created directory. If the write fails, best-effort remove the file and directory created for this task and report the failure (including any cleanup failure); no crash-safe atomicity is promised.
- Render the top-level document as `# <title>`, a `:::shemiq` directive with `type: top-level`, `## Description` containing the supplied description, and `## Context`, `## Interview`, `## Design`, `## Current status`, and `## Tasks` each containing `[TBD]`. Do not create subtasks or other project files.
- Test via the Cobra command and temporary directories: success when a nested invocation discovers an existing project, success when a new `.shemiq/` must be created, missing description/title input, and collision without overwriting an existing task. Check the document and absolute stdout path in success cases. No agent integration or broad helper-level test suite is needed.

## Implementation notes

- Added a Cobra command tree in `main.go` with a one-line stderr title prompt. Cobra's error usage is directed to stderr; only a successful task path is written to stdout.
- `task.go` discovers the nearest existing `.shemiq/`, validates the input and ASCII slug before creating directories, exclusively creates the task directory, and writes the top-level document with best-effort cleanup if writing fails. Creation takes the title as data rather than reading from the prompt.
- Added focused command tests for existing/new projects, document content and absolute output, invalid inputs, and collision preservation. `go test ./...` and `go vet ./...` pass.
