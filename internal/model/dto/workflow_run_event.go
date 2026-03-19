package dto

type WorkflowRunEvent struct {
	Type           string `json:"type"`
	RunID          string `json:"run_id,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	Status         string `json:"status,omitempty"`
	AssignedNodeID string `json:"assigned_node_id,omitempty"`
	EffectiveTag   string `json:"effective_tag,omitempty"`
	AssignedAt     string `json:"assigned_at,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	Output         string `json:"output,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
}
