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

	tasks, err := s.ListTasks(workflowID)
	if err != nil {
		return err
	}

	edges, err := s.ListEdges(workflowID)
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
			run_id, workflow_id, workflow_name, workflow_description, workflow_version, workflow_tag,
			status, created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, 'created', ?, NULL, NULL)
	`, runID, workflow.ID, workflow.Name, workflow.Description, workflow.Version, workflow.Tag, now)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		_, err = tx.Exec(`
			INSERT INTO task_runs (
				run_id, task_id, workflow_id, task_name, task_description, 
				task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
				status, created_at, started_at, finished_at, exit_code, output
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', NULL, 'pending', ?, NULL, NULL, 0, '')
		`, runID, task.ID, workflow.ID, task.Name, task.Description,
			task.Type, task.Config, task.Tag, task.Position, task.NodeType, now)
		if err != nil {
			return err
		}
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
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, workflow_tag,
		status, created_at, started_at, finished_at 
		FROM workflow_runs WHERE run_id = ?
	`, runID).Scan(
		&run.RunID, &run.WorkflowID, &run.WorkflowName, &run.WorkflowDescription,
		&run.WorkflowVersion, &run.WorkflowTag, &run.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
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
		task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs WHERE run_id = ?
		ORDER BY task_id ASC
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		var task entity.TaskRunEntity
		var createdAtStr string
		var assignedAtStr, startedAtStr, finishedAtStr sql.NullString
		err := rows.Scan(
			&task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
			&task.ExitCode, &task.Output,
		)
		if err != nil {
			return nil, err
		}
		task.CreatedAt, _ = parseTime(createdAtStr)
		task.AssignedAt = parseNullableTime(assignedAtStr)
		task.StartedAt = parseNullableTime(startedAtStr)
		task.FinishedAt = parseNullableTime(finishedAtStr)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) GetTaskRun(runID string, taskID string) (entity.TaskRunEntity, error) {
	var task entity.TaskRunEntity
	var createdAtStr string
	var assignedAtStr, startedAtStr, finishedAtStr sql.NullString
	err := s.db.QueryRow(`
		SELECT run_id, task_id, workflow_id, task_name, task_description,
		task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs WHERE run_id = ? AND task_id = ?
	`, runID, taskID).Scan(
		&task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskName, &task.TaskDescription,
		&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
		&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
		&task.ExitCode, &task.Output,
	)
	if err != nil {
		return entity.TaskRunEntity{}, err
	}
	task.CreatedAt, _ = parseTime(createdAtStr)
	task.AssignedAt = parseNullableTime(assignedAtStr)
	task.StartedAt = parseNullableTime(startedAtStr)
	task.FinishedAt = parseNullableTime(finishedAtStr)
	return task, nil
}

func (s *SqliteDB) GetEdgeRuns(runID string) ([]entity.EdgeRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, edge_id, workflow_id, edge_source, edge_target, 
		edge_source_handle, edge_target_handle, created_at 
		FROM edge_runs WHERE run_id = ?
		ORDER BY edge_id ASC
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
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, workflow_tag,
		status, created_at, started_at, finished_at 
		FROM workflow_runs
		ORDER BY datetime(created_at) DESC, run_id DESC
		LIMIT ? OFFSET ?
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
			&run.WorkflowVersion, &run.WorkflowTag, &run.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
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

func (s *SqliteDB) MarkWorkflowRunRunning(runID string) error {
	_, err := s.db.Exec(`
		UPDATE workflow_runs
		SET status = 'running', started_at = DATETIME('now'), finished_at = NULL
		WHERE run_id = ?
	`, runID)
	return err
}

func (s *SqliteDB) FinishWorkflowRun(runID string, status string) error {
	_, err := s.db.Exec(`
		UPDATE workflow_runs
		SET status = ?, finished_at = DATETIME('now')
		WHERE run_id = ?
	`, status, runID)
	return err
}

func (s *SqliteDB) UpdateWorkflowRunStatus(runID string, status string) error {
	_, err := s.db.Exec(`
		UPDATE workflow_runs
		SET status = ?
		WHERE run_id = ?
	`, status, runID)
	return err
}

func (s *SqliteDB) QueueTaskRun(runID string, taskID string, effectiveTag string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'queued', effective_tag = ?, assigned_node_id = '', assigned_at = NULL
		WHERE run_id = ? AND task_id = ?
	`, effectiveTag, runID, taskID)
	return err
}

func (s *SqliteDB) AssignTaskRun(runID string, taskID string, nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'assigned', assigned_node_id = ?, assigned_at = DATETIME('now')
		WHERE run_id = ? AND task_id = ?
	`, nodeID, runID, taskID)
	return err
}

