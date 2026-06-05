package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
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

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO workflow_runs (
			run_id, workflow_id, workflow_name, workflow_description, workflow_config, workflow_version, workflow_tag,
			status, created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'created', ?, NULL, NULL)
	`, runID, workflow.ID, workflow.Name, workflow.Description, workflow.Config, workflow.Version, workflow.Tag, now)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		_, err = tx.Exec(`
			INSERT INTO task_runs (
				task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description, 
				task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
				status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', NULL, 'pending', ?, NULL, NULL, 0, '', '', '', -1, 0)
		`, uuid.NewString(), runID, task.ID, workflow.ID, task.RefOrDefault(), task.Name, task.Description,
			task.Type, task.Config, task.Tag, task.Position, task.NodeType, now)
		if err != nil {
			return err
		}
	}

	for _, edge := range edges {
		_, err = tx.Exec(`
			INSERT INTO edge_runs (
				edge_run_id, run_id, edge_id, workflow_id, edge_source, edge_target, 
				edge_source_handle, edge_target_handle, created_at, loop_id, loop_index, loop_value
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', -1, 0)
		`, uuid.NewString(), runID, edge.ID, workflow.ID, edge.Source, edge.Target,
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
	return time.ParseInLocation("2006-01-02 15:04:05", t, time.UTC)
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
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_config, workflow_version, workflow_tag,
		status, created_at, started_at, finished_at 
		FROM workflow_runs WHERE run_id = ?
	`, runID).Scan(
		&run.RunID, &run.WorkflowID, &run.WorkflowName, &run.WorkflowDescription,
		&run.WorkflowConfig, &run.WorkflowVersion, &run.WorkflowTag, &run.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
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
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description, 
		task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs WHERE run_id = ?
		ORDER BY task_id ASC, loop_index ASC, created_at ASC, task_run_id ASC
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
			&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
			&task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
		task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs WHERE run_id = ? AND task_id = ?
		ORDER BY loop_index DESC, created_at DESC, task_run_id DESC
		LIMIT 1
	`, runID, taskID).Scan(
		&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
		&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
		&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
		&task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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

func (s *SqliteDB) GetTaskRunByID(taskRunID string) (entity.TaskRunEntity, error) {
	var task entity.TaskRunEntity
	var createdAtStr string
	var assignedAtStr, startedAtStr, finishedAtStr sql.NullString
	err := s.db.QueryRow(`
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
		task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs WHERE task_run_id = ?
	`, taskRunID).Scan(
		&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
		&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
		&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr,
		&task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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
		SELECT edge_run_id, run_id, edge_id, workflow_id, edge_source, edge_target, 
		edge_source_handle, edge_target_handle, created_at, loop_id, loop_index, loop_value 
		FROM edge_runs WHERE run_id = ?
		ORDER BY edge_id ASC, loop_index ASC, edge_run_id ASC
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
			&edge.EdgeRunID, &edge.RunID, &edge.EdgeID, &edge.WorkflowID, &edge.EdgeSource, &edge.EdgeTarget,
			&edge.EdgeSourceHandle, &edge.EdgeTargetHandle, &createdAtStr, &edge.LoopID, &edge.LoopIndex, &edge.LoopValue,
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
	return s.ListWorkflowRunsFiltered(offset, limit, "", "", "")
}

func (s *SqliteDB) ListWorkflowRunsFiltered(offset int, limit int, workflowQuery string, runQuery string, status string) ([]entity.WorkflowRunEntity, error) {
	logrus.Debugf("query workflow runs limit %d offset %d", limit, offset)
	workflowLike := sqliteLikePattern(workflowQuery)
	runLike := sqliteLikePattern(runQuery)
	trimmedWorkflow := strings.TrimSpace(workflowQuery)
	trimmedRun := strings.TrimSpace(runQuery)
	trimmedStatus := strings.TrimSpace(strings.ToLower(status))
	rows, err := s.db.Query(`
		SELECT run_id, workflow_id, workflow_name, workflow_description, workflow_version, workflow_tag,
		status, created_at, started_at, finished_at 
		FROM workflow_runs
		WHERE (? = '' OR LOWER(workflow_name) LIKE ? OR LOWER(workflow_id) LIKE ?)
		  AND (? = '' OR LOWER(run_id) LIKE ?)
		  AND (? = '' OR LOWER(status) = ?)
		ORDER BY created_at DESC, run_id DESC
		LIMIT ? OFFSET ?
	`, trimmedWorkflow, workflowLike, workflowLike, trimmedRun, runLike, trimmedStatus, trimmedStatus, limit, offset)
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

func (s *SqliteDB) CountWorkflowRuns() (int, error) {
	return s.CountWorkflowRunsFiltered("", "", "")
}

func (s *SqliteDB) CountWorkflowRunsFiltered(workflowQuery string, runQuery string, status string) (int, error) {
	workflowLike := sqliteLikePattern(workflowQuery)
	runLike := sqliteLikePattern(runQuery)
	trimmedWorkflow := strings.TrimSpace(workflowQuery)
	trimmedRun := strings.TrimSpace(runQuery)
	trimmedStatus := strings.TrimSpace(strings.ToLower(status))
	row := s.db.QueryRow(`
		SELECT COUNT(*) FROM workflow_runs
		WHERE (? = '' OR LOWER(workflow_name) LIKE ? OR LOWER(workflow_id) LIKE ?)
		  AND (? = '' OR LOWER(run_id) LIKE ?)
		  AND (? = '' OR LOWER(status) = ?)
	`, trimmedWorkflow, workflowLike, workflowLike, trimmedRun, runLike, trimmedStatus, trimmedStatus)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
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
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, effectiveTag, runID, taskID)
	return err
}

func (s *SqliteDB) AssignTaskRun(runID string, taskID string, nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'assigned', assigned_node_id = ?, assigned_at = DATETIME('now')
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, nodeID, runID, taskID)
	return err
}

func (s *SqliteDB) ResetAssignedTaskRun(runID string, taskID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'queued', assigned_node_id = '', assigned_at = NULL
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, runID, taskID)
	return err
}

func (s *SqliteDB) PrepareTaskRunsForLoop(runID string, taskIDs []string, loopID string, loopIndex int, loopValue int, createNew bool) error {
	if len(taskIDs) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, taskID := range taskIDs {
		if createNew {
			if _, err := tx.Exec(`
				INSERT INTO task_runs (
					task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
					task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
					status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
				)
				SELECT
					?, run_id, task_id, workflow_id, task_ref, task_name, task_description,
					task_type, task_config, task_tag, task_position, task_node_type, '', '', NULL,
					'pending', DATETIME('now'), NULL, NULL, 0, '', '', ?, ?, ?
				FROM task_runs
				WHERE run_id = ? AND task_id = ?
				ORDER BY loop_index DESC, created_at DESC, task_run_id DESC
				LIMIT 1
			`, uuid.NewString(), loopID, loopIndex, loopValue, runID, taskID); err != nil {
				return err
			}
			continue
		}

		if _, err := tx.Exec(`
			UPDATE task_runs
			SET loop_id = ?, loop_index = ?, loop_value = ?
			WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
		`, loopID, loopIndex, loopValue, runID, taskID); err != nil {
			return err
		}
	}

	for _, targetID := range taskIDs {
		if _, err := tx.Exec(`
			UPDATE edge_runs
			SET loop_id = ?, loop_index = ?, loop_value = ?
			WHERE run_id = ? AND edge_source = ? AND edge_target = ?
		`, loopID, loopIndex, loopValue, runID, loopID, targetID); err != nil {
			return err
		}
		for _, sourceID := range taskIDs {
			if _, err := tx.Exec(`
				UPDATE edge_runs
				SET loop_id = ?, loop_index = ?, loop_value = ?
				WHERE run_id = ? AND edge_source = ? AND edge_target = ?
			`, loopID, loopIndex, loopValue, runID, sourceID, targetID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *SqliteDB) MarkTaskRunRunning(runID string, taskID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'running', started_at = DATETIME('now'), finished_at = NULL
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, runID, taskID)
	return err
}

func (s *SqliteDB) MarkTaskRunRunningByID(taskRunID string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'running', started_at = DATETIME('now'), finished_at = NULL
		WHERE task_run_id = ?
	`, taskRunID)
	return err
}

