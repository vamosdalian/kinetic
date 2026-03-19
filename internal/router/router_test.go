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
