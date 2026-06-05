package workflow

import (
	"testing"
	"time"

	"github.com/vamosdalian/kinetic/internal/model/entity"
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

func TestParseTaskPolicyAppliesDefaultTimeout(t *testing.T) {
	policy, err := ParseTaskPolicy(`{"retry_count":2}`)
	if err != nil {
		t.Fatalf("expected task policy to parse: %v", err)
	}
	if policy.TimeoutSeconds != DefaultTaskTimeoutSeconds {
		t.Fatalf("expected default timeout %d, got %d", DefaultTaskTimeoutSeconds, policy.TimeoutSeconds)
	}

	explicit, err := ParseTaskPolicy(`{"timeout_seconds":30}`)
	if err != nil {
		t.Fatalf("expected task policy to parse: %v", err)
	}
	if explicit.TimeoutSeconds != 30 {
		t.Fatalf("expected explicit timeout 30 to be preserved, got %d", explicit.TimeoutSeconds)
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

func TestValidateDefinitionAllowsForLoopBranches(t *testing.T) {
	forID := "for-1"
	bodyID := "body-1"
	doneID := "done-1"
	rootID := "root-1"

	err := ValidateDefinition([]entity.TaskEntity{
		{ID: rootID, Name: "root", Type: "shell", Config: `{"script":"printf root"}`},
		{ID: forID, Name: "for", Type: "for", Config: `{"start":1,"end":3,"var":"N"}`},
		{ID: bodyID, Name: "body", Type: "shell", Config: `{"script":"printf body"}`},
		{ID: doneID, Name: "done", Type: "shell", Config: `{"script":"printf done"}`},
	}, []entity.EdgeEntity{
		{ID: "edge-root-for", Source: rootID, Target: forID},
		{ID: "edge-for-body", Source: forID, Target: bodyID, SourceHandle: "body"},
		{ID: "edge-for-done", Source: forID, Target: doneID, SourceHandle: "done"},
	})
	if err != nil {
		t.Fatalf("expected for workflow to validate: %v", err)
	}
}

func TestValidateDefinitionAllowsForLoopWithoutInboundOrDone(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{
		{ID: "for", Name: "for", Type: "for", Config: `{"start":1,"end":3,"var":"N"}`},
		{ID: "body", Name: "body", Type: "shell", Config: `{"script":"printf body"}`},
	}, []entity.EdgeEntity{
		{ID: "edge-for-body", Source: "for", Target: "body", SourceHandle: "body"},
	})
	if err != nil {
		t.Fatalf("expected for workflow without inbound or done to validate: %v", err)
	}
}

func TestValidateDefinitionRejectsInvalidForVariable(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{
		{ID: "root", Name: "root", Type: "shell", Config: `{"script":"printf root"}`},
		{ID: "for", Name: "for", Type: "for", Config: `{"start":1,"end":3,"var":"KINETIC_LOOP_VALUE"}`},
		{ID: "body", Name: "body", Type: "shell", Config: `{"script":"printf body"}`},
		{ID: "done", Name: "done", Type: "shell", Config: `{"script":"printf done"}`},
	}, []entity.EdgeEntity{
		{ID: "edge-root-for", Source: "root", Target: "for"},
		{ID: "edge-for-body", Source: "for", Target: "body", SourceHandle: "body"},
		{ID: "edge-for-done", Source: "for", Target: "done", SourceHandle: "done"},
	})
	if err == nil {
		t.Fatal("expected reserved loop variable to fail validation")
	}
}

func TestValidateDefinitionRejectsDuplicateTaskRef(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{
		{ID: "task-1", Ref: "build", Name: "build one", Type: "shell", Config: `{"script":"printf one"}`},
		{ID: "task-2", Ref: "build", Name: "build two", Type: "shell", Config: `{"script":"printf two"}`},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate task refs to fail validation")
	}
}

func TestValidateDefinitionRejectsInvalidTaskRef(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{
		{ID: "task-1", Ref: "build-image", Name: "build", Type: "shell", Config: `{"script":"printf build"}`},
	}, nil)
	if err == nil {
		t.Fatal("expected invalid task ref to fail validation")
	}
}
