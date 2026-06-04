package dto

import (
	"encoding/json"
)

type WorkflowRun struct {
	WorkflowRunListItem
	Description string        `json:"description"`
	Tag         string        `json:"tag,omitempty"`
	TaskNodes   []TaskNodeRun `json:"taskNodes"`
	Edges       []EdgeRun     `json:"edges"`
}

type TaskNodeRun struct {
	TaskRunID      string          `json:"task_run_id"`
	RunID          string          `json:"run_id"`
	TaskID         string          `json:"task_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Type           TaskType        `json:"type"`
	Config         json.RawMessage `json:"config"`
	Tag            string          `json:"tag,omitempty"`
	EffectiveTag   string          `json:"effective_tag,omitempty"`
	AssignedNodeID string          `json:"assigned_node_id,omitempty"`
	Position       Position        `json:"position"`
	NodeType       string          `json:"nodeType"`
	Status         string          `json:"status"`
	CreatedAt      string          `json:"created_at"`
	AssignedAt     string          `json:"assigned_at"`
	StartedAt      string          `json:"started_at"`
	FinishedAt     string          `json:"finished_at"`
	ExitCode       int             `json:"exit_code"`
	Output         string          `json:"output"`
	Result         string          `json:"result,omitempty"`
	LoopID         string          `json:"loop_id,omitempty"`
	LoopIndex      int             `json:"loop_index"`
	LoopValue      int             `json:"loop_value"`
}

type EdgeRun struct {
	EdgeRunID    string `json:"edge_run_id"`
	RunID        string `json:"run_id"`
	EdgeID       string `json:"edge_id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
	LoopID       string `json:"loop_id,omitempty"`
	LoopIndex    int    `json:"loop_index"`
	LoopValue    int    `json:"loop_value"`
}

type WorkflowRunListItem struct {
	RunID      string `json:"run_id"`
	WorkflowID string `json:"workflow_id"`
	Name       string `json:"name"`
	Version    int    `json:"version,string"`
	Status     string `json:"status"`
	CreatedAt  string `json:"create_at"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}
