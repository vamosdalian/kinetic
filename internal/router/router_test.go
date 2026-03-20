package router

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRouter_TimeoutOptionsApply(t *testing.T) {
	r := New(
		WithReadTimeout(5*time.Second),
		WithWriteTimeout(10*time.Second),
	)

	assert.Equal(t, 5*time.Second, r.readTimeout)
	assert.Equal(t, 10*time.Second, r.writeTimeout)
	assert.Equal(t, 60*time.Second, r.idleTimeout)
}

func TestShouldLogRequestAtDebug(t *testing.T) {
	assert.True(t, shouldLogRequestAtDebug("/assets"))
	assert.True(t, shouldLogRequestAtDebug("/assets/app.js"))
	assert.True(t, shouldLogRequestAtDebug("/api/internal/nodes/node-1/heartbeat"))

	assert.False(t, shouldLogRequestAtDebug("/api/nodes"))
	assert.False(t, shouldLogRequestAtDebug("/api/internal/nodes/node-1/stream"))
	assert.False(t, shouldLogRequestAtDebug("/assetz/app.js"))
}
