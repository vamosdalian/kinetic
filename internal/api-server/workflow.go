package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

type WorkflowHandler struct {
	db         database.Database
	runService RunManager
}

func NewWorkflowHandler(db database.Database) *WorkflowHandler {
	return &WorkflowHandler{db: db}
}

type RunManager interface {
	StartWorkflowRun(workflowID string) (string, error)
	RerunWorkflowRun(runID string) (string, error)
	CancelWorkflowRun(runID string) error
	SubscribeRunEvents(runID string) (<-chan dto.WorkflowRunEvent, func(), error)
}

func (h *WorkflowHandler) SetRunService(runService RunManager) {
	h.runService = runService
}

type PageQuerys struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize" binding:"max=100"`
}

type WorkflowListQuery struct {
	PageQuerys
	Query string `form:"query"`
}

func (h *WorkflowHandler) List(c *gin.Context) {
	var pageQuerys WorkflowListQuery
	if err := c.ShouldBindQuery(&pageQuerys); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	page := max(pageQuerys.Page, 1)
	pageSize := pageQuerys.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "pageSize must be <= 100")
		return
	}
	workflows, err := h.db.ListWorkflowsFiltered((page-1)*pageSize, pageSize, pageQuerys.Query)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	count, err := h.db.CountWorkflowsFiltered(pageQuerys.Query)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	response := make([]dto.WorkflowListItem, 0, len(workflows))
	for _, w := range workflows {
		response = append(response, dto.WorkflowListItem{
			ID:        w.ID,
			Name:      w.Name,
			Enable:    w.Enable,
			Version:   w.Version,
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
		})
	}

	ResponseSuccessWithPagination(c, response, page, pageSize, count)
}

func (h *WorkflowHandler) Get(c *gin.Context) {
	id := c.Param("id")

	workflow, err := h.db.GetWorkflowByID(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	tasks, err := h.db.ListTasks(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	edges, err := h.db.ListEdges(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	resp := dto.Workflow{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Config:      workflowcfg.WorkflowConfig{},
		Tag:         workflow.Tag,
		Version:     workflow.Version,
		Enable:      workflow.Enable,
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
	}
	if cfg, err := workflowcfg.ParseWorkflowConfig(workflow.Config); err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	} else {
		resp.Config = cfg
	}

	for _, task := range tasks {
		var position dto.Position
		json.Unmarshal([]byte(task.Position), &position)
		resp.TaskNodes = append(resp.TaskNodes, dto.TaskNode{
			ID:          task.ID,
			Name:        task.Name,
			Type:        dto.TaskType(task.Type),
			Config:      json.RawMessage(task.Config),
			Tag:         task.Tag,
			Position:    position,
			NodeType:    task.NodeType,
			Description: task.Description,
		})
	}

	for _, edge := range edges {
		resp.Edges = append(resp.Edges, dto.Edge{
			ID:           edge.ID,
			Source:       edge.Source,
			Target:       edge.Target,
			SourceHandle: edge.SourceHandle,
			TargetHandle: edge.TargetHandle,
		})
	}

	ResponseSuccess(c, resp)
}

func (h *WorkflowHandler) Save(c *gin.Context) {
	var req dto.Workflow
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	req.ID = c.Param("id")
	workflowConfigBytes, err := json.Marshal(req.Config)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}
	if _, err := workflowcfg.ParseWorkflowConfig(string(workflowConfigBytes)); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	taskEntities := []entity.TaskEntity{}
	for _, task := range req.TaskNodes {
		config, _ := task.Config.MarshalJSON()
		position, _ := json.Marshal(task.Position)
		taskEntities = append(taskEntities, entity.TaskEntity{
			ID:          task.ID,
			WorkflowID:  req.ID,
			Name:        task.Name,
			Description: task.Description,
			Type:        string(task.Type),
			Config:      string(config),
			Tag:         task.Tag,
			Position:    string(position),
			NodeType:    task.NodeType,
		})
	}

	edgeEntities := []entity.EdgeEntity{}
	for _, edge := range req.Edges {
		edgeEntities = append(edgeEntities, entity.EdgeEntity{
			ID:           edge.ID,
			WorkflowID:   req.ID,
			Source:       edge.Source,
			Target:       edge.Target,
			SourceHandle: edge.SourceHandle,
			TargetHandle: edge.TargetHandle,
		})
	}

	if err := workflowcfg.ValidateDefinition(taskEntities, edgeEntities); err != nil {
		ResponseErrorWithDetails(c, http.StatusBadRequest, ErrorCodeInvalidWorkflow, "Workflow validation failed", err.Error())
		return
	}

	workflowEntity := entity.WorkflowEntity{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Config:      string(workflowConfigBytes),
		Tag:         req.Tag,
		Enable:      req.Enable,
	}
	if err := h.db.SaveWorkflowDefinition(workflowEntity, taskEntities, edgeEntities); err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, nil)
}

func (h *WorkflowHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	deleted, err := h.db.DeleteWorkflowDefinition(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	if !deleted {
		ResponseError(c, http.StatusNotFound, ErrorCodeWorkflowNotFound, fmt.Sprintf("Workflow '%s' not found", id))
		return
	}

	ResponseSuccess(c, nil)
}
