package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/koheismh/Asteriade/internal/core"
)

func TestRejectsCrossProjectExecutionAndToolReferences(t *testing.T) {
	s, ctx := openTest(t)
	p, _ := s.CreateProject(ctx, "first", "goal", "plan_confirmation", "one")
	q, _ := s.CreateProject(ctx, "second", "goal", "plan_confirmation", "two")
	if _, err := s.db.ExecContext(ctx, "INSERT INTO tasks(id,project_id,objective,status) VALUES('task',?,'goal','running')", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO executions(id,project_id,task_id,status,attempt) VALUES('wrong',?,'task','running',1)", q.ID); err == nil {
		t.Fatal("cross-project execution accepted")
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO executions(id,project_id,task_id,status,attempt) VALUES('exec',?,'task','running',1)", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO tool_operations(id,project_id,execution_id,operation_key,state) VALUES('wrong',?,'exec','op','dispatched')", q.ID); err == nil {
		t.Fatal("cross-project tool accepted")
	}
}

func TestExpiredDecisionAndCancellationCannotLeaveUsableApproval(t *testing.T) {
	s, ctx := openTest(t)
	p, _ := s.CreateProject(ctx, "test", "goal", "plan_confirmation", "one")
	op := core.Operation{Kind: "external.send", Destination: "https://example.org", Account: "test", Target: "test", PayloadHash: strings.Repeat("a", 64), PolicyRevision: 1}
	a, err := s.RequestApproval(ctx, p.ID, op, "request", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE approvals SET expires_at=? WHERE id=?", time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano), a.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = s.DecideApproval(ctx, a.ID, a.Version, a.OperationHash, "granted", "expired-decision"); !errors.Is(err, ErrConflict) {
			t.Fatalf("expired grant replay returned success: %v", err)
		}
	}
	b, err := s.RequestApproval(ctx, p.ID, op, "second", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DecideApproval(ctx, b.ID, b.Version, b.OperationHash, "granted", "grant"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CancelProject(ctx, p.ID, p.Version, "cancel"); err != nil {
		t.Fatal(err)
	}
	as, err := s.ListApprovals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range as {
		if a.Status == "pending" || a.Status == "granted" {
			t.Fatal("cancelled project's approval remains usable")
		}
	}
}

func TestCorruptDatabaseIsNotReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.db")
	original := []byte("deliberately invalid sqlite data")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("corruption accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("original corrupt file changed", err)
	}
}

func TestConcurrentCreateUsesOneReceiptAndOneProject(t *testing.T) {
	s, ctx := openTest(t)
	var wg sync.WaitGroup
	ids := make(chan string, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := s.CreateProject(ctx, "one", "goal", "plan_confirmation", "same-key")
			ids <- p.ID
			errs <- err
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		} else if id != first {
			t.Fatal("duplicate project from concurrent retry")
		}
	}
	ps, err := s.ListProjects(ctx)
	if err != nil || len(ps) != 1 {
		t.Fatal("wrong project count", err)
	}
	es, err := s.Events(ctx, first, 0)
	if err != nil || len(es) != 1 {
		t.Fatal("duplicate event from concurrent retry", err)
	}
}

func TestSQLiteFullRollsBackAndPreservesExistingData(t *testing.T) {
	s, ctx := openTest(t)
	p := project(t, s, ctx)
	var pages, limit int
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf("PRAGMA max_page_count=%d", pages)).Scan(&limit); err != nil || limit != pages {
		t.Fatal("failed to constrain database", err)
	}
	if _, err := s.CreateProject(ctx, "too large", strings.Repeat("x", 1<<20), "plan_confirmation", "full-write"); err == nil {
		t.Fatal("capacity-limited write unexpectedly succeeded")
	}
	// SQLITE_FULL must not leave a goal, receipt, or event partially committed.
	ps, err := s.ListProjects(ctx)
	if err != nil || len(ps) != 1 || ps[0].ID != p.ID {
		t.Fatal("existing data lost or partial project retained", err)
	}
	var receipts int
	if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM request_receipts WHERE request_key='full-write'").Scan(&receipts); err != nil || receipts != 0 {
		t.Fatal("partial receipt after full disk", err)
	}
	if err = s.Health(ctx); err != nil {
		t.Fatal("capacity failure damaged database", err)
	}
}
