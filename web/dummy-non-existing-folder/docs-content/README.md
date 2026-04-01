# Kinetic Docs

This is the Kinetic user manual. It is written for people who use the web UI to create workflows, run them, and review the results.

## What This Manual Covers

- How to build a workflow in the editor
- How to choose the right task type
- How to use runtime templates in fields and environment variables
- How to read run records and task output
- How to understand node assignment and routing

## Start Here

- [Workflow Basics](workflow-basics.md)
- [Task Types](task-types.md)
- [Workflow Templates](workflow-templates.md)
- [Runs And Nodes](runs-and-nodes.md)

## Product Areas In The UI

### Dashboard

Use the dashboard to get a quick summary of recent workflow activity, success and failure trends, and recent runs that may need attention.

### Workflow

Use the workflow page to create and edit workflow graphs. This is where you add task nodes, connect them, and configure task details.

### Record

Use the record page to inspect workflow runs. It shows status, task output, branch behavior, and any structured result written by tasks.

### Node

Use the node page to inspect available execution nodes, current status, concurrency, and tags used for routing.

### Admin

The admin page is reserved for management features and operational settings.

## Main Concepts

### Workflow

A workflow is the full automation flow. It contains task nodes, edges, workflow-level environment variables, and optional routing tags.

### Task

A task is one step in the workflow. Each task has a type, a config object, and optional execution policy fields such as timeout or retry.

### Run

A run is one execution of a workflow. Every task in the workflow produces its own task run record during that execution.

### Node

A node is a machine or worker process that can execute tasks. Tags let you control where work is allowed to run.

## Typical User Flow

1. Open `Workflow` and create a workflow.
2. Add one or more tasks.
3. Configure scripts, HTTP requests, or condition expressions.
4. Save the workflow.
5. Run the workflow.
6. Open `Record` to inspect the result.
7. If needed, open `Node` to confirm routing and worker availability.

## Good First Workflow

For your first workflow, keep it simple:

1. Create one `shell` task.
2. Set the script to print a short message.
3. Run the workflow.
4. Open the run record and confirm the output.

Once that works, add a second task and connect them with one edge.

## Recommended Reading Order

1. Read [Workflow Basics](workflow-basics.md) to understand the graph model.
2. Read [Task Types](task-types.md) to understand what each node can do.
3. Read [Workflow Templates](workflow-templates.md) when you need dynamic values.
4. Read [Runs And Nodes](runs-and-nodes.md) when you want to inspect and debug execution.