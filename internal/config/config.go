package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type Mode string

const (
	ModeController Mode = "controller"
	ModeWorker     Mode = "worker"
)

type DatabaseType string

const (
	DBTypeSQLite   DatabaseType = "sqlite"
	DBTypeMySQL    DatabaseType = "mysql"
	DBTypePostgres DatabaseType = "postgres"
)

type Config struct {
	Mode     Mode           `yaml:"mode"     env:"KINETIC_MODE, overwrite"`
	API      APIConfig      `yaml:"api"      env:", prefix=KINETIC_API_"`
	Database DatabaseConfig `yaml:"database" env:", prefix=KINETIC_DATABASE_"`
	Worker   WorkerConfig   `yaml:"worker"   env:", prefix=KINETIC_WORKER_"`
	Log      LogConfig      `yaml:"log"      env:", prefix=KINETIC_LOG_"`
}

type APIConfig struct {
	Host string `yaml:"host" env:"HOST, overwrite"`
	Port int    `yaml:"port" env:"PORT, overwrite"`
}

type DatabaseConfig struct {
	Type     DatabaseType `yaml:"type"     env:"TYPE, overwrite"`
	Host     string       `yaml:"host"     env:"HOST, overwrite"`
	Port     int          `yaml:"port"     env:"PORT, overwrite"`
	User     string       `yaml:"user"     env:"USER, overwrite"`
	Password string       `yaml:"password" env:"PASSWORD, overwrite"`
	Database string       `yaml:"database" env:"NAME, overwrite"`
	Path     string       `yaml:"path"     env:"PATH, overwrite"`
}

type WorkerConfig struct {
	ControllerURL  string `yaml:"controller_url"  env:"CONTROLLER_URL, overwrite"`
	MaxConcurrency int    `yaml:"max_concurrency" env:"MAX_CONCURRENCY, overwrite"`
}

type LogConfig struct {
	Level  string `yaml:"level"  env:"LEVEL, overwrite"`
	Format string `yaml:"format" env:"FORMAT, overwrite"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Mode: ModeController,
		API: APIConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Type: DBTypeSQLite,
			Path: filepath.Join(homeDir, ".kinetic", "kinetic.db"),
		},
		Worker: WorkerConfig{
			ControllerURL:  "http://localhost:8080",
			MaxConcurrency: 10,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func configPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kinetic", "config.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	path := configPath()
	if err := cfg.loadFromFile(path); err != nil {
		if os.IsNotExist(err) {
			if err := cfg.save(path); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			fmt.Printf("Created default config at %s\n", path)
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	ctx := context.Background()
	if err := envconfig.Process(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func (c *Config) save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	header := `# Kinetic Configuration File
#
# Mode: controller or worker
#   - controller: runs API server, scheduler, and executor
#   - worker: runs executor only, connects to controller
#
# Environment variables can override any setting:
#   KINETIC_MODE, KINETIC_API_HOST, KINETIC_API_PORT,
#   KINETIC_DATABASE_TYPE, KINETIC_DATABASE_PATH, etc.
#

`
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}

func (c *Config) APIAddr() string {
	return fmt.Sprintf("%s:%d", c.API.Host, c.API.Port)
}

func (c *Config) IsController() bool {
	return c.Mode == ModeController
}

func (c *Config) IsWorker() bool {
	return c.Mode == ModeWorker
}
