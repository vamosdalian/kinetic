package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/executor"
	"github.com/vamosdalian/kinetic/internal/model/dto"
	"github.com/vamosdalian/kinetic/internal/model/entity"
	workflowcfg "github.com/vamosdalian/kinetic/internal/workflow"
)

var ErrRunCancelled = errors.New("workflow run cancelled")

type RunService struct {
	db               database.Database
	executor         *executor.Executor
	mu               sync.RWMutex
	cancels          map[string]context.CancelCauseFunc
	subscribers      map[string]map[int]chan dto.WorkflowRunEvent
	nextSubscriberID int
}

type edgeState string

const (
	edgeStateUnknown  edgeState = "unknown"
	edgeStateActive   edgeState = "active"
	edgeStateInactive edgeState = "inactive"
)

type runtimeEdge struct {
	edge  entity.EdgeRunEntity
	state edgeState
}

type runtimeNode struct {
	task     entity.TaskRunEntity
	inbound  []*runtimeEdge
	outbound []*runtimeEdge
}

type runGraph struct {
	nodes map[string]*runtimeNode
}

type completedTaskState struct {
	Status   string
	ExitCode int
	Output   string
}

type runtimeTaskResult struct {
	taskID         string
	status         string
	result         executor.TaskResult
	selectedBranch string
	err            error
}

func NewRunService(db database.Database, maxConcurrency int) *RunService {
	return &RunService{
		db:          db,
		executor:    executor.NewExecutor(maxConcurrency),
		cancels:     make(map[string]context.CancelCauseFunc),
		subscribers: make(map[string]map[int]chan dto.WorkflowRunEvent),
	}
}

func (s *RunService) StartWorkflowRun(workflowID string) (string, error) {
	runID := uuid.New().String()
	if err := s.db.CreateWorkflowRun(workflowID, runID); err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	s.storeCancel(runID, cancel)
	go s.executeRun(ctx, runID)

	return runID, nil
}

func (s *RunService) RerunWorkflowRun(runID string) (string, error) {
	run, err := s.db.GetWorkflowRun(runID)
	if err != nil {
		return "", err
	}
	return s.StartWorkflowRun(run.WorkflowID)
}

func (s *RunService) CancelWorkflowRun(runID string) error {
	run, err := s.db.GetWorkflowRun(runID)
	if err != nil {
		return err
	}
	if run.Status != "created" && run.Status != "running" {
		return fmt.Errorf("workflow run %s is already in terminal state %s", runID, run.Status)
	}

	cancel := s.getCancel(runID)
	if cancel == nil {
		return fmt.Errorf("workflow run %s cannot be cancelled", runID)
	}

	cancel(ErrRunCancelled)
	return nil
}

func (s *RunService) executeRun(ctx context.Context, runID string) {
	defer s.takeCancel(runID)

	if err := s.finalizeCancelledBeforeStart(ctx, runID); err != nil {
		return
	}

	if err := s.db.MarkWorkflowRunRunning(runID); err != nil {
		logrus.Errorf("failed to mark workflow run %s running: %v", runID, err)
		return
	}
	s.publishRunStatus(runID)

	taskRuns, err := s.db.GetTaskRuns(runID)
	if err != nil {
		s.failRun(runID, "Skipped because task metadata could not be loaded.", err)
		return
	}

	edgeRuns, err := s.db.GetEdgeRuns(runID)
	if err != nil {
		s.failRun(runID, "Skipped because workflow metadata could not be loaded.", err)
		return
	}

	graph, err := buildRunGraph(taskRuns, edgeRuns)
	if err != nil {
		s.failRun(runID, fmt.Sprintf("Skipped because workflow graph is invalid: %v", err), err)
		return
	}

	if len(graph.nodes) == 0 {
		if err := s.finishWorkflowRun(runID, "success"); err != nil {
			logrus.Errorf("failed to finish empty workflow run %s: %v", runID, err)
		}
		return
	}

	if _, err := s.runDAG(ctx, runID, graph); err != nil && !errors.Is(err, ErrRunCancelled) {
		logrus.Errorf("workflow run %s ended with error: %v", runID, err)
	}
}

