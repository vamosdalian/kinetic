package sqlite

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

var versionSql = []string{
	1: `
CREATE TABLE IF NOT EXISTS workflows (
	id TEXT PRIMARY KEY, 
	name TEXT,
	description TEXT, 
	version INTEGER, 
	enable INTEGER, 
	created_at TEXT, 
	updated_at TEXT
);
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY, 
	workflow_id TEXT, 
	name TEXT,
	description TEXT,
	type TEXT, 
	config TEXT, 
	position TEXT, 
	node_type TEXT
);
CREATE TABLE IF NOT EXISTS edges (
	id TEXT PRIMARY KEY, 
	workflow_id TEXT, 
	source TEXT, 
	target TEXT, 
	source_handle TEXT, 
	target_handle TEXT
);
`,
	2: `
CREATE TABLE IF NOT EXISTS workflow_runs (
	run_id TEXT PRIMARY KEY, 
	workflow_id TEXT, 
	workflow_name TEXT, 
	workflow_description TEXT, 
	workflow_version INTEGER, 
	status TEXT, 
	created_at TEXT, 
	started_at TEXT, 
	finished_at TEXT
);
CREATE TABLE IF NOT EXISTS task_runs (
	run_id TEXT, 
	task_id TEXT, 
	workflow_id TEXT, 
	task_name TEXT, 
	task_description TEXT, 
	task_type TEXT, 
	task_config TEXT, 
	task_position TEXT, 
	task_node_type TEXT, 
	status TEXT, 
	created_at TEXT, 
	started_at TEXT, 
	finished_at TEXT, 
	exit_code INTEGER, 
	output TEXT
);
CREATE TABLE IF NOT EXISTS edge_runs (
	run_id TEXT, 
	edge_id TEXT, 
	workflow_id TEXT, 
	edge_source TEXT, 
	edge_target TEXT, 
	edge_source_handle TEXT, 
	edge_target_handle TEXT, 
	created_at TEXT
);
`,
	3: `
ALTER TABLE workflows ADD COLUMN tag TEXT DEFAULT '';
ALTER TABLE tasks ADD COLUMN tag TEXT DEFAULT '';
ALTER TABLE workflow_runs ADD COLUMN workflow_tag TEXT DEFAULT '';
ALTER TABLE task_runs ADD COLUMN task_tag TEXT DEFAULT '';
ALTER TABLE task_runs ADD COLUMN effective_tag TEXT DEFAULT '';
ALTER TABLE task_runs ADD COLUMN assigned_node_id TEXT DEFAULT '';
ALTER TABLE task_runs ADD COLUMN assigned_at TEXT;
CREATE TABLE IF NOT EXISTS nodes (
	node_id TEXT PRIMARY KEY,
	name TEXT,
	ip TEXT,
	kind TEXT,
	status TEXT,
	max_concurrency INTEGER,
	running_count INTEGER,
	last_heartbeat_at TEXT,
	last_stream_at TEXT,
	created_at TEXT,
	updated_at TEXT
);
CREATE TABLE IF NOT EXISTS node_tags (
	node_id TEXT,
	tag TEXT,
	system_managed INTEGER,
	created_at TEXT,
	PRIMARY KEY (node_id, tag)
);
`,
	4: `
CREATE INDEX IF NOT EXISTS idx_workflow_runs_created_at_run_id
	ON workflow_runs(created_at DESC, run_id DESC);
CREATE INDEX IF NOT EXISTS idx_task_runs_status_created_at_task_id
	ON task_runs(status, created_at ASC, task_id ASC);
CREATE INDEX IF NOT EXISTS idx_task_runs_assigned_node_status_created_at
	ON task_runs(assigned_node_id, status, created_at ASC, task_id ASC);
CREATE INDEX IF NOT EXISTS idx_task_runs_run_id_task_id
	ON task_runs(run_id, task_id);
CREATE INDEX IF NOT EXISTS idx_nodes_status_running_count_node_id
	ON nodes(status, running_count ASC, node_id ASC);
`,
}

func (s *SqliteDB) Migrate() error {
	current, err := s.GetCurrentVersion()
	if err != nil {
		return err
	}
	if current >= len(versionSql)-1 {
		logrus.Infof("[Database] Database is up to date at version %d", current)
		return nil
	}
	logrus.Infof("[Database] Current database version: %d, starting migration", current)

	for i := current + 1; i < len(versionSql); i++ {
		_, err := s.db.Exec(versionSql[i])
		if err != nil {
			return fmt.Errorf("[Database] Failed to migrate database to version %d: %w", i, err)
		}
		_, err = s.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", i))
		if err != nil {
			return fmt.Errorf("[Database] Failed to set database version: %w", err)
		}
		logrus.Infof("[Database] Migrated database to version %d", i)
	}
	logrus.Infof("[Database] Migration completed")
	return nil
}

func (s *SqliteDB) GetCurrentVersion() (int, error) {
	var current int
	err := s.db.QueryRow("PRAGMA user_version;").Scan(&current)
	if err != nil {
		return 0, fmt.Errorf("[Database] Failed to get current database version: %w", err)
	}
	return current, nil
}
