package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

type TaskEntity struct {
	ID     string
	Type   string
	Config string
	Env    map[string]string
}

type shellTask struct {
	id     string
	script string
	env    map[string]string
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

type outputCollector struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	onOutput OutputFunc
}

func (c *outputCollector) append(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	c.mu.Lock()
	c.buffer.Write(chunk)
	c.mu.Unlock()
	if c.onOutput != nil {
		c.onOutput(string(chunk))
	}
}

func (c *outputCollector) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.String()
}

type collectorWriter struct {
	collector *outputCollector
}

func (w collectorWriter) Write(p []byte) (int, error) {
	w.collector.append(p)
	return len(p), nil
}

func NewTask(task TaskEntity) (Task, error) {
	switch task.Type {
	case "shell":
		var cfg workflowcfg.ShellConfig
		if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
			return nil, fmt.Errorf("invalid shell config: %w", err)
		}
		if cfg.Script == "" {
			return nil, fmt.Errorf("shell task requires script")
		}
		return &shellTask{id: task.ID, script: cfg.Script, env: resolveTaskEnv(cfg.TaskPolicy.Env, task.Env)}, nil
	case "http":
		var cfg workflowcfg.HTTPConfig
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

func (t *shellTask) Execute(ctx context.Context, onOutput OutputFunc) (TaskResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", t.script)
	if len(t.env) > 0 {
		cmd.Env = append(os.Environ(), flattenEnvMap(t.env)...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return TaskResult{ExitCode: -1}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return TaskResult{ExitCode: -1}, err
	}

	collector := &outputCollector{onOutput: onOutput}
	writer := collectorWriter{collector: collector}

	if err := cmd.Start(); err != nil {
		return TaskResult{ExitCode: -1}, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, stderr)
	}()

	err = cmd.Wait()
	wg.Wait()

	result := TaskResult{
		Output:   collector.String(),
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

func (t *httpTask) Execute(ctx context.Context, onOutput OutputFunc) (TaskResult, error) {
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

	statusLine := fmt.Sprintf("HTTP %d\n", resp.StatusCode)
	if onOutput != nil {
		onOutput(statusLine)
	}
	bodyString := string(body)
	if onOutput != nil && bodyString != "" {
		onOutput(bodyString)
	}
	output := statusLine + bodyString
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

func (t *unsupportedTask) Execute(ctx context.Context, onOutput OutputFunc) (TaskResult, error) {
	_ = ctx
	message := fmt.Sprintf("%s task is not implemented", t.taskType)
	if onOutput != nil {
		onOutput(message)
	}
	return TaskResult{
		Output:   strings.TrimSpace(message),
		ExitCode: -1,
	}, fmt.Errorf("%s task is not implemented", t.taskType)
}

func resolveTaskEnv(configEnv map[string]string, explicitEnv map[string]string) map[string]string {
	if len(configEnv) == 0 && len(explicitEnv) == 0 {
		return nil
	}

	resolved := make(map[string]string, len(configEnv)+len(explicitEnv))
	for key, value := range configEnv {
		resolved[key] = value
	}
	for key, value := range explicitEnv {
		resolved[key] = value
	}
	return resolved
}

func flattenEnvMap(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	flattened := make([]string, 0, len(values))
	for key, value := range values {
		flattened = append(flattened, key+"="+value)
	}
	return flattened
}
