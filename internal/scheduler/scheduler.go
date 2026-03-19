package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type dispatcher interface {
	DispatchQueuedTasks(ctx context.Context, limit int) error
	SweepOfflineNodes(ctx context.Context) error
}

type Scheduler struct {
	mu         sync.Mutex
	stopCh     chan struct{}
	closeOnce  sync.Once
	dispatcher dispatcher
	interval   time.Duration
}

func NewScheduler(dispatcher dispatcher) *Scheduler {
	return &Scheduler{
		stopCh:     make(chan struct{}),
		dispatcher: dispatcher,
		interval:   2 * time.Second,
	}
}

func (s *Scheduler) Run() error {
	logrus.Info("Starting scheduler...")
	if s.dispatcher == nil {
		<-s.stopCh
		return nil
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), s.interval)
			if err := s.dispatcher.SweepOfflineNodes(ctx); err != nil {
				logrus.Warnf("scheduler sweep failed: %v", err)
			}
			if err := s.dispatcher.DispatchQueuedTasks(ctx, 64); err != nil {
				logrus.Warnf("scheduler dispatch failed: %v", err)
			}
			cancel()
		}
	}
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down scheduler...")
	s.closeOnce.Do(func() {
		close(s.stopCh)
	})
	return nil
}
