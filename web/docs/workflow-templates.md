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
- `.workflow.name`
- `.workflow.description`
- `.workflow.config`
- `.workflow.env.KEY`

### task

- The current task being prepared for execution
- `.task.name`
- `.task.type`
- `.task.description`
- `.task.env.KEY`

### runtime

- Execution metadata created by the controller
- `.runtime.runID`
- `.runtime.createdAt`
- `.runtime.startTime`
- `.runtime.env.startTime`

### upstream

Available when the task has one active upstream result.

- `.upstream.status`
- `.upstream.exitCode`
- `.upstream.output`
- `.upstream.outputJSON.field`
- `.upstream.result`
- `.upstream.resultJSON.field`

`previous` is available as an alias of `upstream`.

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

If you need stable downstream access, write structured JSON to `KINETIC_RESULT_PATH` and use `resultJSON` instead of parsing plain text.

## Examples

### Shell Script

```sh
printf '%s' '${{ .workflow.name }}'
```

### HTTP URL

```text
${{ .workflow.env.API_HOST }}/jobs/${{ .runtime.runID }}
```

### Workflow Env Value

```text
service-${{ .task.name }}-${{ .runtime.runID }}
```

### HTTP Body

```json
{
	"workflow": "${{ .workflow.name }}",
	"token": "${{ .upstream.outputJSON.token }}"
}
```

### Condition Expression

```text
json.ok == ${{ .upstream.outputJSON.expected }}
```

### Use Result JSON From A Shell Task

If an upstream shell task writes JSON to `KINETIC_RESULT_PATH`, downstream tasks can read it through `resultJSON`:

```text
${{ .upstream.resultJSON.release.version }}
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

`upstream` is intended for tasks that have one active parent result. If a task has no active upstream result, upstream references are unavailable.

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

If there is no active upstream result for the current task, `upstream` is unavailable.

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
json.retry_count >= ${{ .upstream.outputJSON.threshold }}
```

## Notes

- Missing values fail execution.
- Condition templates must render into a valid condition expression.
- Upstream references are only available when Kinetic can determine a single active upstream result.

## See Also

- [Workflow Basics](workflow-basics.md)
- [Task Types](task-types.md)
- [Runs And Nodes](runs-and-nodes.md)