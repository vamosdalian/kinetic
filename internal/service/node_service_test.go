package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func setupNodeService(t *testing.T, heartbeatTimeout time.Duration) (*RunService, *NodeService) {
	t.Helper()

	db := setupRunServiceDB(t)
	runService := NewRunService(db, 2)
	hub := NewWorkerStreamHub()
	runService.EnableDistributed(hub)

	return runService, NewNodeService(db, runService, hub, heartbeatTimeout)
}

func TestNodeService_RegisterNodeDefaultsAndHeartbeat(t *testing.T) {
	_, nodeService := setupNodeService(t, time.Second)

	node, err := nodeService.RegisterNode(dto.RegisterNodeRequest{
		NodeID:         "node-1",
		IP:             "127.0.0.1",
		MaxConcurrency: 0,
	})
	require.NoError(t, err)

	assert.Equal(t, "node-1", node.NodeID)
	assert.Equal(t, "node-1", node.Name)
	assert.Equal(t, "remote", node.Kind)
	assert.Equal(t, "online", node.Status)
	assert.Equal(t, 1, node.MaxConcurrency)
	assert.NotEmpty(t, node.LastHeartbeatAt)
	assert.NotEmpty(t, node.LastStreamAt)
	assert.Contains(t, node.Tags, dto.NodeTag{Tag: "node-default", SystemManaged: true})
	assert.Contains(t, node.Tags, dto.NodeTag{Tag: "node-127.0.0.1", SystemManaged: true})

	require.NoError(t, nodeService.Heartbeat(node.NodeID))

	updated, err := nodeService.GetNodeDTO(node.NodeID)
	require.NoError(t, err)
	assert.Equal(t, "online", updated.Status)
	assert.NotEmpty(t, updated.LastHeartbeatAt)
}

