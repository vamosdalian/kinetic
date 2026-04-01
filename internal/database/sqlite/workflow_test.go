package sqlite

import (
	"os"
	"testing"

	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func TestSaveWorkflow(t *testing.T) {
	// 创建临时测试数据库
	dbPath := "test_save_workflow.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 测试1: 创建新的 workflow
	t.Run("Create new workflow", func(t *testing.T) {
		workflow := entity.WorkflowEntity{
			ID:          "test-workflow-1",
			Name:        "Test Workflow",
			Description: "This is a test workflow",
			Config:      `{"env":{"API_TOKEN":"secret"}}`,
			Enable:      true,
		}

		err := db.SaveWorkflow(workflow)
		if err != nil {
			t.Fatalf("Failed to save workflow: %v", err)
		}

		// 验证数据是否正确保存
		var savedID, savedName, savedDescription, savedConfig string
		var savedVersion int
		var savedEnable int
		var savedCreatedAt, savedUpdatedAt string

		err = db.db.QueryRow(
			"SELECT id, name, description, config, version, enable, created_at, updated_at FROM workflows WHERE id = ?",
			workflow.ID,
		).Scan(&savedID, &savedName, &savedDescription, &savedConfig, &savedVersion, &savedEnable, &savedCreatedAt, &savedUpdatedAt)

		if err != nil {
			t.Fatalf("Failed to query saved workflow: %v", err)
		}

		if savedID != workflow.ID {
			t.Errorf("Expected ID %s, got %s", workflow.ID, savedID)
		}
		if savedName != workflow.Name {
			t.Errorf("Expected Name %s, got %s", workflow.Name, savedName)
		}
		if savedDescription != workflow.Description {
			t.Errorf("Expected Description %s, got %s", workflow.Description, savedDescription)
		}
		if savedConfig != workflow.Config {
			t.Errorf("Expected Config %s, got %s", workflow.Config, savedConfig)
		}
		if savedVersion != 1 {
			t.Errorf("Expected Version 1, got %d", savedVersion)
		}
		if savedEnable != 1 {
			t.Errorf("Expected Enable true (1), got %d", savedEnable)
		}
		if savedCreatedAt == "" {
			t.Error("CreatedAt should not be empty")
		}
		if savedUpdatedAt == "" {
			t.Error("UpdatedAt should not be empty")
		}
	})

	// 测试2: 更新已存在的 workflow
	t.Run("Update existing workflow", func(t *testing.T) {
		// 先获取当前版本
		var currentVersion int
		err := db.db.QueryRow("SELECT version FROM workflows WHERE id = ?", "test-workflow-1").Scan(&currentVersion)
		if err != nil {
			t.Fatalf("Failed to get current version: %v", err)
		}

		// 更新 workflow
		updatedWorkflow := entity.WorkflowEntity{
			ID:          "test-workflow-1",
			Name:        "Updated Workflow Name",
			Description: "Updated description",
			Config:      `{"env":{"BASE_URL":"https://example.com"}}`,
			Enable:      false,
		}

		err = db.SaveWorkflow(updatedWorkflow)
		if err != nil {
			t.Fatalf("Failed to update workflow: %v", err)
		}

		// 验证数据是否正确更新
		var savedID, savedName, savedDescription, savedConfig string
		var savedVersion int
		var savedEnable int
		var savedUpdatedAt string

		err = db.db.QueryRow(
			"SELECT id, name, description, config, version, enable, updated_at FROM workflows WHERE id = ?",
			updatedWorkflow.ID,
		).Scan(&savedID, &savedName, &savedDescription, &savedConfig, &savedVersion, &savedEnable, &savedUpdatedAt)

		if err != nil {
			t.Fatalf("Failed to query updated workflow: %v", err)
		}

		if savedName != updatedWorkflow.Name {
			t.Errorf("Expected Name %s, got %s", updatedWorkflow.Name, savedName)
		}
		if savedDescription != updatedWorkflow.Description {
			t.Errorf("Expected Description %s, got %s", updatedWorkflow.Description, savedDescription)
		}
		if savedConfig != updatedWorkflow.Config {
			t.Errorf("Expected Config %s, got %s", updatedWorkflow.Config, savedConfig)
		}
		if savedVersion != currentVersion+1 {
			t.Errorf("Expected Version %d, got %d", currentVersion+1, savedVersion)
		}
		if savedEnable != 0 {
			t.Errorf("Expected Enable false (0), got %d", savedEnable)
		}
		if savedUpdatedAt == "" {
			t.Error("UpdatedAt should not be empty")
		}
	})

	// 测试3: 创建另一个新的 workflow
	t.Run("Create another new workflow", func(t *testing.T) {
		workflow := entity.WorkflowEntity{
			ID:          "test-workflow-2",
			Name:        "Second Workflow",
			Description: "Another test workflow",
			Enable:      true,
		}

		err := db.SaveWorkflow(workflow)
		if err != nil {
			t.Fatalf("Failed to save second workflow: %v", err)
		}

		// 验证版本号从1开始
		var savedVersion int
		err = db.db.QueryRow("SELECT version FROM workflows WHERE id = ?", workflow.ID).Scan(&savedVersion)
		if err != nil {
			t.Fatalf("Failed to query version: %v", err)
		}

		if savedVersion != 1 {
			t.Errorf("Expected Version 1 for new workflow, got %d", savedVersion)
		}
	})
}

func TestGetWorkflowByID(t *testing.T) {
	dbPath := "test_get_workflow_by_id.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Run("Get existing workflow", func(t *testing.T) {
		workflow := entity.WorkflowEntity{
			ID:          "test-workflow-get-1",
			Name:        "Test Workflow for Get",
			Description: "Test description",
			Config:      `{"env":{"API_TOKEN":"secret"}}`,
			Enable:      true,
		}

		err := db.SaveWorkflow(workflow)
		if err != nil {
			t.Fatalf("Failed to save workflow: %v", err)
		}

		result, err := db.GetWorkflowByID(workflow.ID)
		if err != nil {
			t.Fatalf("Failed to get workflow: %v", err)
		}

		if result.ID != workflow.ID {
			t.Errorf("Expected ID %s, got %s", workflow.ID, result.ID)
		}
		if result.Name != workflow.Name {
			t.Errorf("Expected Name %s, got %s", workflow.Name, result.Name)
		}
		if result.Description != workflow.Description {
			t.Errorf("Expected Description %s, got %s", workflow.Description, result.Description)
		}
		if result.Config != workflow.Config {
			t.Errorf("Expected Config %s, got %s", workflow.Config, result.Config)
		}
		if result.Enable != workflow.Enable {
			t.Errorf("Expected Enable %v, got %v", workflow.Enable, result.Enable)
		}
		if result.Version != 1 {
			t.Errorf("Expected Version 1, got %d", result.Version)
		}
		if result.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if result.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	})

	t.Run("Get non-existent workflow", func(t *testing.T) {
		_, err := db.GetWorkflowByID("non-existent-id")
		if err == nil {
			t.Error("Expected error when getting non-existent workflow, got nil")
		}
	})

	t.Run("Get workflow with updated data", func(t *testing.T) {
		workflow := entity.WorkflowEntity{
			ID:          "test-workflow-get-2",
			Name:        "Original Name",
			Description: "Original Description",
			Config:      `{"env":{"API_TOKEN":"secret"}}`,
			Enable:      true,
		}

		err := db.SaveWorkflow(workflow)
		if err != nil {
			t.Fatalf("Failed to save workflow: %v", err)
		}

		updatedWorkflow := entity.WorkflowEntity{
			ID:          "test-workflow-get-2",
			Name:        "Updated Name",
			Description: "Updated Description",
			Config:      `{"env":{"BASE_URL":"https://example.com"}}`,
			Enable:      false,
		}

		err = db.SaveWorkflow(updatedWorkflow)
		if err != nil {
			t.Fatalf("Failed to update workflow: %v", err)
		}

		result, err := db.GetWorkflowByID(workflow.ID)
		if err != nil {
			t.Fatalf("Failed to get workflow: %v", err)
		}

		if result.Name != updatedWorkflow.Name {
			t.Errorf("Expected Name %s, got %s", updatedWorkflow.Name, result.Name)
		}
		if result.Description != updatedWorkflow.Description {
			t.Errorf("Expected Description %s, got %s", updatedWorkflow.Description, result.Description)
		}
		if result.Config != updatedWorkflow.Config {
			t.Errorf("Expected Config %s, got %s", updatedWorkflow.Config, result.Config)
		}
		if result.Enable != updatedWorkflow.Enable {
			t.Errorf("Expected Enable %v, got %v", updatedWorkflow.Enable, result.Enable)
		}
		if result.Version != 2 {
			t.Errorf("Expected Version 2, got %d", result.Version)
		}
	})
}

