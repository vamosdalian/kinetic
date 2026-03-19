package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHealthChecker struct {
	err error
}

func (s stubHealthChecker) HealthCheck(ctx context.Context) error {
	return s.err
}

func TestAPIServer_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &APIServer{}
	router := gin.New()
	server.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestAPIServer_Readyz_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &APIServer{db: stubHealthChecker{}}
	router := gin.New()
	server.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", checks["database"])
}

func TestAPIServer_Readyz_DBFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &APIServer{db: stubHealthChecker{err: errors.New("db unavailable")}}
	router := gin.New()
	server.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "db unavailable", checks["database"])
}

func TestAPIServer_Readyz_NoDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &APIServer{}
	router := gin.New()
	server.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "not_configured", checks["database"])
}
