---
description: Refine the task and split to subtasks
argument-hint: "<TASK_PATH>"
---
# Refine: $1
Help building the context and high-level design for addressing the task through natural collaborative dialogue.

Explore the current project context and ask questions one at a time to refine the idea.
Once you understand what you're building, present the design in small sections (200-300 words), checking after each section whether it looks right so far.

Propose the split to the tasks, but limit it to the titles, shemiq directives
and single-sentence descriptions. They will be revined sparately.

## The Process

**Understanding the idea:**
- Check out the current project state:
   - Where in codebase changes are needed for this task
   - What existing patterns/structures to follow
   - Which files need modification
   - What related features/code already exist
- Ask questions one at a time to refine the idea
- Prefer multiple choice questions when possible, but open-ended is fine too
- The options should be numbered for easy selection.
- If you have a clear recommendation, state it with the question
- Only one question per message - if a topic needs more exploration, break it into multiple questions
- Focus on understanding: purpose, constraints, success criteria

**Exploring approaches:**
- Propose 2-3 different approaches with trade-offs
- Present options conversationally with your recommendation and reasoning
- Lead with your recommended option and explain why

**Presenting the design:**
- Once you believe you understand what you're building, present the design
- Break it into concise bullet-points
- Ask after each section whether it looks right so far
- Cover: architecture, components, data flow, error handling, testing
- Be ready to go back and clarify if something doesn't make sense

## The written document

The document's body is later included verbatim in every subtask agent's prompt during implementation, so it must hold only context that applies to the whole task.

## Key Principles

- **One question at a time** - Don't overwhelm with multiple questions
- **Multiple choice preferred** - Easier to answer than open-ended when possible
- **YAGNI ruthlessly** - Remove unnecessary features from all designs
- **Explore alternatives** - Always propose 2-3 approaches before settling
- **Incremental validation** - Present design in sections, validate each
- **Be flexible** - Go back and clarify when something doesn't make sense
- **Testing** - Limit the tests to minimum, mostly when they positively contribute to the development and cover e2e scenario.

## Next step

Once a shared understanding has been reached, Update the $1 with details.

The original context provided to the task should be preserved. It can be
reformulated to match the rest of the description, but the the information should
be preserved.

## Shemiq structure

We use shemiq tool to manage the structure of the documents. Load `shemiq` skill for more details.
