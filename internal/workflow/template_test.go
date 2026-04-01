package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vamosdalian/kinetic/internal/model/entity"
)

func TestRenderStringUsesCustomDelimiters(t *testing.T) {
	rendered, err := RenderString("hello ${{ .workflow.name }}", map[string]any{
		"workflow": map[string]any{"name": "demo"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "hello demo", rendered)
}

func TestRenderStringFailsOnMissingValue(t *testing.T) {
	_, err := RenderString("${{ .workflow.missing }}", map[string]any{
		"workflow": map[string]any{"name": "demo"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestRenderJSONStringsRendersNestedValues(t *testing.T) {
	rendered, err := RenderJSONStrings(`{"script":"printf '%s' '${{ .upstream.resultJSON.message }}'","headers":{"X-Run":"${{ .runtime.runID }}"}}`, map[string]any{
		"runtime": map[string]any{"runID": "run-1"},
		"upstream": map[string]any{
			"resultJSON": map[string]any{"message": "ok"},
		},
	})

	assert.NoError(t, err)
	assert.JSONEq(t, `{"script":"printf '%s' 'ok'","headers":{"X-Run":"run-1"}}`, rendered)
}

func TestValidateDefinitionRejectsInvalidTemplateSyntax(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{{
		ID:     "task-1",
		Name:   "task-1",
		Type:   "shell",
		Config: `{"script":"${{ .workflow.name "}`,
	}}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestValidateDefinitionAllowsTemplatedConditionExpression(t *testing.T) {
	err := ValidateDefinition([]entity.TaskEntity{{
		ID:     "condition-1",
		Name:   "condition-1",
		Type:   "condition",
		Config: `{"expression":"exit_code == ${{ .upstream.outputJSON.expected }}"}`,
	}}, []entity.EdgeEntity{{
		ID:           "edge-in",
		Source:       "source-1",
		Target:       "condition-1",
		SourceHandle: "",
	}, {
		ID:           "edge-true",
		Source:       "condition-1",
		Target:       "target-true",
		SourceHandle: "true",
	}, {
		ID:           "edge-false",
		Source:       "condition-1",
		Target:       "target-false",
		SourceHandle: "false",
	}})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "edge source source-1 not found")
}
