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
	if currentVersion != LatestVersion {
		t.Fatalf("Expected database version %d, got %d", LatestVersion, currentVersion)
	}
}
