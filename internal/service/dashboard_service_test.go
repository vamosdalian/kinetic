package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

type fakeDashboardStore struct {
	totalWorkflows int
	nodes          []entity.NodeEntity
	workflowRuns   []entity.WorkflowRunEntity
	taskRuns       []entity.TaskRunEntity
}

func (s *fakeDashboardStore) CountWorkflows() (int, error) {
	return s.totalWorkflows, nil
}

func (s *fakeDashboardStore) ListNodes() ([]entity.NodeEntity, error) {
	return append([]entity.NodeEntity(nil), s.nodes...), nil
}

func (s *fakeDashboardStore) ListWorkflowRunsByCreatedAt(start time.Time, end time.Time) ([]entity.WorkflowRunEntity, error) {
	filtered := make([]entity.WorkflowRunEntity, 0, len(s.workflowRuns))
	for _, run := range s.workflowRuns {
		if !run.CreatedAt.Before(start) && run.CreatedAt.Before(end) {
			filtered = append(filtered, run)
		}
	}
	return filtered, nil
}

func (s *fakeDashboardStore) ListTaskRunsByRunIDs(runIDs []string) ([]entity.TaskRunEntity, error) {
	if len(runIDs) == 0 {
		return []entity.TaskRunEntity{}, nil
	}

	allowed := make(map[string]struct{}, len(runIDs))
	for _, runID := range runIDs {
		allowed[runID] = struct{}{}
	}

	filtered := make([]entity.TaskRunEntity, 0, len(s.taskRuns))
	for _, taskRun := range s.taskRuns {
		if _, ok := allowed[taskRun.RunID]; ok {
			filtered = append(filtered, taskRun)
		}
	}
	return filtered, nil
}

func (s *fakeDashboardStore) ListTaskRunsByAssignedAt(start time.Time, end time.Time) ([]entity.TaskRunEntity, error) {
	filtered := make([]entity.TaskRunEntity, 0, len(s.taskRuns))
	for _, taskRun := range s.taskRuns {
		if taskRun.AssignedAt == nil || taskRun.AssignedAt.IsZero() {
			continue
		}
		if !taskRun.AssignedAt.Before(start) && taskRun.AssignedAt.Before(end) {
			filtered = append(filtered, taskRun)
		}
	}
	return filtered, nil
}

