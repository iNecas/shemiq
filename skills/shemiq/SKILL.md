---
name: shemiq
description: Metadata tracking in .shemiq/** project files. Load always when working with files in that directory.
---
# Summary

Shemiq is a tool and file format for keeping track of the tasks by leveraging appropriate metadata.

It uses ``:::shemiq` markdown directive to keep metatada about the tasks and links between them.

## Installing and locating the CLI

Some usage scenarios involve using the `shemiq` command. If the command is not present, follow the following steps:

- Locate the *loaded* `skills/shemiq/SKILL.md` in the Shemiq repository (not the target project). The repository root is two directories above that skill file. Run `make install` from that root; if it fails, report the error and stop. Return to the original working directory afterward.
- Use `shemiq` if it is on `PATH`. Otherwise invoke the installed executable directly: use `$(go env GOBIN)/shemiq` when `GOBIN` is nonempty, or `<first entry of go env GOPATH>/bin/shemiq` when it is empty. If the executable cannot be found or run, report the problem. No `PATH` change is required.

## Creating a top-level task with the CLI

- From the **target project's working directory**, run `shemiq task new --title <one-line title> <original description>` using the executable located above. Pass the whole description as **one safely shell-quoted argument** and quote the title too. The CLI discovers the nearest `.shemiq/` upwards or creates one in the working directory, derives the slug from the title, writes `.shemiq/tasks/<slug>/top-level.md`, rejects collisions, and prints the absolute file path. Report that printed path on success; surface CLI errors without guessing a path or creating the file manually.
- This CLI creates **top-level tasks only** and assigns their UUIDs automatically. For subtasks, metadata edits, and other document changes, use the format below; do not claim `task new` supports them.

## Metadata and validation

- Every `:::shemiq` directive without `source:` needs its **own** canonical lowercase UUIDv4 in `uuid:`, even if it contains only `status:`. A directive with `source:` must **not** have `uuid:`; its identity comes from the referenced task. Accepted statuses are `new`, `progress`, and `done`.
- When writing or editing directives by hand, do **not** generate or copy UUIDs yourself. Leave `uuid:` absent on new source-less directives, then run `shemiq validate --fix <file>` **separately for each touched Markdown file** needing repair. `--fix` only inserts missing UUIDs; resolve any remaining validation errors manually and recheck with `shemiq validate <file>`.
- `shemiq validate [path] [--fix]` checks metadata in a Markdown file or recursively in a directory. Without a path it scans the nearest existing `.shemiq/` directory. A failed check reports errors and exits nonzero.

## Common structure

These are illustrative **pre-repair** examples: source-less directives intentionally omit `uuid:`. Do not copy them as final documents; run `validate --fix` on each file after writing it.

### Top level tasks

Top level task serves as an entry-point for the work. It provides the top
level context common for all tasks.

It's purpose is high-level refinement.

The top level is eventually broken down to one or many tasks. The top
level keeps only the descriptions of the tasks. The details are kept
in separate documents.

    # Buy grocery
    :::shemiq
    type: top-level
    :::

    ## Description

    [[User-focused description of the problem]].

    ## Context

    [[Context of the top-level, relevant for all tasks]]
    
    ## Interview
    
    [[Summary of all questions and provided answers from the refinement session]]

    ## Design

    [[High-level design description for the implementers of the tasks.]]

    ## Current status

    [[Notes about the work done so far as part of this feature]

    ## Tasks

    ### Make a list
    :::shemiq
    type: task
    source: ./make-a-list.md
    status: done
    :::

    ### Buy the things
    :::shemiq
    type: task
    source: ./buy-the-things.md
    status: new
    :::

    ### Put the things in place
    :::shemiq
    type: task
    source: ./put-the-things-in-place.md
    status: new
    :::

### Tasks

Tasks are the main unit of actual work. They describe meaningful chunk
that should be possible to do in one session.

    # Make a list
    :::shemiq
    type: task
    parent: ./top-level.md
    status: done
    :::

    ## Context

    [[More details description of impotant information for the implementation.]]

    ## Interview
    
    [[Summary of all questions and provided answers from the refinement session (if applicable)]]

    ## Implementation plan

    [[Key points needed for the implementer finish the task. Focus on important
    decisions and guidance, leave the details for the implementer.]]

    ## Implementation notes

    [[Summary fillsed in by the implementer, describing more details from the implementation,
    focus on the key points and potential deviation from the task.]]

## The status

Even a status-only directive needs a UUID. This is a pre-repair example; run `validate --fix <file>` after writing it:

    :::shemiq
    status: [new|progress|done]
    :::
