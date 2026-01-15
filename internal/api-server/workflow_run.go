package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

func (h *WorkflowHandler) Run(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Workflow ID is required")
		return
	}

	runID := uuid.New().String()
	err := h.db.CreateWorkflowRun(workflowID, runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, gin.H{"run_id": runID})
}

func (h *WorkflowHandler) ListRuns(c *gin.Context) {
	var query PageQuerys
	if err := c.ShouldBindQuery(&query); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	offset := (query.Page - 1) * query.PageSize
	runs, err := h.db.ListWorkflowRuns(offset, query.PageSize)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	dtos := make([]dto.WorkflowRunListItem, len(runs))
	for i, run := range runs {
		dtos[i] = dto.WorkflowRunListItem{
			RunID:      run.RunID,
			WorkflowID: run.WorkflowID,
			Name:       run.WorkflowName,
			Version:    run.WorkflowVersion,
			Status:     run.Status,
			CreatedAt:  formatTime(run.CreatedAt),
			StartedAt:  safeFormatTime(run.StartedAt),
			FinishedAt: safeFormatTime(run.FinishedAt),
		}
	}

	ResponseSuccess(c, dtos)
}

func (h *WorkflowHandler) GetRun(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Run ID is required")
		return
	}

	run, err := h.db.GetWorkflowRun(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	taskRuns, err := h.db.GetTaskRuns(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	edgeRuns, err := h.db.GetEdgeRuns(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	runDto := dto.WorkflowRun{
		WorkflowRunListItem: dto.WorkflowRunListItem{
			RunID:      run.RunID,
			WorkflowID: run.WorkflowID,
			Name:       run.WorkflowName,
			Version:    run.WorkflowVersion,
			Status:     run.Status,
			CreatedAt:  formatTime(run.CreatedAt),
			StartedAt:  safeFormatTime(run.StartedAt),
			FinishedAt: safeFormatTime(run.FinishedAt),
		},
		Description: run.WorkflowDescription,
		TaskNodes:   make([]dto.TaskNodeRun, len(taskRuns)),
		Edges:       make([]dto.EdgeRun, len(edgeRuns)),
	}

	for i, t := range taskRuns {
		var pos dto.Position
		if t.TaskPosition != "" {
			_ = json.Unmarshal([]byte(t.TaskPosition), &pos) // Just best effort, otherwise 0,0
		}

		runDto.TaskNodes[i] = dto.TaskNodeRun{
			RunID:       t.RunID,
			TaskID:      t.TaskID,
			Name:        t.TaskName,
			Description: t.TaskDescription,
			Type:        dto.TaskType(t.TaskType),
			Config:      json.RawMessage(t.TaskConfig),
			Position:    pos,
			NodeType:    t.TaskNodeType,
			Status:      t.Status,
			CreatedAt:   formatTime(t.CreatedAt),
			StartedAt:   safeFormatTime(t.StartedAt),
			FinishedAt:  safeFormatTime(t.FinishedAt),
			ExitCode:    t.ExitCode,
		}
	}

	for i, e := range edgeRuns {
		runDto.Edges[i] = dto.EdgeRun{
			RunID:        e.RunID,
			EdgeID:       e.EdgeID,
			Source:       e.EdgeSource,
			Target:       e.EdgeTarget,
			SourceHandle: e.EdgeSourceHandle,
			TargetHandle: e.EdgeTargetHandle,
		}
	}

	ResponseSuccess(c, runDto)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func safeFormatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
