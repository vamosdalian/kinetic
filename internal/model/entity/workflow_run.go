package entity

import "time"

type WorkflowRunEntity struct {
	RunID               string
	WorkflowID          string
	WorkflowName        string
	WorkflowDescription string
	WorkflowVersion     int
	Status              string
	CreatedAt           time.Time
	StartedAt           *time.Time
	FinishedAt          *time.Time
}

type TaskRunEntity struct {
	RunID           string
	TaskID          string
	WorkflowID      string
	TaskName        string
	TaskDescription string
	TaskType        string
	TaskConfig      string // json string
	TaskPosition    string // json string
	TaskNodeType    string
	Status          string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	ExitCode        int
	Output          string
}

type EdgeRunEntity struct {
	RunID            string
	EdgeID           string
	WorkflowID       string
	EdgeSource       string
	EdgeTarget       string
	EdgeSourceHandle string
	EdgeTargetHandle string
	CreatedAt        time.Time
}
