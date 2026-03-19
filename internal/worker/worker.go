package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/executor"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

type Worker struct {
	cfg          *config.Config
	executor     *executor.Executor
	wg           sync.WaitGroup
	stopCh       chan struct{}
	client       *http.Client
	kind         string
	mu           sync.Mutex
	running      map[string]context.CancelFunc
	streamCancel context.CancelFunc
}

func NewWorker(cfg *config.Config, kind string) *Worker {
	if kind == "" {
		kind = "remote"
	}
	return &Worker{
		cfg:      cfg,
		executor: executor.NewExecutor(cfg.Worker.MaxConcurrency),
		stopCh:   make(chan struct{}),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		kind:    kind,
		running: make(map[string]context.CancelFunc),
	}
}

func (w *Worker) Run() error {
	log.Printf("Starting worker %s, connecting to controller: %s", w.cfg.Worker.ID, w.cfg.Worker.ControllerURL)
	log.Printf("Max concurrency: %d", w.cfg.Worker.MaxConcurrency)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.heartbeatLoop()
	}()

	reconnectDelay := time.Duration(w.cfg.Worker.StreamReconnectSeconds) * time.Second
	if reconnectDelay <= 0 {
		reconnectDelay = 5 * time.Second
	}

	for {
		select {
		case <-w.stopCh:
			return nil
		default:
		}

		if err := w.register(); err != nil {
			log.Printf("worker register failed: %v", err)
			if !w.waitReconnect(reconnectDelay) {
				return nil
			}
			continue
		}

		if err := w.runStream(); err != nil {
			select {
			case <-w.stopCh:
				return nil
			default:
				log.Printf("worker stream disconnected: %v", err)
			}
		}

		if !w.waitReconnect(reconnectDelay) {
			return nil
		}
	}
}

func (w *Worker) Shutdown(ctx context.Context) error {
	log.Println("Shutting down worker...")
	close(w.stopCh)

	w.mu.Lock()
	if w.streamCancel != nil {
		w.streamCancel()
	}
	for _, cancel := range w.running {
		cancel()
	}
	w.mu.Unlock()

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

func (w *Worker) register() error {
	req := dto.RegisterNodeRequest{
		NodeID:         w.cfg.Worker.ID,
		Name:           w.cfg.Worker.Name,
		IP:             w.cfg.Worker.AdvertiseIP,
		Kind:           w.kind,
		MaxConcurrency: w.cfg.Worker.MaxConcurrency,
	}
	return w.postJSON("/api/internal/nodes/register", req, nil)
}

func (w *Worker) heartbeatLoop() {
	interval := time.Duration(w.cfg.Worker.HeartbeatInterval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.postJSON(fmt.Sprintf("/api/internal/nodes/%s/heartbeat", w.cfg.Worker.ID), dto.NodeHeartbeatRequest{}, nil); err != nil {
				log.Printf("worker heartbeat failed: %v", err)
			}
		}
	}
}

