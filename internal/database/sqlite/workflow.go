package sqlite

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

type statementPreparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

func (s *SqliteDB) ListWorkflows(offset int, limit int) ([]entity.WorkflowEntity, error) {
	return s.ListWorkflowsFiltered(offset, limit, "")
}

func (s *SqliteDB) ListWorkflowsFiltered(offset int, limit int, query string) ([]entity.WorkflowEntity, error) {
	logrus.Debugf("query workflow limit %d offset %d", limit, offset)
	like := sqliteLikePattern(query)
	trimmed := strings.TrimSpace(query)
	rows, err := s.db.Query(`
		SELECT id, name, enable, version, created_at, updated_at, tag, trigger_type, trigger_expr, next_run_at, last_run_at
		FROM workflows
		WHERE (? = '' OR LOWER(id) LIKE ? OR LOWER(name) LIKE ? OR LOWER(COALESCE(tag, '')) LIKE ?)
		LIMIT ? OFFSET ?
	`, trimmed, like, like, like, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []entity.WorkflowEntity
	for rows.Next() {
		var workflow entity.WorkflowEntity
		var createdAtStr, updatedAtStr string
		var nextRunAtStr, lastRunAtStr sql.NullString
		err := rows.Scan(&workflow.ID, &workflow.Name, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr, &workflow.Tag, &workflow.TriggerType, &workflow.TriggerExpr, &nextRunAtStr, &lastRunAtStr)
		if err != nil {
			return nil, err
		}
		workflow.CreatedAt, err = parseTime(createdAtStr)
		if err != nil {
			return nil, err
		}
		workflow.UpdatedAt, err = parseTime(updatedAtStr)
		if err != nil {
			return nil, err
		}
		workflow.NextRunAt = parseNullableTime(nextRunAtStr)
		workflow.LastRunAt = parseNullableTime(lastRunAtStr)
		workflows = append(workflows, workflow)
	}
	logrus.Debugf("found %d workflows", len(workflows))
	return workflows, nil
}

func (s *SqliteDB) CountWorkflows() (int, error) {
	return s.CountWorkflowsFiltered("")
}

func (s *SqliteDB) CountWorkflowsFiltered(query string) (int, error) {
	var count int
	like := sqliteLikePattern(query)
	trimmed := strings.TrimSpace(query)
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM workflows
		WHERE (? = '' OR LOWER(id) LIKE ? OR LOWER(name) LIKE ? OR LOWER(COALESCE(tag, '')) LIKE ?)
	`, trimmed, like, like, like).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func sqliteLikePattern(query string) string {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return ""
	}
	return "%" + trimmed + "%"
}

func (s *SqliteDB) GetWorkflowByID(id string) (entity.WorkflowEntity, error) {
	var workflow entity.WorkflowEntity
	var createdAtStr, updatedAtStr string
	var nextRunAtStr, lastRunAtStr sql.NullString
	var rawConfig string
	err := s.db.QueryRow("SELECT id, name, description, config, enable, version, created_at, updated_at, tag, trigger_type, trigger_expr, next_run_at, last_run_at FROM workflows WHERE id = ?", id).
		Scan(&workflow.ID, &workflow.Name, &workflow.Description, &rawConfig, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr, &workflow.Tag, &workflow.TriggerType, &workflow.TriggerExpr, &nextRunAtStr, &lastRunAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}
	if _, err := workflowcfg.ParseWorkflowConfig(rawConfig); err != nil {
		return entity.WorkflowEntity{}, err
	}
	workflow.Config = normalizeJSONText(rawConfig)

	workflow.CreatedAt, err = parseTime(createdAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}

	workflow.UpdatedAt, err = parseTime(updatedAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}
	workflow.NextRunAt = parseNullableTime(nextRunAtStr)
	workflow.LastRunAt = parseNullableTime(lastRunAtStr)

	return workflow, nil
}

func (s *SqliteDB) SaveWorkflow(req entity.WorkflowEntity) error {
	return saveWorkflowWithPreparer(s.db, req)
}

func (s *SqliteDB) UpdateWorkflowEnable(id string, enable bool) error {
	workflow, err := s.GetWorkflowByID(id)
	if err != nil {
		return err
	}

	trigger, err := workflowcfg.NormalizeWorkflowTrigger(workflowcfg.WorkflowTrigger{
		Type:      workflowcfg.WorkflowTriggerType(workflow.TriggerType),
		Expr:      workflow.TriggerExpr,
		NextRunAt: workflow.NextRunAt,
		LastRunAt: workflow.LastRunAt,
	}, enable, time.Now().UTC())
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		UPDATE workflows
		SET enable = ?, next_run_at = ?, updated_at = DATETIME('now')
		WHERE id = ?
	`, enable, formatNullableDBTime(trigger.NextRunAt), id)
	return err
}

