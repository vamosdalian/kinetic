package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

type TaskEntity struct {
	ID     string
	Type   string
	Config string
}

type shellConfig struct {
	Script string `json:"script"`
}

type httpConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type shellTask struct {
	id     string
	script string
}

type httpTask struct {
	id      string
	url     string
	method  string
	headers map[string]string
	body    string
	client  *http.Client
}

type unsupportedTask struct {
	id       string
	taskType string
}

func NewTask(task TaskEntity) (Task, error) {
	switch task.Type {
	case "shell":
		var cfg shellConfig
		if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
			return nil, fmt.Errorf("invalid shell config: %w", err)
		}
		if cfg.Script == "" {
			return nil, fmt.Errorf("shell task requires script")
		}
		return &shellTask{id: task.ID, script: cfg.Script}, nil
	case "http":
		var cfg httpConfig
		if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
			return nil, fmt.Errorf("invalid http config: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("http task requires url")
		}
		if cfg.Method == "" {
			cfg.Method = http.MethodGet
		}
		return &httpTask{
			id:      task.ID,
			url:     cfg.URL,
			method:  cfg.Method,
			headers: cfg.Headers,
			body:    cfg.Body,
			client:  http.DefaultClient,
		}, nil
	case "python", "condition":
		return &unsupportedTask{id: task.ID, taskType: task.Type}, nil
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

func (t *shellTask) ID() string {
	return t.id
}

func (t *shellTask) Type() string {
	return "shell"
}

func (t *shellTask) Execute(ctx context.Context) (TaskResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", t.script)

	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer

	err := cmd.Run()
	result := TaskResult{
		Output:   buffer.String(),
		ExitCode: 0,
	}

	if err == nil {
		return result, nil
	}

	result.ExitCode = -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}

	return result, err
}

func (t *httpTask) ID() string {
	return t.id
}

func (t *httpTask) Type() string {
	return "http"
}

func (t *httpTask) Execute(ctx context.Context) (TaskResult, error) {
	req, err := http.NewRequestWithContext(ctx, t.method, t.url, bytes.NewBufferString(t.body))
	if err != nil {
		return TaskResult{ExitCode: -1}, err
	}

	for key, value := range t.headers {
		req.Header.Set(key, value)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return TaskResult{ExitCode: -1}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaskResult{ExitCode: resp.StatusCode}, err
	}

	output := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))
	result := TaskResult{
		Output:   output,
		ExitCode: resp.StatusCode,
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("http task failed with status %d", resp.StatusCode)
	}

	return result, nil
}

func (t *unsupportedTask) ID() string {
	return t.id
}

func (t *unsupportedTask) Type() string {
	return t.taskType
}

func (t *unsupportedTask) Execute(ctx context.Context) (TaskResult, error) {
	_ = ctx
	return TaskResult{
		Output:   fmt.Sprintf("%s task is not implemented", t.taskType),
		ExitCode: -1,
	}, fmt.Errorf("%s task is not implemented", t.taskType)
}
