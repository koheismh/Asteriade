package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koheismh/Asteriade/internal/core"
	"github.com/koheismh/Asteriade/internal/privatefs"
	_ "github.com/ncruces/go-sqlite3/driver"
)

var (
	ErrNotFound = errors.New("store: not found")
	ErrConflict = errors.New("store: conflict")
	ErrInvalid  = errors.New("store: invalid input")
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if path == "" || path == ":memory:" {
		return nil, fmt.Errorf("%w: a persistent database path is required", ErrInvalid)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	path = abs
	if err := privatefs.EnsureDirectory(filepath.Dir(path)); err != nil {
		return nil, err
	}
	// Create privately before SQLite opens it; never follow a pre-existing link.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		if err = f.Close(); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	if err = privatefs.CheckFile(path); err != nil {
		return nil, err
	}
	for _, sidecar := range []string{path + "-wal", path + "-shm"} {
		if _, err = os.Lstat(sidecar); err == nil {
			if err = privatefs.CheckFile(sidecar); err != nil {
				return nil, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	d, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	s := &Store{db: d}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var integrity string
	if err = d.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		d.Close()
		return nil, err
	}
	if integrity != "ok" {
		d.Close()
		return nil, fmt.Errorf("database integrity check: %s", integrity)
	}
	var schemaVersion int
	if err = d.QueryRowContext(ctx, "PRAGMA user_version").Scan(&schemaVersion); err != nil {
		d.Close()
		return nil, err
	}
	if schemaVersion > 1 {
		d.Close()
		return nil, fmt.Errorf("unsupported schema version %d", schemaVersion)
	}
	if schemaVersion == 1 {
		var recorded string
		if err = d.QueryRowContext(ctx, "SELECT script_hash FROM schema_migrations WHERE schema_version=1").Scan(&recorded); err != nil {
			d.Close()
			return nil, fmt.Errorf("migration ledger unavailable: %w", err)
		}
		if recorded != hash([]byte(strings.Join(schema, "\n"))) {
			d.Close()
			return nil, fmt.Errorf("schema migration checksum mismatch")
		}
	}
	if path != ":memory:" {
		if err = os.Chmod(path, 0600); err != nil {
			d.Close()
			return nil, err
		}
	}
	for _, q := range []string{"PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000", "PRAGMA synchronous=FULL", "PRAGMA locking_mode=EXCLUSIVE"} {
		if _, err = d.ExecContext(ctx, q); err != nil {
			d.Close()
			return nil, err
		}
	}
	var journalMode string
	if err = d.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil || !strings.EqualFold(journalMode, "wal") {
		d.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("WAL mode unavailable: %q", journalMode)
	}
	var foreignKeys int
	if err = d.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		d.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("SQLite foreign key enforcement unavailable")
	}
	var lockingMode string
	if err = d.QueryRowContext(ctx, "PRAGMA locking_mode").Scan(&lockingMode); err != nil || !strings.EqualFold(lockingMode, "exclusive") {
		d.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("exclusive database locking unavailable: %q", lockingMode)
	}
	if path != ":memory:" {
		for _, sidecar := range []string{path + "-wal", path + "-shm"} {
			if err = os.Chmod(sidecar, 0600); err != nil && !errors.Is(err, os.ErrNotExist) {
				d.Close()
				return nil, err
			}
		}
	}
	if err = s.migrate(ctx); err != nil {
		d.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Health(ctx context.Context) error {
	var result string
	if err := s.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("database integrity check: %s", result)
	}
	rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("database foreign key integrity check failed")
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}
func (s *Store) migrate(ctx context.Context) error {
	var version int
	err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}
	if version > 1 {
		return fmt.Errorf("unsupported schema version %d", version)
	}
	if version == 1 {
		var recorded string
		err = s.db.QueryRowContext(ctx, "SELECT script_hash FROM schema_migrations WHERE schema_version=1").Scan(&recorded)
		if err != nil {
			return fmt.Errorf("migration ledger unavailable: %w", err)
		}
		if recorded != hash([]byte(strings.Join(schema, "\n"))) {
			return fmt.Errorf("schema migration checksum mismatch")
		}
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range schema {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration: %w", err)
		}
	}
	all := strings.Join(schema, "\n")
	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(schema_version,script_hash,applied_at) VALUES(1,?,?)", hash([]byte(all)), now()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=1"); err != nil {
		return err
	}
	return tx.Commit()
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (schema_version INTEGER PRIMARY KEY, script_hash TEXT NOT NULL, applied_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY,title TEXT NOT NULL,goal TEXT NOT NULL,lifecycle_status TEXT NOT NULL CHECK(lifecycle_status IN ('draft','planning','running','needs_attention','completed','failed','cancelled')),execution_mode TEXT NOT NULL CHECK(execution_mode IN ('automatic','plan_confirmation')),limits_json TEXT NOT NULL,version INTEGER NOT NULL DEFAULT 1,created_at TEXT NOT NULL,updated_at TEXT NOT NULL,cancel_requested_at TEXT)`,
	`CREATE TABLE IF NOT EXISTS scheduler_holds (id TEXT PRIMARY KEY,hold_key TEXT NOT NULL UNIQUE,scope TEXT NOT NULL CHECK(scope IN ('global','project')),project_id TEXT REFERENCES projects(id),reason TEXT NOT NULL,actor TEXT NOT NULL,version INTEGER NOT NULL DEFAULT 1,created_at TEXT NOT NULL,released_at TEXT,released_by TEXT,CHECK((scope='global' AND project_id IS NULL) OR (scope='project' AND project_id IS NOT NULL)))`,
	`CREATE TABLE IF NOT EXISTS events (sequence INTEGER PRIMARY KEY AUTOINCREMENT,id TEXT NOT NULL UNIQUE,project_id TEXT REFERENCES projects(id),type TEXT NOT NULL,aggregate_id TEXT NOT NULL,aggregate_version INTEGER NOT NULL,payload TEXT NOT NULL,occurred_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS outbox (id TEXT PRIMARY KEY,event_id TEXT NOT NULL REFERENCES events(id),destination TEXT NOT NULL,message_key TEXT NOT NULL,state TEXT NOT NULL CHECK(state IN ('queued','sent','acked','uncertain')),attempts INTEGER NOT NULL DEFAULT 0,UNIQUE(destination,message_key))`,
	`CREATE TABLE IF NOT EXISTS request_receipts (scope TEXT NOT NULL,request_key TEXT NOT NULL,body_hash TEXT NOT NULL,result_ref TEXT,status TEXT NOT NULL,PRIMARY KEY(scope,request_key))`,
	`CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES projects(id),objective TEXT NOT NULL,status TEXT NOT NULL CHECK(status IN ('queued','ready','running','awaiting_approval','awaiting_answer','verifying','completed','failed','cancelled','recovery_required')),version INTEGER NOT NULL DEFAULT 1)`,
	`CREATE TABLE IF NOT EXISTS executions (id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES projects(id),task_id TEXT NOT NULL REFERENCES tasks(id),status TEXT NOT NULL CHECK(status IN ('starting','running','succeeded','failed','interrupt_requested','interrupted','outcome_unknown')),attempt INTEGER NOT NULL,epoch INTEGER NOT NULL DEFAULT 1,version INTEGER NOT NULL DEFAULT 1,interrupt_requested_at TEXT,grace_deadline_at TEXT)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS execution_one_live ON executions(task_id) WHERE status IN ('starting','running','interrupt_requested','outcome_unknown')`,
	`CREATE TABLE IF NOT EXISTS tool_operations (id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES projects(id),execution_id TEXT NOT NULL REFERENCES executions(id),operation_key TEXT NOT NULL UNIQUE,state TEXT NOT NULL CHECK(state IN ('prepared','authorized','dispatched','confirmed','denied','unknown')))`,
	`CREATE TABLE IF NOT EXISTS approvals (id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES projects(id),operation_hash TEXT NOT NULL,operation_json TEXT NOT NULL,scope TEXT NOT NULL CHECK(scope='once'),status TEXT NOT NULL CHECK(status IN ('pending','granted','denied','expired','revoked','consumed')),version INTEGER NOT NULL DEFAULT 1,expires_at TEXT NOT NULL,decision TEXT,decision_key TEXT,created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`,
	`CREATE TRIGGER execution_project_insert BEFORE INSERT ON executions WHEN NOT EXISTS (SELECT 1 FROM tasks WHERE id=NEW.task_id AND project_id=NEW.project_id) BEGIN SELECT RAISE(ABORT,'execution project mismatch'); END`,
	`CREATE TRIGGER execution_project_update BEFORE UPDATE OF project_id,task_id ON executions WHEN NOT EXISTS (SELECT 1 FROM tasks WHERE id=NEW.task_id AND project_id=NEW.project_id) BEGIN SELECT RAISE(ABORT,'execution project mismatch'); END`,
	`CREATE TRIGGER tool_project_insert BEFORE INSERT ON tool_operations WHEN NOT EXISTS (SELECT 1 FROM executions WHERE id=NEW.execution_id AND project_id=NEW.project_id) BEGIN SELECT RAISE(ABORT,'tool project mismatch'); END`,
	`CREATE TRIGGER tool_project_update BEFORE UPDATE OF project_id,execution_id ON tool_operations WHEN NOT EXISTS (SELECT 1 FROM executions WHERE id=NEW.execution_id AND project_id=NEW.project_id) BEGIN SELECT RAISE(ABORT,'tool project mismatch'); END`,
}

func now() string          { return time.Now().UTC().Format(time.RFC3339Nano) }
func id() string           { return core.NewID("st") }
func hash(b []byte) string { x := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(x[:]) }
func (s *Store) event(ctx context.Context, tx *sql.Tx, pid, typ, agg string, v int64, payload string) error {
	eventID := id()
	if _, e := tx.ExecContext(ctx, `INSERT INTO events(id,project_id,type,aggregate_id,aggregate_version,payload,occurred_at) VALUES(?,?,?,?,?,?,?)`, eventID, nullString(pid), typ, agg, v, payload, now()); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO outbox(id,event_id,destination,message_key,state) VALUES(?,?,'internal',?,'queued')`, id(), eventID, eventID)
	return e
}
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func (s *Store) receipt(ctx context.Context, tx *sql.Tx, scope, key string, body []byte) (string, bool, error) {
	if key == "" {
		return "", false, fmt.Errorf("%w: empty idempotency key", ErrInvalid)
	}
	h := hash(body)
	var oldHash string
	var result sql.NullString
	e := tx.QueryRowContext(ctx, "SELECT body_hash,result_ref FROM request_receipts WHERE scope=? AND request_key=?", scope, key).Scan(&oldHash, &result)
	if e == nil {
		if oldHash != h {
			return "", false, ErrConflict
		}
		return result.String, true, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return "", false, e
	}
	return h, false, nil
}
func (s *Store) saveReceipt(ctx context.Context, tx *sql.Tx, scope, key, h, result string) error {
	_, e := tx.ExecContext(ctx, "INSERT INTO request_receipts(scope,request_key,body_hash,result_ref,status) VALUES(?,?,?,?,'completed')", scope, key, h, result)
	return e
}