func saveWorkflowWithPreparer(preparer statementPreparer, req entity.WorkflowEntity) error {
	stmt, err := preparer.Prepare(`
		INSERT INTO workflows (id, name, description, config, tag, version, enable, trigger_type, trigger_expr, next_run_at, last_run_at, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, DATETIME('now'), DATETIME('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			config = excluded.config,
			tag = excluded.tag,
			enable = excluded.enable,
			trigger_type = excluded.trigger_type,
			trigger_expr = excluded.trigger_expr,
			next_run_at = excluded.next_run_at,
			last_run_at = excluded.last_run_at,
			version = version + 1,
			updated_at = DATETIME('now')
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := workflowcfg.ParseWorkflowConfig(req.Config); err != nil {
		return err
	}
	trigger, err := workflowcfg.NormalizeWorkflowTrigger(workflowcfg.WorkflowTrigger{
		Type:      workflowcfg.WorkflowTriggerType(req.TriggerType),
		Expr:      req.TriggerExpr,
		NextRunAt: req.NextRunAt,
		LastRunAt: req.LastRunAt,
	}, req.Enable, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		req.ID,
		req.Name,
		req.Description,
		normalizeJSONText(req.Config),
		req.Tag,
		req.Enable,
		string(trigger.Type),
		trigger.Expr,
		formatNullableDBTime(trigger.NextRunAt),
		formatNullableDBTime(trigger.LastRunAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *SqliteDB) SaveWorkflowDefinition(workflow entity.WorkflowEntity, tasks []entity.TaskEntity, edges []entity.EdgeEntity) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := saveWorkflowWithPreparer(tx, workflow); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM tasks WHERE workflow_id = ?", workflow.ID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM edges WHERE workflow_id = ?", workflow.ID); err != nil {
		return err
	}
	if _, err := saveTasksWithPreparer(tx, tasks); err != nil {
		return err
	}
	if _, err := saveEdgesWithPreparer(tx, edges); err != nil {
		return err
	}

	return tx.Commit()
}

func normalizeJSONText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return trimmed
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return trimmed
	}
	return string(normalized)
}

func formatNullableDBTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format("2006-01-02 15:04:05")
}

func (s *SqliteDB) DeleteWorkflow(id string) error {
	deleted, err := s.DeleteWorkflowDefinition(id)
	if err != nil {
		return err
	}
	if !deleted {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SqliteDB) ListScheduledWorkflows() ([]entity.WorkflowEntity, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, config, enable, version, created_at, updated_at, tag, trigger_type, trigger_expr, next_run_at, last_run_at
		FROM workflows
		WHERE trigger_type = 'cron'
		ORDER BY enable DESC, next_run_at IS NULL ASC, next_run_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []entity.WorkflowEntity
	for rows.Next() {
		workflow, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func (s *SqliteDB) ListDueWorkflowsForScheduling(now time.Time, limit int) ([]entity.WorkflowEntity, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, config, enable, version, created_at, updated_at, tag, trigger_type, trigger_expr, next_run_at, last_run_at
		FROM workflows
		WHERE trigger_type = 'cron' AND enable = 1 AND next_run_at IS NOT NULL AND next_run_at <= ?
		ORDER BY next_run_at ASC, id ASC
		LIMIT ?
	`, now.UTC().Format("2006-01-02 15:04:05"), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []entity.WorkflowEntity
	for rows.Next() {
		workflow, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func scanWorkflow(scanner interface{ Scan(dest ...any) error }) (entity.WorkflowEntity, error) {
	var workflow entity.WorkflowEntity
	var rawConfig string
	var createdAtStr, updatedAtStr string
	var nextRunAtStr, lastRunAtStr sql.NullString
	err := scanner.Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Description,
		&rawConfig,
		&workflow.Enable,
		&workflow.Version,
		&createdAtStr,
		&updatedAtStr,
		&workflow.Tag,
		&workflow.TriggerType,
		&workflow.TriggerExpr,
		&nextRunAtStr,
		&lastRunAtStr,
	)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}
	if _, err := workflowcfg.ParseWorkflowConfig(rawConfig); err != nil {
		return entity.WorkflowEntity{}, err
	}
	workflow.Config = normalizeJSONText(rawConfig)
	workflow.CreatedAt, err = parseTime(createdAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}
	workflow.UpdatedAt, err = parseTime(updatedAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}
	workflow.NextRunAt = parseNullableTime(nextRunAtStr)
	workflow.LastRunAt = parseNullableTime(lastRunAtStr)
	return workflow, nil
}

func (s *SqliteDB) TouchWorkflowLastRun(workflowID string, lastRunAt time.Time) error {
	_, err := s.db.Exec(`
		UPDATE workflows
		SET last_run_at = DATETIME(?)
		WHERE id = ?
	`, lastRunAt.UTC().Format("2006-01-02 15:04:05"), workflowID)
	return err
}

func (s *SqliteDB) CreateScheduledWorkflowRun(workflowID string, runID string, scheduledAt time.Time, nextRunAt *time.Time) (bool, error) {
	workflow, err := s.GetWorkflowByID(workflowID)
	if err != nil {
		return false, err
	}
	if !workflow.Enable || workflow.TriggerType != string(workflowcfg.WorkflowTriggerCron) || workflow.NextRunAt == nil || workflow.NextRunAt.After(scheduledAt.UTC()) {
		return false, nil
	}

	tasks, err := s.ListTasks(workflowID)
	if err != nil {
		return false, err
	}
	edges, err := s.ListEdges(workflowID)
	if err != nil {
		return false, err
	}

	now := scheduledAt.UTC().Format("2006-01-02 15:04:05")

	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var dueNow sql.NullString
	if err := tx.QueryRow(`SELECT next_run_at FROM workflows WHERE id = ?`, workflowID).Scan(&dueNow); err != nil {
		return false, err
	}
	dueAt := parseNullableTime(dueNow)
	if dueAt == nil || dueAt.After(scheduledAt.UTC()) {
		return false, nil
	}

	updateResult, err := tx.Exec(`
		UPDATE workflows
		SET last_run_at = ?, next_run_at = ?
		WHERE id = ? AND enable = 1 AND trigger_type = 'cron' AND next_run_at IS NOT NULL AND next_run_at <= ?
	`, now, formatNullableDBTime(nextRunAt), workflowID, now)
	if err != nil {
		return false, err
	}
	rowsAffected, err := updateResult.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, nil
	}
	_, err = tx.Exec(`
		INSERT INTO workflow_runs (
			run_id, workflow_id, workflow_name, workflow_description, workflow_config, workflow_version, workflow_tag,
			status, created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'created', ?, NULL, NULL)
	`, runID, workflow.ID, workflow.Name, workflow.Description, workflow.Config, workflow.Version, workflow.Tag, now)
	if err != nil {
		return false, err
	}

	for _, task := range tasks {
		if _, err := tx.Exec(`
			INSERT INTO task_runs (
				run_id, task_id, workflow_id, task_name, task_description, 
				task_type, task_config, task_tag, task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
				status, created_at, started_at, finished_at, exit_code, output, result
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', NULL, 'pending', ?, NULL, NULL, 0, '', '')
		`, runID, task.ID, workflow.ID, task.Name, task.Description,
			task.Type, task.Config, task.Tag, task.Position, task.NodeType, now); err != nil {
			return false, err
		}
	}
	for _, edge := range edges {
		if _, err := tx.Exec(`
			INSERT INTO edge_runs (
				run_id, edge_id, workflow_id, edge_source, edge_target, 
				edge_source_handle, edge_target_handle, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, runID, edge.ID, workflow.ID, edge.Source, edge.Target,
			edge.SourceHandle, edge.TargetHandle, now); err != nil {
			return false, err
		}
	}

	return true, tx.Commit()
}

func (s *SqliteDB) DeleteWorkflowDefinition(id string) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	result, err := tx.Exec("DELETE FROM workflows WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, nil
	}
	if _, err := tx.Exec("DELETE FROM tasks WHERE workflow_id = ?", id); err != nil {
		return false, err
	}
	if _, err := tx.Exec("DELETE FROM edges WHERE workflow_id = ?", id); err != nil {
		return false, err
	}

	return true, tx.Commit()
}

func (s *SqliteDB) ListTasks(workflowID string) ([]entity.TaskEntity, error) {
	rows, err := s.db.Query("SELECT id, workflow_id, name, type, description, config, tag, position, node_type FROM tasks WHERE workflow_id = ? ORDER BY id ASC", workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskEntity
	for rows.Next() {
		var task entity.TaskEntity
		err := rows.Scan(&task.ID, &task.WorkflowID, &task.Name, &task.Type, &task.Description, &task.Config, &task.Tag, &task.Position, &task.NodeType)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) SaveTasks(req []entity.TaskEntity) ([]entity.TaskEntity, error) {
	return saveTasksWithPreparer(s.db, req)
}

func saveTasksWithPreparer(preparer statementPreparer, req []entity.TaskEntity) ([]entity.TaskEntity, error) {
	stmt, err := preparer.Prepare(`
		INSERT INTO tasks (id, workflow_id, name, type, description, config, tag, position, node_type) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workflow_id = excluded.workflow_id,
			name = excluded.name,
			type = excluded.type,
			description = excluded.description,
			config = excluded.config,
			tag = excluded.tag,
			position = excluded.position,
			node_type = excluded.node_type
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, task := range req {
		_, err = stmt.Exec(task.ID, task.WorkflowID, task.Name, task.Type, task.Description, task.Config, task.Tag, task.Position, task.NodeType)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (s *SqliteDB) DeleteTasks(workflowID string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE workflow_id = ?", workflowID)
	return err
}

func (s *SqliteDB) DeleteTask(id string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func (s *SqliteDB) ListEdges(workflowID string) ([]entity.EdgeEntity, error) {
	rows, err := s.db.Query("SELECT id, workflow_id, source, target, source_handle, target_handle FROM edges WHERE workflow_id = ? ORDER BY id ASC", workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []entity.EdgeEntity
	for rows.Next() {
		var edge entity.EdgeEntity
		err := rows.Scan(&edge.ID, &edge.WorkflowID, &edge.Source, &edge.Target, &edge.SourceHandle, &edge.TargetHandle)
		if err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func (s *SqliteDB) SaveEdges(req []entity.EdgeEntity) ([]entity.EdgeEntity, error) {
	return saveEdgesWithPreparer(s.db, req)
}

func saveEdgesWithPreparer(preparer statementPreparer, req []entity.EdgeEntity) ([]entity.EdgeEntity, error) {
	stmt, err := preparer.Prepare(`
		INSERT INTO edges (id, workflow_id, source, target, source_handle, target_handle) 
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workflow_id = excluded.workflow_id,
			source = excluded.source,
			target = excluded.target,
			source_handle = excluded.source_handle,
			target_handle = excluded.target_handle
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, edge := range req {
		_, err = stmt.Exec(edge.ID, edge.WorkflowID, edge.Source, edge.Target, edge.SourceHandle, edge.TargetHandle)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (s *SqliteDB) DeleteEdges(workflowID string) error {
	_, err := s.db.Exec("DELETE FROM edges WHERE workflow_id = ?", workflowID)
	return err
}

func (s *SqliteDB) DeleteEdge(id string) error {
	_, err := s.db.Exec("DELETE FROM edges WHERE id = ?", id)
	return err
}
