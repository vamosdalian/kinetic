# Task Types

Each task type is suited to a different kind of work. This page helps you choose the right one and understand what fields matter most in the editor.

## Quick Comparison

| Task Type | Best For | Common Output |
| --- | --- | --- |
| `shell` | Local scripts, CLI tools, filesystem work | Plain text output and optional JSON result |
| `http` | Calling APIs and webhooks | HTTP status and response body |
| `condition` | Branching based on upstream status or JSON | Branch decision |
| `python` | Reserved for future or limited use | Depends on runtime support |

## Shell

Shell tasks execute a shell script on the worker.

### When To Use It

- Running CLI tools
- Calling local scripts
- Reading or writing local files on the worker
- Producing structured JSON for downstream tasks

### Main Config Fields

- `script`
- `timeout_seconds`
- `retry_count`
- `retry_backoff_seconds`
- `env`

### Example

```json
{
  "script": "printf 'hello world'"
}
```

### Result Output

Shell tasks can write JSON result data to `KINETIC_RESULT_PATH`. If valid JSON is written there, Kinetic stores it as structured task result data.

```sh
printf '{"version":"1.0.0"}' > "$KINETIC_RESULT_PATH"
```

Use this when downstream tasks need structured fields, not just plain text output.

## HTTP

HTTP tasks perform an HTTP request as part of the workflow.

### When To Use It

- Triggering external services
- Sending webhooks
- Querying APIs for status or data
- Passing workflow context into remote systems

### Main Config Fields

- `url`
- `method`
- `headers`
- `body`
- `timeout_seconds`
- `retry_count`
- `retry_backoff_seconds`
- `env`

### Example

```json
{
  "url": "https://api.example.com/jobs",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": "{\"ok\":true}"
}
```

### Behavior

- Successful `2xx` responses are treated as success
- Non-`2xx` responses fail the task
- The task output includes the HTTP status line and response body

If the response body contains JSON, a downstream condition task can inspect it through `json.field` on the upstream output.

## Condition

Condition tasks do not execute an external command. They inspect one upstream result and choose the `true` or `false` branch.

### When To Use It

- Branch on HTTP response JSON
- Stop a path when an upstream result is not acceptable
- Route success and failure paths differently

### Main Config Field

- `expression`

### Example

```json
{
  "expression": "json.ok == true"
}
```

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

### Authoring Advice

- Keep expressions short and readable
- Prefer checking JSON fields instead of parsing plain text manually
- Use templates only when the threshold or comparison value needs to come from runtime context

## Python

Python is defined in the backend config model and runtime API, but current support is limited.

### Current Status

- Backend config type exists
- Frontend editing is not available
- Runtime support should be treated as incomplete unless you have validated it in your environment

## Choosing A Task Type

- Use `shell` when work belongs on a worker machine and needs local tools or scripts
- Use `http` for service-to-service calls
- Use `condition` to branch based on upstream status, output, or JSON data
- Use `python` only if you have explicitly validated that path in your deployment

## Example Combinations

### API Polling Flow

1. HTTP task starts a job
2. HTTP task checks status
3. Condition task branches on `json.state`

### Artifact Processing Flow

1. Shell task downloads or builds an artifact
2. Shell task writes JSON result to `KINETIC_RESULT_PATH`
3. HTTP task posts the result to another system