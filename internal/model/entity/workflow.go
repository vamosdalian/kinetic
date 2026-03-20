package entity

import (
	"strings"
	"time"
)

type WorkflowEntity struct {
	ID          string
	Name        string
	Description string
	Config      string
	Tag         string
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
	Tag         string
	Position    string // json string
	NodeType    string
}

func (t TaskEntity) NameOrID() string {
	if strings.TrimSpace(t.Name) != "" {
		return t.Name
	}
	return t.ID
}

type EdgeEntity struct {
	ID           string
	WorkflowID   string
	Source       string
	Target       string
	SourceHandle string
	TargetHandle string
}

type NodeEntity struct {
	NodeID          string
	Name            string
	IP              string
	Kind            string
	Status          string
	MaxConcurrency  int
	RunningCount    int
	LastHeartbeatAt *time.Time
	LastStreamAt    *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type NodeTagEntity struct {
	NodeID        string
	Tag           string
	SystemManaged bool
	CreatedAt     time.Time
}
