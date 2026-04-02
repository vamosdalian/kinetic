package apiserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	"github.com/vamosdalian/kinetic/internal/service"
)

const authContextUserKey = "auth.user"

type AuthManager interface {
	Authenticate(ctx context.Context, username string, password string) (dto.LoginResponse, error)
	GetUserFromToken(token string) (entity.UserEntity, error)
}

type AuthHandler struct {
	auth AuthManager
}

func NewAuthHandler(auth AuthManager) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	resp, err := h.auth.Authenticate(c.Request.Context(), strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ResponseError(c, http.StatusUnauthorized, ErrorCodeInvalidCredentials, "Invalid username or password")
			return
		}
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		ResponseError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized")
		return
	}
	ResponseSuccess(c, serviceUserToAuthUser(user))
}

func AuthMiddleware(auth AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			ResponseError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		user, err := auth.GetUserFromToken(token)
		if err != nil {
			ResponseError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		if user.Permission != entity.UserPermissionAdmin {
			ResponseError(c, http.StatusForbidden, ErrorCodeForbidden, "Forbidden")
			c.Abort()
			return
		}

		c.Set(authContextUserKey, user)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[len("Bearer "):])
	}
	return strings.TrimSpace(c.Query("access_token"))
}

func currentUser(c *gin.Context) (entity.UserEntity, bool) {
	value, ok := c.Get(authContextUserKey)
	if !ok {
		return entity.UserEntity{}, false
	}
	user, ok := value.(entity.UserEntity)
	return user, ok
}

func serviceUserToAuthUser(user entity.UserEntity) dto.AuthUser {
	return dto.AuthUser{
		ID:         user.ID,
		Username:   user.Username,
		Permission: user.Permission,
	}
}
