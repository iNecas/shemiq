---
name: shemiq
description: Metadata tracking in .shemiq/** project files. Load always when working with files in that directory.
---
# Summary

Shemiq is a tool and file format for keeping track of the tasks by leveraging appropriate metadata.

It uses ``:::shemiq` markdown directive to keep metatada about the tasks and links between them.

## Common structure

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
    status: todo
    :::

    ### Put the things in place
    :::shemiq
    type: task
    source: ./put-the-things-in-place.md
    status: todo
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

```
:::shemiq
status: [new|progress|done]
:::
```
