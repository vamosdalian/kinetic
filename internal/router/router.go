package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RouteRegistrar func(engine *gin.Engine)

type Router struct {
	engine       *gin.Engine
	httpServer   *http.Server
	addr         string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
}

type Option func(*Router)

func WithAddr(addr string) Option {
	return func(r *Router) {
		r.addr = addr
	}
}

func WithReadTimeout(d time.Duration) Option {
	return func(r *Router) {
		r.readTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(r *Router) {
		r.writeTimeout = d
	}
}

func New(opts ...Option) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(ginLogger())

	r := &Router{
		engine:       engine,
		addr:         ":9898",
		readTimeout:  30 * time.Second,
		writeTimeout: 30 * time.Second,
		idleTimeout:  60 * time.Second,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) Use(middleware ...gin.HandlerFunc) {
	r.engine.Use(middleware...)
}

func (r *Router) Register(registrars ...RouteRegistrar) {
	for _, reg := range registrars {
		reg(r.engine)
	}
}

func (r *Router) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return r.engine.Group(relativePath, handlers...)
}

func (r *Router) Run() error {
	r.httpServer = &http.Server{
		Addr:         r.addr,
		Handler:      r.engine,
		ReadTimeout:  r.readTimeout,
		WriteTimeout: r.writeTimeout,
		IdleTimeout:  r.idleTimeout,
	}

	logrus.Infof("HTTP server starting on %s", r.addr)
	if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (r *Router) Shutdown(ctx context.Context) error {
	if r.httpServer != nil {
		logrus.Info("HTTP server shutting down...")
		return r.httpServer.Shutdown(ctx)
	}
	return nil
}

func (r *Router) Addr() string {
	return r.addr
}

func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latencyTime := time.Since(startTime)
		requestPath := c.Request.URL.Path
		entry := logrus.WithFields(logrus.Fields{
			"status":    c.Writer.Status(),
			"method":    c.Request.Method,
			"path":      c.Request.RequestURI,
			"client_ip": c.ClientIP(),
			"latency":   latencyTime,
		})

		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		} else {
			entry = entry.WithField("errors", "")
		}

		statusCode := c.Writer.Status()
		if statusCode >= 500 {
			entry.Error("http request completed")
		} else if statusCode >= 400 {
			entry.Warn("http request completed")
		} else if shouldLogRequestAtDebug(requestPath) {
			entry.Debug("http request completed")
		} else {
			entry.Info("http request completed")
		}
	}
}

func shouldLogRequestAtDebug(path string) bool {
	if path == "/assets" || strings.HasPrefix(path, "/assets/") {
		return true
	}
	return strings.HasSuffix(path, "/heartbeat")
}
