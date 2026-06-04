package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/vamosdalian/kinetic/internal/model/entity"
)

var taskRefPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidateDefinition(tasks []entity.TaskEntity, edges []entity.EdgeEntity) error {
	taskMap := make(map[string]entity.TaskEntity, len(tasks))
	taskRefMap := make(map[string]string, len(tasks))
	inbound := make(map[string]int, len(tasks))
	outbound := make(map[string][]entity.EdgeEntity, len(tasks))
	indegree := make(map[string]int, len(tasks))

	for _, task := range tasks {
		if task.ID == "" {
			return fmt.Errorf("task id is required")
		}
		ref := task.RefOrDefault()
		if !taskRefPattern.MatchString(ref) {
			return fmt.Errorf("task %s has invalid ref %s", task.NameOrID(), ref)
		}
		if existingTaskID, ok := taskRefMap[ref]; ok && existingTaskID != task.ID {
			return fmt.Errorf("task ref %s must be unique", ref)
		}
		taskRefMap[ref] = task.ID
		taskMap[task.ID] = task
		inbound[task.ID] = 0
		indegree[task.ID] = 0
		outbound[task.ID] = nil

		if err := validateTaskConfig(task); err != nil {
			return fmt.Errorf("task %s: %w", task.NameOrID(), err)
		}
	}

	for _, edge := range edges {
		if _, ok := taskMap[edge.Source]; !ok {
			return fmt.Errorf("edge source %s not found", edge.Source)
		}
		if _, ok := taskMap[edge.Target]; !ok {
			return fmt.Errorf("edge target %s not found", edge.Target)
		}
		inbound[edge.Target]++
		indegree[edge.Target]++
		outbound[edge.Source] = append(outbound[edge.Source], edge)
	}

	for _, task := range tasks {
		if task.Type == "condition" {
			if inbound[task.ID] != 1 {
				return fmt.Errorf("condition task %s must have exactly one incoming edge", task.NameOrID())
			}
			if len(outbound[task.ID]) != 2 {
				return fmt.Errorf("condition task %s must have exactly two outgoing edges", task.NameOrID())
			}

			handles := map[string]bool{}
			for _, edge := range outbound[task.ID] {
				if edge.SourceHandle == "" {
					return fmt.Errorf("condition task %s requires source handles true and false", task.NameOrID())
				}
				handles[edge.SourceHandle] = true
			}
			if !handles["true"] || !handles["false"] {
				return fmt.Errorf("condition task %s requires source handles true and false", task.NameOrID())
			}
			continue
		}

		if task.Type != "for" {
			continue
		}

		bodyCount := 0
		doneCount := 0
		handles := map[string]bool{}
		for _, edge := range outbound[task.ID] {
			if edge.SourceHandle == "" {
				return fmt.Errorf("for task %s requires source handle body", task.NameOrID())
			}
			handles[edge.SourceHandle] = true
			switch edge.SourceHandle {
			case "body":
				bodyCount++
			case "done":
				doneCount++
			default:
				return fmt.Errorf("for task %s only supports source handles body and done", task.NameOrID())
			}
		}
		if !handles["body"] || bodyCount != 1 {
			return fmt.Errorf("for task %s requires exactly one body source handle", task.NameOrID())
		}
		if doneCount > 1 {
			return fmt.Errorf("for task %s supports at most one done source handle", task.NameOrID())
		}
	}

	queue := make([]string, 0, len(tasks))
	for taskID, count := range indegree {
		if count == 0 {
			queue = append(queue, taskID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		taskID := queue[0]
		queue = queue[1:]
		visited++
		for _, edge := range outbound[taskID] {
			indegree[edge.Target]--
			if indegree[edge.Target] == 0 {
				queue = append(queue, edge.Target)
			}
		}
	}

	if visited != len(tasks) {
		return fmt.Errorf("workflow graph contains a cycle")
	}

	return nil
}

func validateTaskConfig(task entity.TaskEntity) error {
	switch task.Type {
	case "shell":
		var cfg ShellConfig
		if err := decodeConfig(task.Config, &cfg); err != nil {
			return fmt.Errorf("invalid shell config: %w", err)
		}
		if strings.TrimSpace(cfg.Script) == "" {
			return fmt.Errorf("shell script is required")
		}
		if err := ValidateTemplateString(cfg.Script); err != nil {
			return err
		}
		if err := ValidateTemplateEnvValues(cfg.TaskPolicy.Env); err != nil {
			return err
		}
	case "http":
		var cfg HTTPConfig
		if err := decodeConfig(task.Config, &cfg); err != nil {
			return fmt.Errorf("invalid http config: %w", err)
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return fmt.Errorf("http url is required")
		}
		if err := ValidateTemplateString(cfg.URL); err != nil {
			return err
		}
		if err := ValidateTemplateString(cfg.Method); err != nil {
			return err
		}
		if err := ValidateTemplateString(cfg.Body); err != nil {
			return err
		}
		for key, value := range cfg.Headers {
			if err := ValidateTemplateString(value); err != nil {
				return fmt.Errorf("invalid template in header %s: %w", key, err)
			}
		}
		if err := ValidateTemplateEnvValues(cfg.TaskPolicy.Env); err != nil {
			return err
		}
	case "condition":
		var cfg ConditionConfig
		if err := decodeConfig(task.Config, &cfg); err != nil {
			return fmt.Errorf("invalid condition config: %w", err)
		}
		if strings.TrimSpace(cfg.Expression) == "" {
			return fmt.Errorf("condition expression is required")
		}
		if err := ValidateTemplateString(cfg.Expression); err != nil {
			return err
		}
		if !ContainsTemplate(cfg.Expression) {
			if _, err := ParseConditionExpression(cfg.Expression); err != nil {
				return err
			}
		}
		if err := ValidateTemplateEnvValues(cfg.TaskPolicy.Env); err != nil {
			return err
		}
	case "for":
		var cfg ForConfig
		if err := decodeConfig(task.Config, &cfg); err != nil {
			return fmt.Errorf("invalid for config: %w", err)
		}
		if strings.TrimSpace(cfg.Var) != "" {
			if err := ValidateEnvName(strings.TrimSpace(cfg.Var)); err != nil {
				return fmt.Errorf("invalid for variable: %w", err)
			}
		}
		if err := ValidateTemplateEnvValues(cfg.TaskPolicy.Env); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported task type %s", task.Type)
	}

	if _, err := ParseTaskPolicy(task.Config); err != nil {
		return err
	}

	return nil
}

func decodeConfig(raw string, target any) error {
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	return json.Unmarshal([]byte(raw), target)
}

func ValidateTemplateEnvValues(values map[string]string) error {
	for key, value := range values {
		if err := ValidateTemplateString(value); err != nil {
			return fmt.Errorf("invalid template in env %s: %w", key, err)
		}
	}

	return nil
}
