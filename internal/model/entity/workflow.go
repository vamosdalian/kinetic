package entity

import (
	"regexp"
	"strings"
	"time"
)

var nonTaskRefCharacterPattern = regexp.MustCompile(`[^A-Za-z0-9_]`)

type WorkflowEntity struct {
	ID          string
	Name        string
	Description string
	Config      string
	Tag         string
	Version     int
	Enable      bool
	TriggerType string
	TriggerExpr string
	NextRunAt   *time.Time
	LastRunAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaskEntity struct {
	ID          string
	WorkflowID  string
	Ref         string
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

func (t TaskEntity) RefOrDefault() string {
	if strings.TrimSpace(t.Ref) != "" {
		return strings.TrimSpace(t.Ref)
	}
	return DefaultTaskRef(t.ID)
}

func DefaultTaskRef(id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "task"
	}
	ref := nonTaskRefCharacterPattern.ReplaceAllString(trimmed, "_")
	return "task_" + strings.Trim(ref, "_")
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
