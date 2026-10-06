# Shemiq

Opinionated setup for LLM-accelerated development that keeps the developer involved.

## Motivation

The spec-coding, heavy antigenic setups, autonomous loops of swarms of agents
don't work (for me at least). They lead to bloated code nobody is willing to
touch (or even look at). They take away from the developers many opportunities to
learn about the problem at hand and steer as the understanding improves.  The
assumption is: the developer knows all the answers at the beginning, the agents
execute on that and everyone is happy. In reality, only the tokens provider is.

The sufficiently detailed specification of the application is the code. Developing
the specification is the main task.

Instead, shemiq focus on building the changes, not trying to compose a markdown
source of truth, indistinguishable from any other slop.

## Overview

Shemiq is a combination of prompts, skills and simple tools that helps organizing
the work that optimizes for:
- understanding
- ownership of decisions
- cost-effective small-context sessions
- incremental iterative development
- frequent reviews around small set of changes
- file-based work tracking (no databases and external services)

It helps with keeping track of tasks being worked, refinement, planning and
implementation.

At the core, there is just an opinionated markdown layout to track the tasks
and few prompts and skills that can be driven entirely from vanilla coding agent
(works best with [Pi](https://pi.dev/), a motivated reader could make it work
with any other implementation).

The rest of the project is just a set of quality-of-life tools that guide the workflow.
This includes:

- CLI
- TUI (TBD)
- Pi extension (TBD)
- TUI (TBD)
- Emacs integration (TBD)
- Herder integration (TBD)

## Installation

Install [Pi](https://pi.dev/) and make sure `pi` is on `PATH`. Clone this repository
and install its prompts and skill as a Pi package:

```sh
git clone git@github.com:inecas/shemiq.git
cd shemiq
make install
pi install "$(pwd)"
```

At some point, there might be releases and packaging, let's keep it personal for now.
  
## Usage

**1. Create new top-level task**

Create a top-level task with short description of your intent in `.shemiq/tasks/$MY_TASK/top-level.md`, or just use the predefined prompt.

```
/shemiq-new a task I would like to work on
```

**2. Refine and split the top-level task**

Provide more description and intial context in the `top-level.md` file and initiate
the refinement process:

```
/shemiq-refine-and-split .shemiq/tasks/$MY_TASK/top-level.md
```

You can also launch refinement from a terminal with `shemiq refine .shemiq/tasks/$MY_TASK`,
or run `shemiq refine` to choose a task interactively.

This will guide you thought the refinement process, asking questions about the problem,
top-level design and initial tasks split. The goal is to gather more context and split
it to reasonably sized chunks suitable for iterative development.

At the end of this phase, everything is still contained in the single `top-level.md` file,
with short descriptions of each sub-task.

**3 a. Refine a sub-task**

Refining a sub-task will create a separate task file in the same directory with design
and implementation details for the specific task. Unlike `top-level.md` that's intended
for high-level context useful for all sub-tasks, it includes details most relevant
to the particular sub-task.

```
/shemiq-refine-sub-task .shemiq/tasks/$MY_TASK/top-level.md "subtask name"
```

Or use `shemiq refine .shemiq/tasks/$MY_TASK --subtask "subtask name"` from a terminal.
Interactive `shemiq refine` variant lets you choose the sub-tasks from refined top-level tasks.

**4. Implement the refined sub-task**

```
/shemiq-implement-task .shemiq/tasks/$MY_TASK/$SUB_TASK.md
```

**3 b. Implement a sub-task immediately**

If a sub-task is straight forward, all the important information has already been
gathered during top-level refinement and you feel lucky, one can trigger the implementation immediately
without sub-task refinement.

```
/shemiq-implement-task-parent .shemiq/tasks/$MY_TASK/top-level.md "subtask name"
```

This should still produce the separate sub-task file at the end of the implementation.

## Inspiration

* [tasktron](https://kevinlynagh.com/newsletter/2026_09_task_workflow/#a-single-file-llm-task-workflow-harness) — a simple Clojure based implementation of local deterministic harness.
  * I like the concepts, but the changes I needed were big enough to start from scratch.
    Also, while appreciating Lisp and Clojure, it's not my primary language and I find
    go-lang a better fit for me.