func (s *RunService) runDAG(ctx context.Context, runID string, graph runGraph) (string, error) {
	results := make(chan runtimeTaskResult, len(graph.nodes))
	ready := make([]string, 0, len(graph.nodes))
	queued := make(map[string]bool, len(graph.nodes))
	running := make(map[string]bool, len(graph.nodes))
	completed := make(map[string]completedTaskState, len(graph.nodes))
	var firstErr error
	failed := false

	enqueueIfReady := func(taskID string) {
		if queued[taskID] || running[taskID] {
			return
		}
		if _, ok := completed[taskID]; ok {
			return
		}

		node := graph.nodes[taskID]
		activeCount := 0
		unknownCount := 0
		for _, edge := range node.inbound {
			switch edge.state {
			case edgeStateActive:
				activeCount++
			case edgeStateUnknown:
				unknownCount++
			}
		}

		if len(node.inbound) == 0 || (unknownCount == 0 && activeCount > 0) {
			ready = append(ready, taskID)
			queued[taskID] = true
		}
	}

	for taskID, node := range graph.nodes {
		if len(node.inbound) == 0 {
			ready = append(ready, taskID)
			queued[taskID] = true
		}
	}

	launchReady := func() {
		for len(ready) > 0 && ctx.Err() == nil && !failed {
			taskID := ready[0]
			ready = ready[1:]
			queued[taskID] = false
			running[taskID] = true

			node := graph.nodes[taskID]
			conditionInput, err := node.buildConditionInput(completed)
			if err != nil {
				results <- runtimeTaskResult{
					taskID: taskID,
					status: "failed",
					result: executor.TaskResult{ExitCode: -1, Output: err.Error()},
					err:    err,
				}
				continue
			}

			go func(task entity.TaskRunEntity, input *workflowcfg.ConditionInput) {
				results <- s.executeTask(ctx, runID, task, input)
			}(node.task, conditionInput)
		}
	}

	launchReady()

	for len(running) > 0 {
		result := <-results
		delete(running, result.taskID)

		switch result.status {
		case "success":
			completed[result.taskID] = completedTaskState{
				Status:   result.status,
				ExitCode: result.result.ExitCode,
				Output:   result.result.Output,
			}
			if ctx.Err() == nil && !failed {
				graph.activateOutbound(result.taskID, result.selectedBranch)
				for _, edge := range graph.nodes[result.taskID].outbound {
					enqueueIfReady(edge.edge.EdgeTarget)
				}
			}
		case "cancelled":
			if firstErr == nil && result.err != nil {
				firstErr = result.err
			}
		case "failed":
			failed = true
			if firstErr == nil {
				firstErr = result.err
			}
		}

		if ctx.Err() == nil && !failed {
			launchReady()
		}
	}

	if ctx.Err() != nil {
		if err := s.db.CancelPendingTaskRuns(runID, "Cancelled before execution."); err != nil {
			logrus.Errorf("failed to cancel pending tasks for %s: %v", runID, err)
		}
		if err := s.finishWorkflowRun(runID, "cancelled"); err != nil {
			return "cancelled", err
		}
		if firstErr == nil {
			firstErr = context.Cause(ctx)
		}
		return "cancelled", firstErr
	}

	if failed {
		if err := s.db.SkipPendingTaskRuns(runID, "Skipped because another task in the workflow failed."); err != nil {
			logrus.Errorf("failed to skip pending tasks for %s: %v", runID, err)
		}
		if err := s.finishWorkflowRun(runID, "failed"); err != nil {
			return "failed", err
		}
		return "failed", firstErr
	}

	if err := s.db.SkipPendingTaskRuns(runID, "Skipped because the condition branch was not activated."); err != nil {
		logrus.Errorf("failed to finalize inactive branches for %s: %v", runID, err)
	}
	if err := s.finishWorkflowRun(runID, "success"); err != nil {
		return "success", err
	}
	return "success", nil
}