func (s *SqliteDB) ResetAssignedTaskRun(runID string, taskID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'queued', assigned_node_id = '', assigned_at = NULL
		WHERE run_id = ? AND task_id = ?
	`, runID, taskID)
	return err
}

func (s *SqliteDB) MarkTaskRunRunning(runID string, taskID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'running', started_at = DATETIME('now'), finished_at = NULL
		WHERE run_id = ? AND task_id = ?
	`, runID, taskID)
	return err
}

func (s *SqliteDB) MarkTaskRunUnknown(runID string, taskID string, output string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'unknown', output = CASE WHEN ? = '' THEN output ELSE COALESCE(output, '') || ? END
		WHERE run_id = ? AND task_id = ?
	`, output, output, runID, taskID)
	return err
}

func (s *SqliteDB) FinishTaskRun(runID string, taskID string, status string, exitCode int, output string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = ?, finished_at = DATETIME('now'), exit_code = ?, output = ?
		WHERE run_id = ? AND task_id = ?
	`, status, exitCode, output, runID, taskID)
	return err
}

func (s *SqliteDB) SkipPendingTaskRuns(runID string, output string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'skipped', finished_at = DATETIME('now'), output = ?
		WHERE run_id = ? AND status IN ('pending', 'queued')
	`, output, runID)
	return err
}

func (s *SqliteDB) CancelPendingTaskRuns(runID string, output string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'cancelled', finished_at = DATETIME('now'), output = ?
		WHERE run_id = ? AND status IN ('pending', 'queued', 'assigned', 'unknown')
	`, output, runID)
	return err
}

func (s *SqliteDB) AppendTaskRunOutput(runID string, taskID string, chunk string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET output = COALESCE(output, '') || ?
		WHERE run_id = ? AND task_id = ?
	`, chunk, runID, taskID)
	return err
}

func (s *SqliteDB) ListQueuedTaskRuns(limit int) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, task_id, workflow_id, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs
		WHERE status = 'queued'
		ORDER BY datetime(created_at) ASC, task_id ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		var task entity.TaskRunEntity
		var createdAtStr string
		var assignedAtStr, startedAtStr, finishedAtStr sql.NullString
		if err := rows.Scan(
			&task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output,
		); err != nil {
			return nil, err
		}
		task.CreatedAt, _ = parseTime(createdAtStr)
		task.AssignedAt = parseNullableTime(assignedAtStr)
		task.StartedAt = parseNullableTime(startedAtStr)
		task.FinishedAt = parseNullableTime(finishedAtStr)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) ListNodeActiveTaskRuns(nodeID string) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT run_id, task_id, workflow_id, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output
		FROM task_runs
		WHERE assigned_node_id = ? AND status IN ('assigned', 'running', 'unknown')
		ORDER BY datetime(created_at) ASC, task_id ASC
	`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskRunEntity
	for rows.Next() {
		var task entity.TaskRunEntity
		var createdAtStr string
		var assignedAtStr, startedAtStr, finishedAtStr sql.NullString
		if err := rows.Scan(
			&task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output,
		); err != nil {
			return nil, err
		}
		task.CreatedAt, _ = parseTime(createdAtStr)
		task.AssignedAt = parseNullableTime(assignedAtStr)
		task.StartedAt = parseNullableTime(startedAtStr)
		task.FinishedAt = parseNullableTime(finishedAtStr)
		tasks = append(tasks, task)
	}
	return tasks, nil
}
