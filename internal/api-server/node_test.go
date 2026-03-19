package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

type stubNodeManager struct {
	registerReq   dto.RegisterNodeRequest
	registerResp  dto.Node
	registerErr   error
	heartbeatID   string
	heartbeatErr  error
	streamCh      chan dto.NodeCommand
	streamErr     error
	streamClosed  bool
	taskEventNode string
	taskEvent     dto.WorkerTaskEvent
	taskEventErr  error
}

func (s *stubNodeManager) RegisterNode(req dto.RegisterNodeRequest) (dto.Node, error) {
	s.registerReq = req
	return s.registerResp, s.registerErr
}

func (s *stubNodeManager) Heartbeat(nodeID string) error {
	s.heartbeatID = nodeID
	return s.heartbeatErr
}

func (s *stubNodeManager) SubscribeStream(nodeID string) (<-chan dto.NodeCommand, func(), error) {
	cleanup := func() {
		s.streamClosed = true
	}
	return s.streamCh, cleanup, s.streamErr
}

func (s *stubNodeManager) HandleTaskEvent(nodeID string, event dto.WorkerTaskEvent) error {
	s.taskEventNode = nodeID
	s.taskEvent = event
	return s.taskEventErr
}

func (s *stubNodeManager) ListNodeDTOs() ([]dto.Node, error)             { return nil, nil }
func (s *stubNodeManager) GetNodeDTO(nodeID string) (dto.Node, error)    { return dto.Node{}, nil }
func (s *stubNodeManager) AddNodeTag(nodeID string, tag string) error    { return nil }
func (s *stubNodeManager) DeleteNodeTag(nodeID string, tag string) error { return nil }

func TestNodeHandler_RegisterUsesClientIPFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &stubNodeManager{
		registerResp: dto.Node{NodeID: "node-1", IP: "203.0.113.7"},
	}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.POST("/api/internal/nodes/register", handler.Register)

	body := []byte(`{"node_id":"node-1","name":"Node 1","max_concurrency":2}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.7:12345"
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "203.0.113.7", manager.registerReq.IP)
	assert.Equal(t, "node-1", manager.registerReq.NodeID)
}

func TestNodeHandler_Heartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &stubNodeManager{}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.POST("/api/internal/nodes/:id/heartbeat", handler.Heartbeat)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes/node-1/heartbeat", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "node-1", manager.heartbeatID)
}

func TestNodeHandler_StreamWritesSSEEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	streamCh := make(chan dto.NodeCommand, 1)
	streamCh <- dto.NodeCommand{
		Type: "assign",
		Task: &dto.AssignedTask{RunID: "run-1", TaskID: "task-1", Name: "Task 1", Type: dto.TaskTypeShell, Config: json.RawMessage(`{"script":"printf 'ok'"}`)},
	}
	close(streamCh)

	manager := &stubNodeManager{streamCh: streamCh}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.GET("/api/internal/nodes/:id/stream", handler.Stream)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/nodes/node-1/stream", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/event-stream", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Body.String(), "event: assign")
	assert.Contains(t, resp.Body.String(), `"run_id":"run-1"`)
	assert.True(t, manager.streamClosed)
}

func TestNodeHandler_TaskEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &stubNodeManager{}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.POST("/api/internal/nodes/:id/task-events", handler.TaskEvents)

	body := []byte(`{"type":"started","run_id":"run-1","task_id":"task-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes/node-1/task-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "node-1", manager.taskEventNode)
	assert.Equal(t, "started", manager.taskEvent.Type)
	assert.Equal(t, "run-1", manager.taskEvent.RunID)
	assert.Equal(t, "task-1", manager.taskEvent.TaskID)
}

func TestNodeHandler_StreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := &stubNodeManager{streamErr: errors.New("stream unavailable")}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.GET("/api/internal/nodes/:id/stream", handler.Stream)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/nodes/node-1/stream", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "stream unavailable")
}

func TestNodeHandler_StreamCancelsOnContextDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	streamCh := make(chan dto.NodeCommand)
	manager := &stubNodeManager{streamCh: streamCh}
	handler := NewNodeHandler(manager)
	router := gin.New()
	router.GET("/api/internal/nodes/:id/stream", handler.Stream)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/internal/nodes/node-1/stream", nil).WithContext(ctx)
	resp := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(resp, req)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream handler did not stop on context cancellation")
	}
	assert.True(t, manager.streamClosed)
	close(streamCh)
}
