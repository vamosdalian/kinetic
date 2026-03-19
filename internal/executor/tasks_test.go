package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShellTaskSuccess(t *testing.T) {
	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "shell",
		Config: `{"script":"printf 'hello'"}`,
	})
	assert.NoError(t, err)

	result, err := task.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello", result.Output)
}

func TestShellTaskFailure(t *testing.T) {
	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "shell",
		Config: `{"script":"exit 3"}`,
	})
	assert.NoError(t, err)

	result, err := task.Execute(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, 3, result.ExitCode)
}

func TestShellTaskTimeout(t *testing.T) {
	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "shell",
		Config: `{"script":"sleep 2"}`,
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = task.Execute(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}

func TestHTTPTaskSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "http",
		Config: `{"url":"` + server.URL + `","method":"POST","body":"payload"}`,
	})
	assert.NoError(t, err)

	result, err := task.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, result.ExitCode)
	assert.Contains(t, result.Output, "HTTP 200")
	assert.Contains(t, result.Output, "ok")
}

func TestHTTPTaskFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "http",
		Config: `{"url":"` + server.URL + `","method":"GET"}`,
	})
	assert.NoError(t, err)

	result, err := task.Execute(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, 502, result.ExitCode)
	assert.Contains(t, result.Output, "bad gateway")
}

func TestHTTPTaskConnectionFailure(t *testing.T) {
	task, err := NewTask(TaskEntity{
		ID:     "task-1",
		Type:   "http",
		Config: `{"url":"http://127.0.0.1:1","method":"GET"}`,
	})
	assert.NoError(t, err)

	result, err := task.Execute(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, -1, result.ExitCode)
}
