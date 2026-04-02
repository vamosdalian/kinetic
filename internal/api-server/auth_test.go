package apiserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/service"
)

func setupAuthRouter(t *testing.T, bootstrapUsername string, bootstrapPassword string) *gin.Engine {
	t.Helper()

	db := setupTestDB(t)
	authService := service.NewAuthService(db, "test-auth-secret")
	_, err := authService.SyncBootstrapAdmin(t.Context(), bootstrapUsername, bootstrapPassword)
	require.NoError(t, err)
	userService := service.NewUserService(db)

	authHandler := NewAuthHandler(authService)
	adminHandler := NewAdminHandler(userService, bootstrapUsername)

	router := gin.New()
	router.POST("/api/auth/login", authHandler.Login)

	protected := router.Group("/api")
	protected.Use(AuthMiddleware(authService))
	protected.GET("/auth/me", authHandler.Me)
	protected.GET("/admin/users", adminHandler.ListUsers)
	protected.POST("/admin/users", adminHandler.CreateUser)
	protected.PUT("/admin/users/:id/password", adminHandler.UpdateUserPassword)
	protected.DELETE("/admin/users/:id", adminHandler.DeleteUser)

	return router
}

func loginAndExtractToken(t *testing.T, router *gin.Engine, username string, password string) string {
	t.Helper()

	body, err := json.Marshal(dto.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	var payload APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	data := payload.Data.(map[string]any)
	token, ok := data["token"].(string)
	require.True(t, ok)
	return token
}

func TestAuthHandler_Login_SuccessAndInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupAuthRouter(t, "admin", "secret-password")

	reqBody, err := json.Marshal(dto.LoginRequest{
		Username: "admin",
		Password: "secret-password",
	})
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	invalidBody, err := json.Marshal(dto.LoginRequest{
		Username: "admin",
		Password: "wrong-password",
	})
	require.NoError(t, err)
	invalidReq, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(invalidBody))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidResp := httptest.NewRecorder()
	router.ServeHTTP(invalidResp, invalidReq)
	assert.Equal(t, http.StatusUnauthorized, invalidResp.Code)

	var payload APIResponse
	require.NoError(t, json.Unmarshal(invalidResp.Body.Bytes(), &payload))
	require.NotNil(t, payload.Error)
	assert.Equal(t, ErrorCodeInvalidCredentials, payload.Error.Code)
}

func TestAuthMiddleware_ProtectsRoutesAndReturnsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupAuthRouter(t, "admin", "secret-password")

	req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	token := loginAndExtractToken(t, router, "admin", "secret-password")

	protectedReq, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+token)
	protectedResp := httptest.NewRecorder()
	router.ServeHTTP(protectedResp, protectedReq)
	assert.Equal(t, http.StatusOK, protectedResp.Code)
}

func TestAdminUsersAPI_CreateResetDeleteAndRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupAuthRouter(t, "admin", "secret-password")
	token := loginAndExtractToken(t, router, "admin", "secret-password")

	createBody, err := json.Marshal(dto.CreateUserRequest{
		Username: "operator",
		Password: "operator-password",
	})
	require.NoError(t, err)

	createReq, _ := http.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBuffer(createBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	assert.Equal(t, http.StatusOK, createResp.Code)

	listReq, _ := http.NewRequest(http.MethodGet, "/api/admin/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	assert.Equal(t, http.StatusOK, listResp.Code)

	var listPayload APIResponse
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listPayload))
	items := listPayload.Data.([]any)
	require.Len(t, items, 2)

	var createdID string
	var bootstrapID string
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["username"] == "operator" {
			createdID = item["id"].(string)
		}
		if item["username"] == "admin" {
			bootstrapID = item["id"].(string)
		}
	}
	require.NotEmpty(t, createdID)
	require.NotEmpty(t, bootstrapID)

	resetBody, err := json.Marshal(dto.UpdateUserPasswordRequest{Password: "new-password"})
	require.NoError(t, err)
	resetReq, _ := http.NewRequest(http.MethodPut, "/api/admin/users/"+createdID+"/password", bytes.NewBuffer(resetBody))
	resetReq.Header.Set("Authorization", "Bearer "+token)
	resetReq.Header.Set("Content-Type", "application/json")
	resetResp := httptest.NewRecorder()
	router.ServeHTTP(resetResp, resetReq)
	assert.Equal(t, http.StatusOK, resetResp.Code)

	deleteBootstrapReq, _ := http.NewRequest(http.MethodDelete, "/api/admin/users/"+bootstrapID, nil)
	deleteBootstrapReq.Header.Set("Authorization", "Bearer "+token)
	deleteBootstrapResp := httptest.NewRecorder()
	router.ServeHTTP(deleteBootstrapResp, deleteBootstrapReq)
	assert.Equal(t, http.StatusBadRequest, deleteBootstrapResp.Code)

	deleteCreatedReq, _ := http.NewRequest(http.MethodDelete, "/api/admin/users/"+createdID, nil)
	deleteCreatedReq.Header.Set("Authorization", "Bearer "+token)
	deleteCreatedResp := httptest.NewRecorder()
	router.ServeHTTP(deleteCreatedResp, deleteCreatedReq)
	assert.Equal(t, http.StatusOK, deleteCreatedResp.Code)
}
