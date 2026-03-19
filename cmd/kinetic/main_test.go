package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindFlags_ModeDefaultsEmpty(t *testing.T) {
	var opts cliOptions
	fs := flag.NewFlagSet("kinetic", flag.ContinueOnError)
	bindFlags(fs, &opts)

	require.NoError(t, fs.Parse(nil))
	assert.Empty(t, opts.mode)
	assert.False(t, opts.withWorker)
	assert.False(t, opts.showVersion)
}

func TestBindFlags_ParsesExplicitModeOverride(t *testing.T) {
	var opts cliOptions
	fs := flag.NewFlagSet("kinetic", flag.ContinueOnError)
	bindFlags(fs, &opts)

	require.NoError(t, fs.Parse([]string{"--mode", "worker", "--with-worker"}))
	assert.Equal(t, "worker", opts.mode)
	assert.True(t, opts.withWorker)
}
