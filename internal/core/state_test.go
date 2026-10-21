package core

import "testing"

func TestDefaultLimitsAndBounds(t *testing.T) {
	limits := DefaultLimits()
	if err := ValidateLimits(limits); err != nil {
		t.Fatalf("defaults rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Limits){
		"zero depth":           func(l *Limits) { l.MaxDepth = 0 },
		"depth over cap":       func(l *Limits) { l.MaxDepth = 4 },
		"concurrency over cap": func(l *Limits) { l.MaxConcurrent = 4 },
		"negative retries":     func(l *Limits) { l.MaxRetries = -1 },
		"tasks over cap":       func(l *Limits) { l.MaxTasks = 101 },
		"execution over cap":   func(l *Limits) { l.ExecutionMinutes = 31 },
		"project over cap":     func(l *Limits) { l.ProjectMinutes = 121 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := limits
			mutate(&bad)
			if ValidateLimits(bad) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestTaskTransitions(t *testing.T) {
	allowed := [][2]string{{"queued", "ready"}, {"ready", "running"}, {"running", "verifying"}, {"verifying", "completed"}, {"failed", "ready"}, {"recovery_required", "verifying"}}
	for _, edge := range allowed {
		if !CanTransitionTask(edge[0], edge[1]) {
			t.Errorf("expected task transition %q -> %q", edge[0], edge[1])
		}
	}
	for _, edge := range [][2]string{{"queued", "running"}, {"completed", "ready"}, {"cancelled", "ready"}, {"unknown", "ready"}, {"running", "completed"}} {
		if CanTransitionTask(edge[0], edge[1]) {
			t.Errorf("unexpected task transition %q -> %q", edge[0], edge[1])
		}
	}
}

func TestExecutionTransitions(t *testing.T) {
	for _, edge := range [][2]string{{"starting", "running"}, {"running", "interrupt_requested"}, {"interrupt_requested", "interrupted"}, {"outcome_unknown", "failed"}} {
		if !CanTransitionExecution(edge[0], edge[1]) {
			t.Errorf("expected execution transition %q -> %q", edge[0], edge[1])
		}
	}
	for _, edge := range [][2]string{{"starting", "succeeded"}, {"succeeded", "running"}, {"failed", "running"}, {"unknown", "failed"}} {
		if CanTransitionExecution(edge[0], edge[1]) {
			t.Errorf("unexpected execution transition %q -> %q", edge[0], edge[1])
		}
	}
}

func TestProjectTransitions(t *testing.T) {
	for _, edge := range [][2]string{{"draft", "planning"}, {"planning", "running"}, {"running", "completed"}, {"needs_attention", "planning"}, {"completed", "planning"}} {
		if !CanTransitionProject(edge[0], edge[1]) {
			t.Errorf("expected project transition %q -> %q", edge[0], edge[1])
		}
	}
	for _, edge := range [][2]string{{"draft", "running"}, {"cancelled", "planning"}, {"completed", "running"}, {"unknown", "draft"}} {
		if CanTransitionProject(edge[0], edge[1]) {
			t.Errorf("unexpected project transition %q -> %q", edge[0], edge[1])
		}
	}
}
