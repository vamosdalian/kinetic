# Workflow API

## 1. List Workflows
List all workflows with pagination support.

- **Method**: `GET`
- **Path**: `/api/workflows`
- **Description**: Retrieve a paginated list of workflows.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `page` | int | query | No | Page number (default: 1) |
| `pageSize` | int | query | No | Items per page (default: 20, max: 100) |

### Response
```json
{
  "success": true,
  "data": [
    {
      "id": "b18e7753-43c3-4416-9285-xxxxxx",
      "name": "My Workflow",
      "enable": true,
      "version": "1",
      "create_at": "2024-01-01T12:00:00Z",
      "update_at": "2024-01-02T12:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "pageSize": 20,
    "total": 100,
    "totalPages": 5
  }
}
```

---

## 2. Get Workflow
Get detailed information about a specific workflow.

- **Method**: `GET`
- **Path**: `/api/workflows/{id}`
- **Description**: Retrieve full details of a workflow including nodes and edges.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `id` | string | path | Yes | The ID of the workflow. |

### Response
```json
{
  "success": true,
  "data": {
    "id": "b18e7753-43c3-4416-9285-xxxxxx",
    "name": "My Workflow",
    "description": "Workflow description",
    "version": "1",
    "enable": true,
    "taskNodes": [
      {
        "id": "node-1",
        "name": "Shell Task",
        "description": "Run shell script",
        "type": "shell",
        "config": {
          "script": "echo hello"
        },
        "position": {
          "x": 100,
          "y": 200
        },
        "nodeType": "baseNodeFull"
      }
    ],
    "edges": [
      {
        "id": "edge-1",
        "source": "node-1",
        "target": "node-2"
      }
    ],
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-02T12:00:00Z"
  }
}
```

---

## 3. Save Workflow
Create or update a workflow.

- **Method**: `PUT`
- **Path**: `/api/workflows/{id}`
- **Description**: Save workflow details, nodes, and edges.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `id` | string | path | Yes | The ID of the workflow to save. |
| `body` | object | body | Yes | Workflow object. |

### Request Body
```json
{
  "name": "My Workflow",
  "description": "Updated description",
  "enable": true,
  "taskNodes": [
    {
      "id": "node-1",
      "name": "Shell Task",
      "type": "shell",
      "config": {
        "script": "echo updated"
      },
      "position": {
        "x": 120,
        "y": 220
      },
      "nodeType": "baseNodeFull"
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "source": "node-1",
      "target": "node-2"
    }
  ]
}
```

### Response
```json
{
  "success": true,
  "data": null
}
```

---

## 4. Delete Workflow
Delete a workflow.

- **Method**: `DELETE`
- **Path**: `/api/workflows/{id}`
- **Description**: Permanently remove a workflow.

### Parameters
| Name | Type | In | Required | Description |
|---|---|---|---|---|
| `id` | string | path | Yes | The ID of the workflow to delete. |

### Response
```json
{
  "success": true,
  "data": null
}
```
