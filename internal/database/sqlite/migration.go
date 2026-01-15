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
