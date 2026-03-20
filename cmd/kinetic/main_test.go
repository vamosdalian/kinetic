package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/config"
)

func TestParseCLIOptions_DefaultsEmpty(t *testing.T) {
	opts, err := parseCLIOptions(nil)
	require.NoError(t, err)

	assert.Empty(t, opts.ConfigPath)
	assert.Empty(t, opts.Mode)
	assert.False(t, opts.WithWorker)
	assert.False(t, opts.ShowVersion)
}

func TestParseCLIOptions_ParsesExplicitModeOverride(t *testing.T) {
	opts, err := parseCLIOptions([]string{"-c", "/tmp/kinetic/config.yml", "--mode", "worker", "--with-worker"})
	require.NoError(t, err)

	assert.Equal(t, "/tmp/kinetic/config.yml", opts.ConfigPath)
	assert.Equal(t, "worker", opts.Mode)
	assert.True(t, opts.WithWorker)
}

func TestLoadRuntimeConfig_AppliesOverridesInOrder(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("KINETIC_MODE", "controller")

	cfg, configPath, shouldPersist, err := loadRuntimeConfig(cliOptions{
		Mode:       "worker",
		WithWorker: true,
	})
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(homeDir, ".kinetic", "config.yml"), configPath)
	assert.True(t, shouldPersist)
	assert.Equal(t, config.ModeWorker, cfg.Mode)
	assert.True(t, cfg.Controller.EmbeddedWorkerEnabled)
}

func TestLoadRuntimeConfig_UsesExplicitConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: worker\n"), 0o644))

	cfg, resolvedPath, shouldPersist, err := loadRuntimeConfig(cliOptions{
		ConfigPath: configPath,
	})
	require.NoError(t, err)

	assert.Equal(t, configPath, resolvedPath)
	assert.False(t, shouldPersist)
	assert.Equal(t, config.ModeWorker, cfg.Mode)
}

func TestPersistMissingConfig_WritesEffectiveMode(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg, configPath, shouldPersist, err := loadRuntimeConfig(cliOptions{
		Mode: "worker",
	})
	require.NoError(t, err)
	require.True(t, shouldPersist)

	require.NoError(t, persistMissingConfig(cfg, configPath, shouldPersist))

	content, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "mode: worker")
	assert.Contains(t, string(content), "# api:")
	assert.Contains(t, string(content), "# database:")
	assert.Contains(t, string(content), "# controller:")
	assert.Contains(t, string(content), "worker:")
	assert.Contains(t, string(content), "log:")
}

func TestPersistMissingConfig_ControllerModeCommentsRemoteWorkerFields(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg, configPath, shouldPersist, err := loadRuntimeConfig(cliOptions{})
	require.NoError(t, err)
	require.True(t, shouldPersist)

	require.NoError(t, persistMissingConfig(cfg, configPath, shouldPersist))

	content, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "mode: controller")
	assert.Contains(t, string(content), "worker:")
	assert.Contains(t, string(content), "  id:")
	assert.Contains(t, string(content), "  heartbeat_interval:")
	assert.Contains(t, string(content), "#     controller_url:")
	assert.Contains(t, string(content), "#     advertise_ip:")
	assert.Contains(t, string(content), "#     stream_reconnect_interval:")
}
