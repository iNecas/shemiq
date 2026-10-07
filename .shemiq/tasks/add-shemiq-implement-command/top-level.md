# Add shemiq implement command
:::shemiq
type: top-level
uuid: f4febffd-4b70-4c20-b2ba-3d879f59c340
:::

## Description

`shemiq implement` cmd

In .shemiq/archive/2026-10-07-add-shemiq-refine-cli/top-level.md we implemented
the refine command, including interactive selection of new tasks.

Implement will be similar.

It should allow implementing subtasks. They can be either refined or new.
The status should be included in the intractive offering for transparency.

Refined tasks should be handled by /shemiq-implement-task, sub-tasks in new
state should be handled by /shemiq-implement-task-parent. One is always implementing
subtasks (even if top-level would have just one sub-task).

## Context

[TBD]

## Interview

[TBD]

## Design

[TBD]

## Current status

[TBD]

## Tasks

[TBD]