func (s *RunService) executeTask(ctx context.Context, runID string, task entity.TaskRunEntity, conditionInput *workflowcfg.ConditionInput) runtimeTaskResult {
	if err := s.db.MarkTaskRunRunning(runID, task.TaskID); err != nil {
		return runtimeTaskResult{
			taskID: task.TaskID,
			status: "failed",
			result: executor.TaskResult{ExitCode: -1, Output: err.Error()},
			err:    err,
		}
	}
	s.publishTaskStatus(runID, task.TaskID)

	policy, err := workflowcfg.ParseTaskPolicy(task.TaskConfig)
	if err != nil {
		output := fmt.Sprintf("Invalid task policy: %v", err)
		_ = s.db.FinishTaskRun(runID, task.TaskID, "failed", -1, output)
		s.publishTaskStatus(runID, task.TaskID)
		return runtimeTaskResult{
			taskID: task.TaskID,
			status: "failed",
			result: executor.TaskResult{ExitCode: -1, Output: output},
			err:    err,
		}
	}

	var outputBuilder strings.Builder
	emitOutput := func(chunk string) {
		if chunk == "" {
			return
		}
		outputBuilder.WriteString(chunk)
		if err := s.db.AppendTaskRunOutput(runID, task.TaskID, chunk); err != nil {
			logrus.Errorf("failed to append task output for %s/%s: %v", runID, task.TaskID, err)
		}
		s.publishEvent(dto.WorkflowRunEvent{
			Type:   "task_output",
			RunID:  runID,
			TaskID: task.TaskID,
			Output: chunk,
		})
	}

	attempts := policy.RetryCount + 1
	var lastErr error
	var lastResult executor.TaskResult
	var selectedBranch string

	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			retryNotice := fmt.Sprintf("\n[retry %d/%d]\n", attempt-1, policy.RetryCount)
			emitOutput(retryNotice)
			if policy.RetryBackoffSeconds > 0 {
				select {
				case <-time.After(time.Duration(policy.RetryBackoffSeconds) * time.Second):
				case <-ctx.Done():
					lastErr = context.Cause(ctx)
				}
				if ctx.Err() != nil {
					break
				}
			}
		}

		attemptCtx := ctx
		cancelAttempt := func() {}
		if policy.TimeoutSeconds > 0 {
			attemptCtx, cancelAttempt = context.WithTimeout(ctx, time.Duration(policy.TimeoutSeconds)*time.Second)
		}

		lastResult, selectedBranch, lastErr = s.executeTaskAttempt(attemptCtx, task, conditionInput, emitOutput)
		cancelAttempt()

		if lastErr == nil {
			lastResult.Output = outputBuilder.String()
			if err := s.db.FinishTaskRun(runID, task.TaskID, "success", lastResult.ExitCode, lastResult.Output); err != nil {
				return runtimeTaskResult{
					taskID: task.TaskID,
					status: "failed",
					result: executor.TaskResult{ExitCode: -1, Output: err.Error()},
					err:    err,
				}
			}
			s.publishTaskStatus(runID, task.TaskID)
			return runtimeTaskResult{
				taskID:         task.TaskID,
				status:         "success",
				result:         lastResult,
				selectedBranch: selectedBranch,
			}
		}

		if errors.Is(context.Cause(ctx), ErrRunCancelled) || errors.Is(context.Cause(attemptCtx), ErrRunCancelled) {
			emitOutput("\nTask cancelled.\n")
			lastResult.Output = outputBuilder.String()
			_ = s.db.FinishTaskRun(runID, task.TaskID, "cancelled", lastResult.ExitCode, lastResult.Output)
			s.publishTaskStatus(runID, task.TaskID)
			return runtimeTaskResult{
				taskID: task.TaskID,
				status: "cancelled",
				result: lastResult,
				err:    ErrRunCancelled,
			}
		}

		if errors.Is(context.Cause(attemptCtx), context.DeadlineExceeded) {
			emitOutput(fmt.Sprintf("\nTask timed out after %d seconds.\n", policy.TimeoutSeconds))
		} else {
			emitOutput(fmt.Sprintf("\nAttempt %d failed: %v\n", attempt, lastErr))
		}

		if attempt == attempts || ctx.Err() != nil {
			break
		}
	}

	lastResult.Output = outputBuilder.String()
	if err := s.db.FinishTaskRun(runID, task.TaskID, "failed", lastResult.ExitCode, lastResult.Output); err != nil {
		return runtimeTaskResult{
			taskID: task.TaskID,
			status: "failed",
			result: executor.TaskResult{ExitCode: -1, Output: err.Error()},
			err:    err,
		}
	}
	s.publishTaskStatus(runID, task.TaskID)

	return runtimeTaskResult{
		taskID: task.TaskID,
		status: "failed",
		result: lastResult,
		err:    lastErr,
	}
}