func (s *Store) CreateProject(ctx context.Context, title, goal, mode, key string) (core.Project, error) {
	title = strings.TrimSpace(title)
	if title == "" || strings.TrimSpace(goal) == "" || (mode != "automatic" && mode != "plan_confirmation") {
		return core.Project{}, ErrInvalid
	}
	body, _ := json.Marshal([]string{title, goal, mode})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return core.Project{}, e
	}
	defer tx.Rollback()
	h, seen, e := s.receipt(ctx, tx, "create_project", key, body)
	if e != nil {
		return core.Project{}, e
	}
	if seen {
		var p core.Project
		e = scanProject(tx.QueryRowContext(ctx, projectSelect+" WHERE id=?", h), &p)
		return p, e
	}
	p := core.Project{ID: id(), Title: title, Goal: goal, LifecycleStatus: "draft", Status: "draft", ExecutionMode: mode, Version: 1, Limits: core.DefaultLimits(), CreatedAt: time.Now().UTC()}
	lj, _ := json.Marshal(p.Limits)
	t := now()
	_, e = tx.ExecContext(ctx, `INSERT INTO projects(id,title,goal,lifecycle_status,execution_mode,limits_json,version,created_at,updated_at) VALUES(?,?,?,'draft',?,?,1,?,?)`, p.ID, title, goal, mode, string(lj), t, t)
	if e != nil {
		return p, e
	}
	if e = s.event(ctx, tx, p.ID, "ProjectCreated", p.ID, 1, "{}"); e != nil {
		return p, e
	}
	if e = s.saveReceipt(ctx, tx, "create_project", key, h, p.ID); e != nil {
		return p, e
	}
	return p, tx.Commit()
}

