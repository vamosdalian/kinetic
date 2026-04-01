package sqlite

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func TestWorkflowRun(t *testing.T) {
	dbPath := "test_workflow_run.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create a workflow
	workflowID := uuid.New().String()
	workflow := entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "Test Workflow",
		Description: "For Run Test",
		Enable:      true,
		Version:     1,
	}
	if err := db.SaveWorkflow(workflow); err != nil {
		t.Fatalf("Failed to save workflow: %v", err)
	}

	// Create tasks for the workflow
	task1ID := uuid.New().String()
	task2ID := uuid.New().String()

	taskEntities := []entity.TaskEntity{
		{
			ID:          task1ID,
			WorkflowID:  workflowID,
			Name:        "Task 1",
			Description: "First Task",
			Type:        "script",
			Config:      "{}",
			Position:    "0,0",
			NodeType:    "start",
		},
		{
			ID:          task2ID,
			WorkflowID:  workflowID,
			Name:        "Task 2",
			Description: "Second Task",
			Type:        "script",
			Config:      "{}",
			Position:    "100,100",
			NodeType:    "end",
		},
	}
	if _, err := db.SaveTasks(taskEntities); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Create edges
	edgeID := uuid.New().String()
	edges := []entity.EdgeEntity{
		{
			ID:           edgeID,
			WorkflowID:   workflowID,
			Source:       task1ID,
			Target:       task2ID,
			SourceHandle: "a",
			TargetHandle: "b",
		},
	}
	if _, err := db.SaveEdges(edges); err != nil {
		t.Fatalf("Failed to save edges: %v", err)
	}

	// Test CreateWorkflowRun
	runID := uuid.New().String()
	err = db.CreateWorkflowRun(workflowID, runID)
	if err != nil {
		t.Fatalf("Failed to create workflow run: %v", err)
	}

	// Test GetWorkflowRun
	run, err := db.GetWorkflowRun(runID)
	if err != nil {
		t.Fatalf("Failed to get workflow run: %v", err)
	}
	if run.RunID != runID {
		t.Errorf("Expected RunID %s, got %s", runID, run.RunID)
	}
	if run.WorkflowID != workflowID {
		t.Errorf("Expected WorkflowID %s, got %s", workflowID, run.WorkflowID)
	}
	if run.Status != "created" {
		t.Errorf("Expected Status 'created', got %s", run.Status)
	}

	// Test GetTaskRuns
	taskRuns, err := db.GetTaskRuns(runID)
	if err != nil {
		t.Fatalf("Failed to get task runs: %v", err)
	}
	if len(taskRuns) != 2 {
		t.Errorf("Expected 2 task runs, got %d", len(taskRuns))
	}

	// Verify one of the tasks
	foundTask := false
	for _, tr := range taskRuns {
		if tr.TaskID == task1ID {
			foundTask = true
			if tr.RunID != runID {
				t.Errorf("Expected RunID %s in task run, got %s", runID, tr.RunID)
			}
			if tr.Status != "pending" {
				t.Errorf("Expected Status 'pending', got %s", tr.Status)
			}
		}
	}
	if !foundTask {
		t.Error("Task 1 run not found")
	}

	err = db.FinishTaskRun(runID, task1ID, "success", 0, "hello\n", `{"message":"ok"}`)
	if err != nil {
		t.Fatalf("Failed to finish task run: %v", err)
	}

	finishedTaskRun, err := db.GetTaskRun(runID, task1ID)
	if err != nil {
		t.Fatalf("Failed to get finished task run: %v", err)
	}
	if finishedTaskRun.Result != `{"message":"ok"}` {
		t.Fatalf("Expected task result %s, got %s", `{"message":"ok"}`, finishedTaskRun.Result)
	}

	// Test GetEdgeRuns
	edgeRuns, err := db.GetEdgeRuns(runID)
	if err != nil {
		t.Fatalf("Failed to get edge runs: %v", err)
	}
	if len(edgeRuns) != 1 {
		t.Errorf("Expected 1 edge run, got %d", len(edgeRuns))
	}
	if edgeRuns[0].EdgeID != edgeID {
		t.Errorf("Expected EdgeID %s, got %s", edgeID, edgeRuns[0].EdgeID)
	}

	// Test ListWorkflowRuns
	runs, err := db.ListWorkflowRuns(0, 10)
	if err != nil {
		t.Fatalf("Failed to list workflow runs: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("Expected 1 workflow run in list, got %d", len(runs))
	}
	if runs[0].RunID != runID {
		t.Errorf("Expected RunID %s, got %s", runID, runs[0].RunID)
	}

	count, err := db.CountWorkflowRuns()
	if err != nil {
		t.Fatalf("Failed to count workflow runs: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 workflow run in count, got %d", count)
	}
}
