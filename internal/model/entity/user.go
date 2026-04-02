package entity

import "time"

const UserPermissionAdmin = "admin"

type UserEntity struct {
	ID           string
	Username     string
	PasswordHash string
	Permission   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
