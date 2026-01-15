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

func TestWorkflowHandler_ListRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflow-runs", handler.ListRuns)

	workflowID := uuid.New().String()
	runID := uuid.New().String()

	// Create Workflow
	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "List Test Workflow",
		Description: "Desc",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	assert.NoError(t, err)

	// Create Run
	err = db.CreateWorkflowRun(workflowID, runID)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workflow-runs?page=1&pageSize=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Code int                       `json:"code"`
		Msg  string                    `json:"msg"`
		Data []dto.WorkflowRunListItem `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Len(t, response.Data, 1)
	assert.Equal(t, runID, response.Data[0].RunID)
	assert.NotEmpty(t, response.Data[0].CreatedAt)
}

func TestWorkflowHandler_GetRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflow-runs/:run_id", handler.GetRun)

	workflowID := uuid.New().String()
	runID := uuid.New().String()

	// Create Workflow
	err := db.SaveWorkflow(entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "Get Test Workflow",
		Description: "Desc",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	assert.NoError(t, err)

	// Create Run
	err = db.CreateWorkflowRun(workflowID, runID)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/workflow-runs/"+runID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data dto.WorkflowRun `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, runID, response.Data.RunID)
	assert.Equal(t, "Get Test Workflow", response.Data.Name)
	assert.Equal(t, "1", response.Data.Version)
	assert.NotEmpty(t, response.Data.CreatedAt)
}
