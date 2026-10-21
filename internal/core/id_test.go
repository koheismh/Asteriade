package core

import "testing"

func TestNewIDHasPrefixAndIsUnique(t *testing.T) {
	a, b := NewID("task"), NewID("task")
	if len(a) <= len("task_") || a[:5] != "task_" {
		t.Fatalf("unexpected ID format %q", a)
	}
	if a == b {
		t.Fatal("generated duplicate IDs")
	}
}
