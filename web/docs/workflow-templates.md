# Workflow Templates

Kinetic templates let you place runtime values into task fields and environment variables. They are most useful when one task depends on workflow metadata, run metadata, or upstream output.

Kinetic uses Go `text/template` syntax with custom delimiters:

```text
${{ ... }}
```

## Available Scopes

Think of scopes as the data sources you can read from while Kinetic prepares a task.

### workflow

- Workflow metadata and config snapshot for the current run
- `.workflow.id`
- `.workflow.name`
- `.workflow.description`
- `.workflow.version`

### task

- The current task being prepared for execution
- `.task.id`
- `.task.ref`
- `.task.name`
- `.task.description`
- `.task.type`

### runtime

- Execution metadata created by the controller
- `.runtime.workflow.runid`
- `.runtime.workflow.createdAt`
- `.runtime.workflow.startedAt`
- `.runtime.task.runid`
- `.runtime.task.createdAt`
- `.runtime.task.startedAt`
- `.runtime.loop.name`
- `.runtime.loop.index`
- `.runtime.loop.value`
- `.runtime.loop.var`

### upstream

Upstream values are keyed by task `ref`. Task refs must be unique in a workflow and use identifier-safe names such as `build`, `deploy_prod`, or `check_1`.

- `.upstream.<ref>.exitCode`
- `.upstream.<ref>.output`
- `.upstream.<ref>.result`
- `.upstream.<ref>.result.key` when the result is valid JSON

Example:

```text
${{ .upstream.build.result.version }}
```

When an upstream task has multiple successful runs, such as a task inside a loop consumed outside that loop, the ref value is a list. Use Go template's `index` function to choose one run, and refer to the official Go template syntax for more advanced access patterns:

```text
${{ index .upstream.build 0 }}
```

## When To Use Templates

Use templates when a field should change from run to run, task to task, or based on upstream output.

Common examples:

- Build an HTTP URL from workflow env and run ID
- Put upstream JSON values into an HTTP body
- Use a dynamic comparison value in a condition expression
- Derive a task env value from workflow metadata

## Where Templates Are Supported

Templates are evaluated in string fields at task preparation time.

### Supported Today

- Workflow env values in `workflow.config.env`
- Task env values in `task.config.env`
- Shell script content
- HTTP URL
- HTTP method
- HTTP header values
- HTTP body
- Condition expressions

### Important Boundaries

- Missing values fail execution
- Non-string fields are not templated
- Header names should stay static
- Templates must render to a valid final value for the field they are used in

## Writing Templates Safely

### Prefer Shared Values In Workflow Env

If several tasks need the same value, store it once in workflow env and reference it from tasks.

### Keep Upstream Dependencies Obvious

If a task depends on upstream output, make that visible in the task name or description.

### Use Result JSON For Structured Data

If you need stable downstream access, write structured JSON to `KINETIC_RESULT_PATH` and read parsed fields from `.upstream.<ref>.result` instead of parsing plain text.

## Examples

### Shell Script

```sh
printf '%s' '${{ .workflow.name }}'
```

### HTTP URL

```text
https://api.example.com/jobs/${{ .runtime.workflow.runid }}
```

### Task Ref

```text
service-${{ .task.ref }}-${{ .runtime.task.runid }}
```

### HTTP Body

```json
{
	"workflow": "${{ .workflow.name }}",
	"token": "${{ .upstream.auth.result.token }}"
}
```

### Condition Expression

```text
json.ok == ${{ .upstream.check.result.expected }}
```

### Use Result JSON From A Shell Task

If an upstream shell task with ref `build` writes JSON to `KINETIC_RESULT_PATH`, downstream tasks can read parsed fields directly under `result`:

```text
${{ .upstream.build.result.release.version }}
```

## Template Behavior

### Resolution Order

1. Workflow env values are rendered for the current task
2. Task env values are rendered
3. The task config is rendered and then parsed by the task runtime

### Missing Values

Kinetic uses strict template evaluation. If a value does not exist, task preparation fails and the task run is marked failed.

This is intentional. It prevents a task from running with silently broken input.

### Upstream Availability

`upstream` is built from active direct upstream task runs. If a task has no active upstream result for the requested ref, that reference is unavailable.

## Common Mistakes

### Referencing A Missing Key

Example:

```text
${{ .workflow.env.API_TOKEN }}
```

If `API_TOKEN` does not exist, task preparation fails.

### Using A Template In The Wrong Place

Templates only apply to string fields. They do not convert non-string config fields into dynamic values.

### Expecting Upstream Data Without An Active Parent

If there is no active upstream result for the requested ref, that `upstream` entry is unavailable.

## Condition Expressions

Condition expressions still use Kinetic's own condition language after templates are rendered.

### Supported Operands

- `status`
- `exit_code`
- `output`
- `json`
- `json.field`

### Supported Operators

- `contains`
- `==`
- `!=`
- `>`
- `<`
- `>=`
- `<=`

### Example

```text
json.retry_count >= ${{ .upstream.check.result.threshold }}
```

## Notes

- Missing values fail execution.
- Condition templates must render into a valid condition expression.
- Upstream references use task refs, not task names.

## See Also

- [Workflow Basics](workflow-basics.md)
- [Task Types](task-types.md)
- [Runs And Nodes](runs-and-nodes.md)