func TestListWorkflows(t *testing.T) {
	dbPath := "test_list_workflows.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Prepare data
	workflows := []entity.WorkflowEntity{
		{ID: "wf-1", Name: "Workflow 1", Description: "Desc 1", Enable: true},
		{ID: "wf-2", Name: "Workflow 2", Description: "Desc 2", Enable: false},
		{ID: "wf-3", Name: "Workflow 3", Description: "Desc 3", Enable: true},
	}

	for _, wf := range workflows {
		if err := db.SaveWorkflow(wf); err != nil {
			t.Fatalf("Failed to save workflow %s: %v", wf.ID, err)
		}
		// Sleep briefly to ensure CreatedAt might differ if we were sorting by it,
		// but here we rely on insertion order (rowid) which is default for SQLite without ORDER BY
	}

	t.Run("List all workflows", func(t *testing.T) {
		list, err := db.ListWorkflows(0, 10)
		if err != nil {
			t.Fatalf("Failed to list workflows: %v", err)
		}
		if len(list) != 3 {
			t.Errorf("Expected 3 workflows, got %d", len(list))
		}

		// Verify content of one item
		found := false
		for _, item := range list {
			if item.ID == "wf-1" {
				found = true
				if item.Name != "Workflow 1" {
					t.Errorf("Expected name 'Workflow 1', got '%s'", item.Name)
				}
				if !item.Enable {
					t.Error("Expected Enable true")
				}
				if item.Version != 1 {
					t.Errorf("Expected Version 1, got %d", item.Version)
				}
				if item.CreatedAt.IsZero() {
					t.Error("CreatedAt should not be zero")
				}
				if item.UpdatedAt.IsZero() {
					t.Error("UpdatedAt should not be zero")
				}
			}
		}
		if !found {
			t.Error("Workflow wf-1 not found in list")
		}
	})

	t.Run("List with limit", func(t *testing.T) {
		list, err := db.ListWorkflows(0, 2)
		if err != nil {
			t.Fatalf("Failed to list workflows: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("Expected 2 workflows, got %d", len(list))
		}
	})

	t.Run("List with offset", func(t *testing.T) {
		list, err := db.ListWorkflows(2, 10)
		if err != nil {
			t.Fatalf("Failed to list workflows: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("Expected 1 workflow, got %d", len(list))
		}
		// Assuming insertion order
		if list[0].ID != "wf-3" {
			t.Errorf("Expected wf-3, got %s", list[0].ID)
		}
	})
}

func TestTasks(t *testing.T) {
	dbPath := "test_tasks.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	workflowID := "wf-tasks-1"
	tasks := []entity.TaskEntity{
		{ID: "task-1", WorkflowID: workflowID, Name: "Task 1", Type: "start", Config: "{}", Position: `{"x":0,"y":0}`, NodeType: "custom"},
		{ID: "task-2", WorkflowID: workflowID, Name: "Task 2", Type: "process", Config: "{}", Position: `{"x":100,"y":0}`, NodeType: "custom"},
	}

	t.Run("SaveTasks", func(t *testing.T) {
		_, err := db.SaveTasks(tasks)
		if err != nil {
			t.Fatalf("Failed to save tasks: %v", err)
		}
	})

	t.Run("ListTasks", func(t *testing.T) {
		list, err := db.ListTasks(workflowID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(list))
		}
		// Verify content
		for _, task := range list {
			if task.ID == "task-1" {
				if task.Name != "Task 1" {
					t.Errorf("Expected Task 1, got %s", task.Name)
				}
			}
		}
	})

	t.Run("UpdateTasks", func(t *testing.T) {
		updatedTasks := []entity.TaskEntity{
			{ID: "task-1", WorkflowID: workflowID, Name: "Task 1 Updated", Type: "start", Config: "{}", Position: `{"x":10,"y":10}`, NodeType: "custom"},
		}
		_, err := db.SaveTasks(updatedTasks)
		if err != nil {
			t.Fatalf("Failed to update tasks: %v", err)
		}

		list, err := db.ListTasks(workflowID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		found := false
		for _, task := range list {
			if task.ID == "task-1" {
				found = true
				if task.Name != "Task 1 Updated" {
					t.Errorf("Expected Task 1 Updated, got %s", task.Name)
				}
			}
		}
		if !found {
			t.Error("Task 1 not found after update")
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		err := db.DeleteTask("task-1")
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		list, err := db.ListTasks(workflowID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("Expected 1 task, got %d", len(list))
		}
		if list[0].ID == "task-1" {
			t.Error("Task 1 should have been deleted")
		}
	})

	t.Run("DeleteTasks", func(t *testing.T) {
		err := db.DeleteTasks(workflowID)
		if err != nil {
			t.Fatalf("Failed to delete tasks: %v", err)
		}

		list, err := db.ListTasks(workflowID)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("Expected 0 tasks, got %d", len(list))
		}
	})
}

func TestEdges(t *testing.T) {
	dbPath := "test_edges.db"
	defer os.Remove(dbPath)

	db, err := NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	workflowID := "wf-edges-1"
	edges := []entity.EdgeEntity{
		{ID: "edge-1", WorkflowID: workflowID, Source: "task-1", Target: "task-2", SourceHandle: "s1", TargetHandle: "t1"},
		{ID: "edge-2", WorkflowID: workflowID, Source: "task-2", Target: "task-3", SourceHandle: "s2", TargetHandle: "t2"},
	}

	t.Run("SaveEdges", func(t *testing.T) {
		_, err := db.SaveEdges(edges)
		if err != nil {
			t.Fatalf("Failed to save edges: %v", err)
		}
	})

	t.Run("ListEdges", func(t *testing.T) {
		list, err := db.ListEdges(workflowID)
		if err != nil {
			t.Fatalf("Failed to list edges: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("Expected 2 edges, got %d", len(list))
		}
		// Verify content
		for _, edge := range list {
			if edge.ID == "edge-1" {
				if edge.Source != "task-1" {
					t.Errorf("Expected source task-1, got %s", edge.Source)
				}
			}
		}
	})

	t.Run("UpdateEdges", func(t *testing.T) {
		updatedEdges := []entity.EdgeEntity{
			{ID: "edge-1", WorkflowID: workflowID, Source: "task-1-updated", Target: "task-2", SourceHandle: "s1", TargetHandle: "t1"},
		}
		_, err := db.SaveEdges(updatedEdges)
		if err != nil {
			t.Fatalf("Failed to update edges: %v", err)
		}

		list, err := db.ListEdges(workflowID)
		if err != nil {
			t.Fatalf("Failed to list edges: %v", err)
		}

		found := false
		for _, edge := range list {
			if edge.ID == "edge-1" {
				found = true
				if edge.Source != "task-1-updated" {
					t.Errorf("Expected source task-1-updated, got %s", edge.Source)
				}
			}
		}
		if !found {
			t.Error("Edge 1 not found after update")
		}
	})

	t.Run("DeleteEdge", func(t *testing.T) {
		err := db.DeleteEdge("edge-1")
		if err != nil {
			t.Fatalf("Failed to delete edge: %v", err)
		}

		list, err := db.ListEdges(workflowID)
		if err != nil {
			t.Fatalf("Failed to list edges: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("Expected 1 edge, got %d", len(list))
		}
		if list[0].ID == "edge-1" {
			t.Error("Edge 1 should have been deleted")
		}
	})

	t.Run("DeleteEdges", func(t *testing.T) {
		err := db.DeleteEdges(workflowID)
		if err != nil {
			t.Fatalf("Failed to delete edges: %v", err)
		}

		list, err := db.ListEdges(workflowID)
		if err != nil {
			t.Fatalf("Failed to list edges: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("Expected 0 edges, got %d", len(list))
		}
	})
}
