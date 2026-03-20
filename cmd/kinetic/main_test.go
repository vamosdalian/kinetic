package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/config"
)

func TestBindFlags_ModeDefaultsEmpty(t *testing.T) {
	var opts cliOptions
	fs := flag.NewFlagSet("kinetic", flag.ContinueOnError)
	bindFlags(fs, &opts)

	require.NoError(t, fs.Parse(nil))
	assert.Empty(t, opts.configPath)
	assert.Empty(t, opts.mode)
	assert.False(t, opts.withWorker)
	assert.False(t, opts.showVersion)
}

func TestBindFlags_ParsesExplicitModeOverride(t *testing.T) {
	var opts cliOptions
	fs := flag.NewFlagSet("kinetic", flag.ContinueOnError)
	bindFlags(fs, &opts)

	require.NoError(t, fs.Parse([]string{"-c", "/tmp/kinetic/config.yml", "--mode", "worker", "--with-worker"}))
	assert.Equal(t, "/tmp/kinetic/config.yml", opts.configPath)
	assert.Equal(t, "worker", opts.mode)
	assert.True(t, opts.withWorker)
}

func TestLoadRuntimeConfig_AppliesOverridesInOrder(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("KINETIC_MODE", "controller")

	cfg, configPath, shouldPersist, err := loadRuntimeConfig(cliOptions{
		mode:       "worker",
		withWorker: true,
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
		configPath: configPath,
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
		mode: "worker",
	})
	require.NoError(t, err)
	require.True(t, shouldPersist)

	require.NoError(t, persistMissingConfig(cfg, configPath, shouldPersist))

	content, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "mode: worker")
}
