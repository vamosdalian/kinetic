package apiserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/clusterauth"
)

func newInternalAuthRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/internal")
	group.Use(InternalAuthMiddleware(secret))
	group.POST("/nodes/:id/task-events", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return router
}

func TestInternalAuthMiddleware_AllowsSignedRequest(t *testing.T) {
	const secret = "cluster-secret"
	router := newInternalAuthRouter(secret)

	path := "/api/internal/nodes/node-1/task-events"
	body := []byte(`{"type":"started"}`)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range clusterauth.Headers(secret, http.MethodPost, path, body, time.Now()) {
		req.Header.Set(k, v)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestInternalAuthMiddleware_RejectsUnsignedRequest(t *testing.T) {
	router := newInternalAuthRouter("cluster-secret")

	path := "/api/internal/nodes/node-1/task-events"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{"type":"started"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestInternalAuthMiddleware_RejectsWrongSecret(t *testing.T) {
	router := newInternalAuthRouter("cluster-secret")

	path := "/api/internal/nodes/node-1/task-events"
	body := []byte(`{"type":"started"}`)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range clusterauth.Headers("attacker-secret", http.MethodPost, path, body, time.Now()) {
		req.Header.Set(k, v)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestInternalAuthMiddleware_EmptySecretRequiresValidSignature(t *testing.T) {
	router := newInternalAuthRouter("")

	path := "/api/internal/nodes/node-1/task-events"
	body := []byte(`{"type":"started"}`)

	// No signature headers -> rejected even when the secret is empty.
	unsigned := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	unsigned.Header.Set("Content-Type", "application/json")
	unsignedResp := httptest.NewRecorder()
	router.ServeHTTP(unsignedResp, unsigned)
	assert.Equal(t, http.StatusUnauthorized, unsignedResp.Code)

	// Empty-key signature -> accepted (uniform code path, public key).
	signed := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	signed.Header.Set("Content-Type", "application/json")
	for k, v := range clusterauth.Headers("", http.MethodPost, path, body, time.Now()) {
		signed.Header.Set(k, v)
	}
	signedResp := httptest.NewRecorder()
	router.ServeHTTP(signedResp, signed)
	assert.Equal(t, http.StatusOK, signedResp.Code)
}
