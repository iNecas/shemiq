# Add validate command for task metadata
:::shemiq
type: top-level
uuid: ccc11874-4584-472b-9968-f95faa709d0e
:::

## Description

Add a `validate` command to check metadata in `.shemiq` task documents. It accepts a single file or a directory (including a task subdirectory), or scans all Shemiq files when no path is given. An optional `--fix` flag attempts to correct fixable errors.

Introduce a `uuid` field to uniquely identify each task. Every `:::shemiq` directive uses a UUID unless it has `source:`; in that case, the UUID is determined by the source. The `uuid` and `source` fields are mutually exclusive.

## Context

The Go CLI uses Cobra and provides `shemiq task new` and `shemiq validate`; `internal/task/task.go` discovers `.shemiq/` directories and renders new top-level documents. The validator now checks metadata and can insert missing UUIDs. `skills/shemiq/SKILL.md` documents directives with `type`, `parent`, `source`, and `status`, while existing `.shemiq/` documents provide real examples. Existing documents have been repaired and validated; `task new` still lacks UUIDs. The skill's task-entry examples use `todo`, although its status list says `new`, `progress`, and `done`.

## Interview

- Every directive without `source:` needs its own UUID, including a status-only directive; a directive with `source:` must not also carry `uuid:`.
- Validate known metadata fields, allowed values, UUIDs, and local `parent:` targets, but not Markdown headings, prose, filenames, or directive placement. Treat each directive independently rather than inferring requirements from its role in a document.
- Accept only `new`, `progress`, and `done` for `status`. Generate and require canonical lowercase UUIDv4 values.
- `--fix` adds missing UUIDs only; it does not replace malformed or duplicate UUIDs, remove conflicts, or rewrite references.
- Without a path, scan Markdown files throughout the nearest existing `.shemiq/` directory; a directory argument scans its Markdown files recursively. Skip files with no Shemiq directives. Uniqueness applies only to explicit UUIDs in the selected files, not the whole project when a single file is selected.
- Check that `parent:` targets exist, but do not require `source:` targets to exist: listed subtasks may not yet have their own documents. Do not load files outside the selected scope to resolve sources or check uniqueness.
- Expose the command as `shemiq validate [path] [--fix]`. Use a small line-oriented parser rather than a Markdown AST or regex-only substitutions; unknown metadata keys are errors.

## Design

- Add the Cobra command alongside `task new`. Reuse upward project discovery for the no-path case, but do not create `.shemiq/` during validation; an explicit path may be a Markdown file or a directory. Separate path selection from directive parsing and validation.
- Parse complete `:::shemiq` blocks as simple `key: value` fields, retaining file and line locations. Report malformed or unterminated blocks, repeated or unknown keys, invalid `type`/`status` values, invalid UUIDv4 values, `source:`/`uuid:` conflicts, and missing `parent:` files. Resolve `parent:` relative to the containing file. Do not impose a task-document template or require source files to exist.
- Check duplicate explicit UUIDs only across the selected files. Source entries refer to task documents, possibly not yet created, and have no explicit UUID to compare. In `--fix` mode, insert a fresh UUIDv4 into each parseable directive lacking both `uuid:` and `source:`, preserving other file contents; then validate the selected files. A second `--fix` run should not change them.
- Report errors with file and line locations and return a nonzero result for unresolved validation errors. A clean check or fully successful repair returns success. Share a UUIDv4 generator between validation repair and `task new`; the latter adds a UUID to each new top-level document without changing its other behavior.
- Update `skills/shemiq/SKILL.md` to describe UUIDs, the source exception, statuses, and validation; correct the `todo` examples. Bring existing `.shemiq/` documents into compliance with `validate --fix` once it exists.
- Keep tests focused on temporary-project CLI scenarios: selected-file versus directory/default scope and duplicate detection, missing UUID repair and repeat-run stability, invalid metadata, and missing parent versus allowed missing source. Update the existing `task new` document test to check UUID shape without relying on a fixed value.

## Current status

`shemiq validate [path] [--fix]` performs scoped metadata checks and conservative UUID insertion. `task new` now uses the shared UUID generator to create valid top-level documents, and the skill documents UUID/source rules and the manual validation workflow. Both tasks are complete; `go test ./...` and read-only `go run . validate` pass. Existing compliant documents were not migrated again.

## Tasks

### Implement `shemiq validate`
:::shemiq
type: task
source: ./implement-validate-command.md
status: done
:::

Add scoped metadata validation and conservative `--fix` UUID insertion, with focused CLI tests.

### Generate UUIDs for new tasks and adopt the metadata rules
:::shemiq
type: task
source: ./adopt-uuid-metadata.md
status: done
:::

Use the shared UUIDv4 generator in `task new`, update the skill and affected tests.

The skill should instruct the agent to not attempt to generate the uuid on its own, but rather rely on validate --fix to fill in the missing uuids.
