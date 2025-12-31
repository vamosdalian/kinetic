package sqlite

import (
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
	s := &SqliteDB{
		dbPath: dbPath,
		db:     db,
	}

	if s.Migrate() != nil {
		return nil, err
	}

	return s, nil
}

func (s *SqliteDB) Close() error {
	return s.db.Close()
}
