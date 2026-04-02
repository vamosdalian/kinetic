package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrateRemovesLegacyPythonWorkflows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy_python_cleanup.db")
	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open legacy database: %v", err)
	}
	defer legacyDB.Close()

	for version := 1; version <= 6; version++ {
		if _, err := legacyDB.Exec(versionSql[version]); err != nil {
			t.Fatalf("Failed to apply legacy schema version %d: %v", version, err)
		}
	}
	if _, err := legacyDB.Exec("PRAGMA user_version = 6"); err != nil {
		t.Fatalf("Failed to set legacy user_version: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO workflows (id, name, description, config, tag, version, enable, created_at, updated_at)
		VALUES
			('wf-python', 'Python Workflow', '', '{}', '', 1, 1, DATETIME('now'), DATETIME('now')),
			('wf-shell', 'Shell Workflow', '', '{}', '', 1, 1, DATETIME('now'), DATETIME('now'))
	`); err != nil {
		t.Fatalf("Failed to insert workflows: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO tasks (id, workflow_id, name, description, type, config, tag, position, node_type)
		VALUES
			('task-python', 'wf-python', 'Python Task', '', 'python', '{"script":"print(1)"}', '', '{"x":0,"y":0}', 'default'),
			('task-shell', 'wf-shell', 'Shell Task', '', 'shell', '{"script":"echo ok"}', '', '{"x":0,"y":0}', 'default')
	`); err != nil {
		t.Fatalf("Failed to insert tasks: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO edges (id, workflow_id, source, target, source_handle, target_handle)
		VALUES ('edge-python', 'wf-python', 'task-python', 'task-python', '', '')
	`); err != nil {
		t.Fatalf("Failed to insert edges: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO workflow_runs (run_id, workflow_id, workflow_name, workflow_description, workflow_config, workflow_version, workflow_tag, status, created_at, started_at, finished_at)
		VALUES ('run-python', 'wf-python', 'Python Workflow', '', '{}', 1, '', 'failed', DATETIME('now'), DATETIME('now'), DATETIME('now'))
	`); err != nil {
		t.Fatalf("Failed to insert workflow run: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO task_runs (
			run_id, task_id, workflow_id, task_name, task_description, task_type, task_config, task_tag,
			task_position, task_node_type, effective_tag, assigned_node_id, assigned_at,
			status, created_at, started_at, finished_at, exit_code, output, result
		) VALUES (
			'run-python', 'task-python', 'wf-python', 'Python Task', '', 'python', '{"script":"print(1)"}', '',
			'{"x":0,"y":0}', 'default', '', '', NULL,
			'failed', DATETIME('now'), DATETIME('now'), DATETIME('now'), 1, 'boom', ''
		)
	`); err != nil {
		t.Fatalf("Failed to insert task run: %v", err)
	}

	if _, err := legacyDB.Exec(`
		INSERT INTO edge_runs (run_id, edge_id, workflow_id, edge_source, edge_target, edge_source_handle, edge_target_handle, created_at)
		VALUES ('run-python', 'edge-python', 'wf-python', 'task-python', 'task-python', '', '', DATETIME('now'))
	`); err != nil {
		t.Fatalf("Failed to insert edge run: %v", err)
	}

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen migrated database: %v", err)
	}
	defer db.Close()

	assertCount := func(query string, want int) {
		t.Helper()
		var got int
		if err := db.db.QueryRow(query).Scan(&got); err != nil {
			t.Fatalf("Failed to query count for %q: %v", query, err)
		}
		if got != want {
			t.Fatalf("Expected %d rows for %q, got %d", want, query, got)
		}
	}

	currentVersion, err := db.GetCurrentVersion()
	if err != nil {
		t.Fatalf("Failed to get migrated database version: %v", err)
	}
	if currentVersion != len(versionSql)-1 {
		t.Fatalf("Expected database version %d after migration, got %d", len(versionSql)-1, currentVersion)
	}

	assertCount("SELECT COUNT(*) FROM workflows WHERE id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM tasks WHERE workflow_id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM edges WHERE workflow_id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM workflow_runs WHERE workflow_id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM task_runs WHERE workflow_id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM edge_runs WHERE workflow_id = 'wf-python'", 0)
	assertCount("SELECT COUNT(*) FROM workflows WHERE id = 'wf-shell'", 1)

	workflow, err := db.GetWorkflowByID("wf-shell")
	if err != nil {
		t.Fatalf("Expected non-python workflow to survive migration: %v", err)
	}
	if workflow.ID != "wf-shell" {
		t.Fatalf("Expected surviving workflow wf-shell, got %s", workflow.ID)
	}

	if _, err := db.GetWorkflowByID("wf-python"); err == nil {
		t.Fatal("Expected python workflow to be removed during migration")
	}
}

func TestMigrateCreatesUsersTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "users_table.db")
	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'users'
	`).Scan(&count); err != nil {
		t.Fatalf("Failed to check users table: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected users table to exist, got %d matches", count)
	}
}
