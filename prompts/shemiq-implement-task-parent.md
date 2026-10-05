---
description: Implement a subtask from parent
argument-hint: "<PARENT_TASK_PATH> <TASK_NAME>"
---
# Implement: $2
Implement a task from $1 that matches description "$2".

Focus on the description of the task, but take into account other mentioned
tasks during implementation to make other tasks easier to implement.
Primary goal is the complete the task, the secondary goal is to make other
tasks easy to solve (or just solve it if that's more practical, but keep
the tasks there, only add small comment if your work is affecting them).

## Short planning phase
First create a separate shemiq task file for that document, briefly describing the plan.

Ask for calrifying questions if anything isn't clear.

## Implementation

Implement the task based on the description and plan
Limit the tests to minimum, mostly when they positively contribute to the development and cover e2e scenario.

## Summary

Write the notes from the implementation to the task file and update the current status
section in the top level file.

## Shemiq structure

We use shemiq tool to manage the structure of the documents. Load `shemiq` skill for more details.
