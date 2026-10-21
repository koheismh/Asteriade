package core

import "errors"

var ErrInvalidLimits = errors.New("invalid limits")

func DefaultLimits() Limits {
	return Limits{MaxDepth: 3, MaxConcurrent: 3, MaxRetries: 2, MaxTasks: 100, ExecutionMinutes: 30, ProjectMinutes: 120}
}

func ValidateLimits(l Limits) error {
	if l.MaxDepth < 1 || l.MaxDepth > 3 || l.MaxConcurrent < 1 || l.MaxConcurrent > 3 ||
		l.MaxRetries < 0 || l.MaxRetries > 2 || l.MaxTasks < 1 || l.MaxTasks > 100 ||
		l.ExecutionMinutes < 1 || l.ExecutionMinutes > 30 || l.ProjectMinutes < 1 || l.ProjectMinutes > 120 {
		return ErrInvalidLimits
	}
	return nil
}

var taskTransitions = map[string]map[string]bool{
	"queued":            {"ready": true, "cancelled": true, "failed": true},
	"ready":             {"running": true, "awaiting_approval": true, "awaiting_answer": true, "cancelled": true, "failed": true},
	"running":           {"awaiting_approval": true, "awaiting_answer": true, "verifying": true, "failed": true, "cancelled": true, "recovery_required": true},
	"awaiting_approval": {"ready": true, "running": true, "failed": true, "cancelled": true, "recovery_required": true},
	"awaiting_answer":   {"ready": true, "running": true, "failed": true, "cancelled": true, "recovery_required": true},
	"verifying":         {"completed": true, "failed": true, "cancelled": true, "recovery_required": true},
	"failed":            {"ready": true, "cancelled": true},
	"recovery_required": {"verifying": true, "ready": true, "failed": true, "cancelled": true},
}

var executionTransitions = map[string]map[string]bool{
	"starting":            {"running": true, "failed": true, "interrupt_requested": true, "outcome_unknown": true},
	"running":             {"succeeded": true, "failed": true, "interrupt_requested": true, "outcome_unknown": true},
	"interrupt_requested": {"interrupted": true, "succeeded": true, "failed": true, "outcome_unknown": true},
	"outcome_unknown":     {"succeeded": true, "failed": true, "interrupted": true},
}

var projectTransitions = map[string]map[string]bool{
	"draft":           {"planning": true, "cancelled": true},
	"planning":        {"running": true, "needs_attention": true, "failed": true, "cancelled": true},
	"running":         {"completed": true, "needs_attention": true, "failed": true, "cancelled": true},
	"needs_attention": {"planning": true, "running": true, "failed": true, "cancelled": true},
	"failed":          {"planning": true, "cancelled": true},
	"completed":       {"planning": true},
}

func transitionAllowed(transitions map[string]map[string]bool, from, to string) bool {
	return transitions[from][to]
}

func CanTransitionTask(from, to string) bool { return transitionAllowed(taskTransitions, from, to) }
func CanTransitionExecution(from, to string) bool {
	return transitionAllowed(executionTransitions, from, to)
}
func CanTransitionProject(from, to string) bool {
	return transitionAllowed(projectTransitions, from, to)
}
