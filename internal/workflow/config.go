package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ReservedEnvPrefix = "KINETIC_"

type WorkflowConfig struct {
	Env map[string]string `json:"env,omitempty"`
}

type TaskPolicy struct {
	TimeoutSeconds      int               `json:"timeout_seconds,omitempty"`
	RetryCount          int               `json:"retry_count,omitempty"`
	RetryBackoffSeconds int               `json:"retry_backoff_seconds,omitempty"`
	Env                 map[string]string `json:"env,omitempty"`
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

func ParseWorkflowConfig(raw string) (WorkflowConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return WorkflowConfig{}, nil
	}

	var cfg WorkflowConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return WorkflowConfig{}, fmt.Errorf("invalid workflow config: %w", err)
	}

	if err := ValidateEnvMap(cfg.Env); err != nil {
		return WorkflowConfig{}, err
	}

	return cfg, nil

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
	if err := ValidateEnvMap(policy.Env); err != nil {
		return TaskPolicy{}, err
	}

	return policy, nil
}

func ValidateEnvMap(values map[string]string) error {
	for key := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			return fmt.Errorf("env key is required")
		}
		if strings.HasPrefix(trimmed, ReservedEnvPrefix) {
			return fmt.Errorf("env key %s uses reserved prefix %s", trimmed, ReservedEnvPrefix)
		}
	}

	return nil
}
