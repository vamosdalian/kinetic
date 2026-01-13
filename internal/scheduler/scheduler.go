package scheduler

import (
	"context"
	"log"
	"sync"

	"github.com/sirupsen/logrus"
)

// Scheduler 任务调度器
type Scheduler struct {
	mu     sync.Mutex
	stopCh chan struct{}
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		stopCh: make(chan struct{}),
	}
}

// Run 启动调度器
func (s *Scheduler) Run() error {
	logrus.Info("Starting scheduler...")

	// TODO: 实现调度逻辑
	// 1. 定期检查需要执行的定时任务
	// 2. 将任务分发给 Worker

	<-s.stopCh
	return nil
}

// Shutdown 停止调度器
func (s *Scheduler) Shutdown(ctx context.Context) error {
	log.Println("Shutting down scheduler...")
	close(s.stopCh)
	return nil
}
