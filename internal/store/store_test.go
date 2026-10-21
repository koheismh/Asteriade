package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/koheismh/Asteriade/internal/core"
	"github.com/koheismh/Asteriade/internal/privatefs"
)

func openTest(t *testing.T) (*Store, context.Context) {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s, context.Background()
}
func project(t *testing.T, s *Store, ctx context.Context) core.Project {
	t.Helper()
	p, e := s.CreateProject(ctx, "title", "goal", "automatic", "create-1")
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestProjectsIdempotencyAndConflict(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	again, e := s.CreateProject(ctx, "title", "goal", "automatic", "create-1")
	if e != nil || again.ID != p.ID {
		t.Fatalf("idempotent result: %#v %v", again, e)
	}
	_, e = s.CreateProject(ctx, "different", "goal", "automatic", "create-1")
	if !errors.Is(e, ErrConflict) {
		t.Fatalf("expected conflict, got %v", e)
	}
	got, e := s.GetProject(ctx, p.ID)
	if e != nil || got.Status != "draft" {
		t.Fatalf("get project: %#v %v", got, e)
	}
}
func TestPauseResumeAndProjectCancellation(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	h, e := s.Pause(ctx, "project", p.ID, "pause-1")
	if e != nil {
		t.Fatal(e)
	}
	hs, e := s.ListHolds(ctx)
	if e != nil || len(hs) != 1 {
		t.Fatalf("holds: %d %v", len(hs), e)
	}
	if e = s.Resume(ctx, h.ID, h.Version, "resume-1"); e != nil {
		t.Fatal(e)
	}
	got, e := s.GetProject(ctx, p.ID)
	if e != nil || got.Status != "draft" {
		t.Fatalf("resume projection: %#v %v", got, e)
	}
	c, e := s.CancelProject(ctx, p.ID, p.Version, "cancel-1")
	if e != nil || c.LifecycleStatus != "cancelled" {
		t.Fatalf("cancel: %#v %v", c, e)
	}
	if _, e = s.CancelProject(ctx, p.ID, c.Version, "cancel-again"); !errors.Is(e, ErrConflict) {
		t.Fatalf("terminal cancel should conflict: %v", e)
	}
}
func TestRecoveryMarksUnknownWithoutReexecution(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	if _, e := s.db.ExecContext(ctx, "UPDATE projects SET lifecycle_status='running' WHERE id=?", p.ID); e != nil {
		t.Fatal(e)
	}
	task, execID := core.NewID("task"), core.NewID("exec")
	_, e := s.db.ExecContext(ctx, "INSERT INTO tasks(id,project_id,objective,status) VALUES(?,?,?,'running')", task, p.ID, "task")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.ExecContext(ctx, "INSERT INTO executions(id,project_id,task_id,status,attempt) VALUES(?,?,?,'running',1)", execID, p.ID, task)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.ExecContext(ctx, "INSERT INTO tool_operations(id,project_id,execution_id,operation_key,state) VALUES(?,?,?,?,'dispatched')", core.NewID("tool"), p.ID, execID, "op-1")
	if e != nil {
		t.Fatal(e)
	}
	n, e := s.Recover(ctx)
	if e != nil || n != 1 {
		t.Fatalf("recover %d %v", n, e)
	}
	var es, ts, os string
	s.db.QueryRowContext(ctx, "SELECT status FROM executions WHERE id=?", execID).Scan(&es)
	s.db.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id=?", task).Scan(&ts)
	s.db.QueryRowContext(ctx, "SELECT state FROM tool_operations WHERE operation_key='op-1'").Scan(&os)
	if es != "outcome_unknown" || ts != "recovery_required" || os != "unknown" {
		t.Fatalf("recovery states %s %s %s", es, ts, os)
	}
	proj, e := s.GetProject(ctx, p.ID)
	if e != nil || proj.LifecycleStatus != "needs_attention" {
		t.Fatalf("recovery project state %#v %v", proj, e)
	}
	n, e = s.Recover(ctx)
	if e != nil || n != 0 {
		t.Fatalf("second recovery %d %v", n, e)
	}
	if _, e = s.db.ExecContext(ctx, "UPDATE tasks SET status='running' WHERE id=?", task); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.ExecContext(ctx, "INSERT INTO tool_operations(id,project_id,execution_id,operation_key,state) VALUES(?,?,?,?,'dispatched')", core.NewID("tool"), p.ID, execID, "op-stale"); e != nil {
		t.Fatal(e)
	}
	n, e = s.Recover(ctx)
	if e != nil || n != 0 {
		t.Fatalf("unknown-trial recovery %d %v", n, e)
	}
	if e = s.db.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id=?", task).Scan(&ts); e != nil || ts != "recovery_required" {
		t.Fatalf("unknown task state %s %v", ts, e)
	}
	if e = s.db.QueryRowContext(ctx, "SELECT state FROM tool_operations WHERE operation_key='op-stale'").Scan(&os); e != nil || os != "unknown" {
		t.Fatalf("stale dispatched operation %s %v", os, e)
	}
}
func TestCancelWithLiveTrialWaitsForRecovery(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	task, execID := core.NewID("task"), core.NewID("exec")
	if _, e := s.db.ExecContext(ctx, "INSERT INTO tasks(id,project_id,objective,status) VALUES(?,?,?,'running')", task, p.ID, "task"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.ExecContext(ctx, "INSERT INTO executions(id,project_id,task_id,status,attempt) VALUES(?,?,?,'running',1)", execID, p.ID, task); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.ExecContext(ctx, "INSERT INTO tool_operations(id,project_id,execution_id,operation_key,state) VALUES(?,?,?,?,'dispatched')", core.NewID("tool"), p.ID, execID, "cancel-op"); e != nil {
		t.Fatal(e)
	}
	got, e := s.CancelProject(ctx, p.ID, p.Version, "cancel-live")
	if e != nil || got.LifecycleStatus != "needs_attention" || got.CancelRequestedAt == nil {
		t.Fatalf("cancel live: %#v %v", got, e)
	}
	var status string
	s.db.QueryRowContext(ctx, "SELECT status FROM executions WHERE id=?", execID).Scan(&status)
	if status != "outcome_unknown" {
		t.Fatalf("trial status %s", status)
	}
	var operation string
	if e = s.db.QueryRowContext(ctx, "SELECT state FROM tool_operations WHERE operation_key='cancel-op'").Scan(&operation); e != nil || operation != "unknown" {
		t.Fatalf("tool operation state %s: %v", operation, e)
	}
}
func TestApprovalHashDecisionAndExpiry(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	op := core.Operation{Kind: "external.send", Destination: "https://example.test", Account: "acct", Target: "main", PayloadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PolicyRevision: 1}
	a, e := s.RequestApproval(ctx, p.ID, op, "approval-1", time.Now().Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.DecideApproval(ctx, a.ID, a.Version, "wrong", "granted", "decision-1")
	if !errors.Is(e, ErrConflict) {
		t.Fatalf("expected hash conflict: %#v %v", got, e)
	}
	got, e = s.DecideApproval(ctx, a.ID, a.Version, a.OperationHash, "granted", "decision-2")
	if e != nil || got.Status != "granted" {
		t.Fatalf("decision: %#v %v", got, e)
	}
}
func TestBackupIsConsistentAndRefusesOverwrite(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if e := s.Backup(ctx, dest); e != nil {
		t.Fatal(e)
	}
	if st, e := os.Stat(dest); e != nil || (runtime.GOOS != "windows" && st.Mode().Perm() != 0600) {
		t.Fatalf("backup mode/stat: %v %v", st, e)
	}
	if e := privatefs.CheckFile(dest); e != nil {
		t.Fatal(e)
	}
	b, e := Open(dest)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	got, e := b.GetProject(ctx, p.ID)
	if e != nil || got.ID != p.ID {
		t.Fatalf("restore read: %#v %v", got, e)
	}
	if e = s.Backup(ctx, dest); !errors.Is(e, ErrConflict) {
		t.Fatalf("expected existing destination conflict, got %v", e)
	}
}
func TestEventsAreTransactionalWithProjectCreation(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	ev, e := s.Events(ctx, p.ID, 0)
	if e != nil || len(ev) != 1 || ev[0].Type != "ProjectCreated" {
		t.Fatalf("events %#v %v", ev, e)
	}
	if _, e = s.Events(ctx, p.ID, ev[0].Sequence); e != nil {
		t.Fatal(e)
	}
	var outbox int
	if e = s.db.QueryRowContext(ctx, "SELECT count(*) FROM outbox WHERE event_id=?", ev[0].ID).Scan(&outbox); e != nil || outbox != 1 {
		t.Fatalf("event/outbox atomic record: count=%d err=%v", outbox, e)
	}
}

func TestProjectAndEventCommitAtomically(t *testing.T) {
	s, ctx := openTest(t)
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_project_event BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT,'injected event failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateProject(ctx, "should rollback", "goal", "automatic", "rollback-1")
	if err == nil {
		t.Fatal("expected injected event failure")
	}
	var projects, receipts int
	if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM projects").Scan(&projects); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM request_receipts").Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if projects != 0 || receipts != 0 {
		t.Fatalf("partial commit: projects=%d receipts=%d", projects, receipts)
	}
}

func TestCompletedProjectCannotBeCancelled(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	if _, err := s.db.ExecContext(ctx, "UPDATE projects SET lifecycle_status='completed',version=version+1 WHERE id=?", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelProject(ctx, p.ID, p.Version+1, "cancel-completed"); !errors.Is(err, ErrConflict) {
		t.Fatalf("completed cancellation should conflict: %v", err)
	}
	got, err := s.GetProject(ctx, p.ID)
	if err != nil || got.LifecycleStatus != "completed" {
		t.Fatalf("completed project changed: %#v %v", got, err)
	}
}

func TestInterruptRequestsStopWithoutClaimingCompletion(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	taskID, executionID := core.NewID("task"), core.NewID("exec")
	if _, err := s.db.ExecContext(ctx, "UPDATE projects SET lifecycle_status='running',version=version+1 WHERE id=?", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO tasks(id,project_id,objective,status) VALUES(?,?,?,'running')", taskID, p.ID, "task"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO executions(id,project_id,task_id,status,attempt) VALUES(?,?,?,'running',1)", executionID, p.ID, taskID); err != nil {
		t.Fatal(err)
	}
	got, err := s.InterruptProject(ctx, p.ID, p.Version+1, "interrupt-1")
	if err != nil {
		t.Fatal(err)
	}
	var status string
	var deadline sql.NullString
	if err = s.db.QueryRowContext(ctx, "SELECT status,grace_deadline_at FROM executions WHERE id=?", executionID).Scan(&status, &deadline); err != nil {
		t.Fatal(err)
	}
	var taskStatus string
	if err = s.db.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id=?", taskID).Scan(&taskStatus); err != nil {
		t.Fatal(err)
	}
	if status != "interrupt_requested" || !deadline.Valid || taskStatus != "running" || got.LifecycleStatus != "running" {
		t.Fatalf("interrupt outcome exec=%s deadline=%v task=%s project=%s", status, deadline.Valid, taskStatus, got.LifecycleStatus)
	}
}

func TestSecondOpenOfSameDatabaseIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "single.db")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Open(path)
	if err == nil {
		second.Close()
		t.Fatal("second store open unexpectedly succeeded")
	}
}
