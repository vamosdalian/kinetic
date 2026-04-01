package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

const (
	TemplateLeftDelimiter  = "${{"
	TemplateRightDelimiter = "}}"
)

func ContainsTemplate(input string) bool {
	return strings.Contains(input, TemplateLeftDelimiter)
}

func ValidateTemplateString(input string) error {
	if !ContainsTemplate(input) {
		return nil
	}

	_, err := newTemplate("validate").Parse(input)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	return nil
}

func RenderString(input string, data any) (string, error) {
	if !ContainsTemplate(input) {
		return input, nil
	}

	tmpl, err := newTemplate("render").Parse(input)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buffer.String(), nil
}

func RenderStringMapValues(values map[string]string, data any) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	rendered := make(map[string]string, len(values))
	for key, value := range values {
		renderedValue, err := RenderString(value, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", key, err)
		}
		rendered[key] = renderedValue
	}

	return rendered, nil
}

func RenderJSONStrings(raw string, data any) (string, error) {
	if strings.TrimSpace(raw) == "" || !ContainsTemplate(raw) {
		return raw, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", fmt.Errorf("decode json: %w", err)
	}

	rendered, err := RenderJSONValueStrings(value, data)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(rendered)
	if err != nil {
		return "", fmt.Errorf("encode json: %w", err)
	}

	return string(encoded), nil
}

func RenderJSONValueStrings(value any, data any) (any, error) {
	switch typed := value.(type) {
	case string:
		return RenderString(typed, data)
	case []any:
		rendered := make([]any, len(typed))
		for index, item := range typed {
			next, err := RenderJSONValueStrings(item, data)
			if err != nil {
				return nil, err
			}
			rendered[index] = next
		}
		return rendered, nil
	case map[string]any:
		rendered := make(map[string]any, len(typed))
		for key, item := range typed {
			next, err := RenderJSONValueStrings(item, data)
			if err != nil {
				return nil, err
			}
			rendered[key] = next
		}
		return rendered, nil
	default:
		return value, nil
	}
}

func newTemplate(name string) *template.Template {
	return template.New(name).Option("missingkey=error").Delims(TemplateLeftDelimiter, TemplateRightDelimiter)
}
