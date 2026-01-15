# Workflow Run API

## 1. Run Workflow
Trigger a new run for a specific workflow.

- **Method**: `POST`
- **Path**: `/api/workflows/{id}/run`
- **Description**: Creates a new workflow run instance.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `id` | string | path | Yes | The ID of the workflow to run. |

### Response
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "run_id": "generated-uuid"
  }
}
```

---

## 2. List Workflow Runs
List workflow runs with pagination.

- **Method**: `GET`
- **Path**: `/api/workflow_runs`
- **Description**: Retrieves a list of recent workflow runs.

### Parameters
| Name | Type | In | Required | Default | Description |
|---|---|---|---|---|---|
| `page` | int | query | No | 1 | Page number. |
| `pageSize` | int | query | No | 10 | Number of items per page. |

### Response
```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "run_id": "string",
      "workflow_id": "string",
      "name": "string",
      "version": "string",
      "status": "string",
      "create_at": "string (RFC3339)",
      "started_at": "string (RFC3339)",
      "finished_at": "string (RFC3339)"
    }
  ]
}
```

---

## 3. Get Workflow Run
Get detailed information about a specific workflow run.

- **Method**: `GET`
- **Path**: `/api/workflow_runs/{run_id}`
- **Description**: Retrieves detailed info including task node execution status and edges.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `run_id` | string | path | Yes | The ID of the workflow run. |

### Response
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "run_id": "string",
    "workflow_id": "string",
    "name": "string",
    "version": "string",
    "status": "string",
    "create_at": "string (RFC3339)",
    "started_at": "string (RFC3339)",
    "finished_at": "string (RFC3339)",
    "description": "string",
    "taskNodes": [
      {
        "run_id": "string",
        "task_id": "string",
        "name": "string",
        "description": "string",
        "type": "string",
        "config": {},
        "position": {
          "x": 0,
          "y": 0
        },
        "nodeType": "string",
        "status": "string",
        "created_at": "string (RFC3339)",
        "started_at": "string (RFC3339)",
        "finished_at": "string (RFC3339)",
        "exit_code": 0
      }
    ],
    "edges": [
      {
        "run_id": "string",
        "edge_id": "string",
        "source": "string",
        "target": "string",
        "sourceHandle": "string",
        "targetHandle": "string"
      }
    ]
  }
}
```
