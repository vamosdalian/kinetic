# Kinetic

[![Release](https://img.shields.io/github/v/release/vamosdalian/kinetic)](https://github.com/vamosdalian/kinetic/releases)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![GitHub Issues](https://img.shields.io/github/issues/vamosdalian/kinetic)](https://github.com/vamosdalian/kinetic/issues)
[![GitHub Stars](https://img.shields.io/github/stars/vamosdalian/kinetic?style=social)](https://github.com/vamosdalian/kinetic/stargazers)

Kinetic is a lightweight workflow orchestration system with a built-in web UI, HTTP API, scheduler, and distributed worker model. It is designed to be easy to deploy for small teams while still supporting controller/worker separation when you need to run tasks on remote nodes.

## Features

- Built-in web UI served by the backend binary
- Workflow graph editing and execution tracking
- Task types for `shell`, `http`, `python`, and `condition`
- Controller and worker run modes
- Embedded worker support for single-node deployments
- Distributed execution with node registration, heartbeats, and task streaming
- SQLite-backed persistence
- Real-time workflow run event streaming

## Architecture

Kinetic supports two runtime modes:

- `controller`: runs the API server, scheduler, persistence layer, and optionally an embedded local worker
- `worker`: connects to a controller and executes assigned tasks

For local development or a simple self-hosted install, the default controller mode with an embedded worker is the fastest way to get started. For multi-node execution, run one controller and attach one or more workers.

## Quick Start

### Prerequisites

- Go `1.23+`
- Node.js `22+`
- npm

### Run Locally

Build the frontend first so the static assets can be embedded into the Go binary:

```bash
cd web
npm ci
npm run build
cd ..
go run ./cmd/kinetic --mode controller --with-worker
```

Then open:

- UI: [http://localhost:8080](http://localhost:8080)
- Health check: [http://localhost:8080/healthz](http://localhost:8080/healthz)
- Readiness check: [http://localhost:8080/readyz](http://localhost:8080/readyz)

On first start, Kinetic creates a default config file at `~/.kinetic/config.yaml` and a SQLite database at `~/.kinetic/kinetic.db`.

### Build a Binary

```bash
cd web
npm ci
npm run build
cd ..
go build -o kinetic ./cmd/kinetic
./kinetic --mode controller --with-worker
```

## Deployment Modes

### Single Node

Run one controller with the embedded worker enabled:

```bash
./kinetic --mode controller --with-worker
```

### Distributed

Run a controller without the embedded worker:

```bash
KINETIC_MODE=controller \
KINETIC_CONTROLLER_EMBEDDED_WORKER_ENABLED=false \
./kinetic
```

Run a worker that connects to the controller:

```bash
KINETIC_MODE=worker \
KINETIC_WORKER_CONTROLLER_URL=http://controller-host:8080 \
./kinetic
```

## Configuration

Kinetic loads configuration from `~/.kinetic/config.yaml` and allows every field to be overridden with environment variables.

Example configuration:

```yaml
mode: controller

api:
  host: 0.0.0.0
  port: 8080

database:
  type: sqlite
  path: /home/your-user/.kinetic/kinetic.db

controller:
  embedded_worker_enabled: true

worker:
  id: node-local
  name: node-local
  controller_url: http://localhost:8080
  advertise_ip: ""
  heartbeat_interval: 5
  stream_reconnect_interval: 5
  max_concurrency: 10

log:
  level: info
  format: text
```

Common environment variable overrides:

- `KINETIC_MODE`
- `KINETIC_API_HOST`
- `KINETIC_API_PORT`
- `KINETIC_DATABASE_PATH`
- `KINETIC_CONTROLLER_EMBEDDED_WORKER_ENABLED`
- `KINETIC_WORKER_CONTROLLER_URL`
- `KINETIC_WORKER_MAX_CONCURRENCY`
- `KINETIC_LOG_LEVEL`
- `KINETIC_LOG_FORMAT`

## Workflow Model

Kinetic workflows are stored as graph definitions made up of task nodes and edges.

Supported task types:

- `shell`: run shell scripts on a worker
- `http`: make HTTP requests as workflow steps
- `python`: execute Python scripts
- `condition`: branch execution based on an expression

Validation rules include:

- every task must have a valid config
- condition nodes must have exactly one incoming edge
- condition nodes must have exactly two outgoing edges with `true` and `false` handles
- workflow graphs must be acyclic

Workflows and tasks can also carry tags so runs can be routed to matching worker nodes.

## API Overview

Main API groups:

- `GET /healthz`
- `GET /readyz`
- `GET /api/workflows`
- `GET /api/workflows/:id`
- `PUT /api/workflows/:id`
- `DELETE /api/workflows/:id`
- `POST /api/workflows/:id/run`
- `GET /api/workflow_runs`
- `GET /api/workflow_runs/:run_id`
- `GET /api/workflow_runs/:run_id/events`
- `POST /api/workflow_runs/:run_id/rerun`
- `POST /api/workflow_runs/:run_id/cancel`
- `GET /api/nodes`

## Development

### Backend

Run tests:

```bash
go test ./...
```

### Frontend

Start the Vite dev server:

```bash
cd web
npm ci
npm run dev
```

Run frontend tests:

```bash
cd web
npm test
```

Build frontend assets:

```bash
cd web
npm run build
```

## Project Structure

```text
cmd/kinetic/          CLI entrypoint
internal/api-server/  HTTP API and static asset serving
internal/controller/  Controller bootstrap and lifecycle
internal/service/     Workflow run orchestration and node coordination
internal/worker/      Remote/local worker runtime
internal/database/    Persistence abstractions and SQLite implementation
internal/workflow/    Workflow config parsing and validation
web/                  React + Vite frontend
```

## Release

GitHub Releases are built from tagged commits. Release assets are attached automatically, and the release page includes both a short custom highlights section and GitHub-generated release notes.
