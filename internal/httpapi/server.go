// Package httpapi provides the authenticated, loopback-only Stage 0 API.
package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/koheismh/Asteriade/internal/core"
	"github.com/koheismh/Asteriade/internal/runner"
	"github.com/koheismh/Asteriade/internal/store"
	"github.com/koheismh/Asteriade/internal/webui"
)

const cookieName = "asteriade_session"

type session struct {
	csrf    string
	expires time.Time
}

type Server struct {
	store            *store.Store
	origin           string
	host             string
	mux              *http.ServeMux
	mu               sync.Mutex
	bootstrap        string
	bootstrapExpires time.Time
	sessions         map[string]session
}

// New accepts only an explicit IPv4 loopback origin; forwarded headers are ignored.
func New(st *store.Store, origin string) (*Server, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid local origin")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || host != "127.0.0.1" {
		return nil, errors.New("API must use 127.0.0.1")
	}
	pn, err := strconv.Atoi(port)
	if err != nil || pn < 1 || pn > 65535 {
		return nil, errors.New("invalid local port")
	}
	s := &Server{store: st, origin: origin, host: u.Host, mux: http.NewServeMux(), bootstrap: rand.Text(), bootstrapExpires: time.Now().Add(10 * time.Minute), sessions: make(map[string]session)}
	s.routes()
	return s, nil
}

// LaunchURL is for the private local browser handoff only, never application logs.
func (s *Server) LaunchURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.origin + "/#bootstrap=" + s.bootstrap
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'")
		if r.Host != s.host {
			failure(w, http.StatusForbidden, "invalid_host", "許可されていない接続先です。")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.origin {
			failure(w, http.StatusForbidden, "invalid_origin", "別の Origin からの操作は許可されていません。")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Origin") != s.origin {
			failure(w, http.StatusForbidden, "invalid_origin", "Origin の確認が必要です。")
			return
		}
		if r.URL.Path == "/api/v1/session" && r.Method == "POST" {
			s.exchange(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if r.URL.Path == "/api/v1/health" && r.Method == "GET" {
				respond(w, 200, map[string]any{"name": "Asteriade", "stage": "0", "alive": true})
				return
			}
			ss, ok := s.authenticate(r)
			if !ok {
				failure(w, 401, "unauthenticated", "起動ファイルから管理画面を開き直してください。")
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" {
				if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(ss.csrf)) != 1 {
					failure(w, 403, "invalid_csrf", "CSRF token が一致しません。")
					return
				}
				key := r.Header.Get("Idempotency-Key")
				if !validKey(key) {
					failure(w, 400, "invalid_request_key", "Idempotency-Key が必要です。")
					return
				}
			}
		}
		s.mux.ServeHTTP(w, r)
	})
}

