---
description: Create a new subtask in a shemiq top-level task
argument-hint: "<parent-path> <after-title> <description>"
---
# New shemiq subtask

We need to add a new subtask to an existing shemiq top-level task.

- Parent task document: $1
- Insert after the sibling subtask titled: $2
- Original description of the new subtask:

    $3

Choose a concise, one-line title for the user's original description above. Load
the `shemiq` skill for CLI availability and installation instructions. From the
target project's working directory, run `shemiq new-subtask --title <title>
--after <after> <parent> <description>`: pass the chosen title as the `--title`
value, the sibling title above as the `--after` value, the parent path above as
the `<parent>` argument, and the **entire original description** as the final
argument. Shell-quote/escape all values safely; do not paste the description
unquoted into a shell command or split it into words.

Let the CLI insert the subtask heading and description into the parent document
and reject duplicates or invalid structure. Tell the user the path printed by
the command on success. If installation or subtask creation fails, report the
error; do not guess a path or edit the Markdown file by hand.
