package database

import (
	"context"

	"github.com/vamosdalian/kinetic/internal/model/entity"
	_ "modernc.org/sqlite"
)

type Database interface {
	Close() error
	HealthCheck(ctx context.Context) error

	// Workflow
	ListWorkflows(offset int, limit int) ([]entity.WorkflowEntity, error)
	CountWorkflows() (int, error)
	ListWorkflowsFiltered(offset int, limit int, query string) ([]entity.WorkflowEntity, error)
	CountWorkflowsFiltered(query string) (int, error)
	GetWorkflowByID(id string) (entity.WorkflowEntity, error)
	SaveWorkflow(req entity.WorkflowEntity) error
	SaveWorkflowDefinition(workflow entity.WorkflowEntity, tasks []entity.TaskEntity, edges []entity.EdgeEntity) error
	DeleteWorkflow(id string) error
	DeleteWorkflowDefinition(id string) (bool, error)
	GetUserByID(id string) (entity.UserEntity, error)
	GetUserByUsername(username string) (entity.UserEntity, error)
	ListUsers() ([]entity.UserEntity, error)
	CreateUser(user entity.UserEntity) (entity.UserEntity, error)
	UpdateUserPassword(userID string, passwordHash string) error
	UpsertUser(user entity.UserEntity) (entity.UserEntity, error)
	DeleteUser(userID string) error
	ListNodes() ([]entity.NodeEntity, error)
	GetNodeByID(nodeID string) (entity.NodeEntity, error)
	UpsertNode(node entity.NodeEntity) error
	SetNodeStatus(nodeID string, status string) error
	UpdateNodeHeartbeat(nodeID string) error
	UpdateNodeStream(nodeID string) error
	ListNodeTags(nodeID string) ([]entity.NodeTagEntity, error)
	SaveNodeTag(tag entity.NodeTagEntity) error
	DeleteNodeTag(nodeID string, tag string) error

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

	// Workflow Run
	CreateWorkflowRun(workflowID string, runID string) error
	GetWorkflowRun(runID string) (entity.WorkflowRunEntity, error)
	GetTaskRun(runID string, taskID string) (entity.TaskRunEntity, error)
	GetTaskRuns(runID string) ([]entity.TaskRunEntity, error)
	GetEdgeRuns(runID string) ([]entity.EdgeRunEntity, error)
	ListWorkflowRuns(offset int, limit int) ([]entity.WorkflowRunEntity, error)
	CountWorkflowRuns() (int, error)
	ListWorkflowRunsFiltered(offset int, limit int, workflowQuery string, runQuery string, status string) ([]entity.WorkflowRunEntity, error)
	CountWorkflowRunsFiltered(workflowQuery string, runQuery string, status string) (int, error)
	MarkWorkflowRunRunning(runID string) error
	FinishWorkflowRun(runID string, status string) error
	UpdateWorkflowRunStatus(runID string, status string) error
	MarkTaskRunRunning(runID string, taskID string) error
	QueueTaskRun(runID string, taskID string, effectiveTag string) error
	AssignTaskRun(runID string, taskID string, nodeID string) error
	ResetAssignedTaskRun(runID string, taskID string) error
	MarkTaskRunUnknown(runID string, taskID string, output string) error
	FinishTaskRun(runID string, taskID string, status string, exitCode int, output string, result string) error
	SkipPendingTaskRuns(runID string, output string) error
	CancelPendingTaskRuns(runID string, output string) error
	AppendTaskRunOutput(runID string, taskID string, chunk string) error
	ListQueuedTaskRuns(limit int) ([]entity.TaskRunEntity, error)
	ListNodeActiveTaskRuns(nodeID string) ([]entity.TaskRunEntity, error)
	IncrementNodeRunningCount(nodeID string) error
	DecrementNodeRunningCount(nodeID string) error
}
