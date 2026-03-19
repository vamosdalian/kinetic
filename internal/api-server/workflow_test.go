package apiserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/database/sqlite"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func setupTestDB(t *testing.T) database.Database {
	dbPath := filepath.Join(t.TempDir(), "test_workflow_"+uuid.New().String()+".db")
	db, err := sqlite.NewSqliteDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestWorkflowHandler_Save(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		workflowID       string
		requestBody      dto.Workflow
		setupDB          func(db database.Database) error
		expectedStatus   int
		validateResponse func(t *testing.T, response *httptest.ResponseRecorder)
	}{
		{
			name:       "成功保存新工作流",
			workflowID: uuid.New().String(),
			requestBody: func() dto.Workflow {
				taskID := uuid.New().String()
				return dto.Workflow{
					Name:        "测试工作流",
					Description: "这是一个测试工作流",
					Version:     1,
					Enable:      true,
					TaskNodes: []dto.TaskNode{
						{
							ID:       taskID,
							Name:     "任务1",
							Type:     dto.TaskTypeShell,
							Config:   json.RawMessage(`{"script":"echo hello"}`),
							Position: dto.Position{X: 100, Y: 200},
							NodeType: "default",
						},
					},
					Edges: []dto.Edge{},
				}
			}(),
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var apiResponse APIResponse
				err := json.Unmarshal(response.Body.Bytes(), &apiResponse)
				assert.NoError(t, err)
				assert.True(t, apiResponse.Success)
			},
		},
		{
			name:       "成功更新现有工作流",
			workflowID: uuid.New().String(),
			requestBody: dto.Workflow{
				Name:        "更新的工作流",
				Description: "更新后的描述",
				Version:     2,
				Enable:      false,
				TaskNodes:   []dto.TaskNode{},
				Edges:       []dto.Edge{},
			},
			setupDB: func(db database.Database) error {
				// 先创建一个工作流
				workflow := entity.WorkflowEntity{
					ID:          uuid.New().String(),
					Name:        "原始工作流",
					Description: "原始描述",
					Version:     1,
					Enable:      true,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := db.SaveWorkflow(workflow)
				return err
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var apiResponse APIResponse
				err := json.Unmarshal(response.Body.Bytes(), &apiResponse)
				assert.NoError(t, err)
				assert.True(t, apiResponse.Success)
			},
		},
		{
			name:       "保存空工作流（无任务和边）",
			workflowID: uuid.New().String(),
			requestBody: dto.Workflow{
				Name:        "空工作流",
				Description: "没有任务和边的工作流",
				Version:     1,
				Enable:      true,
				TaskNodes:   []dto.TaskNode{},
				Edges:       []dto.Edge{},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				var apiResponse APIResponse
				err := json.Unmarshal(response.Body.Bytes(), &apiResponse)
				assert.NoError(t, err)
				assert.True(t, apiResponse.Success)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)

			if tt.setupDB != nil {
				err := tt.setupDB(db)
				if err != nil {
					t.Fatalf("Setup DB failed: %v", err)
				}
			}

			handler := NewWorkflowHandler(db)
			router := gin.New()
			router.PUT("/api/workflows/:id", handler.Save)

			reqBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/api/workflows/"+tt.workflowID, bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateResponse != nil {
				tt.validateResponse(t, w)
			}
		})
	}
}

