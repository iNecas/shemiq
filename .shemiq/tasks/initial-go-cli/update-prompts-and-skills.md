# Update the prompts and skills to start using shemiq binary
:::shemiq
type: task
parent: ./top-level.md
status: done
:::

## Context

`prompts/shemiq-new.md` currently asks the agent to create `.shemiq/tasks/<slug>/top-level.md` by hand. The existing CLI now supports non-interactive `shemiq task new --title <title> <description...>` and prints the created file's absolute path. `skills/shemiq/SKILL.md` describes the document format but does not yet explain the CLI or how to install it. The CLI only creates top-level tasks; the other prompts still need document-format guidance for subtasks and metadata edits.

## Interview

- Use the CLI rather than a manual-file fallback for creating top-level tasks. If installation or creation fails, report the problem instead of silently writing Markdown by hand.
- When the executable is missing, run `make install` from the Shemiq repository containing the loaded `skills/shemiq/SKILL.md`, not from the user's target project. The skill lives two directories below that repository root.
- If installation succeeds but `shemiq` is still off `PATH`, invoke the installed executable from Go's binary directory (`GOBIN` if set, otherwise `GOPATH/bin`) without requiring a change to the user's `PATH`.

## Implementation plan

- Update `prompts/shemiq-new.md` to have the agent choose a concise one-line title, pass it via `--title`, and pass the user's original description as a single safely quoted argument. Run the CLI from the target project's working directory; let it derive the slug, discover or create `.shemiq/`, generate the document and detect collisions. Use the path printed on success to tell the user where the task is; surface errors rather than guessing a path or manually creating the file.
- Expand `skills/shemiq/SKILL.md` with concise guidance for finding/installing and using `shemiq task new`. Check `PATH` first; when missing, derive the repository from the loaded skill location and run `make install` there. Afterward use `shemiq` from `PATH` or the Go-installed binary location, then restore the target project's working directory before invoking it. Keep the existing document-format guidance for operations the CLI does not support; do not imply it can create subtasks.
- Validate with one end-to-end smoke run from a temporary target project: confirm that the CLI prints a path and creates a top-level document with the original description. No new Go tests are necessary for prompt/skill text changes.

## Implementation notes

- Updated `prompts/shemiq-new.md` to request a one-line title, pass the complete original description as a single safely quoted argument to `shemiq task new --title`, and report only the CLI's printed path or error (no manual-file fallback).
- Added CLI discovery, installation from the loaded skill's repository, Go binary path fallback, target-project working-directory guidance, and the top-level-only limitation to `skills/shemiq/SKILL.md`. Retained the document-format guidance for subtasks and edits.
- Smoke-tested `make install` into a temporary Go binary directory and invoked the installed executable from a temporary project. It printed the expected absolute path and created a top-level document preserving a description containing both apostrophes and quotes. No new Go tests added.
