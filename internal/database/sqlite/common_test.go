package sqlite

import (
	"os"
	"testing"
)

func TestCreateDB(t *testing.T) {
	defer os.Remove("test.db")
	db, err := NewSqliteDB("test.db")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	currentVersion, err := db.GetCurrentVersion()
	if err != nil {
		t.Fatalf("Failed to get current database version: %v", err)
	}
	expectedVersion := len(versionSql) - 1
	if currentVersion != expectedVersion {
		t.Fatalf("Expected database version %d, got %d", expectedVersion, currentVersion)
	}
}
