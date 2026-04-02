package apiserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	"github.com/vamosdalian/kinetic/internal/service"
)

type UserManager interface {
	ListUsers() ([]entity.UserEntity, error)
	CreateUser(username string, password string) (entity.UserEntity, error)
	UpdatePassword(userID string, password string) error
	DeleteUser(userID string) error
}

type AdminHandler struct {
	users             UserManager
	bootstrapUsername string
}

func NewAdminHandler(users UserManager, bootstrapUsername string) *AdminHandler {
	return &AdminHandler{
		users:             users,
		bootstrapUsername: strings.TrimSpace(bootstrapUsername),
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.users.ListUsers()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	items := make([]dto.AdminUser, 0, len(users))
	for _, user := range users {
		items = append(items, h.toAdminUser(user))
	}
	ResponseSuccess(c, items)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	user, err := h.users.CreateUser(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			ResponseError(c, http.StatusConflict, ErrorCodeUserAlreadyExists, "User already exists")
		default:
			ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		}
		return
	}
	ResponseSuccess(c, h.toAdminUser(user))
}

func (h *AdminHandler) UpdateUserPassword(c *gin.Context) {
	var req dto.UpdateUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	if err := h.users.UpdatePassword(c.Param("id"), req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			ResponseError(c, http.StatusNotFound, ErrorCodeUserNotFound, "User not found")
		default:
			ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		}
		return
	}
	ResponseSuccess(c, gin.H{"id": c.Param("id")})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		ResponseError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized")
		return
	}
	targetID := c.Param("id")
	if targetID == user.ID {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "You cannot delete your own account")
		return
	}

	allUsers, err := h.users.ListUsers()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	var target *entity.UserEntity
	for i := range allUsers {
		if allUsers[i].ID == targetID {
			target = &allUsers[i]
			break
		}
	}
	if target == nil {
		ResponseError(c, http.StatusNotFound, ErrorCodeUserNotFound, "User not found")
		return
	}
	if strings.EqualFold(target.Username, h.bootstrapUsername) {
		ResponseError(c, http.StatusBadRequest, ErrorCodeBootstrapAdminDeleteForbidden, "Bootstrap admin cannot be deleted")
		return
	}

	if err := h.users.DeleteUser(targetID); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			ResponseError(c, http.StatusNotFound, ErrorCodeUserNotFound, "User not found")
			return
		}
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	ResponseSuccess(c, gin.H{"id": targetID})
}

func (h *AdminHandler) toAdminUser(user entity.UserEntity) dto.AdminUser {
	return dto.AdminUser{
		ID:               user.ID,
		Username:         user.Username,
		Permission:       user.Permission,
		IsBootstrapAdmin: strings.EqualFold(user.Username, h.bootstrapUsername),
		CreatedAt:        formatAdminTime(user.CreatedAt),
		UpdatedAt:        formatAdminTime(user.UpdatedAt),
	}
}

func formatAdminTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
