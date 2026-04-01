# Workflow Templates

Kinetic uses Go `text/template` syntax with custom delimiters:

```text
${{ ... }}
```

## Available Scopes

### workflow

- `.workflow.name`
- `.workflow.description`
- `.workflow.config`
- `.workflow.env.KEY`

### task

- `.task.name`
- `.task.type`
- `.task.description`
- `.task.env.KEY`

### runtime

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

## Examples

### Shell Script

```sh
printf '%s' '${{ .workflow.name }}'
```

### HTTP URL

```text
${{ .workflow.env.API_HOST }}/jobs/${{ .runtime.runID }}
```

### Condition Expression

```text
json.ok == ${{ .upstream.outputJSON.expected }}
```

## Notes

- Missing values fail execution.
- Condition templates must render into a valid condition expression.
- Upstream references are only available when Kinetic can determine a single active upstream result.