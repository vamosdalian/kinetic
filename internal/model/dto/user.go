package dto

type AdminUser struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Permission       string `json:"permission"`
	IsBootstrapAdmin bool   `json:"is_bootstrap_admin"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateUserPasswordRequest struct {
	Password string `json:"password"`
}
