# Runs And Nodes

This guide explains how to read workflow runs in the UI and how node information helps you understand execution.

## Workflow Runs

A workflow run is created every time you execute a workflow. Kinetic snapshots workflow metadata and creates task run records for each task in the graph.

In daily use, this is the main object you inspect after clicking run.

### Typical Run Lifecycle

- `created`
- `running`
- `success`
- `failed`
- `cancelled`

## Task Runs

Each task in the workflow gets its own task run record.

Open a task run when you want to answer one of these questions:

- Did the task actually start?
- Did it run on the expected node?
- What output did it produce?
- Did it fail during execution or before execution began?

### Common Task Statuses

- `pending`
- `queued`
- `assigned`
- `running`
- `success`
- `failed`
- `skipped`
- `cancelled`
- `unknown`

## Output And Result

### Output

Output is the plain text log captured during task execution.

### Result

Result is structured data captured separately from shell tasks through `KINETIC_RESULT_PATH`.

Use output for logs and result for machine-readable data that downstream tasks may consume.

As a rule of thumb:

- Use `output` for humans
- Use `result` for downstream automation

## Record Page

The `Record` area is the main place to inspect workflow runs.

### What To Look For

- Overall run status
- Per-task status
- Task output
- Structured result data
- Branching behavior for condition nodes

## How To Read A Failed Run

1. Confirm whether the workflow failed or was cancelled.
2. Find the first failed task, not just the last visible task.
3. Open the task output and read from the top of the failing attempt.
4. Check whether the failure came from the task itself, template resolution, timeout, or routing.
5. If the workflow contains condition nodes, verify which branch was activated.

## Node Page

The `Node` page shows registered worker nodes and operational details.

### Key Fields

- Node ID
- Name
- IP
- Status
- Max concurrency
- Running count
- Tags
- Heartbeat timestamp
- Stream timestamp

If a task did not start where you expected, the node page is the first place to check.

## Tags And Assignment

Tags are used to route tasks in distributed mode.

- Workflow tag acts as the default
- Task tag overrides the workflow tag
- Matching nodes receive the assignment

If a task is waiting too long, compare the task tag with the tags available on active nodes.

## Failure Investigation Checklist

1. Check the workflow run status first.
2. Open the failed task.
3. Read the task output for the first real error, not just the final retry notice.
4. If templates are involved, confirm the referenced values actually exist.
5. If the task was skipped, inspect the upstream condition branch.
6. If the task stayed queued or unknown, inspect node availability and tags.

## Interpreting Common Statuses

### queued

The task is ready but waiting for an eligible node.

### assigned

The task has been routed to a node and is waiting to start.

### unknown

The task was assigned, but the worker state became uncertain, usually because the worker disconnected.

### skipped

The task did not run because an upstream failure or inactive condition branch prevented execution.

## Common Failure Patterns

### Template Resolution Failure

Cause:
Referenced value does not exist.

Typical symptom:
Task fails before actual execution starts.

What to check:

- The referenced scope exists
- The referenced key exists
- The task actually has an active upstream result if `upstream` is used

### HTTP Task Failure

Cause:
Non-`2xx` response or connection error.

Typical symptom:
Task output begins with an HTTP status line and error body.

What to check:

- URL and method
- Header values
- Body content
- Upstream values used in templates

### Shell Task Failure

Cause:
Script exits with a non-zero code.

Typical symptom:
Task output shows command stderr or partial logs before failure.

What to check:

- Shell command syntax
- Required tools on the worker
- Environment variable values
- Timeout and retry settings

### Node Routing Issue

Cause:
No eligible node matches the task tag, or a node goes offline.

Typical symptom:
Task remains queued, assigned, or moves to `unknown`.

What to check:

- Task tag and workflow tag
- Node status
- Node heartbeat recency
- Node concurrency saturation

## See Also

- [Workflow Basics](workflow-basics.md)
- [Task Types](task-types.md)
- [Workflow Templates](workflow-templates.md)