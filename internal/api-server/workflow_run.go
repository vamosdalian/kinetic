package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/model/dto"
)

func (h *WorkflowHandler) Run(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Workflow ID is required")
		return
	}

	if h.runService == nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, "Run service is not configured")
		return
	}

	runID, err := h.runService.StartWorkflowRun(workflowID)
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

	runDto, err := h.buildRunDTO(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, runDto)
}

func (h *WorkflowHandler) Rerun(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Run ID is required")
		return
	}
	if h.runService == nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, "Run service is not configured")
		return
	}

	newRunID, err := h.runService.RerunWorkflowRun(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	ResponseSuccess(c, gin.H{"run_id": newRunID})
}

func (h *WorkflowHandler) Cancel(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Run ID is required")
		return
	}
	if h.runService == nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, "Run service is not configured")
		return
	}

	if err := h.runService.CancelWorkflowRun(runID); err != nil {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	ResponseSuccess(c, gin.H{"run_id": runID})
}

func (h *WorkflowHandler) RunEvents(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "Run ID is required")
		return
	}
	if h.runService == nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, "Run service is not configured")
		return
	}

	snapshot, err := h.buildRunDTO(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}

	eventCh, cleanup, err := h.runService.SubscribeRunEvents(runID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		return
	}
	defer cleanup()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	writeSSEEvent(c, "snapshot", snapshot)
	if isTerminalRunStatus(snapshot.Status) {
		return
	}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			writeSSEEvent(c, event.Type, event)
			if event.Type == "run_status" && isTerminalRunStatus(event.Status) {
				return
			}
		case <-keepalive.C:
			writeSSEEvent(c, "keepalive", gin.H{"ts": time.Now().Format(time.RFC3339)})
		}
	}
}

func (h *WorkflowHandler) buildRunDTO(runID string) (dto.WorkflowRun, error) {
	run, err := h.db.GetWorkflowRun(runID)
	if err != nil {
		return dto.WorkflowRun{}, err
	}

	taskRuns, err := h.db.GetTaskRuns(runID)
	if err != nil {
		return dto.WorkflowRun{}, err
	}

	edgeRuns, err := h.db.GetEdgeRuns(runID)
	if err != nil {
		return dto.WorkflowRun{}, err
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
			_ = json.Unmarshal([]byte(t.TaskPosition), &pos)
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
			Output:      t.Output,
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

	return runDto, nil
}

func writeSSEEvent(c *gin.Context, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}

func isTerminalRunStatus(status string) bool {
	switch status {
	case "success", "failed", "cancelled":
		return true
	default:
		return false
	}
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
