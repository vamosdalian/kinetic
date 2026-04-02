package service

import (
	"path"
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
	t.Setenv("HOME", t.TempDir())

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

func seedWorkflowWithConfig(t *testing.T, db database.Database, workflowConfig string, tasks []entity.TaskEntity, edges []entity.EdgeEntity) string {
	t.Helper()

	workflowID := uuid.New().String()
	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "test workflow",
		Description: "test workflow description",
		Config:      workflowConfig,
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
	t.Setenv("HOME", t.TempDir())

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

func TestRunService_StartWorkflowRunRejectsDisabledWorkflow(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 1)

	workflowID := uuid.New().String()
	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "disabled workflow",
		Description: "disabled",
		Enable:      false,
	})
	assert.NoError(t, err)

	_, err = service.StartWorkflowRun(workflowID)
	assert.ErrorIs(t, err, ErrWorkflowDisabled)
}

func TestRunService_PersistsTaskResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	db := setupRunServiceDB(t)
	service := NewRunService(db, 1)

	taskID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       taskID,
			Name:     "task-with-result",
			Type:     "shell",
			Config:   `{"script":"printf 'done'; printf '{\"count\":1}' > \"$KINETIC_RESULT_PATH\""}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "success")

	taskRun, err := db.GetTaskRun(runID, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "done", taskRun.Output)
	assert.JSONEq(t, `{"count":1}`, taskRun.Result)
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

func TestRunService_CancelWorkflowRun(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	taskID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       taskID,
			Name:     "long-task",
			Type:     "shell",
			Config:   `{"script":"sleep 5"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	time.Sleep(150 * time.Millisecond)
	err = service.CancelWorkflowRun(runID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "cancelled")

	taskRun, err := db.GetTaskRun(runID, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "cancelled", taskRun.Status)
}

func TestRunService_RetryEventuallySucceeds(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	marker := path.Join(t.TempDir(), "retry-marker")
	taskID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       taskID,
			Name:     "retry-task",
			Type:     "shell",
			Config:   `{"script":"if [ -f '` + marker + `' ]; then printf 'ok'; else touch '` + marker + `'; printf 'fail'; exit 1; fi","retry_count":1}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "success")

	taskRun, err := db.GetTaskRun(runID, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "success", taskRun.Status)
	assert.Contains(t, taskRun.Output, "[retry 1/1]")
	assert.Contains(t, taskRun.Output, "ok")
}

func TestRunService_TimeoutFailsTask(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	taskID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       taskID,
			Name:     "timeout-task",
			Type:     "shell",
			Config:   `{"script":"sleep 2","timeout_seconds":1}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, nil)

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "failed")

	taskRun, err := db.GetTaskRun(runID, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "failed", taskRun.Status)
	assert.Contains(t, taskRun.Output, "Task timed out")
}

func TestRunService_ConditionRoutesTrueBranch(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	rootID := uuid.New().String()
	conditionID := uuid.New().String()
	trueID := uuid.New().String()
	falseID := uuid.New().String()

	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       rootID,
			Name:     "root",
			Type:     "shell",
			Config:   `{"script":"printf '{\"ok\":true}'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       conditionID,
			Name:     "condition",
			Type:     "condition",
			Config:   `{"expression":"json.ok == true"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       trueID,
			Name:     "true-branch",
			Type:     "shell",
			Config:   `{"script":"printf 'true'"}`,
			Position: `{"x":2,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       falseID,
			Name:     "false-branch",
			Type:     "shell",
			Config:   `{"script":"printf 'false'"}`,
			Position: `{"x":3,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{ID: uuid.New().String(), Source: rootID, Target: conditionID},
		{ID: uuid.New().String(), Source: conditionID, Target: trueID, SourceHandle: "true"},
		{ID: uuid.New().String(), Source: conditionID, Target: falseID, SourceHandle: "false"},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "success")

	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 4) {
		runsByTaskID := make(map[string]entity.TaskRunEntity, len(taskRuns))
		for _, taskRun := range taskRuns {
			runsByTaskID[taskRun.TaskID] = taskRun
		}
		assert.Equal(t, "success", runsByTaskID[conditionID].Status)
		assert.Contains(t, runsByTaskID[conditionID].Output, "evaluated to true")
		assert.Equal(t, "success", runsByTaskID[trueID].Status)
		assert.Equal(t, "skipped", runsByTaskID[falseID].Status)
	}
}

