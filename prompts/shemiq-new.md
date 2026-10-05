---
description: Create new top-level shemiq task
argument-hint: "<task description>"
---
# New shemiq task

Here is a task we need to create a shemiq top-level task for:

    $@

Choose a concise, one-line title for the user's original description above. Load the
`shemiq` skill for CLI availability and installation instructions. From the target
project's working directory, run `shemiq task new --title <title> <description>`:
pass the title as the `--title` value and the **entire original description** as
one argument. Shell-quote/escape both values safely; do not paste the description
unquoted into a shell command or split it into words.

Let the CLI find or create `.shemiq/`, choose the slug, generate the top-level
document (including the original description), and reject collisions. Tell the
user the path printed by the command on success. If installation or task creation
fails, report the error; do not guess a path or create the Markdown file by hand.
