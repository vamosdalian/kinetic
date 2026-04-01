# Workflow Basics

Kinetic workflows are built as a graph of task nodes connected by directed edges. This page explains what you create in the editor and how Kinetic interprets it at runtime.

## Before You Start

When you open the workflow editor, think in terms of steps and transitions:

- Each node is one step
- Each edge defines which step runs next
- A workflow starts from tasks with no incoming edges
- A workflow ends when there are no more active downstream tasks

## Main Building Blocks

### Workflow

A workflow contains:

- Name
- Description
- Optional tag
- Workflow-level config
- Task nodes
- Edges between nodes

### Task Node

A task node represents one unit of work. In the editor, you usually give it a clear name, choose a type, and fill in its config.

### Edge

An edge defines execution order. If task A points to task B, task B can only run after task A completes on an active path.

## How To Build A Workflow In The UI

1. Create a new workflow.
2. Fill in the workflow name and description.
3. Add task nodes to the canvas.
4. Connect nodes with edges.
5. Click a task to edit its details in the side panel.
6. Save the workflow before running it.

If you are creating a new workflow pattern, start with the smallest version that proves the path works.

## Supported Task Types

- `shell`
- `http`
- `python`
- `condition`

See [Task Types](task-types.md) for details.

## Workflow-Level Settings

Workflow-level settings are shared values that apply across the graph.

### Workflow Environment Variables

Use workflow environment variables for values that are reused by multiple tasks, such as:

- Base URLs
- Shared tokens or identifiers
- Environment labels
- Common runtime options

## Validation Rules

Kinetic validates workflow definitions before execution.

- Every task must have a valid config
- Workflow graphs must not contain cycles
- Condition nodes must have exactly one incoming edge
- Condition nodes must have exactly two outgoing edges
- Condition outgoing edges must use `true` and `false` source handles

These rules protect you from saving a graph that cannot be executed correctly.

## Workflow And Task Config

### Workflow Config

Current workflow-level config fields:

- `env`

Workflow env values are inherited by tasks unless overridden.

This is the best place for shared values.

### Task Policy

Most executable task types support policy fields inside task config:

- `timeout_seconds`
- `retry_count`
- `retry_backoff_seconds`
- `env`

These settings control how Kinetic executes the task, not what the task does.

## Timeout And Retry

Use timeout and retry deliberately:

- Add `timeout_seconds` when a task should not run forever
- Add `retry_count` when transient failures are expected
- Add `retry_backoff_seconds` when the target system may need a short recovery period

## Tags And Routing

Workflows and tasks can carry tags. Use them when different work must run on different nodes.

### Workflow Tag

Acts as a default tag for the workflow.

### Task Tag

Overrides the workflow tag for that specific task.

If you do not need special routing, keep your tag strategy simple.

## Environment Precedence

At runtime, task environment values are assembled from multiple sources.

1. System-provided `KINETIC_*` values
2. Workflow env values
3. Task env values

Current reserved variables include:

- `KINETIC_WORKFLOW_NAME`
- `KINETIC_TASK_NAME`
- `KINETIC_RESULT_PATH`

Keys starting with `KINETIC_` are reserved and cannot be defined by users.

If you see a validation error around a reserved name, rename your custom variable.

## Authoring Tips

- Start with one or two shell tasks before building larger graphs
- Add condition nodes only when you need explicit branching
- Keep task names stable and descriptive for easier run debugging
- Put shared values in workflow env and only override when necessary

## Common Modeling Patterns

### Linear Workflow

Use a straight line of tasks when each step must finish before the next begins.

### HTTP Then Condition

Use an HTTP task to fetch data, then a condition task to branch based on the response.

### Shell Produces Result, Downstream Consumes It

Use a shell task to write structured JSON result data, then reference it from downstream templates.

## See Also

- [Task Types](task-types.md)
- [Workflow Templates](workflow-templates.md)
- [Runs And Nodes](runs-and-nodes.md)