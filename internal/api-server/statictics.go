package apiserver

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/web"
)

// StaticHandler 静态文件处理器
type StaticHandler struct {
	distFS       fs.FS
	staticServer http.Handler
}

// NewStaticHandler 创建静态文件处理器
func NewStaticHandler() *StaticHandler {
	distFS, err := web.DistFS()
	if err != nil {
		logrus.WithError(err).Warn("Failed to load static files")
		return nil
	}

	return &StaticHandler{
		distFS:       distFS,
		staticServer: http.FileServer(http.FS(distFS)),
	}
}

// RegisterRoutes 注册静态文件路由
func (h *StaticHandler) RegisterRoutes(engine *gin.Engine) {
	if h == nil || h.distFS == nil {
		logrus.Warn("Static handler not initialized, skipping static routes")
		return
	}

	// 静态资源目录 (js, css, images 等)
	engine.GET("/assets/*filepath", func(c *gin.Context) {
		c.Request.URL.Path = "/assets" + c.Param("filepath")
		h.staticServer.ServeHTTP(c.Writer, c.Request)
	})

	// favicon
	engine.GET("/favicon.ico", func(c *gin.Context) {
		h.staticServer.ServeHTTP(c.Writer, c.Request)
	})

	// Docs assets must be served as real static files. If they fall through to
	// the SPA fallback, the docs iframe renders the application shell instead.
	engine.GET("/docsify/*filepath", func(c *gin.Context) {
		h.serveStaticFile(c, "docsify", "index.html")
	})
	engine.GET("/docs-content/*filepath", func(c *gin.Context) {
		h.serveStaticFile(c, "docs-content", "README.md")
	})

	// SPA fallback: 所有非 API、非静态资源的请求都返回 index.html
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 跳过 API 请求
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// 尝试提供静态文件
		filePath := strings.TrimPrefix(path, "/")
		if filePath != "" {
			if _, err := fs.Stat(h.distFS, filePath); err == nil {
				h.staticServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 返回 index.html (SPA 路由)
		indexFile, err := fs.ReadFile(h.distFS, "index.html")
		if err != nil {
			c.String(500, "Internal Server Error")
			return
		}
		c.Data(200, "text/html; charset=utf-8", indexFile)
	})
}

func (h *StaticHandler) serveStaticFile(c *gin.Context, root string, defaultFile string) {
	requestPath := strings.TrimPrefix(c.Request.URL.Path, "/")
	filePath := path.Clean(requestPath)
	if filePath == "." || filePath == root {
		filePath = path.Join(root, defaultFile)
	}
	if !strings.HasPrefix(filePath, root+"/") {
		c.String(http.StatusForbidden, "Forbidden")
		return
	}

	info, err := fs.Stat(h.distFS, filePath)
	if err != nil {
		c.String(http.StatusNotFound, "404 page not found")
		return
	}
	if info.IsDir() {
		filePath = path.Join(filePath, defaultFile)
	}

	content, err := fs.ReadFile(h.distFS, filePath)
	if err != nil {
		c.String(http.StatusNotFound, "404 page not found")
		return
	}

	c.Data(http.StatusOK, contentTypeForStaticFile(filePath, content), content)
}

func contentTypeForStaticFile(filePath string, content []byte) string {
	if strings.HasSuffix(filePath, ".md") {
		return "text/markdown; charset=utf-8"
	}
	if contentType := mime.TypeByExtension(path.Ext(filePath)); contentType != "" {
		return contentType
	}
	return http.DetectContentType(content)
}