const projectSelect = `SELECT id,title,goal,lifecycle_status,execution_mode,limits_json,version,created_at,cancel_requested_at FROM projects`

type rowScan interface{ Scan(...any) error }

func scanProject(r rowScan, p *core.Project) error {
	var limits, created string
	var cancel sql.NullString
	if e := r.Scan(&p.ID, &p.Title, &p.Goal, &p.LifecycleStatus, &p.ExecutionMode, &limits, &p.Version, &created, &cancel); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return ErrNotFound
		}
		return e
	}
	p.Status = p.LifecycleStatus
	var e error
	if e = json.Unmarshal([]byte(limits), &p.Limits); e != nil {
		return e
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if cancel.Valid {
		x, _ := time.Parse(time.RFC3339Nano, cancel.String)
		p.CancelRequestedAt = &x
	}
	return nil
}
func (s *Store) GetProject(ctx context.Context, id string) (core.Project, error) {
	var p core.Project
	e := scanProject(s.db.QueryRowContext(ctx, projectSelect+" WHERE id=?", id), &p)
	if e != nil {
		return p, e
	}
	var n int
	if e := s.db.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_holds WHERE released_at IS NULL AND (scope='global' OR project_id=?)", id).Scan(&n); e != nil {
		return core.Project{}, e
	}
	if n > 0 && (p.LifecycleStatus == "planning" || p.LifecycleStatus == "running") {
		p.Status = "paused"
	}
	return p, nil
}
func (s *Store) ListProjects(ctx context.Context) ([]core.Project, error) {
	r, e := s.db.QueryContext(ctx, projectSelect+" ORDER BY created_at,id")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	ids := []string{}
	for r.Next() {
		var p core.Project
		if e = scanProject(r, &p); e != nil {
			return nil, e
		}
		ids = append(ids, p.ID)
	}
	if e = r.Err(); e != nil {
		r.Close()
		return nil, e
	}
	if e = r.Close(); e != nil {
		return nil, e
	}
	out := make([]core.Project, 0, len(ids))
	for _, projectID := range ids {
		p, err := s.GetProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) Pause(ctx context.Context, scope, pid, key string) (core.Hold, error) {
	if scope != "global" && scope != "project" {
		return core.Hold{}, ErrInvalid
	}
	if scope == "project" && pid == "" || scope == "global" && pid != "" {
		return core.Hold{}, ErrInvalid
	}
	body, _ := json.Marshal([]string{scope, pid})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return core.Hold{}, e
	}
	defer tx.Rollback()
	h, seen, e := s.receipt(ctx, tx, "pause:"+scope+":"+pid, key, body)
	if e != nil {
		return core.Hold{}, e
	}
	if seen {
		var out core.Hold
		e = scanHold(tx.QueryRowContext(ctx, holdSelect+" WHERE id=?", h), &out)
		return out, e
	}
	if scope == "project" {
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM projects WHERE id=?", pid).Scan(&n); e != nil {
			return core.Hold{}, e
		}
		if n == 0 {
			return core.Hold{}, ErrNotFound
		}
	}
	hold := core.Hold{ID: id(), Scope: scope, ProjectID: pid, Reason: "user_pause", Actor: "user", Version: 1}
	holdKey := scope + ":" + pid + ":" + key
	t := now()
	_, e = tx.ExecContext(ctx, `INSERT INTO scheduler_holds(id,hold_key,scope,project_id,reason,actor,created_at) VALUES(?,?,?,?,?,?,?)`, hold.ID, holdKey, scope, nullString(pid), hold.Reason, hold.Actor, t)
	if e != nil {
		return hold, e
	}
	if e = s.event(ctx, tx, pid, "SchedulerPaused", hold.ID, 1, "{}"); e != nil {
		return hold, e
	}
	if e = s.saveReceipt(ctx, tx, "pause:"+scope+":"+pid, key, h, hold.ID); e != nil {
		return hold, e
	}
	return hold, tx.Commit()
}

