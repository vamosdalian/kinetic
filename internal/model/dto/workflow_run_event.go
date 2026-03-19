package dto

type WorkflowRunEvent struct {
	Type       string `json:"type"`
	RunID      string `json:"run_id,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	Status     string `json:"status,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Output     string `json:"output,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
}