func TestNodeService_DispatchQueuedTasksAssignsToSubscribedNode(t *testing.T) {
	runService, nodeService := setupNodeService(t, 5*time.Second)

	node, err := nodeService.RegisterNode(dto.RegisterNodeRequest{
		NodeID:         "node-a",
		Name:           "Node A",
		MaxConcurrency: 2,
	})
	require.NoError(t, err)

	stream, cleanup, err := nodeService.SubscribeStream(node.NodeID)
	require.NoError(t, err)
	defer cleanup()

	workflowID := seedWorkflow(t, runService.db, []entity.TaskEntity{
		{
			ID:       uuid.New().String(),
			Name:     "task-1",
			Type:     "shell",
			Config:   `{"script":"printf 'hello'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := runService.StartWorkflowRun(workflowID)
	require.NoError(t, err)

	require.NoError(t, nodeService.DispatchQueuedTasks(context.Background(), 64))

	select {
	case command := <-stream:
		require.Equal(t, "assign", command.Type)
		require.NotNil(t, command.Task)
		assert.Equal(t, runID, command.Task.RunID)
	case <-time.After(2 * time.Second):
		t.Fatal("expected assign command to be published")
	}

	taskRuns, err := runService.db.GetTaskRuns(runID)
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	assert.Equal(t, "assigned", taskRuns[0].Status)
	assert.Equal(t, node.NodeID, taskRuns[0].AssignedNodeID)

	updatedNode, err := nodeService.GetNodeDTO(node.NodeID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedNode.RunningCount)
}

func TestNodeService_SweepOfflineNodesResetsAssignedTasks(t *testing.T) {
	runService, nodeService := setupNodeService(t, time.Second)

	node, err := nodeService.RegisterNode(dto.RegisterNodeRequest{
		NodeID:         "node-offline",
		MaxConcurrency: 1,
	})
	require.NoError(t, err)

	_, cleanup, err := nodeService.SubscribeStream(node.NodeID)
	require.NoError(t, err)
	defer cleanup()

	workflowID := seedWorkflow(t, runService.db, []entity.TaskEntity{
		{
			ID:       uuid.New().String(),
			Name:     "task-offline",
			Type:     "shell",
			Config:   `{"script":"printf 'offline'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := runService.StartWorkflowRun(workflowID)
	require.NoError(t, err)
	require.NoError(t, nodeService.DispatchQueuedTasks(context.Background(), 64))

	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, nodeService.SweepOfflineNodes(context.Background()))

	updatedNode, err := nodeService.GetNodeDTO(node.NodeID)
	require.NoError(t, err)
	assert.Equal(t, "offline", updatedNode.Status)
	assert.Equal(t, 0, updatedNode.RunningCount)

	taskRuns, err := runService.db.GetTaskRuns(runID)
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	assert.Equal(t, "queued", taskRuns[0].Status)
	assert.Empty(t, taskRuns[0].AssignedNodeID)
}

func TestNodeService_SweepOfflineNodesKeepsNodeOnlineWithFreshHeartbeat(t *testing.T) {
	_, nodeService := setupNodeService(t, time.Second)

	oldStream := time.Now().UTC().Add(-2 * time.Second)
	freshHeartbeat := time.Now().UTC()
	require.NoError(t, nodeService.db.UpsertNode(entity.NodeEntity{
		NodeID:          "node-heartbeat-fresh",
		Name:            "node-heartbeat-fresh",
		Kind:            "remote",
		Status:          "online",
		MaxConcurrency:  1,
		LastHeartbeatAt: &freshHeartbeat,
		LastStreamAt:    &oldStream,
	}))

	require.NoError(t, nodeService.SweepOfflineNodes(context.Background()))

	updatedNode, err := nodeService.GetNodeDTO("node-heartbeat-fresh")
	require.NoError(t, err)
	assert.Equal(t, "online", updatedNode.Status)
}

func TestNodeService_SweepOfflineNodesMarksNodeOfflineWithStaleHeartbeat(t *testing.T) {
	_, nodeService := setupNodeService(t, time.Second)

	staleHeartbeat := time.Now().UTC().Add(-2 * time.Second)
	freshStream := time.Now().UTC()
	require.NoError(t, nodeService.db.UpsertNode(entity.NodeEntity{
		NodeID:          "node-heartbeat-stale",
		Name:            "node-heartbeat-stale",
		Kind:            "remote",
		Status:          "online",
		MaxConcurrency:  1,
		LastHeartbeatAt: &staleHeartbeat,
		LastStreamAt:    &freshStream,
	}))

	require.NoError(t, nodeService.SweepOfflineNodes(context.Background()))

	updatedNode, err := nodeService.GetNodeDTO("node-heartbeat-stale")
	require.NoError(t, err)
	assert.Equal(t, "offline", updatedNode.Status)
}

func TestNodeService_DispatchQueuedTasksRespectsCapacityWithinBatch(t *testing.T) {
	runService, nodeService := setupNodeService(t, 5*time.Second)

	firstNode, err := nodeService.RegisterNode(dto.RegisterNodeRequest{
		NodeID:         "node-1",
		Name:           "Node 1",
		MaxConcurrency: 1,
	})
	require.NoError(t, err)

	secondNode, err := nodeService.RegisterNode(dto.RegisterNodeRequest{
		NodeID:         "node-2",
		Name:           "Node 2",
		MaxConcurrency: 1,
	})
	require.NoError(t, err)

	firstStream, firstCleanup, err := nodeService.SubscribeStream(firstNode.NodeID)
	require.NoError(t, err)
	defer firstCleanup()

	secondStream, secondCleanup, err := nodeService.SubscribeStream(secondNode.NodeID)
	require.NoError(t, err)
	defer secondCleanup()

	workflowID := seedWorkflow(t, runService.db, []entity.TaskEntity{
		{
			ID:       uuid.New().String(),
			Name:     "task-1",
			Type:     "shell",
			Config:   `{"script":"printf 'one'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       uuid.New().String(),
			Name:     "task-2",
			Type:     "shell",
			Config:   `{"script":"printf 'two'"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := runService.StartWorkflowRun(workflowID)
	require.NoError(t, err)

	require.NoError(t, nodeService.DispatchQueuedTasks(context.Background(), 64))

	assignedNodes := make(map[string]int)
	for i := 0; i < 2; i++ {
		select {
		case command := <-firstStream:
			if command.Type == "assign" && command.Task != nil {
				assignedNodes[firstNode.NodeID]++
			}
		case command := <-secondStream:
			if command.Type == "assign" && command.Task != nil {
				assignedNodes[secondNode.NodeID]++
			}
		case <-time.After(2 * time.Second):
			t.Fatal("expected both tasks to be assigned within the dispatch batch")
		}
	}

	assert.Equal(t, 1, assignedNodes[firstNode.NodeID])
	assert.Equal(t, 1, assignedNodes[secondNode.NodeID])

	taskRuns, err := runService.db.GetTaskRuns(runID)
	require.NoError(t, err)
	require.Len(t, taskRuns, 2)
	assert.NotEqual(t, taskRuns[0].AssignedNodeID, taskRuns[1].AssignedNodeID)

	updatedFirstNode, err := nodeService.GetNodeDTO(firstNode.NodeID)
	require.NoError(t, err)
	updatedSecondNode, err := nodeService.GetNodeDTO(secondNode.NodeID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedFirstNode.RunningCount)
	assert.Equal(t, 1, updatedSecondNode.RunningCount)
}

func TestNodeService_FailedTaskRequeuesWithinBudgetThenFails(t *testing.T) {
	runService, nodeService := setupNodeService(t, 5*time.Second)

	node, err := nodeService.RegisterNode(dto.RegisterNodeRequest{NodeID: "node-retry", MaxConcurrency: 1})
	require.NoError(t, err)
	_, cleanup, err := nodeService.SubscribeStream(node.NodeID)
	require.NoError(t, err)
	defer cleanup()

	workflowID := seedWorkflow(t, runService.db, []entity.TaskEntity{
		{
			ID:       uuid.New().String(),
			Name:     "task-retry",
			Type:     "shell",
			Config:   `{"script":"exit 1","retry_count":1}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := runService.StartWorkflowRun(workflowID)
	require.NoError(t, err)

	taskRuns, err := runService.db.GetTaskRuns(runID)
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	taskID := taskRuns[0].TaskID

	failOnce := func() {
		require.NoError(t, nodeService.DispatchQueuedTasks(context.Background(), 64))
		require.NoError(t, runService.HandleWorkerTaskEvent(node.NodeID, dto.WorkerTaskEvent{Type: "started", RunID: runID, TaskID: taskID}))
		exit := 1
		require.NoError(t, runService.HandleWorkerTaskEvent(node.NodeID, dto.WorkerTaskEvent{Type: "failed", RunID: runID, TaskID: taskID, ExitCode: &exit}))
	}

	// First attempt fails but the retry budget (retry_count=1) remains, so the
	// task is requeued and the run keeps running.
	failOnce()
	assert.Equal(t, 1, runService.getAttemptCount(runID, taskID))
	got, err := runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	assert.Equal(t, "queued", got.Status)
	run, err := runService.db.GetWorkflowRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "running", run.Status)

	// Second attempt exhausts the shared budget, so the run fails. The in-memory
	// retry state is cleared once the task reaches a terminal failure.
	failOnce()
	assert.Equal(t, 0, runService.getAttemptCount(runID, taskID))
	got, err = runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	assert.Equal(t, "failed", got.Status)
	run, err = runService.db.GetWorkflowRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "failed", run.Status)
}

func TestNodeService_StaleUnknownTaskRequeuesWithinBudget(t *testing.T) {
	runService, nodeService := setupNodeService(t, 5*time.Second)

	clock := time.Now().UTC()
	runService.now = func() time.Time { return clock }

	runID, taskID, node := seedRunningTaskOnNode(t, runService, nodeService, `{"script":"sleep 1","retry_count":1}`)

	require.NoError(t, runService.HandleNodeOffline(node))
	got, err := runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	require.Equal(t, "unknown", got.Status)

	timeout := UnknownTaskTimeoutSeconds * time.Second

	// Not stale yet: nothing happens.
	require.NoError(t, runService.RequeueStaleUnknownTasks(timeout))
	got, err = runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	assert.Equal(t, "unknown", got.Status)

	// Advance past the timeout: budget remains so the task is requeued.
	clock = clock.Add(timeout + time.Second)
	require.NoError(t, runService.RequeueStaleUnknownTasks(timeout))
	got, err = runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	assert.Equal(t, "queued", got.Status)
	run, err := runService.db.GetWorkflowRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "running", run.Status)
}

func TestNodeService_StaleUnknownTaskFailsRunWhenBudgetExhausted(t *testing.T) {
	runService, nodeService := setupNodeService(t, 5*time.Second)

	clock := time.Now().UTC()
	runService.now = func() time.Time { return clock }

	runID, taskID, node := seedRunningTaskOnNode(t, runService, nodeService, `{"script":"sleep 1"}`)

	require.NoError(t, runService.HandleNodeOffline(node))

	timeout := UnknownTaskTimeoutSeconds * time.Second
	clock = clock.Add(timeout + time.Second)
	require.NoError(t, runService.RequeueStaleUnknownTasks(timeout))

	got, err := runService.db.GetTaskRun(runID, taskID)
	require.NoError(t, err)
	assert.Equal(t, "failed", got.Status)
	run, err := runService.db.GetWorkflowRun(runID)
	require.NoError(t, err)
	assert.Equal(t, "failed", run.Status)
}

// seedRunningTaskOnNode creates a single-shell-task workflow, dispatches it to a
// freshly registered node, and drives it into the running state. It returns the
// run id, task id, and node id.
func seedRunningTaskOnNode(t *testing.T, runService *RunService, nodeService *NodeService, config string) (string, string, string) {
	t.Helper()

	node, err := nodeService.RegisterNode(dto.RegisterNodeRequest{NodeID: "node-unknown", MaxConcurrency: 1})
	require.NoError(t, err)
	_, cleanup, err := nodeService.SubscribeStream(node.NodeID)
	require.NoError(t, err)
	t.Cleanup(cleanup)

	workflowID := seedWorkflow(t, runService.db, []entity.TaskEntity{
		{
			ID:       uuid.New().String(),
			Name:     "task-unknown",
			Type:     "shell",
			Config:   config,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := runService.StartWorkflowRun(workflowID)
	require.NoError(t, err)

	taskRuns, err := runService.db.GetTaskRuns(runID)
	require.NoError(t, err)
	require.Len(t, taskRuns, 1)
	taskID := taskRuns[0].TaskID

	require.NoError(t, nodeService.DispatchQueuedTasks(context.Background(), 64))
	require.NoError(t, runService.HandleWorkerTaskEvent(node.NodeID, dto.WorkerTaskEvent{Type: "started", RunID: runID, TaskID: taskID}))

	return runID, taskID, node.NodeID
}
