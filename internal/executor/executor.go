package executor

import (
	"context"
	"log"
	"sync"
)

// Executor 任务执行器
type Executor struct {
	maxConcurrency int
	semaphore      chan struct{}
	wg             sync.WaitGroup
}

type TaskResult struct {
	Output   string
	ExitCode int
}

type OutputFunc func(chunk string)

// NewExecutor 创建执行器
func NewExecutor(maxConcurrency int) *Executor {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &Executor{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// Task 任务接口
type Task interface {
	ID() string
	Type() string
	Execute(ctx context.Context, onOutput OutputFunc) (TaskResult, error)
}

// Execute 执行任务
func (e *Executor) Execute(ctx context.Context, task Task, onOutput OutputFunc) (TaskResult, error) {
	// 获取信号量
	select {
	case e.semaphore <- struct{}{}:
	case <-ctx.Done():
		return TaskResult{}, ctx.Err()
	}
	defer func() { <-e.semaphore }()

	log.Printf("Executing task %s (type: %s)", task.ID(), task.Type())

	result, err := task.Execute(ctx, onOutput)
	if err != nil {
		log.Printf("Task %s failed: %v", task.ID(), err)
		return result, err
	}

	log.Printf("Task %s completed", task.ID())
	return result, nil
}

// ExecuteAsync 异步执行任务
func (e *Executor) ExecuteAsync(ctx context.Context, task Task) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		_, _ = e.Execute(ctx, task, nil)
	}()
}

// Wait 等待所有任务完成
func (e *Executor) Wait() {
	e.wg.Wait()
}
