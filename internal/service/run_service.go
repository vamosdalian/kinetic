package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vamosdalian/kinetic/internal/database"
	"github.com/vamosdalian/kinetic/internal/executor"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

type RunService struct {
	db       database.Database
	executor *executor.Executor
}

type taskResult struct {
	taskID string
	result executor.TaskResult
	err    error
}

type dag struct {
	tasks      map[string]entity.TaskRunEntity
	children   map[string][]string
	indegree   map[string]int
	topoCount  int
	totalTasks int
}

func NewRunService(db database.Database, maxConcurrency int) *RunService {
	return &RunService{
		db:       db,
		executor: executor.NewExecutor(maxConcurrency),
	}
}

func (s *RunService) StartWorkflowRun(workflowID string) (string, error) {
	runID := uuid.New().String()
	if err := s.db.CreateWorkflowRun(workflowID, runID); err != nil {
		return "", err
	}

	go s.executeRun(runID)

	return runID, nil
}

func (s *RunService) executeRun(runID string) {
	if err := s.db.MarkWorkflowRunRunning(runID); err != nil {
		logrus.Errorf("failed to mark workflow run %s running: %v", runID, err)
		return
	}

	taskRuns, err := s.db.GetTaskRuns(runID)
	if err != nil {
		_ = s.db.FinishWorkflowRun(runID, "failed")
		logrus.Errorf("failed to load task runs for %s: %v", runID, err)
		return
	}

	edgeRuns, err := s.db.GetEdgeRuns(runID)
	if err != nil {
		_ = s.db.SkipPendingTaskRuns(runID, "Skipped because workflow metadata could not be loaded.")
		_ = s.db.FinishWorkflowRun(runID, "failed")
		logrus.Errorf("failed to load edge runs for %s: %v", runID, err)
		return
	}

	graph, err := buildDAG(taskRuns, edgeRuns)
	if err != nil {
		_ = s.db.SkipPendingTaskRuns(runID, fmt.Sprintf("Skipped because workflow graph is invalid: %v", err))
		_ = s.db.FinishWorkflowRun(runID, "failed")
		logrus.Warnf("workflow run %s graph validation failed: %v", runID, err)
		return
	}

	if graph.totalTasks == 0 {
		if err := s.db.FinishWorkflowRun(runID, "success"); err != nil {
			logrus.Errorf("failed to finish empty workflow run %s: %v", runID, err)
		}
		return
	}

	if err := s.runDAG(context.Background(), runID, graph); err != nil {
		logrus.Errorf("workflow run %s failed: %v", runID, err)
		return
	}
}

func (s *RunService) runDAG(ctx context.Context, runID string, graph dag) error {
	remainingDeps := make(map[string]int, len(graph.indegree))
	for taskID, count := range graph.indegree {
		remainingDeps[taskID] = count
	}

	statuses := make(map[string]string, len(graph.tasks))
	ready := make([]string, 0, len(graph.tasks))
	for taskID, count := range remainingDeps {
		if count == 0 {
			ready = append(ready, taskID)
		}
		statuses[taskID] = "pending"
	}

	results := make(chan taskResult, len(graph.tasks))
	var once sync.Once
	var runErr error
	running := 0
	failed := false

	launch := func(taskID string) {
		running++
		statuses[taskID] = "running"
		go func(task entity.TaskRunEntity) {
			result, err := s.executeTask(ctx, runID, task)
			results <- taskResult{
				taskID: task.TaskID,
				result: result,
				err:    err,
			}
		}(graph.tasks[taskID])
	}

	for len(ready) > 0 && !failed {
		taskID := ready[0]
		ready = ready[1:]
		launch(taskID)
	}

	for running > 0 {
		res := <-results
		running--
		if res.err != nil {
			failed = true
			once.Do(func() {
				runErr = res.err
			})
			statuses[res.taskID] = "failed"
		} else {
			statuses[res.taskID] = "success"
			for _, childID := range graph.children[res.taskID] {
				remainingDeps[childID]--
				if remainingDeps[childID] == 0 && !failed {
					ready = append(ready, childID)
				}
			}
		}

		for len(ready) > 0 && !failed {
			taskID := ready[0]
			ready = ready[1:]
			launch(taskID)
		}
	}

	if failed {
		if err := s.db.SkipPendingTaskRuns(runID, "Skipped because another task in the workflow failed."); err != nil {
			logrus.Errorf("failed to skip pending tasks for %s: %v", runID, err)
		}
		if err := s.db.FinishWorkflowRun(runID, "failed"); err != nil {
			return err
		}
		return runErr
	}

	return s.db.FinishWorkflowRun(runID, "success")
}

func (s *RunService) executeTask(ctx context.Context, runID string, task entity.TaskRunEntity) (executor.TaskResult, error) {
	if err := s.db.MarkTaskRunRunning(runID, task.TaskID); err != nil {
		return executor.TaskResult{ExitCode: -1}, err
	}

	execTask, err := executor.NewTask(executor.TaskEntity{
		ID:     task.TaskID,
		Type:   task.TaskType,
		Config: task.TaskConfig,
	})
	if err != nil {
		output := fmt.Sprintf("Failed to prepare task: %v", err)
		finishErr := s.db.FinishTaskRun(runID, task.TaskID, "failed", -1, output)
		if finishErr != nil {
			logrus.Errorf("failed to persist task preparation failure for %s/%s: %v", runID, task.TaskID, finishErr)
		}
		return executor.TaskResult{Output: output, ExitCode: -1}, err
	}

	result, err := s.executor.Execute(ctx, execTask)
	status := "success"
	if err != nil {
		status = "failed"
		if result.Output == "" {
			result.Output = err.Error()
		}
	}

	if finishErr := s.db.FinishTaskRun(runID, task.TaskID, status, result.ExitCode, result.Output); finishErr != nil {
		return result, finishErr
	}

	return result, err
}

func buildDAG(tasks []entity.TaskRunEntity, edges []entity.EdgeRunEntity) (dag, error) {
	graph := dag{
		tasks:      make(map[string]entity.TaskRunEntity, len(tasks)),
		children:   make(map[string][]string, len(tasks)),
		indegree:   make(map[string]int, len(tasks)),
		totalTasks: len(tasks),
	}

	for _, task := range tasks {
		graph.tasks[task.TaskID] = task
		graph.indegree[task.TaskID] = 0
		graph.children[task.TaskID] = nil
	}

	for _, edge := range edges {
		if _, ok := graph.tasks[edge.EdgeSource]; !ok {
			return dag{}, fmt.Errorf("edge source %s not found", edge.EdgeSource)
		}
		if _, ok := graph.tasks[edge.EdgeTarget]; !ok {
			return dag{}, fmt.Errorf("edge target %s not found", edge.EdgeTarget)
		}

		graph.children[edge.EdgeSource] = append(graph.children[edge.EdgeSource], edge.EdgeTarget)
		graph.indegree[edge.EdgeTarget]++
	}

	queue := make([]string, 0, len(graph.tasks))
	indegree := make(map[string]int, len(graph.indegree))
	for taskID, count := range graph.indegree {
		indegree[taskID] = count
		if count == 0 {
			queue = append(queue, taskID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		taskID := queue[0]
		queue = queue[1:]
		visited++

		for _, childID := range graph.children[taskID] {
			indegree[childID]--
			if indegree[childID] == 0 {
				queue = append(queue, childID)
			}
		}
	}

	graph.topoCount = visited
	if visited != len(graph.tasks) {
		return dag{}, fmt.Errorf("cycle detected")
	}

	return graph, nil
}
