package sqlite

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

type SqliteDB struct {
	dbPath string
	db     *sql.DB
}

func NewSqliteDB(dbPath string) (*SqliteDB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &SqliteDB{
		dbPath: dbPath,
		db:     db,
	}

	if err := s.Migrate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SqliteDB) Close() error {
	return s.db.Close()
}

func (s *SqliteDB) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