func validKey(k string) bool {
	if len(k) == 0 || len(k) > 128 {
		return false
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func (s *Server) authenticate(r *http.Request) (session, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[c.Value]
	if ok && time.Now().Before(ss.expires) {
		return ss, true
	}
	delete(s.sessions, c.Value)
	return session{}, false
}

func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &body) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bootstrap == "" || time.Now().After(s.bootstrapExpires) || subtle.ConstantTimeCompare([]byte(body.Token), []byte(s.bootstrap)) != 1 {
		failure(w, 401, "invalid_bootstrap", "起動リンクは期限切れか使用済みです。アプリを再起動してください。")
		return
	}
	sid := rand.Text()
	ss := session{csrf: rand.Text(), expires: time.Now().Add(8 * time.Hour)}
	s.sessions[sid] = ss
	s.bootstrap = ""
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: sid, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 8 * 60 * 60})
	respond(w, 200, map[string]string{"csrf_token": ss.csrf})
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/session", func(w http.ResponseWriter, r *http.Request) {
		ss, _ := s.authenticate(r)
		respond(w, 200, map[string]string{"csrf_token": ss.csrf})
	})
	s.mux.HandleFunc("GET /api/v1/diagnostics", s.diagnostics)
	s.mux.HandleFunc("GET /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		p, e := s.store.ListProjects(r.Context())
		if e != nil {
			storeFailure(w, e)
			return
		}
		respond(w, 200, map[string]any{"projects": p})
	})
	s.mux.HandleFunc("POST /api/v1/projects", s.createProject)
	s.mux.HandleFunc("GET /api/v1/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, e := s.store.GetProject(r.Context(), r.PathValue("id"))
		if e != nil {
			storeFailure(w, e)
			return
		}
		respond(w, 200, p)
	})
	s.mux.HandleFunc("GET /api/v1/holds", func(w http.ResponseWriter, r *http.Request) {
		h, e := s.store.ListHolds(r.Context())
		if e != nil {
			storeFailure(w, e)
			return
		}
		respond(w, 200, map[string]any{"holds": h})
	})
	s.mux.HandleFunc("POST /api/v1/scheduler/pause", s.pause)
	s.mux.HandleFunc("POST /api/v1/holds/{id}/resume", s.resume)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/cancel", s.cancel)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/interrupt", s.interrupt)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.store.GetProject(r.Context(), r.PathValue("id")); err != nil {
			storeFailure(w, err)
			return
		}
		// No HTTP input, approval, environment variable, or JSON profile can enable it.
		_ = runner.CheckReady()
		failure(w, 409, "runner_unverified", "Runner の実効安全境界が未検証のため実行できません。Windows 検証が必要です。")
	})
	s.mux.HandleFunc("GET /api/v1/projects/{id}/events", s.events)
	s.mux.HandleFunc("GET /api/v1/approvals", func(w http.ResponseWriter, r *http.Request) {
		a, e := s.store.ListApprovals(r.Context())
		if e != nil {
			storeFailure(w, e)
			return
		}
		respond(w, 200, map[string]any{"approvals": a})
	})
	s.mux.HandleFunc("POST /api/v1/projects/{id}/approvals", s.requestApproval)
	s.mux.HandleFunc("POST /api/v1/approvals/{id}/decision", s.decideApproval)
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if path != "index.html" && path != "app.js" && path != "style.css" {
			http.NotFound(w, r)
			return
		}
		b, err := webui.Files.ReadFile(path)
		if err != nil {
			failure(w, 500, "asset_unavailable", "画面を読み込めません。")
			return
		}
		ct := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}
		w.Header().Set("Content-Type", ct[path])
		w.Write(b)
	})
}

func (s *Server) diagnostics(w http.ResponseWriter, r *http.Request) {
	ready := s.store.Health(r.Context()) == nil
	respond(w, 200, map[string]any{"name": "Asteriade", "stage": "0", "database_healthy": ready, "execution_enabled": false, "runner": runner.Describe(), "providers": []map[string]string{
		{"name": "Codex", "status": "unchecked", "reason": "認証・Windows sandbox・App Server 実経路は未検証です。"},
		{"name": "Bedrock", "status": "unsupported", "reason": "Stage 2 対象です。従量 API は無効です。"},
		{"name": "Kiro", "status": "unsupported", "reason": "利用条件を確認するまで自動連携しません。"},
	}})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Title string `json:"title"`
		Goal  string `json:"goal"`
		Mode  string `json:"execution_mode"`
	}
	if !decode(w, r, &b) {
		return
	}
	b.Title = strings.TrimSpace(b.Title)
	b.Goal = strings.TrimSpace(b.Goal)
	if b.Title == "" || len(b.Title) > 200 || b.Goal == "" || len(b.Goal) > 16000 {
		failure(w, 400, "invalid_project", "タイトルと目標を指定してください（200/16000 bytes 以内）。")
		return
	}
	if b.Mode == "" {
		b.Mode = "plan_confirmation"
	}
	p, e := s.store.CreateProject(r.Context(), b.Title, b.Goal, b.Mode, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 201, p)
}

func (s *Server) pause(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Scope     string `json:"scope"`
		ProjectID string `json:"project_id"`
	}
	if !decode(w, r, &b) {
		return
	}
	h, e := s.store.Pause(r.Context(), b.Scope, b.ProjectID, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 201, h)
}

func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Version int64 `json:"expected_version"`
	}
	if !decode(w, r, &b) {
		return
	}
	e := s.store.Resume(r.Context(), r.PathValue("id"), b.Version, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 200, map[string]bool{"resumed": true})
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Version int64 `json:"expected_version"`
	}
	if !decode(w, r, &b) {
		return
	}
	p, e := s.store.CancelProject(r.Context(), r.PathValue("id"), b.Version, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 200, p)
}

