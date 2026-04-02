package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

var ErrInvalidDashboardRange = errors.New("invalid dashboard range")

type DashboardStore interface {
	CountWorkflows() (int, error)
	ListNodes() ([]entity.NodeEntity, error)
	ListScheduledWorkflows() ([]entity.WorkflowEntity, error)
	ListWorkflowRunsByCreatedAt(start time.Time, end time.Time) ([]entity.WorkflowRunEntity, error)
	ListTaskRunsByRunIDs(runIDs []string) ([]entity.TaskRunEntity, error)
	ListTaskRunsByAssignedAt(start time.Time, end time.Time) ([]entity.TaskRunEntity, error)
}

type DashboardService struct {
	store DashboardStore
	now   func() time.Time
}

func NewDashboardService(store DashboardStore) *DashboardService {
	return &DashboardService{
		store: store,
		now:   time.Now,
	}
}

func ParseDashboardRange(rangeKey string) (int, error) {
	switch strings.TrimSpace(rangeKey) {
	case "", "30d":
		return 30, nil
	case "7d":
		return 7, nil
	case "90d":
		return 90, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrInvalidDashboardRange, rangeKey)
	}
}

func ResolveDashboardLocation(timezone string) (string, *time.Location) {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return time.UTC.String(), time.UTC
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC.String(), time.UTC
	}

	return timezone, loc
}

func (s *DashboardService) GetDashboard(rangeKey string, timezone string) (dto.DashboardResponse, error) {
	if s.store == nil {
		return dto.DashboardResponse{}, errors.New("dashboard store is not configured")
	}

	rangeDays, err := ParseDashboardRange(rangeKey)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	normalizedTZ, loc := ResolveDashboardLocation(timezone)
	nowUTC := s.now().UTC()
	nowLocal := nowUTC.In(loc)
	startOfTodayLocal := startOfDay(nowLocal)
	endOfTodayLocal := startOfTodayLocal.AddDate(0, 0, 1)
	startOfChartLocal := startOfTodayLocal.AddDate(0, 0, -(rangeDays - 1))

	workflowRuns, err := s.store.ListWorkflowRunsByCreatedAt(startOfChartLocal.UTC(), endOfTodayLocal.UTC())
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	totalWorkflows, err := s.store.CountWorkflows()
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	nodes, err := s.store.ListNodes()
	if err != nil {
		return dto.DashboardResponse{}, err
	}
	scheduledWorkflows, err := s.store.ListScheduledWorkflows()
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	todayRuns := filterWorkflowRunsByLocalWindow(workflowRuns, startOfTodayLocal, endOfTodayLocal, loc)
	todayTaskRuns, err := s.store.ListTaskRunsByAssignedAt(startOfTodayLocal.UTC(), endOfTodayLocal.UTC())
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	nodeLabels, err := s.buildNodeLabels(todayRuns)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	response := dto.DashboardResponse{
		Summary: dto.DashboardSummary{
			WorkflowRuns:   len(workflowRuns),
			TotalWorkflows: totalWorkflows,
			TotalNodes:     len(nodes),
			SuccessRate:    calculateWorkflowSuccessRate(workflowRuns),
		},
		Chart: dto.DashboardChart{
			Range:    normalizeDashboardRangeKey(rangeDays),
			Timezone: normalizedTZ,
			Points:   buildChartPoints(workflowRuns, startOfChartLocal, rangeDays, loc),
		},
		Tables: dto.DashboardTables{
			TodayWorkflows: dto.DashboardWorkflowTable{
				Count: len(todayRuns),
				Items: buildWorkflowRows(todayRuns, nodeLabels, nowUTC),
			},
			ScheduledRuns: dto.DashboardScheduledRunTable{
				Count: len(scheduledWorkflows),
				Items: buildScheduledRunRows(scheduledWorkflows),
			},
			FailedRuns: dto.DashboardWorkflowTable{
				Items: buildFailedWorkflowRows(todayRuns, nodeLabels, nowUTC),
			},
			NodeActivity: dto.DashboardNodeActivityTable{
				Items: buildNodeActivityRows(nodes, todayTaskRuns),
			},
		},
	}

	response.Tables.FailedRuns.Count = len(response.Tables.FailedRuns.Items)
	response.Tables.NodeActivity.Count = len(response.Tables.NodeActivity.Items)

	return response, nil
}

