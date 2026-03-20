package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func (s *SqliteDB) ListWorkflowRunsByCreatedAt(start time.Time, end time.Time) ([]entity.WorkflowRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, workflow_tag,
		       status, created_at, started_at, finished_at
		FROM workflow_runs
		WHERE created_at >= ? AND created_at < ?
		ORDER BY created_at DESC, run_id DESC
	`, start.UTC().Format("2006-01-02 15:04:05"), end.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []entity.WorkflowRunEntity
	for rows.Next() {
		run, err := scanWorkflowRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *SqliteDB) ListTaskRunsByRunIDs(runIDs []string) ([]entity.TaskRunEntity, error) {
	if len(runIDs) == 0 {
		return []entity.TaskRunEntity{}, nil
	}

	query := `
		SELECT run_id, task_id, workflow_id, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs
		WHERE run_id IN (` + placeholders(len(runIDs)) + `)
		ORDER BY run_id ASC, task_id ASC
	`

	args := make([]any, 0, len(runIDs))
	for _, runID := range runIDs {
		args = append(args, runID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		task, err := scanTaskRun(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) ListTaskRunsByAssignedAt(start time.Time, end time.Time) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, task_id, workflow_id, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs
		WHERE assigned_at IS NOT NULL AND assigned_at >= ? AND assigned_at < ?
		ORDER BY assigned_at DESC, task_id ASC
	`, start.UTC().Format("2006-01-02 15:04:05"), end.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		task, err := scanTaskRun(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func scanWorkflowRun(scanner interface{ Scan(dest ...any) error }) (entity.WorkflowRunEntity, error) {
	var run entity.WorkflowRunEntity
	var createdAt sql.NullString
	var startedAt sql.NullString
	var finishedAt sql.NullString

	err := scanner.Scan(
		&run.RunID,
		&run.WorkflowID,
		&run.WorkflowName,
		&run.WorkflowDescription,
		&run.WorkflowVersion,
		&run.WorkflowTag,
		&run.Status,
		&createdAt,
		&startedAt,
		&finishedAt,
	)
	if err != nil {
		return entity.WorkflowRunEntity{}, err
	}

	run.CreatedAt = *parseNullableTime(createdAt)
	run.StartedAt = parseNullableTime(startedAt)
	run.FinishedAt = parseNullableTime(finishedAt)
	return run, nil
}

func scanTaskRun(scanner interface{ Scan(dest ...any) error }) (entity.TaskRunEntity, error) {
	var task entity.TaskRunEntity
	var createdAt string
	var assignedAt sql.NullString
	var startedAt sql.NullString
	var finishedAt sql.NullString

	err := scanner.Scan(
		&task.RunID,
		&task.TaskID,
		&task.WorkflowID,
		&task.TaskName,
		&task.TaskDescription,
		&task.TaskType,
		&task.TaskConfig,
		&task.TaskTag,
		&task.TaskPosition,
		&task.TaskNodeType,
		&task.EffectiveTag,
		&task.AssignedNodeID,
		&assignedAt,
		&task.Status,
		&createdAt,
		&startedAt,
		&finishedAt,
		&task.ExitCode,
		&task.Output,
	)
	if err != nil {
		return entity.TaskRunEntity{}, err
	}

	task.CreatedAt, _ = parseTime(createdAt)
	task.AssignedAt = parseNullableTime(assignedAt)
	task.StartedAt = parseNullableTime(startedAt)
	task.FinishedAt = parseNullableTime(finishedAt)
	return task, nil
}

func placeholders(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}
