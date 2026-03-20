package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"
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

type cliOptions struct {
	ShowVersion bool   `long:"version" description:"Show version information"`
	ConfigPath  string `short:"c" long:"config" value-name:"PATH" description:"Path to config.yml"`
	Mode        string `long:"mode" choice:"controller" choice:"worker" description:"Run mode: controller or worker (overrides config)"`
	WithWorker  bool   `long:"with-worker" description:"Enable embedded worker when running in controller mode"`
}

func parseCLIOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.ParseArgs(args)
	return opts, err
}

func main() {
	opts, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		var flagErr *flags.Error
		if errors.As(err, &flagErr) && flagErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		logrus.WithError(err).Fatal("Failed to parse CLI options")
	}

	if opts.ShowVersion {
		fmt.Printf("Kinetic %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	cfg, configPath, shouldPersistConfig, err := loadRuntimeConfig(opts)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to prepare config")
	}

	if err := validateMode(cfg.Mode); err != nil {
		logrus.WithError(err).Fatal("Invalid runtime config")
	}

	if err := persistMissingConfig(cfg, configPath, shouldPersistConfig); err != nil {
		logrus.WithError(err).Fatal("Failed to persist config")
	}

	configureLogger(cfg.Log)
	logrus.WithFields(logrus.Fields{
		"config_path":       configPath,
		"mode":              cfg.Mode,
		"embedded_worker":   cfg.Controller.EmbeddedWorkerEnabled,
		"api_addr":          cfg.APIAddr(),
		"database_type":     cfg.Database.Type,
		"database_path":     cfg.Database.Path,
		"worker_controller": cfg.Worker.ControllerURL,
	}).Info("Starting Kinetic")

	switch cfg.Mode {
	case config.ModeController:
		runController(cfg)
	case config.ModeWorker:
		runWorker(cfg)
	default:
		logrus.WithField("mode", cfg.Mode).Fatal("Unknown mode")
	}
}

func loadRuntimeConfig(opts cliOptions) (*config.Config, string, bool, error) {
	result, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, "", false, err
	}

	cfg := result.Config
	applyCLIOverrides(cfg, opts)

	return cfg, result.Path, !result.FileExists, nil
}

func applyCLIOverrides(cfg *config.Config, opts cliOptions) {
	if opts.Mode != "" {
		cfg.Mode = config.Mode(opts.Mode)
	}
	if opts.WithWorker {
		cfg.Controller.EmbeddedWorkerEnabled = true
	}
}

func validateMode(mode config.Mode) error {
	switch mode {
	case config.ModeController, config.ModeWorker:
		return nil
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

func persistMissingConfig(cfg *config.Config, path string, shouldPersist bool) error {
	if !shouldPersist {
		return nil
	}

	if err := cfg.Save(path); err != nil {
		return err
	}

	fmt.Printf("Created config at %s\n", path)
	return nil
}

func configureLogger(cfg config.LogConfig) {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, falling back to info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
}

func runController(cfg *config.Config) {
	ctrl, err := controller.NewController(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create controller")
	}

	go func() {
		if err := ctrl.Run(); err != nil {
			logrus.WithError(err).Fatal("Controller error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ctrl.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("Controller shutdown error")
	}
}

// runWorker 以 Worker 模式运行
// Worker 包含: Executor
func runWorker(cfg *config.Config) {
	w := worker.NewWorker(cfg, "remote")

	// 启动 Worker
	go func() {
		if err := w.Run(); err != nil {
			logrus.WithError(err).Fatal("Worker error")
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
		logrus.WithError(err).Fatal("Worker shutdown error")
	}
}
