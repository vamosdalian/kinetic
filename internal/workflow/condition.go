package workflow

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ConditionInput struct {
	Status   string
	ExitCode int
	Output   string
	Result   string
}

type ConditionExpression struct {
	Left     string
	Operator string
	Right    any
}

var supportedOperators = []string{
	" contains ",
	" == ",
	" != ",
	" >= ",
	" <= ",
	" > ",
	" < ",
}

func ParseConditionExpression(expr string) (ConditionExpression, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ConditionExpression{}, fmt.Errorf("condition expression is required")
	}

	for _, op := range supportedOperators {
		index := strings.Index(expr, op)
		if index == -1 {
			continue
		}

		left := strings.TrimSpace(expr[:index])
		right := strings.TrimSpace(expr[index+len(op):])
		if left == "" || right == "" {
			return ConditionExpression{}, fmt.Errorf("invalid condition expression: %s", expr)
		}

		value, err := parseLiteral(right)
		if err != nil {
			return ConditionExpression{}, err
		}

		return ConditionExpression{
			Left:     left,
			Operator: strings.TrimSpace(op),
			Right:    value,
		}, nil
	}

	return ConditionExpression{}, fmt.Errorf("unsupported condition expression: %s", expr)
}

func (c ConditionExpression) Evaluate(input ConditionInput) (bool, error) {
	left, err := resolveOperand(c.Left, input)
	if err != nil {
		return false, err
	}

	switch c.Operator {
	case "contains":
		leftString, ok := left.(string)
		if !ok {
			return false, fmt.Errorf("contains operator requires a string left operand")
		}
		rightString, ok := c.Right.(string)
		if !ok {
			return false, fmt.Errorf("contains operator requires a string right operand")
		}
		return strings.Contains(leftString, rightString), nil
	case "==":
		return compareEquality(left, c.Right), nil
	case "!=":
		return !compareEquality(left, c.Right), nil
	case ">":
		return compareOrdered(left, c.Right, func(a, b float64) bool { return a > b })
	case "<":
		return compareOrdered(left, c.Right, func(a, b float64) bool { return a < b })
	case ">=":
		return compareOrdered(left, c.Right, func(a, b float64) bool { return a >= b })
	case "<=":
		return compareOrdered(left, c.Right, func(a, b float64) bool { return a <= b })
	default:
		return false, fmt.Errorf("unsupported operator %s", c.Operator)
	}
}

func parseLiteral(value string) (any, error) {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1], nil
		}
	}

	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}

	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return number, nil
	}

	return nil, fmt.Errorf("unsupported literal %s", value)
}

func resolveOperand(path string, input ConditionInput) (any, error) {
	switch path {
	case "status":
		return input.Status, nil
	case "exit_code":
		return float64(input.ExitCode), nil
	case "output":
		return input.Output, nil
	case "json":
		return parseJSON(input.Output)
	default:
		if strings.HasPrefix(path, "json.") {
			jsonValue, err := parseJSON(input.Output)
			if err != nil {
				return nil, nil
			}
			return resolveJSONPath(jsonValue, strings.TrimPrefix(path, "json."))
		}
	}

	return nil, fmt.Errorf("unsupported condition operand %s", path)
}

func parseJSON(output string) (any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func resolveJSONPath(value any, path string) (any, error) {
	current := value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("json path %s is not an object", path)
		}
		current = object[segment]
	}

	return current, nil
}

func compareEquality(left any, right any) bool {
	leftNumber, leftIsNumber := normalizeNumber(left)
	rightNumber, rightIsNumber := normalizeNumber(right)
	if leftIsNumber && rightIsNumber {
		return math.Abs(leftNumber-rightNumber) < 1e-9
	}

	return fmt.Sprint(left) == fmt.Sprint(right)
}

func compareOrdered(left any, right any, compare func(a, b float64) bool) (bool, error) {
	leftNumber, leftIsNumber := normalizeNumber(left)
	rightNumber, rightIsNumber := normalizeNumber(right)
	if !leftIsNumber || !rightIsNumber {
		return false, fmt.Errorf("ordered comparison requires numeric operands")
	}
	return compare(leftNumber, rightNumber), nil
}

func normalizeNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case json.Number:
		value, err := v.Float64()
		if err == nil {
			return value, true
		}
	}
	return 0, false
}
