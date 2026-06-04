package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/clusterauth"
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
	stopOnce     sync.Once
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
	logger := w.logger()
	logger.WithField("max_concurrency", w.cfg.Worker.MaxConcurrency).Info("Starting worker")

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
			logger.WithError(err).Warn("Worker register failed")
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
				if isExpectedStreamDisconnect(err) {
					logger.WithError(err).Warn("Worker stream closed, reconnecting")
				} else {
					logger.WithError(err).Warn("Worker stream disconnected")
				}
			}
		}

		if !w.waitReconnect(reconnectDelay) {
			return nil
		}
	}
}

func (w *Worker) Shutdown(ctx context.Context) error {
	logger := w.logger()
	logger.Info("Shutting down worker")
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})

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
		logger.Info("Worker stopped")
		return nil
	case <-ctx.Done():
		logger.WithError(ctx.Err()).Warn("Worker shutdown timeout")
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
				w.logger().WithError(err).Warn("Worker heartbeat failed")
			}
		}
	}
}

func (w *Worker) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"mode":           w.cfg.Mode,
		"worker_id":      w.cfg.Worker.ID,
		"worker_kind":    w.kind,
		"controller_url": w.cfg.Worker.ControllerURL,
	})
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

	path := fmt.Sprintf("/api/internal/nodes/%s/stream", w.cfg.Worker.ID)
	url := strings.TrimRight(w.cfg.Worker.ControllerURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	w.signRequest(req, http.MethodGet, path, nil)
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
	reportCtx, cancelReport := w.taskReportContext()
	defer cancelReport()

	var outputMu sync.Mutex
	var outputSequence int64
	reportEvent := func(event dto.WorkerTaskEvent) bool {
		if err := w.postTaskEventWithRetry(reportCtx, event); err != nil {
			w.logger().WithError(err).WithFields(logrus.Fields{
				"run_id":  event.RunID,
				"task_id": event.TaskID,
				"type":    event.Type,
			}).Warn("Worker task event delivery stopped")
			return false
		}
		return true
	}
	reportOutput := func(chunk string) {
		if chunk == "" {
			return
		}

		outputMu.Lock()
		defer outputMu.Unlock()

		outputSequence++
		reportEvent(dto.WorkerTaskEvent{
			Type:      "output",
			RunID:     task.RunID,
			TaskRunID: task.TaskRunID,
			TaskID:    task.TaskID,
			Sequence:  outputSequence,
			Output:    chunk,
		})
	}

	if !reportEvent(dto.WorkerTaskEvent{
		Type:      "started",
		RunID:     task.RunID,
		TaskRunID: task.TaskRunID,
		TaskID:    task.TaskID,
	}) {
		return
	}

	policy, err := workflowcfg.ParseTaskPolicy(string(task.Config))
	if err != nil {
		exitCode := -1
		reportOutput(fmt.Sprintf("Invalid task policy: %v", err))
		reportEvent(dto.WorkerTaskEvent{
			Type:      "failed",
			RunID:     task.RunID,
			TaskRunID: task.TaskRunID,
			TaskID:    task.TaskID,
			ExitCode:  &exitCode,
		})
		return
	}

	// Retries are driven by the controller (it requeues failed tasks). The worker
	// only ever runs a single attempt, bounded by the task timeout.
	attemptCtx := ctx
	cancel := func() {}
	if policy.TimeoutSeconds > 0 {
		attemptCtx, cancel = context.WithTimeout(ctx, time.Duration(policy.TimeoutSeconds)*time.Second)
	}
	result, selectedBranch, err := w.runTaskAttempt(attemptCtx, task, reportOutput)
	cancel()

	if err == nil {
		exitCode := result.ExitCode
		reportEvent(dto.WorkerTaskEvent{
			Type:           "finished",
			RunID:          task.RunID,
			TaskRunID:      task.TaskRunID,
			TaskID:         task.TaskID,
			SelectedBranch: selectedBranch,
			Result:         result.Result,
			ExitCode:       &exitCode,
		})
		return
	}

	if ctx.Err() != nil {
		exitCode := result.ExitCode
		reportEvent(dto.WorkerTaskEvent{
			Type:      "cancelled",
			RunID:     task.RunID,
			TaskRunID: task.TaskRunID,
			TaskID:    task.TaskID,
			Result:    result.Result,
			ExitCode:  &exitCode,
		})
		return
	}

	if errorsIsDeadline(attemptCtx) {
		reportOutput(fmt.Sprintf("\nTask timed out after %d seconds.\n", policy.TimeoutSeconds))
	} else {
		reportOutput(fmt.Sprintf("\nTask failed: %v\n", err))
	}

	exitCode := result.ExitCode
	if exitCode == 0 {
		exitCode = -1
	}
	reportEvent(dto.WorkerTaskEvent{
		Type:      "failed",
		RunID:     task.RunID,
		TaskRunID: task.TaskRunID,
		TaskID:    task.TaskID,
		Result:    result.Result,
		ExitCode:  &exitCode,
	})
}

func (w *Worker) runTaskAttempt(ctx context.Context, task dto.AssignedTask, onOutput executor.OutputFunc) (executor.TaskResult, string, error) {
	if task.Type == dto.TaskTypeCondition || task.Type == dto.TaskTypeFor {
		return executor.TaskResult{ExitCode: -1}, "", fmt.Errorf("task type %s is controller-only", task.Type)
	}

	execTask, err := executor.NewTask(executor.TaskEntity{
		RunID:  task.RunID,
		ID:     task.TaskID,
		Type:   string(task.Type),
		Config: string(task.Config),
		Env:    task.Env,
	})
	if err != nil {
		return executor.TaskResult{ExitCode: -1}, "", err
	}

	result, err := w.executor.Execute(ctx, execTask, onOutput)
	return result, "", err
}

func (w *Worker) postTaskEvent(event dto.WorkerTaskEvent) error {
	return w.postTaskEventWithContext(context.Background(), event)
}

func (w *Worker) postJSON(path string, payload any, out any) error {
	return w.postJSONWithContext(context.Background(), path, payload, out)
}

func (w *Worker) postTaskEventWithRetry(ctx context.Context, event dto.WorkerTaskEvent) error {
	for {
		if err := w.postTaskEventWithContext(ctx, event); err != nil {
			timer := time.NewTimer(workerTaskEventRetryDelay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}
}

func (w *Worker) postTaskEventWithContext(ctx context.Context, event dto.WorkerTaskEvent) error {
	return w.postJSONWithContext(ctx, fmt.Sprintf("/api/internal/nodes/%s/task-events", w.cfg.Worker.ID), event, nil)
}

func (w *Worker) postJSONWithContext(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := strings.TrimRight(w.cfg.Worker.ControllerURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	w.signRequest(req, http.MethodPost, path, body)

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

func (w *Worker) taskReportContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-w.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func (w *Worker) signRequest(req *http.Request, method string, path string, body []byte) {
	for key, value := range clusterauth.Headers(w.cfg.ClusterSecret, method, path, body, time.Now()) {
		req.Header.Set(key, value)
	}
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

const workerTaskEventRetryDelay = 200 * time.Millisecond

func isExpectedStreamDisconnect(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, net.ErrClosed)
}
