package apiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticHandler_DocsPageAndAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewStaticHandler()
	require.NotNil(t, handler)

	router := gin.New()
	handler.RegisterRoutes(router)

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		contains    string
	}{
		{
			name:        "docs spa route returns app shell",
			path:        "/docs",
			status:      http.StatusOK,
			contentType: "text/html",
			contains:    `<div id="root"></div>`,
		},
		{
			name:        "docsify iframe html is served directly",
			path:        "/docsify/index.html",
			status:      http.StatusOK,
			contentType: "text/html",
			contains:    `Kinetic Docs`,
		},
		{
			name:        "docs markdown content is served directly",
			path:        "/docs-content/README.md",
			status:      http.StatusOK,
			contentType: "text/markdown",
			contains:    `# Kinetic Docs`,
		},
		{
			name:        "missing docs asset does not fall back to app shell",
			path:        "/docsify/missing.js",
			status:      http.StatusNotFound,
			contentType: "text/plain",
			contains:    "404 page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tt.status, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), tt.contentType)
			assert.Contains(t, w.Body.String(), tt.contains)
		})
	}
}
