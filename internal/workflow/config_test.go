package workflow

import (
	"testing"
	"time"
)

func TestParseWorkflowConfigRejectsReservedPrefix(t *testing.T) {
	_, err := ParseWorkflowConfig(`{"env":{"KINETIC_TASK_NAME":"override"}}`)
	if err == nil {
		t.Fatal("expected reserved env prefix to be rejected")
	}
}

func TestParseTaskPolicyRejectsReservedPrefix(t *testing.T) {
	_, err := ParseTaskPolicy(`{"env":{"KINETIC_WORKFLOW_NAME":"override"}}`)
	if err == nil {
		t.Fatal("expected reserved env prefix to be rejected")
	}
}

func TestParseWorkflowConfigAcceptsUserEnv(t *testing.T) {
	config, err := ParseWorkflowConfig(`{"env":{"API_TOKEN":"secret"}}`)
	if err != nil {
		t.Fatalf("expected workflow config to parse: %v", err)
	}
	if config.Env["API_TOKEN"] != "secret" {
		t.Fatalf("expected API_TOKEN env to round-trip, got %q", config.Env["API_TOKEN"])
	}
}

func TestNormalizeWorkflowTrigger(t *testing.T) {
	now := time.Date(2026, 4, 2, 10, 5, 0, 0, time.UTC)

	trigger, err := NormalizeWorkflowTrigger(WorkflowTrigger{}, true, now)
	if err != nil {
		t.Fatalf("expected manual trigger to normalize: %v", err)
	}
	if trigger.Type != WorkflowTriggerManual || trigger.Expr != "" || trigger.NextRunAt != nil {
		t.Fatalf("unexpected manual trigger normalization: %+v", trigger)
	}

	trigger, err = NormalizeWorkflowTrigger(WorkflowTrigger{
		Type: WorkflowTriggerCron,
		Expr: "*/15 * * * *",
	}, true, now)
	if err != nil {
		t.Fatalf("expected cron trigger to normalize: %v", err)
	}
	if trigger.NextRunAt == nil || !trigger.NextRunAt.Equal(time.Date(2026, 4, 2, 10, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next run: %+v", trigger.NextRunAt)
	}

	trigger, err = NormalizeWorkflowTrigger(WorkflowTrigger{
		Type: WorkflowTriggerCron,
		Expr: "0 * * * *",
	}, false, now)
	if err != nil {
		t.Fatalf("expected disabled cron trigger to normalize: %v", err)
	}
	if trigger.NextRunAt != nil {
		t.Fatal("expected disabled cron trigger to clear next_run_at")
	}
}

func TestNormalizeWorkflowTriggerRejectsInvalidCron(t *testing.T) {
	if _, err := NormalizeWorkflowTrigger(WorkflowTrigger{
		Type: WorkflowTriggerCron,
	}, true, time.Now().UTC()); err == nil {
		t.Fatal("expected missing cron expr to fail")
	}

	if _, err := NormalizeWorkflowTrigger(WorkflowTrigger{
		Type: WorkflowTriggerCron,
		Expr: "invalid cron",
	}, true, time.Now().UTC()); err == nil {
		t.Fatal("expected invalid cron expr to fail")
	}
}
