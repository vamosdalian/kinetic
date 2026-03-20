package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_CreatesDefaultConfigWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, ModeController, cfg.Mode)
	assert.True(t, cfg.Controller.EmbeddedWorkerEnabled)
	assert.Equal(t, "http://localhost:9898", cfg.Worker.ControllerURL)
	assert.Equal(t, 5, cfg.Worker.HeartbeatInterval)
	assert.Equal(t, 10, cfg.Worker.MaxConcurrency)
	assert.Equal(t, filepath.Join(homeDir, ".kinetic", "kinetic.db"), cfg.Database.Path)

	configFile := filepath.Join(homeDir, ".kinetic", "config.yaml")
	content, readErr := os.ReadFile(configFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "KINETIC_WORKER_CONTROLLER_URL")
	assert.Contains(t, string(content), "embedded_worker_enabled: true")
	assert.Contains(t, string(content), "heartbeat_interval: 5")
	assert.Contains(t, string(content), "max_concurrency: 10")
}

func TestLoad_AppliesFileThenEnvironmentOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("KINETIC_API_PORT", "9090")
	t.Setenv("KINETIC_MODE", "controller")
	t.Setenv("KINETIC_WORKER_CONTROLLER_URL", "http://env-controller:9898")

	configDir := filepath.Join(homeDir, ".kinetic")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	configBody := strings.TrimSpace(`
mode: worker
api:
  host: 127.0.0.1
  port: 8081
database:
  type: sqlite
  path: /tmp/kinetic.db
controller:
  embedded_worker_enabled: false
worker:
  id: file-worker
  name: File Worker
  controller_url: http://file-controller:9898
  heartbeat_interval: 11
  stream_reconnect_interval: 12
  max_concurrency: 7
log:
  level: warn
  format: json
`) + "\n"

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configBody), 0o644))

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, ModeController, cfg.Mode)
	assert.Equal(t, "127.0.0.1", cfg.API.Host)
	assert.Equal(t, 9090, cfg.API.Port)
	assert.False(t, cfg.Controller.EmbeddedWorkerEnabled)
	assert.Equal(t, "file-worker", cfg.Worker.ID)
	assert.Equal(t, "File Worker", cfg.Worker.Name)
	assert.Equal(t, "http://env-controller:9898", cfg.Worker.ControllerURL)
	assert.Equal(t, 11, cfg.Worker.HeartbeatInterval)
	assert.Equal(t, 12, cfg.Worker.StreamReconnectSeconds)
	assert.Equal(t, 7, cfg.Worker.MaxConcurrency)
	assert.Equal(t, "warn", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "/tmp/kinetic.db", cfg.Database.Path)
}