func (s *SqliteDB) MarkTaskRunUnknown(runID string, taskID string, output string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = 'unknown', output = CASE WHEN ? = '' THEN output ELSE COALESCE(output, '') || ? END
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, output, output, runID, taskID)
	return err
}

func (s *SqliteDB) FinishTaskRun(runID string, taskID string, status string, exitCode int, output string, result string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = ?, finished_at = DATETIME('now'), exit_code = ?, output = ?, result = ?
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, status, exitCode, output, normalizeTaskResultText(result), runID, taskID)
	return err
}

func (s *SqliteDB) FinishTaskRunByID(taskRunID string, status string, exitCode int, output string, result string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET status = ?, finished_at = DATETIME('now'), exit_code = ?, output = ?, result = ?
		WHERE task_run_id = ?
	`, status, exitCode, output, normalizeTaskResultText(result), taskRunID)
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
		WHERE task_run_id = (SELECT task_run_id FROM task_runs WHERE run_id = ? AND task_id = ? ORDER BY loop_index DESC, created_at DESC, task_run_id DESC LIMIT 1)
	`, chunk, runID, taskID)
	return err
}

func (s *SqliteDB) AppendTaskRunOutputByID(taskRunID string, chunk string) error {
	_, err := s.db.Exec(`
		UPDATE task_runs
		SET output = COALESCE(output, '') || ?
		WHERE task_run_id = ?
	`, chunk, taskRunID)
	return err
}

