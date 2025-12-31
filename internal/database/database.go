package database

import (
	"github.com/vamosdalian/kinetic/internal/model/entity"
	_ "modernc.org/sqlite"
)

type Database interface {
	Close() error

	// Workflow
	ListWorkflows(offset int, limit int) ([]entity.WorkflowEntity, error)
	CountWorkflows() (int, error)
	GetWorkflowByID(id string) (entity.WorkflowEntity, error)
	SaveWorkflow(req entity.WorkflowEntity) error
	DeleteWorkflow(id string) error

	// Task
	ListTasks(workflowID string) ([]entity.TaskEntity, error)
	SaveTasks(req []entity.TaskEntity) ([]entity.TaskEntity, error)
	DeleteTask(id string) error
	DeleteTasks(workflowID string) error

	// Edge
	ListEdges(workflowID string) ([]entity.EdgeEntity, error)
	SaveEdges(req []entity.EdgeEntity) ([]entity.EdgeEntity, error)
	DeleteEdges(workflowID string) error
	DeleteEdge(id string) error
}