func (s *DashboardService) buildNodeLabels(runs []entity.WorkflowRunEntity) (map[string]string, error) {
	runIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.RunID)
	}
	if len(runIDs) == 0 {
		return map[string]string{}, nil
	}

	taskRuns, err := s.store.ListTaskRunsByRunIDs(runIDs)
	if err != nil {
		return nil, err
	}

	nodeSets := make(map[string]map[string]struct{}, len(runIDs))
	for _, taskRun := range taskRuns {
		if strings.TrimSpace(taskRun.AssignedNodeID) == "" {
			continue
		}
		if nodeSets[taskRun.RunID] == nil {
			nodeSets[taskRun.RunID] = make(map[string]struct{})
		}
		nodeSets[taskRun.RunID][taskRun.AssignedNodeID] = struct{}{}
	}

	result := make(map[string]string, len(runIDs))
	for _, runID := range runIDs {
		uniqueNodes := nodeSets[runID]
		if len(uniqueNodes) == 0 {
			result[runID] = "-"
			continue
		}

		nodes := make([]string, 0, len(uniqueNodes))
		for nodeID := range uniqueNodes {
			nodes = append(nodes, nodeID)
		}
		sort.Strings(nodes)

		if len(nodes) == 1 {
			result[runID] = nodes[0]
			continue
		}

		result[runID] = fmt.Sprintf("Multiple (%d)", len(nodes))
	}

	return result, nil
}