func (s *RunService) executeTaskAttempt(ctx context.Context, task entity.TaskRunEntity, conditionInput *workflowcfg.ConditionInput, onOutput executor.OutputFunc) (executor.TaskResult, string, error) {
	if task.TaskType == "condition" {
		var cfg workflowcfg.ConditionConfig
		if err := json.Unmarshal([]byte(task.TaskConfig), &cfg); err != nil {
			return executor.TaskResult{ExitCode: -1}, "", fmt.Errorf("invalid condition config: %w", err)
		}
		if conditionInput == nil {
			return executor.TaskResult{ExitCode: -1}, "", fmt.Errorf("condition task %s is missing upstream input", task.TaskName)
		}
		expr, err := workflowcfg.ParseConditionExpression(cfg.Expression)
		if err != nil {
			return executor.TaskResult{ExitCode: -1}, "", err
		}
		matched, err := expr.Evaluate(*conditionInput)
		if err != nil {
			return executor.TaskResult{ExitCode: -1}, "", err
		}
		selectedBranch := "false"
		if matched {
			selectedBranch = "true"
		}
		message := fmt.Sprintf("Condition %q evaluated to %t", cfg.Expression, matched)
		if onOutput != nil {
			onOutput(message)
		}
		return executor.TaskResult{
			Output:   message,
			ExitCode: 0,
		}, selectedBranch, nil
	}

	execTask, err := executor.NewTask(executor.TaskEntity{
		ID:     task.TaskID,
		Type:   task.TaskType,
		Config: task.TaskConfig,
	})
	if err != nil {
		if onOutput != nil {
			onOutput(fmt.Sprintf("Failed to prepare task: %v", err))
		}
		return executor.TaskResult{ExitCode: -1}, "", err
	}

	result, err := s.executor.Execute(ctx, execTask, onOutput)
	return result, "", err
}

func buildRunGraph(tasks []entity.TaskRunEntity, edges []entity.EdgeRunEntity) (runGraph, error) {
	taskEntities := make([]entity.TaskEntity, 0, len(tasks))
	for _, task := range tasks {
		taskEntities = append(taskEntities, entity.TaskEntity{
			ID:          task.TaskID,
			WorkflowID:  task.WorkflowID,
			Name:        task.TaskName,
			Description: task.TaskDescription,
			Type:        task.TaskType,
			Config:      task.TaskConfig,
			Position:    task.TaskPosition,
			NodeType:    task.TaskNodeType,
		})
	}

	edgeEntities := make([]entity.EdgeEntity, 0, len(edges))
	for _, edge := range edges {
		edgeEntities = append(edgeEntities, entity.EdgeEntity{
			ID:           edge.EdgeID,
			WorkflowID:   edge.WorkflowID,
			Source:       edge.EdgeSource,
			Target:       edge.EdgeTarget,
			SourceHandle: edge.EdgeSourceHandle,
			TargetHandle: edge.EdgeTargetHandle,
		})
	}

	if err := workflowcfg.ValidateDefinition(taskEntities, edgeEntities); err != nil {
		return runGraph{}, err
	}

	graph := runGraph{
		nodes: make(map[string]*runtimeNode, len(tasks)),
	}
	for _, task := range tasks {
		graph.nodes[task.TaskID] = &runtimeNode{
			task:     task,
			inbound:  nil,
			outbound: nil,
		}
	}
	for _, edge := range edges {
		runtimeEdge := &runtimeEdge{
			edge:  edge,
			state: edgeStateUnknown,
		}
		graph.nodes[edge.EdgeSource].outbound = append(graph.nodes[edge.EdgeSource].outbound, runtimeEdge)
		graph.nodes[edge.EdgeTarget].inbound = append(graph.nodes[edge.EdgeTarget].inbound, runtimeEdge)
	}

	return graph, nil
}

