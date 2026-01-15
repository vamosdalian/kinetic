package sqlite

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func (s *SqliteDB) CreateWorkflowRun(workflowID string, runID string) error {
	workflow, err := s.GetWorkflowByID(workflowID)
	if err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO workflow_runs (
			run_id, workflow_id, workflow_name, workflow_description, workflow_version, 
			status, created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, 'created', ?, NULL, NULL)
	`, runID, workflow.ID, workflow.Name, workflow.Description, workflow.Version, now)
	if err != nil {
		return err
	}

	// Create TaskRuns for each task in the workflow
	tasks, err := s.ListTasks(workflowID)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		_, err = tx.Exec(`
			INSERT INTO task_runs (
				run_id, task_id, workflow_id, task_name, task_description, 
				task_type, task_config, task_position, task_node_type, 
				status, created_at, started_at, finished_at, exit_code, output
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, NULL, NULL, 0, '')
		`, runID, task.ID, workflow.ID, task.Name, task.Description,
			task.Type, task.Config, task.Position, task.NodeType, now)
		if err != nil {
			return err
		}
	}

	// Create EdgeRuns for each edge in the workflow
	edges, err := s.ListEdges(workflowID)
	if err != nil {
		return err
	}

	for _, edge := range edges {
		_, err = tx.Exec(`
			INSERT INTO edge_runs (
				run_id, edge_id, workflow_id, edge_source, edge_target, 
				edge_source_handle, edge_target_handle, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, runID, edge.ID, workflow.ID, edge.Source, edge.Target,
			edge.SourceHandle, edge.TargetHandle, now)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func parseTime(t string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", t)
}

func parseNullableTime(t sql.NullString) *time.Time {
	if !t.Valid {
		return nil
	}
	parsed, err := parseTime(t.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func (s *SqliteDB) GetWorkflowRun(runID string) (entity.WorkflowRunEntity, error) {
	var run entity.WorkflowRunEntity
	var createdAtStr string
	var startedAtStr, finishedAtStr sql.NullString
	err := s.db.QueryRow(`
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, 
		status, created_at, started_at, finished_at 
		FROM workflow_runs WHERE run_id = ?
	`, runID).Scan(
		&run.RunID, &run.WorkflowID, &run.WorkflowName, &run.WorkflowDescription,
		&run.WorkflowVersion, &run.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
	)
	if err != nil {
		return entity.WorkflowRunEntity{}, err
	}
	run.CreatedAt, _ = parseTime(createdAtStr)
	run.StartedAt = parseNullableTime(startedAtStr)
	run.FinishedAt = parseNullableTime(finishedAtStr)
	return run, nil
}

func (s *SqliteDB) GetTaskRuns(runID string) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, task_id, workflow_id, task_name, task_description, 
		task_type, task_config, task_position, task_node_type, 
		status, created_at, started_at, finished_at, exit_code, output 
		FROM task_runs WHERE run_id = ?
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		var task entity.TaskRunEntity
		var createdAtStr string
		var startedAtStr, finishedAtStr sql.NullString
		err := rows.Scan(
			&task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskPosition, &task.TaskNodeType,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
			&task.ExitCode, &task.Output,
		)
		if err != nil {
			return nil, err
		}
		task.CreatedAt, _ = parseTime(createdAtStr)
		task.StartedAt = parseNullableTime(startedAtStr)
		task.FinishedAt = parseNullableTime(finishedAtStr)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) GetEdgeRuns(runID string) ([]entity.EdgeRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, edge_id, workflow_id, edge_source, edge_target, 
		edge_source_handle, edge_target_handle, created_at 
		FROM edge_runs WHERE run_id = ?
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []entity.EdgeRunEntity
	for rows.Next() {
		var edge entity.EdgeRunEntity
		var createdAtStr string
		err := rows.Scan(
			&edge.RunID, &edge.EdgeID, &edge.WorkflowID, &edge.EdgeSource, &edge.EdgeTarget,
			&edge.EdgeSourceHandle, &edge.EdgeTargetHandle, &createdAtStr,
		)
		if err != nil {
			return nil, err
		}
		edge.CreatedAt, _ = parseTime(createdAtStr)
		edges = append(edges, edge)
	}
	return edges, nil
}

func (s *SqliteDB) ListWorkflowRuns(offset int, limit int) ([]entity.WorkflowRunEntity, error) {
	logrus.Debugf("query workflow runs limit %d offset %d", limit, offset)
	rows, err := s.db.Query(`
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, 
		status, created_at, started_at, finished_at 
		FROM workflow_runs LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []entity.WorkflowRunEntity
	for rows.Next() {
		var run entity.WorkflowRunEntity
		var createdAtStr, startedAtStr, finishedAtStr sql.NullString
		err := rows.Scan(
			&run.RunID, &run.WorkflowID, &run.WorkflowName, &run.WorkflowDescription,
			&run.WorkflowVersion, &run.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
		)
		if err != nil {
			return nil, err
		}
		run.CreatedAt = *parseNullableTime(createdAtStr)
		run.StartedAt = parseNullableTime(startedAtStr)
		run.FinishedAt = parseNullableTime(finishedAtStr)
		runs = append(runs, run)
	}
	return runs, nil
}