func TestRunService_RendersTemplatesAcrossWorkflowAndUpstreamContext(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	producerID := uuid.New().String()
	consumerID := uuid.New().String()
	workflowID := seedWorkflowWithConfig(t, db, `{"env":{"GREETING":"hello-${{ .task.name }}"}}`, []entity.TaskEntity{
		{
			ID:       producerID,
			Name:     "producer",
			Type:     "shell",
			Config:   `{"script":"printf 'producer'; printf '{\"message\":\"ok\"}' > \"$KINETIC_RESULT_PATH\""}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       consumerID,
			Name:     "consumer",
			Type:     "shell",
			Config:   `{"script":"printf '%s|%s|%s' \"$GREETING\" \"${{ .upstream.resultJSON.message }}\" \"${{ .runtime.env.startTime }}\""}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{{
		ID:     uuid.New().String(),
		Source: producerID,
		Target: consumerID,
	}})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	run := waitForRunStatus(t, db, runID, "success")
	taskRun, err := db.GetTaskRun(runID, consumerID)
	assert.NoError(t, err)
	assert.Equal(t, "success", taskRun.Status)
	assert.Contains(t, taskRun.Output, "hello-consumer|ok|")
	if assert.NotNil(t, run.StartedAt) {
		assert.Contains(t, taskRun.Output, run.StartedAt.UTC().Format(time.RFC3339))
	}
}

func TestRunService_FailsOnMissingTemplateValue(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 1)

	taskID := uuid.New().String()
	workflowID := seedWorkflow(t, db, []entity.TaskEntity{{
		ID:       taskID,
		Name:     "missing-template",
		Type:     "shell",
		Config:   `{"script":"printf '${{ .workflow.env.MISSING }}'"}`,
		Position: `{"x":0,"y":0}`,
		NodeType: "baseNodeFull",
	}}, nil)

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "failed")
	taskRun, err := db.GetTaskRun(runID, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "failed", taskRun.Status)
	assert.Contains(t, taskRun.Output, "Invalid task execution context")
	assert.Contains(t, taskRun.Output, "MISSING")
}

func TestRunService_ConditionExpressionSupportsTemplates(t *testing.T) {
	db := setupRunServiceDB(t)
	service := NewRunService(db, 2)

	rootID := uuid.New().String()
	conditionID := uuid.New().String()
	trueID := uuid.New().String()
	falseID := uuid.New().String()

	workflowID := seedWorkflow(t, db, []entity.TaskEntity{
		{
			ID:       rootID,
			Name:     "root",
			Type:     "shell",
			Config:   `{"script":"printf '{\"expected\":true}'"}`,
			Position: `{"x":0,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       conditionID,
			Name:     "condition",
			Type:     "condition",
			Config:   `{"expression":"json.expected == ${{ .upstream.outputJSON.expected }}"}`,
			Position: `{"x":1,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       trueID,
			Name:     "true-branch",
			Type:     "shell",
			Config:   `{"script":"printf 'true'"}`,
			Position: `{"x":2,"y":0}`,
			NodeType: "baseNodeFull",
		},
		{
			ID:       falseID,
			Name:     "false-branch",
			Type:     "shell",
			Config:   `{"script":"printf 'false'"}`,
			Position: `{"x":3,"y":0}`,
			NodeType: "baseNodeFull",
		},
	}, []entity.EdgeEntity{
		{ID: uuid.New().String(), Source: rootID, Target: conditionID},
		{ID: uuid.New().String(), Source: conditionID, Target: trueID, SourceHandle: "true"},
		{ID: uuid.New().String(), Source: conditionID, Target: falseID, SourceHandle: "false"},
	})

	runID, err := service.StartWorkflowRun(workflowID)
	assert.NoError(t, err)

	_ = waitForRunStatus(t, db, runID, "success")
	taskRuns, err := db.GetTaskRuns(runID)
	assert.NoError(t, err)
	if assert.Len(t, taskRuns, 4) {
		runsByTaskID := make(map[string]entity.TaskRunEntity, len(taskRuns))
		for _, taskRun := range taskRuns {
			runsByTaskID[taskRun.TaskID] = taskRun
		}
		assert.Equal(t, "success", runsByTaskID[conditionID].Status)
		assert.Contains(t, runsByTaskID[conditionID].Output, `json.expected == true`)
		assert.Equal(t, "success", runsByTaskID[trueID].Status)
		assert.Equal(t, "skipped", runsByTaskID[falseID].Status)
	}
}
