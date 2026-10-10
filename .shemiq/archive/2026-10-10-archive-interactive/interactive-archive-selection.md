# Add interactive archive selection
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 291ce8e7-0291-41bb-8eb0-322fcc43da04
:::

## Context

- Allow `shemiq archive` without a path to select a recognized immediate-child `top-level.md` task, across all statuses.
- Preserve explicit-path behavior and existing archive safety checks; prompts go to stderr and only the destination goes to stdout.

## Implementation plan

- Reuse task-store discovery and the console picker; filter candidates to `.shemiq/tasks/<task>/top-level.md` and label them `Title [status]`.
- Accept zero or one argument, always prompt without a path, and archive the selected directory immediately using the existing operation.
- Add one happy-path CLI test covering selection, the directory move, output separation, and preserved contents.
- Update command help, `README.md`, and `skills/shemiq/SKILL.md`; validate linked task metadata and record results here.

## Implementation notes

- `cmd/archive.go` accepts zero or one path; no-path selection uses `task.Store` and `Console.Pick` with numbered `Title [status]` labels across all statuses.
- Candidate filtering requires immediate-child `top-level.md` documents; duplicate titles remain independently selectable. Discovery errors and picker cancellation propagate without moving tasks.
- The existing archive operation and explicit-path metadata bypass are unchanged; selection archives immediately, with prompts on stderr and only the absolute destination on stdout.
- Added one happy-path CLI test covering all statuses, duplicate titles, candidate filtering, nearest-project discovery, output separation, and preserved Markdown/binary contents.
- Updated command help, `README.md`, and `skills/shemiq/SKILL.md`; no changes to shared store or picker APIs were needed.
- Checks passed: archive-focused tests, `go test ./...`, and `go vet ./...`.
