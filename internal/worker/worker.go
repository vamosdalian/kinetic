package worker

import (
	"context"
	"log"
	"sync"

	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/executor"
)

// Worker 工作节点
type Worker struct {
	cfg      *config.Config
	executor *executor.Executor
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

// NewWorker 创建工作节点
func NewWorker(cfg *config.Config) *Worker {
	return &Worker{
		cfg:      cfg,
		executor: executor.NewExecutor(cfg.Worker.MaxConcurrency),
		stopCh:   make(chan struct{}),
	}
}

// Run 启动工作节点
func (w *Worker) Run() error {
	log.Printf("Starting worker, connecting to controller: %s", w.cfg.Worker.ControllerURL)
	log.Printf("Max concurrency: %d", w.cfg.Worker.MaxConcurrency)

	// TODO: 实现与 Controller 的通信
	// 1. 注册到 Controller
	// 2. 定期心跳
	// 3. 接收任务并执行

	// 暂时只是等待停止信号
	<-w.stopCh
	return nil
}

// Shutdown 优雅关闭工作节点
func (w *Worker) Shutdown(ctx context.Context) error {
	log.Println("Shutting down worker...")
	close(w.stopCh)

	// 等待所有任务完成
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Worker stopped")
		return nil
	case <-ctx.Done():
		log.Println("Worker shutdown timeout")
		return ctx.Err()
	}
}
