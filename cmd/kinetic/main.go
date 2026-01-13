package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/controller"
	"github.com/vamosdalian/kinetic/internal/worker"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		showVersion bool
		mode        string
	)
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.StringVar(&mode, "mode", "controller", "Run mode: controller or worker (overrides config)")
	flag.Parse()

	if showVersion {
		fmt.Printf("Kinetic %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if mode != "" {
		cfg.Mode = config.Mode(mode)
	}

	logrus.SetLevel(logrus.DebugLevel)
	logrus.Infof("Starting Kinetic in %s mode...", cfg.Mode)

	switch cfg.Mode {
	case config.ModeController:
		runController(cfg)
	case config.ModeWorker:
		runWorker(cfg)
	default:
		log.Fatalf("Unknown mode: %s", cfg.Mode)
	}
}

func runController(cfg *config.Config) {
	ctrl, err := controller.NewController(cfg)
	if err != nil {
		log.Fatalf("Failed to create controller: %v", err)
	}

	go func() {
		if err := ctrl.Run(); err != nil {
			log.Fatalf("Controller error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ctrl.Shutdown(ctx); err != nil {
		log.Fatalf("Controller shutdown error: %v", err)
	}
}

// runWorker 以 Worker 模式运行
// Worker 包含: Executor
func runWorker(cfg *config.Config) {
	w := worker.NewWorker(cfg)

	// 启动 Worker
	go func() {
		if err := w.Run(); err != nil {
			log.Fatalf("Worker error: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.Shutdown(ctx); err != nil {
		log.Fatalf("Worker shutdown error: %v", err)
	}
}