func (s *SqliteDB) ListQueuedTaskRuns(limit int) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs
		WHERE status = 'queued'
		ORDER BY created_at ASC, task_id ASC
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
			&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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

func (s *SqliteDB) ListUnknownTaskRuns() ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs
		WHERE status = 'unknown'
		ORDER BY created_at ASC, task_id ASC
	`)
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
			&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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

func (s *SqliteDB) ListAssignedTaskRunsBefore(cutoff time.Time) ([]entity.TaskRunEntity, error) {
	rows, err := s.db.Query(`
		SELECT t.task_run_id, t.run_id, t.task_id, t.workflow_id, t.task_ref, t.task_name, t.task_description,
		       t.task_type, t.task_config, t.task_tag, t.task_position, t.task_node_type, t.effective_tag, t.assigned_node_id, t.assigned_at,
		       t.status, t.created_at, t.started_at, t.finished_at, t.exit_code, t.output, t.result, t.loop_id, t.loop_index, t.loop_value
		FROM task_runs t
		JOIN workflow_runs r ON r.run_id = t.run_id
		WHERE t.status = 'assigned'
		  AND t.assigned_at IS NOT NULL
		  AND t.assigned_at <= ?
		  AND r.status = 'running'
		ORDER BY t.assigned_at ASC, t.task_id ASC
	`, cutoff.UTC().Format("2006-01-02 15:04:05"))
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
			&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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
		SELECT task_run_id, run_id, task_id, workflow_id, task_ref, task_name, task_description,
		       task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
		       status, created_at, started_at, finished_at, exit_code, output, result, loop_id, loop_index, loop_value
		FROM task_runs
		WHERE assigned_node_id = ? AND status IN ('assigned', 'running', 'unknown')
		ORDER BY created_at ASC, task_id ASC
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
			&task.TaskRunID, &task.RunID, &task.TaskID, &task.WorkflowID, &task.TaskRef, &task.TaskName, &task.TaskDescription,
			&task.TaskType, &task.TaskConfig, &task.TaskTag, &task.TaskPosition, &task.TaskNodeType, &task.EffectiveTag, &task.AssignedNodeID, &assignedAtStr,
			&task.Status, &createdAtStr, &startedAtStr, &finishedAtStr, &task.ExitCode, &task.Output, &task.Result, &task.LoopID, &task.LoopIndex, &task.LoopValue,
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