func TestWorkflowHandler_Save_WithTasksAndEdges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.PUT("/api/workflows/:id", handler.Save)

	workflowID := uuid.New().String()
	task1ID := uuid.New().String()
	task2ID := uuid.New().String()
	requestBody := dto.Workflow{
		Name:        "完整工作流测试",
		Description: "包含任务和边的完整工作流",
		Version:     1,
		Enable:      true,
		TaskNodes: []dto.TaskNode{
			{
				ID:       task1ID,
				Name:     "Shell任务",
				Type:     dto.TaskTypeShell,
				Config:   json.RawMessage(`{"script":"echo 'Hello World'"}`),
				Position: dto.Position{X: 100, Y: 100},
				NodeType: "default",
			},
			{
				ID:       task2ID,
				Name:     "HTTP任务",
				Type:     dto.TaskTypeHTTP,
				Config:   json.RawMessage(`{"url": "https://api.example.com", "method": "GET"}`),
				Position: dto.Position{X: 300, Y: 100},
				NodeType: "default",
			},
		},
		Edges: []dto.Edge{
			{
				ID:     uuid.New().String(),
				Source: task1ID,
				Target: task2ID,
			},
		},
	}

	reqBody, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req, _ := http.NewRequest("PUT", "/api/workflows/"+workflowID, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var apiResponse APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &apiResponse)
	assert.NoError(t, err)
	assert.True(t, apiResponse.Success)
}

func TestWorkflowHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflows", handler.List)

	// 插入测试数据
	for i := 0; i < 15; i++ {
		workflow := entity.WorkflowEntity{
			ID:        uuid.New().String(),
			Name:      "Workflow " + uuid.New().String(),
			Version:   1,
			Enable:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = db.SaveWorkflow(workflow)
	}

	// 测试默认分页
	req, _ := http.NewRequest("GET", "/api/workflows", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var apiResponse APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &apiResponse)
	assert.NoError(t, err)
	assert.True(t, apiResponse.Success)
	assert.NotNil(t, apiResponse.Meta)
	assert.Equal(t, 15, apiResponse.Meta.Total)

	// 测试自定义分页
	req, _ = http.NewRequest("GET", "/api/workflows?page=2&pageSize=5", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &apiResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, apiResponse.Meta.Page)
	assert.Equal(t, 5, apiResponse.Meta.PageSize)
}

func TestWorkflowHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.GET("/api/workflows/:id", handler.Get)

	workflowID := uuid.New().String()
	workflow := entity.WorkflowEntity{
		ID:          workflowID,
		Name:        "Test Workflow",
		Description: "Description",
		Version:     1,
		Enable:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_ = db.SaveWorkflow(workflow)

	// 插入任务和边
	task := entity.TaskEntity{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Name:       "Task 1",
		Type:       "shell",
		Config:     "{}",
		Position:   `{"x":0,"y":0}`,
		NodeType:   "default",
	}
	_, _ = db.SaveTasks([]entity.TaskEntity{task})

	req, _ := http.NewRequest("GET", "/api/workflows/"+workflowID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var apiResponse APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &apiResponse)
	assert.NoError(t, err)
	assert.True(t, apiResponse.Success)

	dataMap := apiResponse.Data.(map[string]interface{})
	assert.Equal(t, workflowID, dataMap["id"])
	assert.Equal(t, "Test Workflow", dataMap["name"])
}

func TestWorkflowHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	handler := NewWorkflowHandler(db)
	router := gin.New()
	router.DELETE("/api/workflows/:id", handler.Delete)

	workflowID := uuid.New().String()
	workflow := entity.WorkflowEntity{
		ID:        workflowID,
		Name:      "To Delete",
		Version:   1,
		Enable:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = db.SaveWorkflow(workflow)

	// 插入任务和边
	task := entity.TaskEntity{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Name:       "Task To Delete",
		Type:       "shell",
		Config:     "{}",
		Position:   `{"x":0,"y":0}`,
		NodeType:   "default",
	}
	_, _ = db.SaveTasks([]entity.TaskEntity{task})

	edge := entity.EdgeEntity{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Source:     "source",
		Target:     "target",
	}
	_, _ = db.SaveEdges([]entity.EdgeEntity{edge})

	req, _ := http.NewRequest("DELETE", "/api/workflows/"+workflowID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证已删除
	_, err := db.GetWorkflowByID(workflowID)
	assert.Error(t, err)

	tasks, err := db.ListTasks(workflowID)
	assert.NoError(t, err)
	assert.Empty(t, tasks)

	edges, err := db.ListEdges(workflowID)
	assert.NoError(t, err)
	assert.Empty(t, edges)
}