func TestDashboardService_GetDashboard(t *testing.T) {
	now := time.Date(2026, 3, 20, 2, 0, 0, 0, time.UTC)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	store := &fakeDashboardStore{
		totalWorkflows: 5,
		nodes: []entity.NodeEntity{
			{
				NodeID:          "controller-01",
				Name:            "controller-01",
				Kind:            "controller",
				Status:          "online",
				RunningCount:    1,
				LastHeartbeatAt: ptrTime(now.Add(-2 * time.Minute)),
			},
			{
				NodeID:          "worker-01",
				Name:            "worker-01",
				Kind:            "worker",
				Status:          "online",
				RunningCount:    2,
				LastHeartbeatAt: ptrTime(now.Add(-1 * time.Minute)),
			},
			{
				NodeID:          "worker-02",
				Name:            "worker-02",
				Kind:            "worker",
				Status:          "offline",
				RunningCount:    0,
				LastHeartbeatAt: ptrTime(now.Add(-12 * time.Minute)),
			},
		},
		workflowRuns: []entity.WorkflowRunEntity{
			newWorkflowRun("run-boundary-success", "workflow-boundary", "Boundary Success", "success", localTime(shanghai, 2026, 3, 20, 0, 15), localTimePtr(shanghai, 2026, 3, 20, 0, 16), localTimePtr(shanghai, 2026, 3, 20, 0, 18)),
			newWorkflowRun("run-failed-today", "workflow-daily", "Daily Failure", "failed", localTime(shanghai, 2026, 3, 20, 9, 0), localTimePtr(shanghai, 2026, 3, 20, 9, 1), localTimePtr(shanghai, 2026, 3, 20, 9, 2)),
			newWorkflowRun("run-running-today", "workflow-live", "Live Pipeline", "running", localTime(shanghai, 2026, 3, 20, 9, 30), localTimePtr(shanghai, 2026, 3, 20, 9, 31), nil),
			newWorkflowRun("run-success-month", "workflow-month", "Monthly Sync", "success", localTime(shanghai, 2026, 3, 5, 12, 0), localTimePtr(shanghai, 2026, 3, 5, 12, 1), localTimePtr(shanghai, 2026, 3, 5, 12, 4)),
			newWorkflowRun("run-cancelled", "workflow-cancelled", "Cancelled Flow", "cancelled", localTime(shanghai, 2026, 2, 25, 11, 0), localTimePtr(shanghai, 2026, 2, 25, 11, 1), localTimePtr(shanghai, 2026, 2, 25, 11, 5)),
			newWorkflowRun("run-before-today", "workflow-yesterday", "Previous Day", "failed", localTime(shanghai, 2026, 3, 19, 23, 45), localTimePtr(shanghai, 2026, 3, 19, 23, 46), localTimePtr(shanghai, 2026, 3, 19, 23, 50)),
			newWorkflowRun("run-old-success", "workflow-old", "Old Success", "success", time.Date(2025, 12, 1, 8, 0, 0, 0, time.UTC), ptrTime(time.Date(2025, 12, 1, 8, 1, 0, 0, time.UTC)), ptrTime(time.Date(2025, 12, 1, 8, 5, 0, 0, time.UTC))),
		},
		taskRuns: []entity.TaskRunEntity{
			newTaskRun("run-boundary-success", "task-1", "worker-01", "success", localTimePtr(shanghai, 2026, 3, 20, 0, 16)),
			newTaskRun("run-failed-today", "task-1", "worker-02", "failed", localTimePtr(shanghai, 2026, 3, 20, 9, 1)),
			newTaskRun("run-running-today", "task-1", "worker-01", "running", localTimePtr(shanghai, 2026, 3, 20, 9, 31)),
			newTaskRun("run-running-today", "task-2", "controller-01", "success", localTimePtr(shanghai, 2026, 3, 20, 9, 35)),
			newTaskRun("run-before-today", "task-1", "worker-01", "failed", localTimePtr(shanghai, 2026, 3, 19, 23, 46)),
		},
	}

	service := NewDashboardService(store)
	service.now = func() time.Time { return now }

	response, err := service.GetDashboard("7d", "Asia/Shanghai")
	require.NoError(t, err)

	assert.Equal(t, 4, response.Summary.WorkflowRuns)
	assert.Equal(t, 5, response.Summary.TotalWorkflows)
	assert.Equal(t, 3, response.Summary.TotalNodes)
	assert.InDelta(t, 33.33, response.Summary.SuccessRate, 0.01)

	require.Len(t, response.Chart.Points, 7)
	assert.Equal(t, "7d", response.Chart.Range)
	assert.Equal(t, "Asia/Shanghai", response.Chart.Timezone)
	assert.Equal(t, 1, chartPoint(response.Chart.Points, "2026-03-19").Failure)
	assert.Equal(t, 1, chartPoint(response.Chart.Points, "2026-03-20").Success)
	assert.Equal(t, 1, chartPoint(response.Chart.Points, "2026-03-20").Failure)

	assert.Equal(t, 3, response.Tables.TodayWorkflows.Count)
	require.Len(t, response.Tables.TodayWorkflows.Items, 3)
	assert.Equal(t, "run-running-today", response.Tables.TodayWorkflows.Items[0].RunID)
	assert.Equal(t, "Multiple (2)", response.Tables.TodayWorkflows.Items[0].NodeLabel)
	require.NotNil(t, response.Tables.TodayWorkflows.Items[0].DurationSeconds)
	assert.Equal(t, 1740, *response.Tables.TodayWorkflows.Items[0].DurationSeconds)
	assert.Equal(t, "worker-02", response.Tables.TodayWorkflows.Items[1].NodeLabel)
	assert.Equal(t, "worker-01", response.Tables.TodayWorkflows.Items[2].NodeLabel)

	assert.Equal(t, 1, response.Tables.FailedRuns.Count)
	require.Len(t, response.Tables.FailedRuns.Items, 1)
	assert.Equal(t, "run-failed-today", response.Tables.FailedRuns.Items[0].RunID)

	assert.Equal(t, 0, response.Tables.ScheduledRuns.Count)
	assert.Empty(t, response.Tables.ScheduledRuns.Items)

	assert.Equal(t, 3, response.Tables.NodeActivity.Count)
	require.Len(t, response.Tables.NodeActivity.Items, 3)
	assert.Equal(t, 100.0, response.Tables.NodeActivity.Items[0].SuccessRate)
	assert.Equal(t, 1, response.Tables.NodeActivity.Items[0].TodayRuns)
	assert.Equal(t, 100.0, response.Tables.NodeActivity.Items[1].SuccessRate)
	assert.Equal(t, 2, response.Tables.NodeActivity.Items[1].TodayRuns)
	assert.Equal(t, 0.0, response.Tables.NodeActivity.Items[2].SuccessRate)
	assert.Equal(t, 1, response.Tables.NodeActivity.Items[2].TodayRuns)
}

func TestDashboardService_GetDashboard_DefaultsTimezone(t *testing.T) {
	service := NewDashboardService(&fakeDashboardStore{})
	service.now = func() time.Time {
		return time.Date(2026, 3, 20, 2, 0, 0, 0, time.UTC)
	}

	response, err := service.GetDashboard("30d", "Bad/Timezone")
	require.NoError(t, err)
	assert.Equal(t, "UTC", response.Chart.Timezone)
}

func chartPoint(points []dto.DashboardChartPoint, date string) dto.DashboardChartPoint {
	for _, point := range points {
		if point.Date == date {
			return point
		}
	}
	return dto.DashboardChartPoint{}
}

func newWorkflowRun(runID string, workflowID string, workflowName string, status string, createdAt time.Time, startedAt *time.Time, finishedAt *time.Time) entity.WorkflowRunEntity {
	return entity.WorkflowRunEntity{
		RunID:        runID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		Status:       status,
		CreatedAt:    createdAt,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
	}
}

func newTaskRun(runID string, taskID string, nodeID string, status string, assignedAt *time.Time) entity.TaskRunEntity {
	return entity.TaskRunEntity{
		RunID:          runID,
		TaskID:         taskID,
		AssignedNodeID: nodeID,
		Status:         status,
		AssignedAt:     assignedAt,
	}
}

func localTime(loc *time.Location, year int, month time.Month, day int, hour int, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, loc).UTC()
}

func localTimePtr(loc *time.Location, year int, month time.Month, day int, hour int, minute int) *time.Time {
	value := localTime(loc, year, month, day, hour, minute)
	return &value
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
