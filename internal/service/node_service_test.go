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
	runService, nodeService := setupNodeService(t, time.Millisecond)

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

	time.Sleep(10 * time.Millisecond)
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