func filterWorkflowRunsByLocalWindow(runs []entity.WorkflowRunEntity, start time.Time, end time.Time, loc *time.Location) []entity.WorkflowRunEntity {
	filtered := make([]entity.WorkflowRunEntity, 0, len(runs))
	for _, run := range runs {
		runTime := run.CreatedAt.In(loc)
		if !runTime.Before(start) && runTime.Before(end) {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

func buildChartPoints(runs []entity.WorkflowRunEntity, start time.Time, days int, loc *time.Location) []dto.DashboardChartPoint {
	points := make([]dto.DashboardChartPoint, 0, days)
	indexByDate := make(map[string]int, days)

	for offset := 0; offset < days; offset++ {
		day := start.AddDate(0, 0, offset)
		key := day.Format("2006-01-02")
		indexByDate[key] = len(points)
		points = append(points, dto.DashboardChartPoint{Date: key})
	}

	for _, run := range runs {
		localDay := run.CreatedAt.In(loc)
		key := localDay.Format("2006-01-02")
		index, ok := indexByDate[key]
		if !ok {
			continue
		}

		switch run.Status {
		case "success":
			points[index].Success++
		case "failed":
			points[index].Failure++
		}
	}

	return points
}

func buildWorkflowRows(runs []entity.WorkflowRunEntity, nodeLabels map[string]string, now time.Time) []dto.DashboardWorkflowRow {
	rows := make([]dto.DashboardWorkflowRow, 0, len(runs))
	for _, run := range sortWorkflowRuns(runs) {
		rows = append(rows, dto.DashboardWorkflowRow{
			RunID:           run.RunID,
			WorkflowID:      run.WorkflowID,
			WorkflowName:    run.WorkflowName,
			Status:          run.Status,
			StartedAt:       safeFormatTime(run.StartedAt),
			DurationSeconds: calculateDurationSeconds(run, now),
			NodeLabel:       nodeLabelForRun(run.RunID, nodeLabels),
		})
	}
	return rows
}

func buildFailedWorkflowRows(runs []entity.WorkflowRunEntity, nodeLabels map[string]string, now time.Time) []dto.DashboardWorkflowRow {
	failed := make([]entity.WorkflowRunEntity, 0, len(runs))
	for _, run := range runs {
		if run.Status == "failed" {
			failed = append(failed, run)
		}
	}
	return buildWorkflowRows(failed, nodeLabels, now)
}

func buildScheduledRunRows(workflows []entity.WorkflowEntity) []dto.DashboardScheduledRunRow {
	rows := make([]dto.DashboardScheduledRunRow, 0, len(workflows))
	for _, workflow := range workflows {
		status := "disabled"
		if workflow.Enable {
			status = "enabled"
		}
		rows = append(rows, dto.DashboardScheduledRunRow{
			WorkflowID:   workflow.ID,
			WorkflowName: workflow.Name,
			Mode:         workflow.TriggerType,
			Status:       status,
			NextRunAt:    safeFormatTime(workflow.NextRunAt),
			LastRunAt:    safeFormatTime(workflow.LastRunAt),
			TargetTag:    workflow.Tag,
		})
	}
	return rows
}

func buildNodeActivityRows(nodes []entity.NodeEntity, taskRuns []entity.TaskRunEntity) []dto.DashboardNodeActivityRow {
	type aggregate struct {
		total    int
		success  int
		terminal int
	}

	statsByNode := make(map[string]*aggregate, len(nodes))
	for _, taskRun := range taskRuns {
		nodeID := strings.TrimSpace(taskRun.AssignedNodeID)
		if nodeID == "" {
			continue
		}

		stats := statsByNode[nodeID]
		if stats == nil {
			stats = &aggregate{}
			statsByNode[nodeID] = stats
		}

		stats.total++
		if isDashboardTerminalTaskStatus(taskRun.Status) {
			stats.terminal++
			if taskRun.Status == "success" {
				stats.success++
			}
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})

	rows := make([]dto.DashboardNodeActivityRow, 0, len(nodes))
	for _, node := range nodes {
		stats := statsByNode[node.NodeID]
		todayRuns := 0
		successRate := 0.0
		if stats != nil {
			todayRuns = stats.total
			if stats.terminal > 0 {
				successRate = (float64(stats.success) / float64(stats.terminal)) * 100
			}
		}

		rows = append(rows, dto.DashboardNodeActivityRow{
			NodeID:          node.NodeID,
			NodeName:        node.Name,
			Role:            node.Kind,
			Status:          node.Status,
			TodayRuns:       todayRuns,
			SuccessRate:     successRate,
			RunningCount:    node.RunningCount,
			LastHeartbeatAt: safeFormatTime(node.LastHeartbeatAt),
		})
	}

	return rows
}

func sortWorkflowRuns(runs []entity.WorkflowRunEntity) []entity.WorkflowRunEntity {
	sorted := append([]entity.WorkflowRunEntity(nil), runs...)
	sort.Slice(sorted, func(i, j int) bool {
		left := workflowRowSortTime(sorted[i])
		right := workflowRowSortTime(sorted[j])
		if left.Equal(right) {
			return sorted[i].RunID > sorted[j].RunID
		}
		return left.After(right)
	})
	return sorted
}

func workflowRowSortTime(run entity.WorkflowRunEntity) time.Time {
	if run.StartedAt != nil && !run.StartedAt.IsZero() {
		return *run.StartedAt
	}
	return run.CreatedAt
}

func calculateWorkflowSuccessRate(runs []entity.WorkflowRunEntity) float64 {
	successes := 0
	terminal := 0
	for _, run := range runs {
		switch run.Status {
		case "success":
			successes++
			terminal++
		case "failed", "cancelled":
			terminal++
		}
	}
	if terminal == 0 {
		return 0
	}
	return (float64(successes) / float64(terminal)) * 100
}

func calculateDurationSeconds(run entity.WorkflowRunEntity, now time.Time) *int {
	if run.StartedAt == nil || run.StartedAt.IsZero() {
		return nil
	}

	endTime := now
	if run.FinishedAt != nil && !run.FinishedAt.IsZero() {
		endTime = *run.FinishedAt
	}

	duration := int(endTime.Sub(*run.StartedAt).Seconds())
	if duration < 0 {
		duration = 0
	}
	return &duration
}

func nodeLabelForRun(runID string, labels map[string]string) string {
	if label, ok := labels[runID]; ok {
		return label
	}
	return "-"
}

func normalizeDashboardRangeKey(days int) string {
	switch days {
	case 7:
		return "7d"
	case 90:
		return "90d"
	default:
		return "30d"
	}
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func isDashboardTerminalTaskStatus(status string) bool {
	switch status {
	case "success", "failed", "cancelled", "unknown", "skipped":
		return true
	default:
		return false
	}
}

func safeFormatTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