func (s *Server) interrupt(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Version int64 `json:"expected_version"`
	}
	if !decode(w, r, &b) {
		return
	}
	p, e := s.store.InterruptProject(r.Context(), r.PathValue("id"), b.Version, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 200, p)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.GetProject(r.Context(), r.PathValue("id")); err != nil {
		storeFailure(w, err)
		return
	}
	after := int64(0)
	var err error
	if q := r.URL.Query().Get("after"); q != "" {
		after, err = strconv.ParseInt(q, 10, 64)
	}
	if err != nil || after < 0 {
		failure(w, 400, "invalid_cursor", "event cursor が不正です。")
		return
	}
	es, e := s.store.Events(r.Context(), r.PathValue("id"), after)
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 200, map[string]any{"events": es})
}

func (s *Server) requestApproval(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Operation core.Operation `json:"operation"`
		Expires   time.Time      `json:"expires_at"`
	}
	if !decode(w, r, &b) {
		return
	}
	if time.Until(b.Expires) <= 0 || time.Until(b.Expires) > time.Hour {
		failure(w, 400, "invalid_expiry", "承認期限は現在から1時間以内にしてください。")
		return
	}
	a, e := s.store.RequestApproval(r.Context(), r.PathValue("id"), b.Operation, r.Header.Get("Idempotency-Key"), b.Expires)
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 201, a)
}

func (s *Server) decideApproval(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Version  int64  `json:"expected_version"`
		Hash     string `json:"operation_hash"`
		Decision string `json:"decision"`
	}
	if !decode(w, r, &b) {
		return
	}
	switch b.Decision {
	case "grant":
		b.Decision = "granted"
	case "deny":
		b.Decision = "denied"
	default:
		failure(w, 400, "invalid_decision", "grant または deny を指定してください。")
		return
	}
	a, e := s.store.DecideApproval(r.Context(), r.PathValue("id"), b.Version, b.Hash, b.Decision, r.Header.Get("Idempotency-Key"))
	if e != nil {
		storeFailure(w, e)
		return
	}
	respond(w, 200, a)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		failure(w, 415, "invalid_content_type", "application/json が必要です。")
		return false
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || uniqueJSONKeys(raw) != nil {
		failure(w, 400, "invalid_json", "JSON の重複項目・サイズ・形式が不正です。")
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		failure(w, 400, "invalid_json", "JSON の形式または項目が不正です。")
		return false
	}
	if e := d.Decode(new(any)); e != io.EOF {
		failure(w, 400, "invalid_json", "JSON は1文書だけ指定してください。")
		return false
	}
	return true
}

// Reject ambiguous documents before security-sensitive structs are decoded.
func uniqueJSONKeys(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	var value func(int) error
	value = func(depth int) error {
		if depth > 32 {
			return errors.New("JSON nesting limit")
		}
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'):
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return e
				}
				k, ok := key.(string)
				if !ok || seen[k] {
					return errors.New("duplicate or invalid key")
				}
				seen[k] = true
				if e = value(depth + 1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return errors.New("invalid object")
			}
		case json.Delim('['):
			for d.More() {
				if err = value(depth + 1); err != nil {
					return err
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return errors.New("invalid array")
			}
		case json.Delim('}'), json.Delim(']'):
			return errors.New("unexpected delimiter")
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func storeFailure(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, store.ErrNotFound):
		failure(w, 404, "not_found", "対象が見つかりません。")
	case errors.Is(e, store.ErrConflict):
		failure(w, 409, "conflict", "版または操作キーが一致しません。最新状態を読み直してください。")
	case errors.Is(e, store.ErrInvalid):
		failure(w, 400, "invalid_request", "操作の入力が不正です。")
	default:
		failure(w, 503, "storage_unavailable", "保存処理を完了できません。データを保全し、診断を確認してください。")
	}
}
func failure(w http.ResponseWriter, status int, code, msg string) {
	respond(w, status, map[string]string{"code": code, "error": msg})
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) String() string { return fmt.Sprintf("Asteriade Stage 0 (%s)", s.host) }
