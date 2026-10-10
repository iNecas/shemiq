---
description: Refine the task and split to subtasks
argument-hint: "<TASK_PATH>"
---
# Refine: $1
Help building the context and high-level design for addressing the task through natural collaborative dialogue.

Explore the current project context and ask questions one at a time to refine the idea.
Once you understand what you're building, present the design in small sections (200-300 words), checking after each section whether it looks right so far.

Propose the split to the tasks, but limit it to the titles, shemiq directives
and single-sentence descriptions. They will be refined sparately.

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

**Exploring the design:**
- Once you believe you understand what you're building, continue with the questions
  about the high-level design common for all tasks (leave low-level for sub-tasks refinement phase).
- Cover: architecture, components, data flow, error handling, testing
- Continue using the questions/options style.

**Summarizing the answers**
- Once all the questions has been answered, summarize the answers gathered in the interview 
- Break it into concise bullet-points
- Ask after each section whether it looks right so far
- Cover: architecture, components, data flow, error handling, testing
- Be ready to go back and clarify if something doesn't make sense

**Splitting to sub-tasks**
- The last phase before writing down the results is to propose the split
  to subtasks to follow during the further development.

**Updating the document**
- Use the information from the interview to fill in the rest of the document
- **Don't include anything that was not discussed explicitly during the interview**
- Use concise bullet-points style.

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

Once the user approves the design and task split, update $1 with the agreed
context, design, and subtasks. Only after these are written, set `status: refined`
in its top-level `:::shemiq` directive (add the field if absent). Do not mark it
refined merely because this session ends or Pi exits. Keep each new subtask's
parent-list status `new`.

The original context provided to the task should be preserved. It can be
reformulated to match the rest of the description, but the the information should
be preserved.

## Shemiq structure

We use shemiq tool to manage the structure of the documents. Load `shemiq` skill for more details.
