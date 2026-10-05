---
description: Refine parent task's subtask
argument-hint: "<PARENT_TASK_PATH> <TASK_NAME>"
---
# Refine: $1 $2
We are refining a task from $1 that matches description "$2".

Focus on the description of the task, but take into account other mentioned
tasks during implementation to make other tasks easier to implement.
Primary goal is the complete the task, the secondary goal is to make other
tasks easy to solve (or just solve it if that's more practical, but keep
the tasks there, only add small comment if your work is affecting them).

The goal is to fill in necessary information for succesful implementation of the sub-task.
The purpose is not to list every implementation details, just the important questions
that would have big impact on the implementation.

Help refining the task through natural collaborative dialogue.

Read the task and potential parent files. If necessary, explore the current
project context and ask questions one at a time to refine the idea.
Once you understand what you're building, present the design in small sections (200-300 words), checking after each section whether it looks right so far.

## The Process

**Understanding the task:**
- Read the task and potential parent tasks.
- Ask questions one at a time to refine the plan
- Prefer multiple choice questions when possible, but open-ended is fine too
- The options should be numbered for easy selection.
- If you have a clear recommendation, state it with the question
- Only one question per message - if a topic needs more exploration, break it into multiple questions
- Focus on important implementaiton details: user-facing interface, high-level API, types and interfaces definitions,
  particular algorithms and approach, the testing approach.

**Exploring approaches:**
- Propose 2-3 different approaches with trade-offs
- Present options conversationally with your recommendation and reasoning
- Lead with your recommended option and explain why

**Presenting the design:**
- Once you believe you have clear idea about the implementation plan, present it to the user
- Break it into concise bullet-points
- Ask after each section whether it looks right so far
- Be ready to go back and clarify if something doesn't make sense

## The written document

Once everything is clear, write the results in separate shemiq task file in the same directory as the parent document.
Ensure the shemiq parent/source metadata are on both sides.

The document's body is later included verbatim in every subtask agent's implementation.


## Key Principles

- **One question at a time** - Don't overwhelm with multiple questions
- **Multiple choice preferred** - Easier to answer than open-ended when possible
- **YAGNI ruthlessly** - Remove unnecessary features from all designs
- **Explore alternatives** - Always propose 2-3 approaches before settling
- **Incremental validation** - Present design in sections, validate each
- **Be flexible** - Go back and clarify when something doesn't make sense
- **Testing** - Limit the tests to minimum, mostly when they positively contribute to the development and cover e2e scenario.

## Implementation

We don't do any actual code changes, only changes in the documents and
experiments to validate assumptions are allowed.

## Shemiq structure

We use shemiq tool to manage the structure of the documents. Load `shemiq` skill for more details.