const holdSelect = `SELECT id,scope,coalesce(project_id,''),reason,actor,version,released_at FROM scheduler_holds`

func scanHold(r rowScan, h *core.Hold) error {
	var rel sql.NullString
	e := r.Scan(&h.ID, &h.Scope, &h.ProjectID, &h.Reason, &h.Actor, &h.Version, &rel)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	if rel.Valid {
		x, _ := time.Parse(time.RFC3339Nano, rel.String)
		h.ReleasedAt = &x
	}
	return nil
}
func (s *Store) Resume(ctx context.Context, id string, version int64, key string) error {
	body, _ := json.Marshal([]any{id, version})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	h, seen, e := s.receipt(ctx, tx, "resume:"+id, key, body)
	if e != nil {
		return e
	}
	if seen {
		return nil
	}
	var pid, reason, actor string
	var old int64
	var released sql.NullString
	e = tx.QueryRowContext(ctx, "SELECT coalesce(project_id,''),reason,actor,version,released_at FROM scheduler_holds WHERE id=?", id).Scan(&pid, &reason, &actor, &old, &released)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	if old != version {
		return ErrConflict
	}
	if reason != "user_pause" || actor != "user" {
		return ErrConflict
	}
	if !released.Valid {
		t := now()
		res, e := tx.ExecContext(ctx, "UPDATE scheduler_holds SET released_at=?,released_by='user',version=version+1 WHERE id=? AND version=?", t, id, version)
		if e != nil {
			return e
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return ErrConflict
		}
		if e = s.event(ctx, tx, pid, "SchedulerResumed", id, version+1, "{}"); e != nil {
			return e
		}
	}
	if e = s.saveReceipt(ctx, tx, "resume:"+id, key, h, id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) CancelProject(ctx context.Context, id string, version int64, key string) (core.Project, error) {
	body, _ := json.Marshal([]any{id, version})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return core.Project{}, e
	}
	defer tx.Rollback()
	h, seen, e := s.receipt(ctx, tx, "cancel:"+id, key, body)
	if e != nil {
		return core.Project{}, e
	}
	if seen {
		var p core.Project
		e = scanProject(tx.QueryRowContext(ctx, projectSelect+" WHERE id=?", h), &p)
		return p, e
	}
	var p core.Project
	if e = scanProject(tx.QueryRowContext(ctx, projectSelect+" WHERE id=?", id), &p); e != nil {
		return p, e
	}
	if p.Version != version {
		return p, ErrConflict
	}
	if p.LifecycleStatus == "completed" || p.LifecycleStatus == "cancelled" {
		return p, ErrConflict
	}
	var live int
	e = tx.QueryRowContext(ctx, "SELECT count(*) FROM executions WHERE project_id=? AND status IN ('starting','running','interrupt_requested','outcome_unknown')", id).Scan(&live)
	if e != nil {
		return p, e
	}
	t := now()
	life := "cancelled"
	if live > 0 {
		life = "needs_attention"
		if _, e = tx.ExecContext(ctx, "UPDATE executions SET status='outcome_unknown',version=version+1 WHERE project_id=? AND status IN ('starting','running','interrupt_requested')", id); e != nil {
			return p, e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE tasks SET status='recovery_required',version=version+1 WHERE project_id=? AND id IN (SELECT task_id FROM executions WHERE project_id=? AND status='outcome_unknown') AND status NOT IN ('completed','failed','cancelled')`, id, id); e != nil {
			return p, e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE tool_operations SET state='unknown' WHERE state='dispatched' AND execution_id IN (SELECT id FROM executions WHERE project_id=? AND status='outcome_unknown')`, id); e != nil {
			return p, e
		}
	}
	if _, e = tx.ExecContext(ctx, `UPDATE tasks SET status='cancelled',version=version+1 WHERE project_id=? AND status NOT IN ('completed','failed','cancelled','recovery_required') AND id NOT IN (SELECT task_id FROM executions WHERE project_id=? AND status='outcome_unknown')`, id, id); e != nil {
		return p, e
	}
	res, e := tx.ExecContext(ctx, "UPDATE projects SET lifecycle_status=?,cancel_requested_at=?,version=version+1,updated_at=? WHERE id=? AND version=?", life, t, t, id, version)
	if e != nil {
		return p, e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return p, ErrConflict
	}
	p.LifecycleStatus = life
	p.Status = life
	p.CancelRequestedAt = &time.Time{}
	*p.CancelRequestedAt, _ = time.Parse(time.RFC3339Nano, t)
	p.Version++
	if _, e = tx.ExecContext(ctx, "UPDATE approvals SET status='revoked',version=version+1,updated_at=? WHERE project_id=? AND status IN ('pending','granted')", t, id); e != nil {
		return p, e
	}
	if e = s.event(ctx, tx, id, "ProjectCancelRequested", id, p.Version, fmt.Sprintf(`{"recovery_required":%t}`, live > 0)); e != nil {
		return p, e
	}
	if e = s.saveReceipt(ctx, tx, "cancel:"+id, key, h, id); e != nil {
		return p, e
	}
	return p, tx.Commit()
}
func (s *Store) Events(ctx context.Context, pid string, after int64) ([]core.Event, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,sequence,coalesce(project_id,''),type,aggregate_id,aggregate_version,payload,occurred_at FROM events WHERE project_id=? AND sequence>? ORDER BY sequence`, pid, after)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []core.Event{}
	for rows.Next() {
		var x core.Event
		var ts string
		if e = rows.Scan(&x.ID, &x.Sequence, &x.ProjectID, &x.Type, &x.AggregateID, &x.AggregateVersion, &x.Payload, &ts); e != nil {
			return nil, e
		}
		x.OccurredAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) Recover(ctx context.Context) (int, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	projects, e := tx.QueryContext(ctx, `SELECT DISTINCT p.id,p.lifecycle_status,p.version FROM projects p JOIN executions x ON x.project_id=p.id WHERE x.status IN ('starting','running','interrupt_requested','outcome_unknown')`)
	if e != nil {
		return 0, e
	}
	type projectRecovery struct {
		id, status string
		version    int64
	}
	projectRows := []projectRecovery{}
	for projects.Next() {
		var p projectRecovery
		if e = projects.Scan(&p.id, &p.status, &p.version); e != nil {
			projects.Close()
			return 0, e
		}
		projectRows = append(projectRows, p)
	}
	if e = projects.Err(); e != nil {
		projects.Close()
		return 0, e
	}
	projects.Close()
	for _, p := range projectRows {
		if p.status != "needs_attention" && p.status != "completed" && p.status != "cancelled" && p.status != "failed" {
			res, updateErr := tx.ExecContext(ctx, "UPDATE projects SET lifecycle_status='needs_attention',version=version+1,updated_at=? WHERE id=? AND version=?", now(), p.id, p.version)
			if updateErr != nil {
				return 0, updateErr
			}
			n, _ := res.RowsAffected()
			if n != 1 {
				return 0, ErrConflict
			}
			if e = s.event(ctx, tx, p.id, "ProjectRecoveryRequired", p.id, p.version+1, `{"reason":"execution_outcome_unknown"}`); e != nil {
				return 0, e
			}
		}
	}
	rows, e := tx.QueryContext(ctx, "SELECT id,project_id,task_id,version FROM executions WHERE status IN ('starting','running','interrupt_requested')")
	if e != nil {
		return 0, e
	}
	type ex struct {
		id, p, t string
		version  int64
	}
	all := []ex{}
	for rows.Next() {
		var x ex
		if e = rows.Scan(&x.id, &x.p, &x.t, &x.version); e != nil {
			rows.Close()
			return 0, e
		}
		all = append(all, x)
	}
	rows.Close()
	for _, x := range all {
		if _, e = tx.ExecContext(ctx, "UPDATE executions SET status='outcome_unknown',version=version+1 WHERE id=?", x.id); e != nil {
			return 0, e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE tasks SET status='recovery_required',version=version+1 WHERE id=? AND status NOT IN ('completed','failed','cancelled')", x.t); e != nil {
			return 0, e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE tool_operations SET state='unknown' WHERE execution_id=? AND state='dispatched'", x.id); e != nil {
			return 0, e
		}
		if e = s.event(ctx, tx, x.p, "RecoveryRequired", x.id, x.version+1, `{"execution_status":"outcome_unknown"}`); e != nil {
			return 0, e
		}
	}
	unknownTasks, e := tx.QueryContext(ctx, `SELECT x.project_id,t.id,t.version,x.id FROM executions x JOIN tasks t ON t.id=x.task_id WHERE x.status='outcome_unknown' AND t.status NOT IN ('completed','failed','cancelled','recovery_required')`)
	if e != nil {
		return 0, e
	}
	type recoverTask struct {
		projectID, taskID, executionID string
		version                        int64
	}
	recoveryTasks := []recoverTask{}
	for unknownTasks.Next() {
		var x recoverTask
		if e = unknownTasks.Scan(&x.projectID, &x.taskID, &x.version, &x.executionID); e != nil {
			unknownTasks.Close()
			return 0, e
		}
		recoveryTasks = append(recoveryTasks, x)
	}
	if e = unknownTasks.Err(); e != nil {
		unknownTasks.Close()
		return 0, e
	}
	unknownTasks.Close()
	for _, x := range recoveryTasks {
		if _, e = tx.ExecContext(ctx, "UPDATE tasks SET status='recovery_required',version=version+1 WHERE id=? AND version=?", x.taskID, x.version); e != nil {
			return 0, e
		}
		if e = s.event(ctx, tx, x.projectID, "RecoveryRequired", x.taskID, x.version+1, fmt.Sprintf(`{"execution_id":%q,"task_status":"recovery_required"}`, x.executionID)); e != nil {
			return 0, e
		}
	}
	if _, e = tx.ExecContext(ctx, `UPDATE tool_operations SET state='unknown' WHERE state='dispatched' AND execution_id IN (SELECT id FROM executions WHERE status='outcome_unknown')`); e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return len(all), nil
}
func (s *Store) RequestApproval(ctx context.Context, pid string, op core.Operation, key string, expires time.Time) (core.Approval, error) {
	if expires.IsZero() || !expires.After(time.Now()) {
		return core.Approval{}, ErrInvalid
	}
	if e := core.ValidateOperation(op); e != nil {
		return core.Approval{}, fmt.Errorf("%w: %v", ErrInvalid, e)
	}
	oh, e := core.OperationHash(op)
	if e != nil {
		return core.Approval{}, e
	}
	raw, _ := json.Marshal(op)
	body, _ := json.Marshal(struct {
		Project   string
		Operation core.Operation
		Expires   int64
	}{pid, op, expires.UTC().UnixNano()})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return core.Approval{}, e
	}
	defer tx.Rollback()
	rh, seen, e := s.receipt(ctx, tx, "approval:"+pid, key, body)
	if e != nil {
		return core.Approval{}, e
	}
	if seen {
		var a core.Approval
		e = scanApproval(tx.QueryRowContext(ctx, approvalSelect+" WHERE id=?", rh), &a)
		return a, e
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM projects WHERE id=?", pid).Scan(&n); e != nil {
		return core.Approval{}, e
	}
	if n == 0 {
		return core.Approval{}, ErrNotFound
	}
	a := core.Approval{ID: id(), ProjectID: pid, OperationHash: oh, Operation: op, Scope: "once", Status: "pending", Version: 1, ExpiresAt: expires.UTC()}
	t := now()
	_, e = tx.ExecContext(ctx, `INSERT INTO approvals(id,project_id,operation_hash,operation_json,scope,status,version,expires_at,created_at,updated_at) VALUES(?,?,?,?,?,'pending',1,?,?,?)`, a.ID, pid, oh, string(raw), a.Scope, a.ExpiresAt.Format(time.RFC3339Nano), t, t)
	if e != nil {
		return a, e
	}
	if e = s.event(ctx, tx, pid, "ApprovalRequested", a.ID, 1, `{"operation_hash":"`+oh+`"}`); e != nil {
		return a, e
	}
	if e = s.saveReceipt(ctx, tx, "approval:"+pid, key, rh, a.ID); e != nil {
		return a, e
	}
	return a, tx.Commit()
}

const approvalSelect = `SELECT id,project_id,operation_hash,operation_json,scope,status,version,expires_at FROM approvals`

func scanApproval(r rowScan, a *core.Approval) error {
	var raw, expires string
	e := r.Scan(&a.ID, &a.ProjectID, &a.OperationHash, &raw, &a.Scope, &a.Status, &a.Version, &expires)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	if e != nil {
		return e
	}
	if e = json.Unmarshal([]byte(raw), &a.Operation); e != nil {
		return e
	}
	a.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return nil
}
func (s *Store) DecideApproval(ctx context.Context, id string, version int64, hash, decision, key string) (core.Approval, error) {
	if decision != "granted" && decision != "denied" {
		return core.Approval{}, ErrInvalid
	}
	body, _ := json.Marshal([]any{id, version, hash, decision})
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return core.Approval{}, e
	}
	defer tx.Rollback()
	rh, seen, e := s.receipt(ctx, tx, "decide_approval:"+id, key, body)
	if e != nil {
		return core.Approval{}, e
	}
	if seen {
		var a core.Approval
		e = scanApproval(tx.QueryRowContext(ctx, approvalSelect+" WHERE id=?", rh), &a)
		if e == nil && a.Status == "expired" {
			return a, ErrConflict
		}
		return a, e
	}
	var a core.Approval
	if e = scanApproval(tx.QueryRowContext(ctx, approvalSelect+" WHERE id=?", id), &a); e != nil {
		return a, e
	}
	if a.Version != version || a.OperationHash != hash {
		return a, ErrConflict
	}
	if a.Status != "pending" {
		return a, ErrConflict
	}
	if !a.ExpiresAt.After(time.Now()) {
		t := now()
		if _, e = tx.ExecContext(ctx, "UPDATE approvals SET status='expired',version=version+1,updated_at=? WHERE id=? AND version=?", t, id, version); e != nil {
			return a, e
		}
		a.Status = "expired"
		a.Version++
		if e = s.event(ctx, tx, a.ProjectID, "ApprovalExpired", id, a.Version, `{}`); e != nil {
			return a, e
		}
		if e = s.saveReceipt(ctx, tx, "decide_approval:"+id, key, rh, id); e != nil {
			return a, e
		}
		if e = tx.Commit(); e != nil {
			return a, e
		}
		return a, ErrConflict
	}
	t := now()
	res, e := tx.ExecContext(ctx, "UPDATE approvals SET status=?,decision=?,decision_key=?,version=version+1,updated_at=? WHERE id=? AND version=? AND status='pending'", decision, decision, key, t, id, version)
	if e != nil {
		return a, e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return a, ErrConflict
	}
	a.Status = decision
	a.Version++
	if e = s.event(ctx, tx, a.ProjectID, "ApprovalDecided", id, a.Version, `{"decision":"`+decision+`"}`); e != nil {
		return a, e
	}
	if e = s.saveReceipt(ctx, tx, "decide_approval:"+id, key, rh, id); e != nil {
		return a, e
	}
	return a, tx.Commit()
}
func (s *Store) ListHolds(ctx context.Context) ([]core.Hold, error) {
	r, e := s.db.QueryContext(ctx, holdSelect+" ORDER BY created_at,id")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []core.Hold{}
	for r.Next() {
		var h core.Hold
		if e = scanHold(r, &h); e != nil {
			return nil, e
		}
		out = append(out, h)
	}
	return out, r.Err()
}
func (s *Store) ListApprovals(ctx context.Context) ([]core.Approval, error) {
	r, e := s.db.QueryContext(ctx, approvalSelect+" ORDER BY created_at,id")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []core.Approval{}
	for r.Next() {
		var a core.Approval
		if e = scanApproval(r, &a); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, r.Err()
}
func (s *Store) Backup(ctx context.Context, dest string) error {
	if dest == "" || dest == ":memory:" {
		return ErrInvalid
	}
	if _, e := os.Stat(dest); e == nil {
		return ErrConflict
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if e := privatefs.EnsureDirectory(filepath.Dir(dest)); e != nil {
		return e
	}
	abs, e := filepath.Abs(dest)
	if e != nil {
		return e
	}
	if strings.ContainsRune(abs, '\x00') {
		return ErrInvalid
	}
	tmp := abs + ".backup-" + id()
	defer os.Remove(tmp)
	if _, e = s.db.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(tmp, "'", "''")+"'"); e != nil {
		return e
	}
	if e = os.Chmod(tmp, 0600); e != nil {
		return e
	}
	if e = privatefs.CheckFile(tmp); e != nil {
		return e
	}
	d, e := sql.Open("sqlite3", tmp)
	if e != nil {
		return e
	}
	defer d.Close()
	var result string
	if e = d.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); e != nil {
		return e
	}
	if result != "ok" {
		return fmt.Errorf("backup integrity check: %s", result)
	}
	rows, e := d.QueryContext(ctx, "PRAGMA foreign_key_check")
	if e != nil {
		return e
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("backup foreign key check failed")
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if e = rows.Close(); e != nil {
		return e
	}
	if e = d.Close(); e != nil {
		return e
	}
	if _, e = os.Stat(abs); e == nil {
		return ErrConflict
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if e = os.Link(tmp, abs); e != nil {
		return e
	}
	return nil
}

// InterruptProject records a request to stop live executions. It never claims
// that a worker has stopped; resolution is left to the runner/recovery path.
func (s *Store) InterruptProject(ctx context.Context, projectID string, expectedVersion int64, key string) (core.Project, error) {
	body, _ := json.Marshal([]any{projectID, expectedVersion})
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return core.Project{}, err
	}
	defer tx.Rollback()
	rh, seen, err := s.receipt(ctx, tx, "interrupt:"+projectID, key, body)
	if err != nil {
		return core.Project{}, err
	}
	if seen {
		var p core.Project
		err = scanProject(tx.QueryRowContext(ctx, projectSelect+" WHERE id=?", rh), &p)
		return p, err
	}
	var p core.Project
	if err = scanProject(tx.QueryRowContext(ctx, projectSelect+" WHERE id=?", projectID), &p); err != nil {
		return p, err
	}
	if p.Version != expectedVersion {
		return p, ErrConflict
	}
	if p.LifecycleStatus == "cancelled" || p.LifecycleStatus == "completed" || p.LifecycleStatus == "failed" {
		return p, ErrConflict
	}
	var live int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM executions WHERE project_id=? AND status IN ('starting','running','interrupt_requested')", projectID).Scan(&live); err != nil {
		return p, err
	}
	if live == 0 {
		return p, ErrConflict
	}
	t := now()
	res, err := tx.ExecContext(ctx, `UPDATE executions SET status='interrupt_requested',interrupt_requested_at=coalesce(interrupt_requested_at,?),grace_deadline_at=coalesce(grace_deadline_at,?),version=version+1 WHERE project_id=? AND status IN ('starting','running')`, t, time.Now().UTC().Add(30*time.Second).Format(time.RFC3339Nano), projectID)
	if err != nil {
		return p, err
	}
	n, _ := res.RowsAffected()
	if n == 0 { // repeated requests are accepted only if an interrupt is already pending.
		var pending int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM executions WHERE project_id=? AND status='interrupt_requested'", projectID).Scan(&pending); err != nil {
			return p, err
		}
		if pending == 0 {
			return p, ErrConflict
		}
	}
	p.Version++
	if _, err = tx.ExecContext(ctx, "UPDATE projects SET version=version+1,updated_at=? WHERE id=? AND version=?", t, projectID, expectedVersion); err != nil {
		return p, err
	}
	if err = s.event(ctx, tx, projectID, "ProjectInterruptRequested", projectID, p.Version, `{"live_executions":true}`); err != nil {
		return p, err
	}
	if err = s.saveReceipt(ctx, tx, "interrupt:"+projectID, key, rh, projectID); err != nil {
		return p, err
	}
	return p, tx.Commit()
}
