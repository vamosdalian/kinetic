package apiserver

import (
	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/router"
	"github.com/vamosdalian/kinetic/internal/scheduler"
)

type APIServer struct {
	scheduler       *scheduler.Scheduler
	workflowHandler *WorkflowHandler
	nodeHandler     *NodeHandler
	staticHandler   *StaticHandler
}

func NewAPIServer(db database.Database, scheduler *scheduler.Scheduler, r *router.Router, runService RunManager, nodeService NodeManager) *APIServer {
	workflowHandler := NewWorkflowHandler(db)
	workflowHandler.SetRunService(runService)

	apiServer := &APIServer{
		scheduler:       scheduler,
		workflowHandler: workflowHandler,
		nodeHandler:     NewNodeHandler(nodeService),
		staticHandler:   NewStaticHandler(),
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
	// Health check
	engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := engine.Group("/api")
	{
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
