package apiserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

type WorkflowHandler struct {
	db database.Database
}

func NewWorkflowHandler(db database.Database) *WorkflowHandler {
	return &WorkflowHandler{db: db}
}

type PageQuerys struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize" binding:"max=100"`
}

func (h *WorkflowHandler) List(c *gin.Context) {
	var pageQuerys PageQuerys
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
	workflows, err := h.db.ListWorkflows(page, pageSize)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	count, err := h.db.CountWorkflows()
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
			Version:   fmt.Sprintf("%d", w.Version),
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
		Version:     workflow.Version,
		Enable:      workflow.Enable,
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
	}

	for _, task := range tasks {
		var position dto.Position
		json.Unmarshal([]byte(task.Position), &position)
		resp.TaskNodes = append(resp.TaskNodes, dto.TaskNode{
			ID:          task.ID,
			Name:        task.Name,
			Type:        dto.TaskType(task.Type),
			Config:      json.RawMessage(task.Config),
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

	workflowEntity := entity.WorkflowEntity{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Enable:      req.Enable,
	}
	err := h.db.SaveWorkflow(workflowEntity)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	taskEntities := []entity.TaskEntity{}
	for _, task := range req.TaskNodes {
		config, _ := task.Config.MarshalJSON()
		position, _ := json.Marshal(task.Position)
		taskEntities = append(taskEntities, entity.TaskEntity{
			ID:         task.ID,
			WorkflowID: req.ID,
			Name:       task.Name,
			Type:       string(task.Type),
			Config:     string(config),
			Position:   string(position),
			NodeType:   task.NodeType,
		})
	}
	_, err = h.db.SaveTasks(taskEntities)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	edgeEntities := []entity.EdgeEntity{}
	for _, edge := range req.Edges {
		edgeEntities = append(edgeEntities, entity.EdgeEntity{
			ID:         edge.ID,
			WorkflowID: req.ID,
			Source:     edge.Source,
			Target:     edge.Target,
		})
	}
	_, err = h.db.SaveEdges(edgeEntities)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, nil)
}

func (h *WorkflowHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.db.DeleteWorkflow(id)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, ErrorCodeWorkflowNotFound, fmt.Sprintf("Workflow '%s' not found", id))
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	err = h.db.DeleteTasks(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	err = h.db.DeleteEdges(id)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, nil)
}

func (h *WorkflowHandler) Run(c *gin.Context) {
	ResponseSuccess(c, nil)
}
