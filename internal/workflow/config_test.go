package workflow

import "testing"

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