func (g runGraph) activateOutbound(taskID string, selectedBranch string) {
	node := g.nodes[taskID]
	if node == nil {
		return
	}

	for _, edge := range node.outbound {
		if node.task.TaskType == "condition" {
			if edge.edge.EdgeSourceHandle == selectedBranch {
				edge.state = edgeStateActive
			} else {
				edge.state = edgeStateInactive
			}
			continue
		}
		edge.state = edgeStateActive
	}
}

func (n *runtimeNode) buildConditionInput(completed map[string]completedTaskState) (*workflowcfg.ConditionInput, error) {
	if n.task.TaskType != "condition" {
		return nil, nil
	}

	for _, edge := range n.inbound {
		if edge.state != edgeStateActive {
			continue
		}
		parent, ok := completed[edge.edge.EdgeSource]
		if !ok {
			return nil, fmt.Errorf("condition task %s is missing upstream result", n.task.TaskName)
		}
		return &workflowcfg.ConditionInput{
			Status:   parent.Status,
			ExitCode: parent.ExitCode,
			Output:   parent.Output,
		}, nil
	}

	return nil, fmt.Errorf("condition task %s requires an active upstream result", n.task.TaskName)
}

func (s *RunService) finalizeCancelledBeforeStart(ctx context.Context, runID string) error {
	if ctx.Err() == nil {
		return nil
	}
	if err := s.db.CancelPendingTaskRuns(runID, "Cancelled before execution."); err != nil {
		logrus.Errorf("failed to cancel pending tasks for %s: %v", runID, err)
	}
	if err := s.finishWorkflowRun(runID, "cancelled"); err != nil {
		return err
	}
	return ErrRunCancelled
}

func (s *RunService) failRun(runID string, skipMessage string, runErr error) {
	if err := s.db.SkipPendingTaskRuns(runID, skipMessage); err != nil {
		logrus.Errorf("failed to skip pending tasks for %s: %v", runID, err)
	}
	if err := s.finishWorkflowRun(runID, "failed"); err != nil {
		logrus.Errorf("failed to finish workflow run %s: %v", runID, err)
	}
	logrus.Warnf("workflow run %s failed: %v", runID, runErr)
}

func (s *RunService) finishWorkflowRun(runID string, status string) error {
	if err := s.db.FinishWorkflowRun(runID, status); err != nil {
		return err
	}
	s.publishRunStatus(runID)
	return nil
}

func (s *RunService) publishRunStatus(runID string) {
	run, err := s.db.GetWorkflowRun(runID)
	if err != nil {
		return
	}
	s.publishEvent(dto.WorkflowRunEvent{
		Type:       "run_status",
		RunID:      runID,
		Status:     run.Status,
		StartedAt:  formatTimePointer(run.StartedAt),
		FinishedAt: formatTimePointer(run.FinishedAt),
	})
}

func (s *RunService) publishTaskStatus(runID string, taskID string) {
	task, err := s.db.GetTaskRun(runID, taskID)
	if err != nil {
		return
	}
	exitCode := task.ExitCode
	s.publishEvent(dto.WorkflowRunEvent{
		Type:       "task_status",
		RunID:      runID,
		TaskID:     taskID,
		Status:     task.Status,
		StartedAt:  formatTimePointer(task.StartedAt),
		FinishedAt: formatTimePointer(task.FinishedAt),
		ExitCode:   &exitCode,
	})
}

func formatTimePointer(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
