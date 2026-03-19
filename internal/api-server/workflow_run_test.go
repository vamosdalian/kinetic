package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

type stubRunStarter struct {
	runID      string
	workflowID string
	err        error
}

func (s *stubRunStarter) StartWorkflowRun(workflowID string) (string, error) {
	s.workflowID = workflowID
	return s.runID, s.err
}

func TestWorkflowHandler_Run(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	starter := &stubRunStarter{runID: "run-123"}
	handler.SetRunService(starter)

	router := gin.New()
	router.POST("/api/workflows/:id/run", handler.Run)

	req, _ := http.NewRequest("POST", "/api/workflows/workflow-123/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	dataMap := response.Data.(map[string]interface{})
	assert.Equal(t, "run-123", dataMap["run_id"])
	assert.Equal(t, "workflow-123", starter.workflowID)
}

func TestWorkflowHandler_ListRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflow_runs", handler.ListRuns)

	workflowID := uuid.New().String()
	runID := uuid.New().String()

	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "List Test Workflow",
		Description: "Desc",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	assert.NoError(t, err)

	err = db.CreateWorkflowRun(workflowID, runID)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workflow_runs?page=1&pageSize=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	rawItems, ok := response.Data.([]interface{})
	assert.True(t, ok)
	assert.Len(t, rawItems, 1)

	itemMap, ok := rawItems[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, runID, itemMap["run_id"])
	assert.NotEmpty(t, itemMap["create_at"])
}

func TestWorkflowHandler_GetRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflow_runs/:run_id", handler.GetRun)

	workflowID := uuid.New().String()
	runID := uuid.New().String()
	taskID := uuid.New().String()

	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "Get Test Workflow",
		Description: "Desc",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	assert.NoError(t, err)

	_, err = db.SaveTasks([]entity.TaskEntity{
		{
			ID:          taskID,
			WorkflowID:  workflowID,
			Name:        "Task 1",
			Description: "Task desc",
			Type:        "shell",
			Config:      `{"script":"echo hello"}`,
			Position:    `{"x":10,"y":20}`,
			NodeType:    "baseNodeFull",
		},
	})
	assert.NoError(t, err)

	err = db.CreateWorkflowRun(workflowID, runID)
	assert.NoError(t, err)
	err = db.FinishTaskRun(runID, taskID, "success", 0, "hello\n")
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workflow_runs/"+runID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	body, err := json.Marshal(response.Data)
	assert.NoError(t, err)

	var run dto.WorkflowRun
	err = json.Unmarshal(body, &run)
	assert.NoError(t, err)
	assert.Equal(t, runID, run.RunID)
	assert.Equal(t, "Get Test Workflow", run.Name)
	assert.Equal(t, 1, run.Version)
	assert.NotEmpty(t, run.CreatedAt)
	if assert.Len(t, run.TaskNodes, 1) {
		assert.Equal(t, "success", run.TaskNodes[0].Status)
		assert.Equal(t, "hello\n", run.TaskNodes[0].Output)
	}
}
