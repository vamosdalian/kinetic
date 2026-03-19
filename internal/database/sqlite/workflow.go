package sqlite

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func (s *SqliteDB) ListWorkflows(offset int, limit int) ([]entity.WorkflowEntity, error) {
	logrus.Debugf("query workflow limit %d offset %d", limit, offset)
	rows, err := s.db.Query("SELECT id, name, enable, version, created_at, updated_at FROM workflows LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []entity.WorkflowEntity
	for rows.Next() {
		var workflow entity.WorkflowEntity
		var createdAtStr, updatedAtStr string
		err := rows.Scan(&workflow.ID, &workflow.Name, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr)
		if err != nil {
			return nil, err
		}
		workflow.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, err
		}
		workflow.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}
	logrus.Debugf("found %d workflows", len(workflows))
	return workflows, nil
}

func (s *SqliteDB) CountWorkflows() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM workflows").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *SqliteDB) GetWorkflowByID(id string) (entity.WorkflowEntity, error) {
	var workflow entity.WorkflowEntity
	var createdAtStr, updatedAtStr string
	err := s.db.QueryRow("SELECT id, name, description, enable, version, created_at, updated_at FROM workflows WHERE id = ?", id).
		Scan(&workflow.ID, &workflow.Name, &workflow.Description, &workflow.Enable, &workflow.Version, &createdAtStr, &updatedAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}

	workflow.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}

	workflow.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err != nil {
		return entity.WorkflowEntity{}, err
	}

	return workflow, nil
}

func (s *SqliteDB) SaveWorkflow(req entity.WorkflowEntity) error {
	stmt, err := s.db.Prepare(`
		INSERT INTO workflows (id, name, description, version, enable, created_at, updated_at) 
		VALUES (?, ?, ?, 1, ?, DATETIME('now'), DATETIME('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			enable = excluded.enable,
			version = version + 1,
			updated_at = DATETIME('now')
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(req.ID, req.Name, req.Description, req.Enable)
	if err != nil {
		return err
	}

	return nil
}

func (s *SqliteDB) DeleteWorkflow(id string) error {
	_, err := s.db.Exec("DELETE FROM workflows WHERE id = ?", id)
	return err
}

func (s *SqliteDB) ListTasks(workflowID string) ([]entity.TaskEntity, error) {
	rows, err := s.db.Query("SELECT id, workflow_id, name, type, description, config, position, node_type FROM tasks WHERE workflow_id = ? ORDER BY id ASC", workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.TaskEntity
	for rows.Next() {
		var task entity.TaskEntity
		err := rows.Scan(&task.ID, &task.WorkflowID, &task.Name, &task.Type, &task.Description, &task.Config, &task.Position, &task.NodeType)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *SqliteDB) SaveTasks(req []entity.TaskEntity) ([]entity.TaskEntity, error) {
	stmt, err := s.db.Prepare(`
		INSERT INTO tasks (id, workflow_id, name, type, description, config, position, node_type) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workflow_id = excluded.workflow_id,
			name = excluded.name,
			type = excluded.type,
			description = excluded.description,
			config = excluded.config,
			position = excluded.position,
			node_type = excluded.node_type
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, task := range req {
		_, err = stmt.Exec(task.ID, task.WorkflowID, task.Name, task.Type, task.Description, task.Config, task.Position, task.NodeType)
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
