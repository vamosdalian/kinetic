package entity

import (
	"time"
)

type WorkflowEntity struct {
	ID          string
	Name        string
	Description string
	Version     int
	Enable      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaskEntity struct {
	ID          string
	WorkflowID  string
	Name        string
	Description string
	Type        string
	Config      string // json string
	Position    string // json string
	NodeType    string
}

type EdgeEntity struct {
	ID           string
	WorkflowID   string
	Source       string
	Target       string
	SourceHandle string
	TargetHandle string
}
