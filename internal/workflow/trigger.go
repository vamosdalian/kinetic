package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type WorkflowTriggerType string

const (
	WorkflowTriggerManual WorkflowTriggerType = "manual"
	WorkflowTriggerCron   WorkflowTriggerType = "cron"
)

type WorkflowTrigger struct {
	Type      WorkflowTriggerType `json:"type"`
	Expr      string              `json:"expr,omitempty"`
	NextRunAt *time.Time          `json:"next_run_at,omitempty"`
	LastRunAt *time.Time          `json:"last_run_at,omitempty"`
}

func NormalizeWorkflowTrigger(trigger WorkflowTrigger, enabled bool, now time.Time) (WorkflowTrigger, error) {
	trigger.Type = WorkflowTriggerType(strings.TrimSpace(string(trigger.Type)))
	trigger.Expr = strings.TrimSpace(trigger.Expr)

	switch trigger.Type {
	case "", WorkflowTriggerManual:
		trigger.Type = WorkflowTriggerManual
		trigger.Expr = ""
		trigger.NextRunAt = nil
		return trigger, nil
	case WorkflowTriggerCron:
		if trigger.Expr == "" {
			return WorkflowTrigger{}, fmt.Errorf("trigger expr is required for cron workflows")
		}
		nextRunAt, err := NextCronTime(trigger.Expr, now.UTC())
		if err != nil {
			return WorkflowTrigger{}, err
		}
		if enabled {
			trigger.NextRunAt = &nextRunAt
		} else {
			trigger.NextRunAt = nil
		}
		return trigger, nil
	default:
		return WorkflowTrigger{}, fmt.Errorf("unsupported trigger type %q", trigger.Type)
	}
}

func ParseWorkflowTrigger(triggerType string, triggerExpr string, nextRunAt *time.Time, lastRunAt *time.Time) (WorkflowTrigger, error) {
	trigger, err := NormalizeWorkflowTrigger(WorkflowTrigger{
		Type:      WorkflowTriggerType(triggerType),
		Expr:      triggerExpr,
		NextRunAt: nextRunAt,
		LastRunAt: lastRunAt,
	}, nextRunAt != nil, time.Now().UTC())
	if err != nil {
		return WorkflowTrigger{}, err
	}
	trigger.NextRunAt = nextRunAt
	trigger.LastRunAt = lastRunAt
	return trigger, nil
}

type cronField struct {
	allowed map[int]bool
	any     bool
}

type cronSchedule struct {
	minute  cronField
	hour    cronField
	day     cronField
	month   cronField
	weekday cronField
}

func NextCronTime(expr string, after time.Time) (time.Time, error) {
	schedule, err := parseCronSchedule(expr)
	if err != nil {
		return time.Time{}, err
	}

	candidate := after.UTC().Truncate(time.Minute).Add(time.Minute)
	deadline := candidate.AddDate(5, 0, 0)
	for !candidate.After(deadline) {
		if schedule.matches(candidate) {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("cron expr %q did not produce a run within 5 years", expr)
}

func parseCronSchedule(expr string) (cronSchedule, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return cronSchedule{}, fmt.Errorf("cron expr must contain 5 fields")
	}

	minute, err := parseCronField(parts[0], 0, 59, false)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23, false)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("invalid hour field: %w", err)
	}
	day, err := parseCronField(parts[2], 1, 31, false)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	month, err := parseCronField(parts[3], 1, 12, false)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("invalid month field: %w", err)
	}
	weekday, err := parseCronField(parts[4], 0, 7, true)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return cronSchedule{
		minute:  minute,
		hour:    hour,
		day:     day,
		month:   month,
		weekday: weekday,
	}, nil
}

func parseCronField(raw string, min int, max int, normalizeWeekday bool) (cronField, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cronField{}, fmt.Errorf("field is required")
	}
	if raw == "*" {
		return cronField{any: true}, nil
	}

	field := cronField{allowed: make(map[int]bool)}
	for _, segment := range strings.Split(raw, ",") {
		if err := addCronSegment(field.allowed, strings.TrimSpace(segment), min, max, normalizeWeekday); err != nil {
			return cronField{}, err
		}
	}
	if len(field.allowed) == 0 {
		return cronField{}, fmt.Errorf("field does not allow any values")
	}
	return field, nil
}

func addCronSegment(allowed map[int]bool, segment string, min int, max int, normalizeWeekday bool) error {
	if segment == "" {
		return fmt.Errorf("empty segment")
	}

	step := 1
	base := segment
	if strings.Contains(segment, "/") {
		parts := strings.Split(segment, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid step segment %q", segment)
		}
		base = parts[0]
		value, err := strconv.Atoi(parts[1])
		if err != nil || value <= 0 {
			return fmt.Errorf("invalid step %q", parts[1])
		}
		step = value
	}

	rangeMin := min
	rangeMax := max
	switch {
	case base == "*" || base == "":
	case strings.Contains(base, "-"):
		rangeParts := strings.Split(base, "-")
		if len(rangeParts) != 2 {
			return fmt.Errorf("invalid range %q", base)
		}
		left, err := parseCronValue(rangeParts[0], min, max, normalizeWeekday)
		if err != nil {
			return err
		}
		right, err := parseCronValue(rangeParts[1], min, max, normalizeWeekday)
		if err != nil {
			return err
		}
		if left > right {
			return fmt.Errorf("invalid range %q", base)
		}
		rangeMin, rangeMax = left, right
	default:
		value, err := parseCronValue(base, min, max, normalizeWeekday)
		if err != nil {
			return err
		}
		rangeMin, rangeMax = value, value
	}

	for value := rangeMin; value <= rangeMax; value += step {
		allowed[value] = true
	}
	return nil
}

func parseCronValue(raw string, min int, max int, normalizeWeekday bool) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", raw)
	}
	if normalizeWeekday && value == 7 {
		value = 0
	}
	if value < min || value > max {
		return 0, fmt.Errorf("value %d is out of range", value)
	}
	return value, nil
}

func (f cronField) matches(value int) bool {
	if f.any {
		return true
	}
	return f.allowed[value]
}

func (s cronSchedule) matches(ts time.Time) bool {
	if !s.minute.matches(ts.Minute()) || !s.hour.matches(ts.Hour()) || !s.month.matches(int(ts.Month())) {
		return false
	}

	dayMatches := s.day.matches(ts.Day())
	weekdayMatches := s.weekday.matches(int(ts.Weekday()))
	if s.day.any && s.weekday.any {
		return true
	}
	if s.day.any {
		return weekdayMatches
	}
	if s.weekday.any {
		return dayMatches
	}
	return dayMatches || weekdayMatches
}