func (w *Worker) runStream() error {
	ctx, cancel := context.WithCancel(context.Background())
	w.mu.Lock()
	w.streamCancel = cancel
	w.mu.Unlock()
	defer func() {
		cancel()
		w.mu.Lock()
		w.streamCancel = nil
		w.mu.Unlock()
	}()

	url := strings.TrimRight(w.cfg.Worker.ControllerURL, "/") + fmt.Sprintf("/api/internal/nodes/%s/stream", w.cfg.Worker.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected stream status: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var eventType string
	var dataLines []string

	flush := func() error {
		if eventType == "keepalive" {
			eventType = ""
			dataLines = nil
			return nil
		}
		if len(dataLines) == 0 {
			eventType = ""
			return nil
		}
		var command dto.NodeCommand
		if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &command); err != nil {
			eventType = ""
			dataLines = nil
			return err
		}
		if command.Type == "" {
			command.Type = eventType
		}
		switch command.Type {
		case "assign":
			if command.Task != nil {
				w.startTask(*command.Task)
			}
		case "cancel":
			if command.Task != nil {
				w.cancelTask(command.Task.RunID, command.Task.TaskID)
			}
		}
		eventType = ""
		dataLines = nil
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}

func (w *Worker) startTask(task dto.AssignedTask) {
	key := taskKey(task.RunID, task.TaskID)

	w.mu.Lock()
	if _, exists := w.running[key]; exists {
		w.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.running[key] = cancel
	w.mu.Unlock()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer func() {
			w.mu.Lock()
			delete(w.running, key)
			w.mu.Unlock()
		}()
		w.executeAssignedTask(ctx, task)
	}()
}

func (w *Worker) cancelTask(runID string, taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if cancel, ok := w.running[taskKey(runID, taskID)]; ok {
		cancel()
	}
}

func (w *Worker) executeAssignedTask(ctx context.Context, task dto.AssignedTask) {
	_ = w.postTaskEvent(dto.WorkerTaskEvent{
		Type:   "started",
		RunID:  task.RunID,
		TaskID: task.TaskID,
	})

	policy, err := workflowcfg.ParseTaskPolicy(string(task.Config))
	if err != nil {
		exitCode := -1
		_ = w.postTaskEvent(dto.WorkerTaskEvent{
			Type:     "failed",
			RunID:    task.RunID,
			TaskID:   task.TaskID,
			Output:   fmt.Sprintf("Invalid task policy: %v", err),
			ExitCode: &exitCode,
		})
		return
	}

	reportOutput := func(chunk string) {
		if chunk == "" {
			return
		}
		_ = w.postTaskEvent(dto.WorkerTaskEvent{
			Type:   "output",
			RunID:  task.RunID,
			TaskID: task.TaskID,
			Output: chunk,
		})
	}

	attempts := policy.RetryCount + 1
	var lastResult executor.TaskResult
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			reportOutput(fmt.Sprintf("\n[retry %d/%d]\n", attempt-1, policy.RetryCount))
			if policy.RetryBackoffSeconds > 0 {
				select {
				case <-time.After(time.Duration(policy.RetryBackoffSeconds) * time.Second):
				case <-ctx.Done():
					lastErr = ctx.Err()
				}
				if ctx.Err() != nil {
					break
				}
			}
		}

		attemptCtx := ctx
		cancel := func() {}
		if policy.TimeoutSeconds > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, time.Duration(policy.TimeoutSeconds)*time.Second)
		}
		lastResult, lastErr = w.runTaskAttempt(attemptCtx, task, reportOutput)
		cancel()

		if lastErr == nil {
			exitCode := lastResult.ExitCode
			_ = w.postTaskEvent(dto.WorkerTaskEvent{
				Type:     "finished",
				RunID:    task.RunID,
				TaskID:   task.TaskID,
				ExitCode: &exitCode,
			})
			return
		}

		if ctx.Err() != nil {
			break
		}

		if errorsIsDeadline(attemptCtx) {
			reportOutput(fmt.Sprintf("\nTask timed out after %d seconds.\n", policy.TimeoutSeconds))
		} else {
			reportOutput(fmt.Sprintf("\nAttempt %d failed: %v\n", attempt, lastErr))
		}

		if attempt == attempts {
			break
		}
	}

	if ctx.Err() != nil {
		exitCode := lastResult.ExitCode
		_ = w.postTaskEvent(dto.WorkerTaskEvent{
			Type:     "cancelled",
			RunID:    task.RunID,
			TaskID:   task.TaskID,
			ExitCode: &exitCode,
		})
		return
	}

	exitCode := lastResult.ExitCode
	if exitCode == 0 {
		exitCode = -1
	}
	_ = w.postTaskEvent(dto.WorkerTaskEvent{
		Type:     "failed",
		RunID:    task.RunID,
		TaskID:   task.TaskID,
		ExitCode: &exitCode,
	})
}

func (w *Worker) runTaskAttempt(ctx context.Context, task dto.AssignedTask, onOutput executor.OutputFunc) (executor.TaskResult, error) {
	if task.Type == dto.TaskTypeCondition {
		var cfg workflowcfg.ConditionConfig
		if err := json.Unmarshal(task.Config, &cfg); err != nil {
			return executor.TaskResult{ExitCode: -1}, fmt.Errorf("invalid condition config: %w", err)
		}
		if task.ConditionInput == nil {
			return executor.TaskResult{ExitCode: -1}, fmt.Errorf("condition task is missing input")
		}
		expr, err := workflowcfg.ParseConditionExpression(cfg.Expression)
		if err != nil {
			return executor.TaskResult{ExitCode: -1}, err
		}
		matched, err := expr.Evaluate(workflowcfg.ConditionInput{
			Status:   task.ConditionInput.Status,
			ExitCode: task.ConditionInput.ExitCode,
			Output:   task.ConditionInput.Output,
		})
		if err != nil {
			return executor.TaskResult{ExitCode: -1}, err
		}
		message := fmt.Sprintf("Condition %q evaluated to %t", cfg.Expression, matched)
		onOutput(message)
		return executor.TaskResult{Output: message, ExitCode: 0}, nil
	}

	execTask, err := executor.NewTask(executor.TaskEntity{
		ID:     task.TaskID,
		Type:   string(task.Type),
		Config: string(task.Config),
	})
	if err != nil {
		return executor.TaskResult{ExitCode: -1}, err
	}

	return w.executor.Execute(ctx, execTask, onOutput)
}

func (w *Worker) postTaskEvent(event dto.WorkerTaskEvent) error {
	return w.postJSON(fmt.Sprintf("/api/internal/nodes/%s/task-events", w.cfg.Worker.ID), event, nil)
}

func (w *Worker) postJSON(path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := strings.TrimRight(w.cfg.Worker.ControllerURL, "/") + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (w *Worker) waitReconnect(delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-w.stopCh:
		return false
	case <-timer.C:
		return true
	}
}

func taskKey(runID string, taskID string) string {
	return runID + ":" + taskID
}

func errorsIsDeadline(ctx context.Context) bool {
	return ctx.Err() == context.DeadlineExceeded
}
