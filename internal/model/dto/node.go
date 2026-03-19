package dto

import "encoding/json"

type NodeTag struct {
	Tag           string `json:"tag"`
	SystemManaged bool   `json:"system_managed"`
}

type Node struct {
	NodeID          string    `json:"node_id"`
	Name            string    `json:"name"`
	IP              string    `json:"ip"`
	Kind            string    `json:"kind"`
	Status          string    `json:"status"`
	MaxConcurrency  int       `json:"max_concurrency"`
	RunningCount    int       `json:"running_count"`
	LastHeartbeatAt string    `json:"last_heartbeat_at"`
	LastStreamAt    string    `json:"last_stream_at"`
	Tags            []NodeTag `json:"tags"`
}

type RegisterNodeRequest struct {
	NodeID         string `json:"node_id"`
	Name           string `json:"name"`
	IP             string `json:"ip"`
	Kind           string `json:"kind"`
	MaxConcurrency int    `json:"max_concurrency"`
}

type NodeHeartbeatRequest struct{}

type AddNodeTagRequest struct {
	Tag string `json:"tag"`
}

type WorkerTaskEvent struct {
	Type       string `json:"type"`
	RunID      string `json:"run_id"`
	TaskID     string `json:"task_id"`
	Status     string `json:"status,omitempty"`
	Output     string `json:"output,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

type NodeCommand struct {
	Type string        `json:"type"`
	Task *AssignedTask `json:"task,omitempty"`
}

type AssignedTask struct {
	RunID          string          `json:"run_id"`
	TaskID         string          `json:"task_id"`
	Name           string          `json:"name"`
	Type           TaskType        `json:"type"`
	Config         json.RawMessage `json:"config"`
	ConditionInput *ConditionInput `json:"condition_input,omitempty"`
}

type ConditionInput struct {
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}
