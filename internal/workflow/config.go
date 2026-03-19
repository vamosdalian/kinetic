package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TaskPolicy struct {
	TimeoutSeconds      int `json:"timeout_seconds,omitempty"`
	RetryCount          int `json:"retry_count,omitempty"`
	RetryBackoffSeconds int `json:"retry_backoff_seconds,omitempty"`
}

type ShellConfig struct {
	Script string `json:"script"`
	TaskPolicy
}

type HTTPConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	TaskPolicy
}

type PythonConfig struct {
	Script       string   `json:"script"`
	Requirements []string `json:"requirements,omitempty"`
	TaskPolicy
}

type ConditionConfig struct {
	Expression string `json:"expression"`
	TaskPolicy
}

func ParseTaskPolicy(raw string) (TaskPolicy, error) {
	if strings.TrimSpace(raw) == "" {
		return TaskPolicy{}, nil
	}

	var policy TaskPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return TaskPolicy{}, fmt.Errorf("invalid task policy: %w", err)
	}

	if policy.TimeoutSeconds < 0 {
		return TaskPolicy{}, fmt.Errorf("timeout_seconds must be >= 0")
	}
	if policy.RetryCount < 0 {
		return TaskPolicy{}, fmt.Errorf("retry_count must be >= 0")
	}
	if policy.RetryBackoffSeconds < 0 {
		return TaskPolicy{}, fmt.Errorf("retry_backoff_seconds must be >= 0")
	}

	return policy, nil
}
