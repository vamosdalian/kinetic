package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	apiserver "github.com/vamosdalian/kinetic/internal/api-server"
	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/database/sqlite"
	"github.com/vamosdalian/kinetic/internal/router"
	"github.com/vamosdalian/kinetic/internal/scheduler"
	"github.com/vamosdalian/kinetic/internal/service"
	"github.com/vamosdalian/kinetic/internal/worker"
)

type Controller struct {
	cfg            *config.Config
	db             database.Database
	router         *router.Router
	scheduler      *scheduler.Scheduler
	apiServer      *apiserver.APIServer
	runService     *service.RunService
	nodeService    *service.NodeService
	embeddedWorker *worker.Worker
}

func NewController(cfg *config.Config) (*Controller, error) {
	db, err := sqlite.NewSqliteDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	r := router.New(router.WithAddr(cfg.APIAddr()))

	runService := service.NewRunService(db, cfg.Worker.MaxConcurrency)
	authService := service.NewAuthService(db, cfg.Controller.AuthSecret)
	if _, err := authService.SyncBootstrapAdmin(context.Background(), cfg.Controller.AdminUsername, cfg.Controller.AdminPassword); err != nil {
		return nil, fmt.Errorf("failed to sync bootstrap admin: %w", err)
	}
	userService := service.NewUserService(db)
	streamHub := service.NewWorkerStreamHub()
	runService.EnableDistributed(streamHub)
	heartbeatTimeout := time.Duration(cfg.Worker.HeartbeatInterval*3) * time.Second
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = 15 * time.Second
	}
	nodeService := service.NewNodeService(db, runService, streamHub, heartbeatTimeout)
	schedulerInterval := time.Duration(cfg.Controller.SchedulerInterval) * time.Second
	sched := scheduler.NewSchedulerWithInterval(nodeService, schedulerInterval)

	apiServer := apiserver.NewAPIServer(db, sched, r, runService, nodeService, authService, userService, cfg.Controller.AdminUsername)

	var embeddedWorker *worker.Worker
	if cfg.Controller.EmbeddedWorkerEnabled {
		embeddedCfg := *cfg
		embeddedCfg.Mode = config.ModeWorker
		embeddedCfg.Worker.ID = cfg.Worker.ID + "-embedded"
		embeddedCfg.Worker.Name = cfg.Worker.Name + " (local)"
		embeddedWorker = worker.NewWorker(&embeddedCfg, "local")
	}

	return &Controller{
		cfg:            cfg,
		db:             db,
		router:         r,
		scheduler:      sched,
		apiServer:      apiServer,
		runService:     runService,
		nodeService:    nodeService,
		embeddedWorker: embeddedWorker,
	}, nil
}

func (c *Controller) Run() error {
	logger := logrus.WithFields(logrus.Fields{
		"mode":            c.cfg.Mode,
		"embedded_worker": c.embeddedWorker != nil,
	})
	logger.Info("Starting controller")

	go func() {
		if err := c.scheduler.Run(); err != nil {
			logger.WithError(err).Error("Scheduler error")
		}
	}()
	if c.embeddedWorker != nil {
		go func() {
			if err := c.embeddedWorker.Run(); err != nil {
				logger.WithError(err).Error("Embedded worker error")
			}
		}()
	}

	return c.router.Run()
}

func (c *Controller) Shutdown(ctx context.Context) error {
	logger := logrus.WithField("mode", c.cfg.Mode)
	logger.Info("Shutting down controller")

	if err := c.scheduler.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Scheduler shutdown error")
	}
	if c.embeddedWorker != nil {
		if err := c.embeddedWorker.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Embedded worker shutdown error")
		}
	}

	if err := c.router.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Router shutdown error")
	}

	if err := c.db.Close(); err != nil {
		logger.WithError(err).Error("Database close error")
	}

	logger.Info("Controller stopped")
	return nil
}
