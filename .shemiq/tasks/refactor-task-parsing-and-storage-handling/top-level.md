# Refactor task parsing and storage handling
:::shemiq
type: top-level
uuid: 62f017d6-7eae-48d9-8a42-50fb8ae3408b
:::

## Description

refactor tasks parsing and storage handling

There are multiple in internal/task that handle some parts
of the parsing and it's not combined well together. For example:

- internal/task/refine.go and internal/task/outline.go
  doing it's own outline parsing and top level path detection
- internal/task/metadata.go
  doing raw parsing
- internal/task/validate.go
  doing more semantic validation, but not providing way to work with the data once validated

Overall, the API is terrible and leads to bloated and hard to maintain codebase.

Let's refactor the code and split it to the following:

- internal/task/parse.go
  - focues on parsing single markdown documents with focus on the structure we are interested in:
     - the nodes should be just sections with title, position, directives and children sections
     - the parsing will not go beyond single document
     - the parsing errors on this level should focus mostly on syntax errors (not semantics).
- internal/task/store.go
  - higher level parsing
  - will leverage parse for low-level data extraction, will keep Task representation of the loaded tasks, will provide an API to validate, create and query the tasks (in the scope of the current functionality).

Ideally, once refactoring is done, the most of the business logic from internal/{metadata,outline,archive,refine,validate}  would be either in parse.go or store.go, or inside corresponding cmd files.

Let's start first with refining the desired interface for store and parse, while considering the current code-base.

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
