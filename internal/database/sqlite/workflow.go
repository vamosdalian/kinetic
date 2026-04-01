package sqlite

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

func (s *SqliteDB) ListWorkflows(offset int, limit int) ([]entity.WorkflowEntity, error) {
	return s.ListWorkflowsFiltered(offset, limit, "")
}

func (s *SqliteDB) ListWorkflowsFiltered(offset int, limit int, query string) ([]entity.WorkflowEntity, error) {
	logrus.Debugf("query workflow limit %d offset %d", limit, offset)
	like := sqliteLikePattern(query)
	trimmed := strings.TrimSpace(query)
	rows, err := s.db.Query(`
		SELECT id, name, enable, version, created_at, updated_at, tag
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
		err := rows.Scan(&workflow.ID, &workflow.Name, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr, &workflow.Tag)
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
	var rawConfig string
	err := s.db.QueryRow("SELECT id, name, description, config, enable, version, created_at, updated_at, tag FROM workflows WHERE id = ?", id).
		Scan(&workflow.ID, &workflow.Name, &workflow.Description, &rawConfig, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr, &workflow.Tag)
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

	return workflow, nil
}

func (s *SqliteDB) SaveWorkflow(req entity.WorkflowEntity) error {
	stmt, err := s.db.Prepare(`
		INSERT INTO workflows (id, name, description, config, tag, version, enable, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, 1, ?, DATETIME('now'), DATETIME('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			config = excluded.config,
			tag = excluded.tag,
			enable = excluded.enable,
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
	_, err = stmt.Exec(req.ID, req.Name, req.Description, normalizeJSONText(req.Config), req.Tag, req.Enable)
	if err != nil {
		return err
	}

	return nil
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

func (s *SqliteDB) DeleteWorkflow(id string) error {
	_, err := s.db.Exec("DELETE FROM workflows WHERE id = ?", id)
	return err
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
	stmt, err := s.db.Prepare(`
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
	stmt, err := s.db.Prepare(`
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
