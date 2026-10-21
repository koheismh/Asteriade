package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/koheismh/Asteriade/internal/core"
	"github.com/koheismh/Asteriade/internal/store"
)

type fixture struct {
	s         *Server
	st        *store.Store
	cookie    *http.Cookie
	csrf      string
	bootstrap string
}

const testOrigin = "http://127.0.0.1:19874"

func setup(t *testing.T) *fixture {
	t.Helper()
	st, e := store.Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { st.Close() })
	s, e := New(st, testOrigin)
	if e != nil {
		t.Fatal(e)
	}
	f := &fixture{s: s, st: st}
	u, _ := url.Parse(s.LaunchURL())
	token := strings.TrimPrefix(u.Fragment, "bootstrap=")
	f.bootstrap = token
	w := f.req("POST", "/api/v1/session", fmt.Sprintf(`{"token":%q}`, token), nil)
	if w.Code != 200 {
		t.Fatalf("bootstrap status %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			f.cookie = c
		}
	}
	if f.cookie == nil || !f.cookie.HttpOnly || f.cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("session cookie protection missing")
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	f.csrf = body["csrf_token"]
	if f.csrf == "" {
		t.Fatal("missing CSRF token")
	}
	return f
}

func (f *fixture) req(method, path, body string, change func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, testOrigin+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", testOrigin)
	r.Header.Set("Idempotency-Key", "test-key")
	r.Header.Set("X-CSRF-Token", f.csrf)
	if f.cookie != nil {
		r.AddCookie(f.cookie)
	}
	if change != nil {
		change(r)
	}
	w := httptest.NewRecorder()
	f.s.Handler().ServeHTTP(w, r)
	return w
}

func TestBootstrapSingleUseAndBoundedLocalOrigin(t *testing.T) {
	f := setup(t)
	if _, err := New(f.st, "http://0.0.0.0:8000"); err == nil {
		t.Fatal("accepted public bind")
	}
	for _, origin := range []string{"https://127.0.0.1:8000", "http://localhost:8000", "http://127.0.0.1:8000/path", "http://user@127.0.0.1:8000"} {
		if _, err := New(f.st, origin); err == nil {
			t.Errorf("accepted %s", origin)
		}
	}
	// The exchange already consumed this server's only bootstrap credential.
	if w := f.req("POST", "/api/v1/session", fmt.Sprintf(`{"token":%q}`, f.bootstrap), nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := f.req("GET", "/", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Asteriade") {
		t.Fatal("embedded UI unavailable")
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("security headers missing")
	}
	if w := f.req("GET", "/app.js", "", nil); w.Code != 200 {
		t.Fatal("embedded JS unavailable")
	}
	if w := f.req("GET", "/api/v1/diagnostics", "", func(r *http.Request) { r.Header.Del("Cookie") }); w.Code != 401 {
		t.Fatalf("unauthenticated diagnostics: %d", w.Code)
	}
}

func TestHostOriginCSRFAndRequestKeyRejectBeforeMutation(t *testing.T) {
	f := setup(t)
	body := `{"title":"Asteriade","goal":"保存テスト"}`
	cases := []struct {
		name   string
		change func(*http.Request)
		want   int
	}{
		{"host", func(r *http.Request) { r.Host = "attacker.example" }, 403},
		{"origin", func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example") }, 403},
		{"missing origin", func(r *http.Request) { r.Header.Del("Origin") }, 403},
		{"csrf", func(r *http.Request) { r.Header.Set("X-CSRF-Token", "wrong") }, 403},
		{"missing key", func(r *http.Request) { r.Header.Del("Idempotency-Key") }, 400},
		{"session", func(r *http.Request) { r.Header.Del("Cookie") }, 401},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := f.req("POST", "/api/v1/projects", body, tc.change)
			if w.Code != tc.want {
				t.Fatalf("got %d want %d", w.Code, tc.want)
			}
		})
	}
	ps, e := f.st.ListProjects(context.Background())
	if e != nil || len(ps) != 0 {
		t.Fatal("rejected requests changed storage", e)
	}
}

