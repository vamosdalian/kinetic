package sqlite

import (
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func (s *SqliteDB) GetUserByID(id string) (entity.UserEntity, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, permission, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id)
	return scanUser(row)
}

func (s *SqliteDB) GetUserByUsername(username string) (entity.UserEntity, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, permission, created_at, updated_at
		FROM users
		WHERE username = ?
	`, strings.TrimSpace(username))
	return scanUser(row)
}

func (s *SqliteDB) ListUsers() ([]entity.UserEntity, error) {
	rows, err := s.db.Query(`
		SELECT id, username, password_hash, permission, created_at, updated_at
		FROM users
		ORDER BY username ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]entity.UserEntity, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *SqliteDB) CreateUser(user entity.UserEntity) (entity.UserEntity, error) {
	if strings.TrimSpace(user.ID) == "" {
		user.ID = uuid.NewString()
	}
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, permission, created_at, updated_at)
		VALUES (?, ?, ?, ?, DATETIME('now'), DATETIME('now'))
	`, user.ID, strings.TrimSpace(user.Username), user.PasswordHash, user.Permission)
	if err != nil {
		return entity.UserEntity{}, err
	}
	return s.GetUserByID(user.ID)
}

func (s *SqliteDB) UpdateUserPassword(userID string, passwordHash string) error {
	_, err := s.db.Exec(`
		UPDATE users
		SET password_hash = ?, updated_at = DATETIME('now')
		WHERE id = ?
	`, passwordHash, userID)
	return err
}

func (s *SqliteDB) UpsertUser(user entity.UserEntity) (entity.UserEntity, error) {
	if strings.TrimSpace(user.ID) == "" {
		existing, err := s.GetUserByUsername(user.Username)
		if err == nil {
			user.ID = existing.ID
		} else if err == sql.ErrNoRows {
			user.ID = uuid.NewString()
		} else {
			return entity.UserEntity{}, err
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, permission, created_at, updated_at)
		VALUES (?, ?, ?, ?, DATETIME('now'), DATETIME('now'))
		ON CONFLICT(username) DO UPDATE SET
			password_hash = excluded.password_hash,
			permission = excluded.permission,
			updated_at = DATETIME('now')
	`, user.ID, strings.TrimSpace(user.Username), user.PasswordHash, user.Permission)
	if err != nil {
		return entity.UserEntity{}, err
	}
	return s.GetUserByUsername(user.Username)
}

func (s *SqliteDB) DeleteUser(userID string) error {
	_, err := s.db.Exec(`
		DELETE FROM users
		WHERE id = ?
	`, userID)
	return err
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (entity.UserEntity, error) {
	var user entity.UserEntity
	var createdAt string
	var updatedAt string
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Permission,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return entity.UserEntity{}, err
	}
	user.CreatedAt, _ = parseTime(createdAt)
	user.UpdatedAt, _ = parseTime(updatedAt)
	return user, nil
}
