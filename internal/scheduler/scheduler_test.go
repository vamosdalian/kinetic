package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDispatcher struct {
	mu            sync.Mutex
	dispatchCalls int
	sweepCalls    int
}

func (s *stubDispatcher) DispatchQueuedTasks(ctx context.Context, limit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatchCalls++
	return nil
}

func (s *stubDispatcher) SweepOfflineNodes(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepCalls++
	return nil
}

func (s *stubDispatcher) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dispatchCalls, s.sweepCalls
}

func TestScheduler_RunInvokesDispatcher(t *testing.T) {
	dispatcher := &stubDispatcher{}
	scheduler := NewScheduler(dispatcher)
	scheduler.interval = 10 * time.Millisecond

	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run()
	}()

	require.Eventually(t, func() bool {
		dispatchCalls, sweepCalls := dispatcher.counts()
		return dispatchCalls > 0 && sweepCalls > 0
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, scheduler.Shutdown(context.Background()))
	require.NoError(t, <-done)

	dispatchCalls, sweepCalls := dispatcher.counts()
	assert.Greater(t, dispatchCalls, 0)
	assert.Greater(t, sweepCalls, 0)
}

func TestScheduler_RunWithoutDispatcherStopsCleanly(t *testing.T) {
	scheduler := NewScheduler(nil)

	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run()
	}()

	require.NoError(t, scheduler.Shutdown(context.Background()))
	require.NoError(t, <-done)
}

func TestScheduler_ShutdownIsIdempotent(t *testing.T) {
	scheduler := NewScheduler(nil)

	require.NoError(t, scheduler.Shutdown(context.Background()))
	require.NoError(t, scheduler.Shutdown(context.Background()))
}