func TestProjectHoldCancelAndIdempotentMutation(t *testing.T) {
	f := setup(t)
	body := `{"title":"Asteriade","goal":"Stage 0 の保存確認"}`
	w := f.req("POST", "/api/v1/projects", body, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var p core.Project
	json.Unmarshal(w.Body.Bytes(), &p)
	w = f.req("POST", "/api/v1/projects", body, nil)
	var replay core.Project
	json.Unmarshal(w.Body.Bytes(), &replay)
	if w.Code != 201 || replay.ID != p.ID {
		t.Fatal("retry duplicated project")
	}
	if w = f.req("POST", "/api/v1/projects", `{"title":"changed","goal":"different"}`, nil); w.Code != 409 {
		t.Fatal("changed payload accepted with old key")
	}
	w = f.req("GET", "/api/v1/projects", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var list struct {
		Projects []core.Project `json:"projects"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Projects) != 1 {
		t.Fatal("list mismatch")
	}
	w = f.req("POST", "/api/v1/scheduler/pause", `{"scope":"global"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = f.req("POST", "/api/v1/scheduler/pause", fmt.Sprintf(`{"scope":"project","project_id":%q}`, p.ID), nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var h core.Hold
	json.Unmarshal(w.Body.Bytes(), &h)
	w = f.req("POST", "/api/v1/holds/"+h.ID+"/resume", fmt.Sprintf(`{"expected_version":%d}`, h.Version), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	hs, e := f.st.ListHolds(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	active := 0
	for _, h := range hs {
		if h.ReleasedAt == nil {
			active++
			if h.Scope != "global" {
				t.Fatal("wrong hold survived")
			}
		}
	}
	if active != 1 {
		t.Fatal("project resume released global hold")
	}
	w = f.req("POST", "/api/v1/projects/"+p.ID+"/cancel", fmt.Sprintf(`{"expected_version":%d}`, p.Version), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var cancelled core.Project
	json.Unmarshal(w.Body.Bytes(), &cancelled)
	if cancelled.LifecycleStatus != "cancelled" {
		t.Fatal("cancel not persisted")
	}
	w = f.req("GET", "/api/v1/projects/"+p.ID+"/events?after=0", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "ProjectCancelRequested") {
		t.Fatal("missing audit event")
	}
}

func TestJSONValidationAndApprovalCannotEnableExecution(t *testing.T) {
	f := setup(t)
	for _, body := range []string{`{"title":"ok","goal":"ok","runner_verified":true}`, `{"title":"ok","goal":"ok"} {}`, `{"title":"ok","title":"changed","goal":"ok"}`, strings.Repeat(" ", 65537) + `{}`} {
		w := f.req("POST", "/api/v1/projects", body, nil)
		if w.Code != 400 {
			t.Fatalf("bad JSON accepted: %d", w.Code)
		}
	}
	w := f.req("POST", "/api/v1/projects", `{"title":"approval","goal":"外部操作は実施しない"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var p core.Project
	json.Unmarshal(w.Body.Bytes(), &p)
	op := core.Operation{Kind: "external.send", Destination: "https://example.org/api", Account: "test", Target: "test", PayloadHash: strings.Repeat("a", 64), PolicyRevision: 1}
	a, e := f.st.RequestApproval(context.Background(), p.ID, op, "request-approval", time.Now().Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	w = f.req("POST", "/api/v1/approvals/"+a.ID+"/decision", fmt.Sprintf(`{"expected_version":%d,"operation_hash":%q,"decision":"grant"}`, a.Version, a.OperationHash), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = f.req("POST", "/api/v1/projects/"+p.ID+"/start", `{"runner_verified":true}`, nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "runner_unverified") {
		t.Fatal("approval enabled runner")
	}
	w = f.req("GET", "/api/v1/diagnostics", "", nil)
	var d struct {
		ExecutionEnabled bool               `json:"execution_enabled"`
		Runner           core.RunnerProfile `json:"runner"`
	}
	json.Unmarshal(w.Body.Bytes(), &d)
	if w.Code != 200 || d.ExecutionEnabled || d.Runner.Status != "unverified" {
		t.Fatal("false ready diagnosis")
	}
	for _, cap := range d.Runner.Capabilities {
		if cap {
			t.Fatal("unverified capability enabled")
		}
	}
}

func TestConcurrentBootstrapHasExactlyOneWinner(t *testing.T) {
	f := setup(t)
	second, err := New(f.st, testOrigin)
	if err != nil {
		t.Fatal(err)
	}
	f.s = second
	f.cookie = nil
	f.csrf = ""
	u, _ := url.Parse(second.LaunchURL())
	token := strings.TrimPrefix(u.Fragment, "bootstrap=")
	var wg sync.WaitGroup
	codes := make(chan int, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- f.req("POST", "/api/v1/session", fmt.Sprintf(`{"token":%q}`, token), nil).Code
		}()
	}
	wg.Wait()
	close(codes)
	wins := 0
	for code := range codes {
		if code == 200 {
			wins++
		} else if code != 401 {
			t.Fatalf("unexpected status %d", code)
		}
	}
	if wins != 1 {
		t.Fatalf("bootstrap exchanges succeeded: %d", wins)
	}
}
