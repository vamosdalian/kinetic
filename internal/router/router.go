package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RouteRegistrar func(engine *gin.Engine)

type Router struct {
	engine     *gin.Engine
	httpServer *http.Server
	addr       string
}

type Option func(*Router)

func WithAddr(addr string) Option {
	return func(r *Router) {
		r.addr = addr
	}
}

func WithReadTimeout(d time.Duration) Option {
	return func(r *Router) {
		if r.httpServer != nil {
			r.httpServer.ReadTimeout = d
		}
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(r *Router) {
		if r.httpServer != nil {
			r.httpServer.WriteTimeout = d
		}
	}
}

func New(opts ...Option) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(ginLogger())

	r := &Router{
		engine: engine,
		addr:   ":8080",
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
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logrus.Infof("HTTP server starting on %s", r.addr)
	if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (r *Router) Shutdown(ctx context.Context) error {
	if r.httpServer != nil {
		log.Println("HTTP server shutting down...")
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
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)
		reqMethod := c.Request.Method
		reqUri := c.Request.RequestURI
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		if len(c.Errors) > 0 {
			logrus.Error(c.Errors.String())
		} else {
			msg := fmt.Sprintf("[%d] %s %s | %s | %s",
				statusCode,
				reqMethod,
				reqUri,
				clientIP,
				latencyTime,
			)

			if statusCode >= 500 {
				logrus.Error(msg)
			} else if statusCode >= 400 {
				logrus.Warn(msg)
			} else {
				logrus.Info(msg)
			}
		}
	}
}
