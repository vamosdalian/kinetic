package dto

import (
	"encoding/json"
	"time"

	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Workflow struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Config      workflowcfg.WorkflowConfig  `json:"config,omitempty"`
	Tag         string                      `json:"tag,omitempty"`
	Version     int                         `json:"version,string"`
	Enable      bool                        `json:"enable"`
	Trigger     workflowcfg.WorkflowTrigger `json:"trigger"`
	TaskNodes   []TaskNode                  `json:"taskNodes"`
	Edges       []Edge                      `json:"edges"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

type TaskType string

const (
	TaskTypeShell     TaskType = "shell"
	TaskTypeHTTP      TaskType = "http"
	TaskTypeCondition TaskType = "condition"
)

type TaskNode struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Type        TaskType        `json:"type"`
	Config      json.RawMessage `json:"config"`
	Tag         string          `json:"tag,omitempty"`
	Position    Position        `json:"position"`
	NodeType    string          `json:"nodeType"`
}

type Edge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

type WorkflowListItem struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	Enable    bool                        `json:"enable"`
	Trigger   workflowcfg.WorkflowTrigger `json:"trigger"`
	Version   int                         `json:"version,string"`
	CreatedAt time.Time                   `json:"create_at"`
	UpdatedAt time.Time                   `json:"update_at"`
}
