package apiserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/router"
	"github.com/vamosdalian/kinetic/internal/scheduler"
	"github.com/vamosdalian/kinetic/internal/service"
)

type healthChecker interface {
	HealthCheck(ctx context.Context) error
}

type APIServer struct {
	scheduler        *scheduler.Scheduler
	db               healthChecker
	workflowHandler  *WorkflowHandler
	nodeHandler      *NodeHandler
	dashboardHandler *DashboardHandler
	staticHandler    *StaticHandler
}

func NewAPIServer(db database.Database, scheduler *scheduler.Scheduler, r *router.Router, runService RunManager, nodeService NodeManager) *APIServer {
	workflowHandler := NewWorkflowHandler(db)
	workflowHandler.SetRunService(runService)

	var dashboardHandler *DashboardHandler
	if store, ok := db.(service.DashboardStore); ok {
		dashboardHandler = NewDashboardHandler(service.NewDashboardService(store))
	}

	apiServer := &APIServer{
		scheduler:        scheduler,
		db:               db,
		workflowHandler:  workflowHandler,
		nodeHandler:      NewNodeHandler(nodeService),
		dashboardHandler: dashboardHandler,
		staticHandler:    NewStaticHandler(),
	}

	// 注册 API 路由
	r.Register(apiServer.RegisterRoutes)

	// 注册静态文件路由（放在最后，因为包含 NoRoute 处理）
	if apiServer.staticHandler != nil {
		r.Register(apiServer.staticHandler.RegisterRoutes)
	}

	return apiServer
}

func (a *APIServer) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/healthz", a.healthz)
	engine.GET("/readyz", a.readyz)
	engine.GET("/api/health", a.healthz)

	api := engine.Group("/api")
	{
		if a.dashboardHandler != nil {
			api.GET("/dashboard", a.dashboardHandler.Get)
		}

		// Workflow routes
		workflows := api.Group("/workflows")
		{
			workflows.GET("", a.workflowHandler.List)
			workflows.GET("/:id", a.workflowHandler.Get)
			workflows.PUT("/:id", a.workflowHandler.Save)
			workflows.DELETE("/:id", a.workflowHandler.Delete)
			workflows.POST("/:id/run", a.workflowHandler.Run)
		}

		runs := api.Group("/workflow_runs")
		{
			runs.GET("", a.workflowHandler.ListRuns)
			runs.GET("/:run_id", a.workflowHandler.GetRun)
			runs.GET("/:run_id/events", a.workflowHandler.RunEvents)
			runs.POST("/:run_id/rerun", a.workflowHandler.Rerun)
			runs.POST("/:run_id/cancel", a.workflowHandler.Cancel)
		}

		nodes := api.Group("/nodes")
		{
			nodes.GET("", a.nodeHandler.List)
			nodes.GET("/:id", a.nodeHandler.Get)
			nodes.POST("/:id/tags", a.nodeHandler.AddTag)
			nodes.DELETE("/:id/tags/:tag", a.nodeHandler.DeleteTag)
		}

		internal := api.Group("/internal")
		{
			internal.POST("/nodes/register", a.nodeHandler.Register)
			internal.GET("/nodes/:id/stream", a.nodeHandler.Stream)
			internal.POST("/nodes/:id/heartbeat", a.nodeHandler.Heartbeat)
			internal.POST("/nodes/:id/task-events", a.nodeHandler.TaskEvents)
		}
	}
}

func (a *APIServer) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (a *APIServer) readyz(c *gin.Context) {
	status := http.StatusOK
	response := gin.H{
		"status": "ready",
		"checks": gin.H{
			"database": "ok",
		},
	}

	if a.db == nil {
		status = http.StatusServiceUnavailable
		response["status"] = "not_ready"
		response["checks"] = gin.H{
			"database": "not_configured",
		}
		c.JSON(status, response)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := a.db.HealthCheck(ctx); err != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "not_ready"
		response["checks"] = gin.H{
			"database": err.Error(),
		}
	}

	c.JSON(status, response)
}
