package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/config"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

func testWorkerConfig(controllerURL string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Mode = config.ModeWorker
	cfg.Worker.ID = "worker-test"
	cfg.Worker.Name = "Worker Test"
	cfg.Worker.ControllerURL = controllerURL
	cfg.Worker.HeartbeatInterval = 1
	cfg.Worker.StreamReconnectSeconds = 1
	cfg.Worker.MaxConcurrency = 2
	return cfg
}

func TestWorker_Register(t *testing.T) {
	var mu sync.Mutex
	var got dto.RegisterNodeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/nodes/register" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		require.Equal(t, http.MethodPost, r.Method)
		defer r.Body.Close()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	require.NoError(t, worker.register())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "worker-test", got.NodeID)
	assert.Equal(t, "Worker Test", got.Name)
	assert.Equal(t, "remote", got.Kind)
	assert.Equal(t, 2, got.MaxConcurrency)
}

func TestWorker_HeartbeatLoop(t *testing.T) {
	beats := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/heartbeat") {
			beats <- r.URL.Path
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	worker.cfg.Worker.HeartbeatInterval = 1

	go worker.heartbeatLoop()
	defer func() {
		close(worker.stopCh)
	}()

	select {
	case path := <-beats:
		assert.Equal(t, "/api/internal/nodes/worker-test/heartbeat", path)
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("expected heartbeat request")
	}
}

func TestWorker_RunStreamProcessesAssignCommand(t *testing.T) {
	events := make(chan dto.WorkerTaskEvent, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/internal/nodes/worker-test/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "event: assign\n")
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"assign\",\"task\":{\"run_id\":\"run-1\",\"task_id\":\"task-1\",\"name\":\"task-1\",\"type\":\"shell\",\"config\":{\"script\":\"printf 'ok'\"}}}\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case "/api/internal/nodes/worker-test/task-events":
			defer r.Body.Close()
			var event dto.WorkerTaskEvent
			require.NoError(t, json.NewDecoder(r.Body).Decode(&event))
			events <- event
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	require.NoError(t, worker.runStream())

	seenStarted := false
	seenFinished := false
	deadline := time.After(2 * time.Second)
	for !(seenStarted && seenFinished) {
		select {
		case event := <-events:
			if event.Type == "started" {
				seenStarted = true
			}
			if event.Type == "finished" {
				seenFinished = true
			}
		case <-deadline:
			t.Fatalf("expected started and finished events, got started=%t finished=%t", seenStarted, seenFinished)
		}
	}
}

func TestWorker_Run_ReconnectsAfterStreamDisconnect(t *testing.T) {
	var mu sync.Mutex
	registerCount := 0
	streamCount := 0
	ready := make(chan struct{}, 1)
	blockStream := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/nodes/register":
			mu.Lock()
			registerCount++
			current := registerCount
			mu.Unlock()
			if current >= 2 {
				select {
				case ready <- struct{}{}:
				default:
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.URL.Path == "/api/internal/nodes/worker-test/stream":
			mu.Lock()
			streamCount++
			current := streamCount
			mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			if current == 1 {
				return
			}
			<-blockStream
		case strings.Contains(r.URL.Path, "/heartbeat"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	worker.cfg.Worker.StreamReconnectSeconds = 1
	worker.cfg.Worker.HeartbeatInterval = 1

	done := make(chan error, 1)
	go func() {
		done <- worker.Run()
	}()

	select {
	case <-ready:
	case <-time.After(4 * time.Second):
		t.Fatal("worker did not reconnect after stream disconnect")
	}

	close(blockStream)
	require.NoError(t, worker.Shutdown(context.Background()))
	require.NoError(t, <-done)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, registerCount, 2)
	assert.GreaterOrEqual(t, streamCount, 2)
}

func TestWorker_RunStreamUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	err := worker.runStream()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected stream status")
}

func TestWorker_RunStreamKeepaliveIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writer := bufio.NewWriter(w)
		_, _ = writer.WriteString("event: keepalive\n")
		_, _ = writer.WriteString("data: {\"ts\":\"2026-03-19T00:00:00Z\"}\n\n")
		_ = writer.Flush()
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	defer server.Close()

	worker := NewWorker(testWorkerConfig(server.URL), "remote")
	require.NoError(t, worker.runStream())
	assert.Empty(t, worker.running)
}
