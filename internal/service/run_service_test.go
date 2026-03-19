package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/database/sqlite"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func setupRunServiceDB(t *testing.T) database.Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test_run_service_"+uuid.New().String()+".db")
	db, err := sqlite.NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func waitForRunStatus(t *testing.T, db database.Database, runID string, expected string) entity.WorkflowRunEntity {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		run, err := db.GetWorkflowRun(runID)
		if err == nil && run.Status == expected {
			return run
		}
		time.Sleep(50 * time.Millisecond)
	}

	run, err := db.GetWorkflowRun(runID)
	if err != nil {
		t.Fatalf("failed to get run %s: %v", runID, err)
	}
	t.Fatalf("run %s did not reach status %s, got %s", runID, expected, run.Status)
	return entity.WorkflowRunEntity{}
}

func seedWorkflow(t *testing.T, db database.Database, tasks []entity.TaskEntity, edges []entity.EdgeEntity) string {
	t.Helper()

	workflowID := uuid.New().String()
	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "test workflow",
		Description: "test workflow description",
		Enable:      true,
	})
	assert.NoError(t, err)

	for i := range tasks {
		tasks[i].WorkflowID = workflowID
	}
	for i := range edges {
		edges[i].WorkflowID = workflowID
	}

	_, err = db.SaveTasks(tasks)
	assert.NoError(t, err)
	_, err = db.SaveEdges(edges)
	assert.NoError(t, err)

	return workflowID
}

func TestRunService_LinearWorkflowSuccess(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	task1ID := uuid.New().String()
	task2ID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       task1ID,
			Name:     "task-1",
			Type:     "shell",
			Config:   `{"script":"printf 'first'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       task2ID,
			Name:     "task-2",
			Type:     "shell",
			Config:   `{"script":"printf 'second'"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{
			ID:     uuid.New().String(),
			Source: task1ID,
			Target: task2ID,
		},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	run := waitForRunStatus(t, db, runID, "success")
	assert.NotNil(t, run.StartedAt)
	assert.NotNil(t, run.FinishedAt)

	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 2) {
		runsByTaskID := make(map[string]entity.TaskRunEntity, len(taskRuns))
		for _, taskRun := range taskRuns {
			runsByTaskID[taskRun.TaskID] = taskRun
		}
		assert.Equal(t, "success", runsByTaskID[task1ID].Status)
		assert.Equal(t, "first", runsByTaskID[task1ID].Output)
		assert.Equal(t, "success", runsByTaskID[task2ID].Status)
		assert.Equal(t, "second", runsByTaskID[task2ID].Output)
	}
}

func TestRunService_BranchedWorkflowSuccess(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 4)

	rootAID := uuid.New().String()
	rootBID := uuid.New().String()
	finalID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       rootAID,
			Name:     "root-a",
			Type:     "shell",
			Config:   `{"script":"printf 'A'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       rootBID,
			Name:     "root-b",
			Type:     "shell",
			Config:   `{"script":"printf 'B'"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       finalID,
			Name:     "final",
			Type:     "shell",
			Config:   `{"script":"printf 'C'"}`,
			Position: `{"x":2,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{ID: uuid.New().String(), Source: rootAID, Target: finalID},
		{ID: uuid.New().String(), Source: rootBID, Target: finalID},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "success")

	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 3) {
		for _, taskRun := range taskRuns {
			assert.Equal(t, "success", taskRun.Status)
		}
	}
}

func TestRunService_FailureSkipsDownstream(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	failID := uuid.New().String()
	downstreamID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       failID,
			Name:     "fail",
			Type:     "shell",
			Config:   `{"script":"exit 2"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       downstreamID,
			Name:     "downstream",
			Type:     "shell",
			Config:   `{"script":"printf 'should-not-run'"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{ID: uuid.New().String(), Source: failID, Target: downstreamID},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "failed")

	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 2) {
		runsByTaskID := make(map[string]entity.TaskRunEntity, len(taskRuns))
		for _, taskRun := range taskRuns {
			runsByTaskID[taskRun.TaskID] = taskRun
		}
		assert.Equal(t, "failed", runsByTaskID[failID].Status)
		assert.Equal(t, 2, runsByTaskID[failID].ExitCode)
		assert.Equal(t, "skipped", runsByTaskID[downstreamID].Status)
		assert.Contains(t, runsByTaskID[downstreamID].Output, "Skipped because another task")
	}
}

func TestRunService_CycleFailsRun(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	task1ID := uuid.New().String()
	task2ID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       task1ID,
			Name:     "task-1",
			Type:     "shell",
			Config:   `{"script":"printf 'one'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       task2ID,
			Name:     "task-2",
			Type:     "shell",
			Config:   `{"script":"printf 'two'"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{ID: uuid.New().String(), Source: task1ID, Target: task2ID},
		{ID: uuid.New().String(), Source: task2ID, Target: task1ID},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "failed")

	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 2) {
		for _, taskRun := range taskRuns {
			assert.Equal(t, "skipped", taskRun.Status)
			assert.Contains(t, taskRun.Output, "workflow graph is invalid")
		}
	}
}
