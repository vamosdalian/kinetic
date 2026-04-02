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

	result, err := Load("")
	require.NoError(t, err)

	cfg := result.Config
	assert.Equal(t, ModeController, cfg.Mode)
	assert.True(t, cfg.Controller.EmbeddedWorkerEnabled)
	assert.Equal(t, 5, cfg.Controller.SchedulerInterval)
	assert.Equal(t, "http://localhost:9898", cfg.Worker.ControllerURL)
	assert.Equal(t, 5, cfg.Worker.HeartbeatInterval)
	assert.Equal(t, 10, cfg.Worker.MaxConcurrency)
	assert.Equal(t, filepath.Join(homeDir, ".kinetic", "kinetic.db"), cfg.Database.Path)
	assert.Equal(t, "kinetic", cfg.Controller.AdminUsername)
	assert.Equal(t, "kinetic", cfg.Controller.AdminPassword)
	assert.Empty(t, cfg.Controller.AuthSecret)

	assert.False(t, result.FileExists)
	assert.Equal(t, filepath.Join(homeDir, ".kinetic", "config.yml"), result.Path)

	_, statErr := os.Stat(result.Path)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestLoad_AppliesFileThenEnvironmentOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("KINETIC_API_PORT", "9090")
	t.Setenv("KINETIC_MODE", "controller")
	t.Setenv("KINETIC_WORKER_CONTROLLER_URL", "http://env-controller:9898")
	t.Setenv("KINETIC_CONTROLLER_ADMIN_USERNAME", "env-admin")
	t.Setenv("KINETIC_CONTROLLER_AUTH_SECRET", "env-secret")

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
  scheduler_interval: 9
  admin_username: file-admin
  admin_password: file-password
  auth_secret: file-secret
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

	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(configBody), 0o644))

	result, err := Load("")
	require.NoError(t, err)

	cfg := result.Config
	assert.True(t, result.FileExists)
	assert.Equal(t, filepath.Join(configDir, "config.yml"), result.Path)
	assert.Equal(t, ModeController, cfg.Mode)
	assert.Equal(t, "127.0.0.1", cfg.API.Host)
	assert.Equal(t, 9090, cfg.API.Port)
	assert.False(t, cfg.Controller.EmbeddedWorkerEnabled)
	assert.Equal(t, 9, cfg.Controller.SchedulerInterval)
	assert.Equal(t, "env-admin", cfg.Controller.AdminUsername)
	assert.Equal(t, "file-password", cfg.Controller.AdminPassword)
	assert.Equal(t, "env-secret", cfg.Controller.AuthSecret)
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

func TestLoad_FallsBackToLegacyConfigYAML(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, ".kinetic")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("mode: worker\n"), 0o644))

	result, err := Load("")
	require.NoError(t, err)

	assert.True(t, result.FileExists)
	assert.Equal(t, filepath.Join(configDir, "config.yaml"), result.Path)
	assert.Equal(t, ModeWorker, result.Config.Mode)
}

func TestRenderConfigBody_WorkerModeCommentsUnusedSections(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeWorker

	body, err := renderConfigBody(cfg)
	require.NoError(t, err)

	assert.Contains(t, body, "mode: worker")
	assert.Contains(t, body, "# api:")
	assert.Contains(t, body, "# database:")
	assert.Contains(t, body, "# controller:")
	assert.Contains(t, body, "worker:")
	assert.Contains(t, body, "log:")
	assert.Contains(t, body, "#     scheduler_interval:")
	assert.Contains(t, body, "#     admin_username:")
	assert.Contains(t, body, "#     admin_password:")
	assert.Contains(t, body, "#     auth_secret:")
	assert.NotContains(t, body, "# worker:")
	assert.NotContains(t, body, "# log:")
}

func TestRenderConfigBody_ControllerModeCommentsRemoteWorkerFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeController

	body, err := renderConfigBody(cfg)
	require.NoError(t, err)

	assert.Contains(t, body, "api:")
	assert.Contains(t, body, "database:")
	assert.Contains(t, body, "controller:")
	assert.Contains(t, body, "  scheduler_interval:")
	assert.Contains(t, body, "  admin_username:")
	assert.Contains(t, body, "  admin_password:")
	assert.Contains(t, body, "  auth_secret:")
	assert.Contains(t, body, "worker:")
	assert.Contains(t, body, "  id:")
	assert.Contains(t, body, "  name:")
	assert.Contains(t, body, "  heartbeat_interval:")
	assert.Contains(t, body, "  max_concurrency:")
	assert.Contains(t, body, "#     controller_url:")
	assert.Contains(t, body, "#     advertise_ip:")
	assert.Contains(t, body, "#     stream_reconnect_interval:")
	assert.NotContains(t, body, "# worker:")
	assert.NotContains(t, body, "# api:")
}
