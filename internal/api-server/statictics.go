package apiserver

import (
	"io/fs"
	"net/http"
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
