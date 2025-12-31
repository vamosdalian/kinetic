package sqlite

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

const LatestVersion = 1

const (
	version1Sql = `
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
`
)

func (s *SqliteDB) Migrate() error {
	var current int
	s.db.QueryRow("PRAGMA user_version;").Scan(&current)
	if current >= LatestVersion {
		logrus.Infof("[Database] Database is up to date at version %d", current)
		return nil
	}
	logrus.Infof("[Database] Current database version: %d, starting migration", current)
	switch current {
	case 0:
		_, err := s.db.Exec(version1Sql)
		if err != nil {
			return fmt.Errorf("[Database] Failed to migrate database to version 1: %w", err)
		}
		_, err = s.db.Exec("PRAGMA user_version = 1")
		if err != nil {
			return fmt.Errorf("[Database] Failed to set database version: %w", err)
		}
		logrus.Infof("[Database] Migrated database to version 1")
	default:
		logrus.Errorf("[Database] Unknown database version: %d", current)
		return fmt.Errorf("[Database] Unknown database version: %d", current)
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
