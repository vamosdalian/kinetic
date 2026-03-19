package sqlite

import (
	"database/sql"
	"time"

	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func (s *SqliteDB) ListNodes() ([]entity.NodeEntity, error) {
	rows, err := s.db.Query(`
		SELECT node_id, name, ip, kind, status, max_concurrency, running_count,
		       last_heartbeat_at, last_stream_at, created_at, updated_at
		FROM nodes
		ORDER BY node_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []entity.NodeEntity
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *SqliteDB) GetNodeByID(nodeID string) (entity.NodeEntity, error) {
	row := s.db.QueryRow(`
		SELECT node_id, name, ip, kind, status, max_concurrency, running_count,
		       last_heartbeat_at, last_stream_at, created_at, updated_at
		FROM nodes
		WHERE node_id = ?
	`, nodeID)
	return scanNode(row)
}

func (s *SqliteDB) UpsertNode(node entity.NodeEntity) error {
	_, err := s.db.Exec(`
		INSERT INTO nodes (
			node_id, name, ip, kind, status, max_concurrency, running_count,
			last_heartbeat_at, last_stream_at, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			COALESCE(?, DATETIME('now')), COALESCE(?, DATETIME('now')), DATETIME('now'), DATETIME('now')
		)
		ON CONFLICT(node_id) DO UPDATE SET
			name = excluded.name,
			ip = excluded.ip,
			kind = excluded.kind,
			status = excluded.status,
			max_concurrency = excluded.max_concurrency,
			updated_at = DATETIME('now')
	`, node.NodeID, node.Name, node.IP, node.Kind, node.Status, node.MaxConcurrency, node.RunningCount,
		formatNullableTime(node.LastHeartbeatAt), formatNullableTime(node.LastStreamAt))
	return err
}

func (s *SqliteDB) SetNodeStatus(nodeID string, status string) error {
	_, err := s.db.Exec(`
		UPDATE nodes
		SET status = ?, updated_at = DATETIME('now')
		WHERE node_id = ?
	`, status, nodeID)
	return err
}

func (s *SqliteDB) UpdateNodeHeartbeat(nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE nodes
		SET status = 'online', last_heartbeat_at = DATETIME('now'), updated_at = DATETIME('now')
		WHERE node_id = ?
	`, nodeID)
	return err
}

func (s *SqliteDB) UpdateNodeStream(nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE nodes
		SET status = 'online', last_stream_at = DATETIME('now'), updated_at = DATETIME('now')
		WHERE node_id = ?
	`, nodeID)
	return err
}

func (s *SqliteDB) ListNodeTags(nodeID string) ([]entity.NodeTagEntity, error) {
	rows, err := s.db.Query(`
		SELECT node_id, tag, system_managed, created_at
		FROM node_tags
		WHERE node_id = ?
		ORDER BY tag ASC
	`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []entity.NodeTagEntity
	for rows.Next() {
		var tag entity.NodeTagEntity
		var systemManaged int
		var createdAt string
		if err := rows.Scan(&tag.NodeID, &tag.Tag, &systemManaged, &createdAt); err != nil {
			return nil, err
		}
		tag.SystemManaged = systemManaged == 1
		tag.CreatedAt, _ = parseTime(createdAt)
		tags = append(tags, tag)
	}
	return tags, nil
}

func (s *SqliteDB) SaveNodeTag(tag entity.NodeTagEntity) error {
	systemManaged := 0
	if tag.SystemManaged {
		systemManaged = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO node_tags (node_id, tag, system_managed, created_at)
		VALUES (?, ?, ?, DATETIME('now'))
		ON CONFLICT(node_id, tag) DO UPDATE SET
			system_managed = MAX(node_tags.system_managed, excluded.system_managed)
	`, tag.NodeID, tag.Tag, systemManaged)
	return err
}

func (s *SqliteDB) DeleteNodeTag(nodeID string, tag string) error {
	_, err := s.db.Exec(`
		DELETE FROM node_tags
		WHERE node_id = ? AND tag = ? AND system_managed = 0
	`, nodeID, tag)
	return err
}

func (s *SqliteDB) IncrementNodeRunningCount(nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE nodes
		SET running_count = running_count + 1, updated_at = DATETIME('now')
		WHERE node_id = ?
	`, nodeID)
	return err
}

func (s *SqliteDB) DecrementNodeRunningCount(nodeID string) error {
	_, err := s.db.Exec(`
		UPDATE nodes
		SET running_count = CASE WHEN running_count > 0 THEN running_count - 1 ELSE 0 END,
		    updated_at = DATETIME('now')
		WHERE node_id = ?
	`, nodeID)
	return err
}

type nodeScanner interface {
	Scan(dest ...any) error
}

func scanNode(scanner nodeScanner) (entity.NodeEntity, error) {
	var node entity.NodeEntity
	var lastHeartbeat, lastStream sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(
		&node.NodeID,
		&node.Name,
		&node.IP,
		&node.Kind,
		&node.Status,
		&node.MaxConcurrency,
		&node.RunningCount,
		&lastHeartbeat,
		&lastStream,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return entity.NodeEntity{}, err
	}
	node.LastHeartbeatAt = parseNullableTime(lastHeartbeat)
	node.LastStreamAt = parseNullableTime(lastStream)
	node.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	node.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return node, nil
}

func formatNullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.Format("2006-01-02 15:04:05")
}
