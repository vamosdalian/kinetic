package dto

type DashboardResponse struct {
	Summary DashboardSummary `json:"summary"`
	Chart   DashboardChart   `json:"chart"`
	Tables  DashboardTables  `json:"tables"`
}

type DashboardSummary struct {
	WorkflowRuns   int     `json:"workflow_runs"`
	TotalWorkflows int     `json:"total_workflows"`
	TotalNodes     int     `json:"total_nodes"`
	SuccessRate    float64 `json:"success_rate"`
}

type DashboardChart struct {
	Range    string                `json:"range"`
	Timezone string                `json:"timezone"`
	Points   []DashboardChartPoint `json:"points"`
}

type DashboardChartPoint struct {
	Date    string `json:"date"`
	Success int    `json:"success"`
	Failure int    `json:"failure"`
}

type DashboardTables struct {
	TodayWorkflows DashboardWorkflowTable     `json:"today_workflows"`
	ScheduledRuns  DashboardScheduledRunTable `json:"scheduled_runs"`
	FailedRuns     DashboardWorkflowTable     `json:"failed_workflows"`
	NodeActivity   DashboardNodeActivityTable `json:"node_activity"`
}

type DashboardWorkflowTable struct {
	Count int                    `json:"count"`
	Items []DashboardWorkflowRow `json:"items"`
}

type DashboardWorkflowRow struct {
	RunID           string `json:"run_id"`
	WorkflowID      string `json:"workflow_id"`
	WorkflowName    string `json:"workflow_name"`
	Status          string `json:"status"`
	StartedAt       string `json:"started_at"`
	DurationSeconds *int   `json:"duration_seconds"`
	NodeLabel       string `json:"node_label"`
}

type DashboardScheduledRunTable struct {
	Count int                        `json:"count"`
	Items []DashboardScheduledRunRow `json:"items"`
}

type DashboardScheduledRunRow struct {
	WorkflowName string `json:"workflow_name"`
	Mode         string `json:"mode"`
	Status       string `json:"status"`
	NextRunAt    string `json:"next_run_at"`
	LastRunAt    string `json:"last_run_at"`
	TargetNode   string `json:"target_node"`
}

type DashboardNodeActivityTable struct {
	Count int                        `json:"count"`
	Items []DashboardNodeActivityRow `json:"items"`
}

type DashboardNodeActivityRow struct {
	NodeID          string  `json:"node_id"`
	NodeName        string  `json:"node_name"`
	Role            string  `json:"role"`
	Status          string  `json:"status"`
	TodayRuns       int     `json:"today_runs"`
	SuccessRate     float64 `json:"success_rate"`
	RunningCount    int     `json:"running_count"`
	LastHeartbeatAt string  `json:"last_heartbeat_at"`
}
