# Kinetic Docs

Welcome to the Kinetic documentation workspace.

## What This Covers

- Workflow editor basics
- Runtime template syntax
- Common authoring patterns

## Quick Start

1. Open the workflow editor.
2. Create or edit a workflow.
3. Use task and workflow environment variables where needed.
4. Run the workflow and inspect records for rendered results.

## Template Syntax

Kinetic supports runtime templates with the syntax below:

```text
${{ .workflow.name }}
${{ .runtime.runID }}
${{ .upstream.outputJSON.token }}
```

Templates are resolved on the backend before a task starts.

## Next Reading

- [Workflow Templates](workflow-templates.md)