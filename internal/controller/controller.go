package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/sirupsen/logrus"
	apiserver "github.com/vamosdalian/kinetic/internal/api-server"
	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/database/sqlite"
	"github.com/vamosdalian/kinetic/internal/router"
	"github.com/vamosdalian/kinetic/internal/scheduler"
	"github.com/vamosdalian/kinetic/internal/service"
)

type Controller struct {
	cfg       *config.Config
	db        database.Database
	router    *router.Router
	scheduler *scheduler.Scheduler
	apiServer *apiserver.APIServer
}

func NewController(cfg *config.Config) (*Controller, error) {
	db, err := sqlite.NewSqliteDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	r := router.New(router.WithAddr(cfg.APIAddr()))

	sched := scheduler.NewScheduler()
	runService := service.NewRunService(db, cfg.Worker.MaxConcurrency)

	apiServer := apiserver.NewAPIServer(db, sched, r, runService)

	return &Controller{
		cfg:       cfg,
		db:        db,
		router:    r,
		scheduler: sched,
		apiServer: apiServer,
	}, nil
}

func (c *Controller) Run() error {
	logrus.Info("Starting controller...")

	go func() {
		if err := c.scheduler.Run(); err != nil {
			log.Printf("Scheduler error: %v", err)
		}
	}()

	return c.router.Run()
}

func (c *Controller) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down controller...")

	if err := c.scheduler.Shutdown(ctx); err != nil {
		log.Printf("Scheduler shutdown error: %v", err)
	}

	if err := c.router.Shutdown(ctx); err != nil {
		log.Printf("Router shutdown error: %v", err)
	}

	if err := c.db.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}

	logrus.Info("Controller stopped")
	return nil
}
