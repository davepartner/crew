package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/httpapi"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/store"
)

// fakePaytrackify stands in for paytrackify.com so the whole sign-in and work
// loop can be tested without email or a second server.
type fakePaytrackify struct {
	mu     sync.Mutex
	nextID int
	emails map[string]string // requestId -> email
	tokens map[string]string // token -> email
	names  map[string]string // email -> display name
}

func newFakePaytrackify() *fakePaytrackify {
	return &fakePaytrackify{
		emails: map[string]string{},
		tokens: map[string]string{},
		names:  map[string]string{},
	}
}

func (f *fakePaytrackify) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/otp/request", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email string `json:"email"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		f.mu.Lock()
		f.nextID++
		requestID := fmt.Sprintf("req-%d", f.nextID)
		f.emails[requestID] = body.Email
		if _, ok := f.names[body.Email]; !ok {
			f.names[body.Email] = "Jane Doe"
		}
		f.mu.Unlock()

		writeTestJSON(w, 200, map[string]any{
			"ok": true, "requestId": requestID, "resend_after": 30,
		})
	})

	mux.HandleFunc("POST /api/auth/otp/verify", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RequestID string `json:"requestId"`
			Code      string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body.Code != "12345678" {
			writeTestJSON(w, 422, map[string]any{"ok": false, "error": "That code is not right."})
			return
		}

		f.mu.Lock()
		email := f.emails[body.RequestID]
		token := "tok-" + email
		f.tokens[token] = email
		name := f.names[email]
		f.mu.Unlock()

		if email == "" {
			writeTestJSON(w, 422, map[string]any{"ok": false, "error": "That code is not right."})
			return
		}

		writeTestJSON(w, 200, map[string]any{
			"ok":          true,
			"token":       token,
			"is_new_user": true,
			"user": map[string]any{
				"id": email, "email": email, "name": name, "email_verified": true,
			},
		})
	})

	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

		f.mu.Lock()
		email, ok := f.tokens[token]
		name := f.names[email]
		f.mu.Unlock()

		if !ok {
			writeTestJSON(w, 401, map[string]any{"ok": false, "error": "Please sign in again."})
			return
		}

		writeTestJSON(w, 200, map[string]any{
			"ok":   true,
			"user": map[string]any{"id": email, "email": email, "name": name, "email_verified": true},
		})
	})

	mux.HandleFunc("PATCH /api/auth/profile", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		f.mu.Lock()
		email := f.tokens[token]
		f.names[email] = body.Name
		f.mu.Unlock()

		writeTestJSON(w, 200, map[string]any{
			"ok":   true,
			"user": map[string]any{"id": email, "email": email, "name": body.Name, "email_verified": true},
		})
	})

	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		f.mu.Lock()
		delete(f.tokens, token)
		f.mu.Unlock()
		writeTestJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("GET /api/auth/sessions", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, 200, map[string]any{"ok": true, "sessions": []map[string]any{
			{"id": "s1", "device": "Mac", "ip_address": "127.0.0.1", "last_used_at": time.Now().UTC().Format(time.RFC3339)},
		}})
	})

	mux.HandleFunc("DELETE /api/auth/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, 200, map[string]any{"ok": true})
	})

	return mux
}

func writeTestJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type harness struct {
	base string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	upstream := httptest.NewServer(newFakePaytrackify().handler())
	t.Cleanup(upstream.Close)

	st, err := store.Open(filepath.Join(t.TempDir(), "crew.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := config.Config{
		Addr:            "127.0.0.1:0",
		DBPath:          "test",
		PaytrackifyURL:  upstream.URL,
		PaytrackifyTime: 5 * time.Second,
		SessionCacheTTL: time.Minute,
		Env:             "test",
		ServiceName:     "Crew",
	}

	server := httpapi.New(cfg, st, payauth.NewClient(cfg.PaytrackifyURL, cfg.PaytrackifyTime))
	base := httptest.NewServer(server.Handler())
	t.Cleanup(base.Close)

	return &harness{base: base.URL}
}

type browser struct {
	h    *harness
	http *http.Client
	jar  *cookiejar.Jar
}

func (h *harness) newBrowser(t *testing.T) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("jar: %v", err)
	}
	return &browser{h: h, jar: jar, http: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
}

func (b *browser) baseURL(t *testing.T) *url.URL {
	t.Helper()
	parsed, err := url.Parse(b.h.base)
	if err != nil {
		t.Fatalf("parse base: %v", err)
	}
	return parsed
}

func (b *browser) csrf(t *testing.T) string {
	t.Helper()

	resp, err := b.http.Get(b.h.base + "/api/csrf")
	if err != nil {
		t.Fatalf("csrf: %v", err)
	}
	resp.Body.Close()

	for _, cookie := range b.jar.Cookies(b.baseURL(t)) {
		if cookie.Name == "crew_csrf" {
			return cookie.Value
		}
	}

	t.Fatal("no csrf cookie")
	return ""
}

func (b *browser) do(t *testing.T, method, path string, body any, out any) (int, string) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, b.h.base+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set("X-CSRF-Token", b.csrf(t))
	}

	resp, err := b.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode %s %s: %v (%s)", method, path, err, raw)
		}
	}

	return resp.StatusCode, string(raw)
}

func (b *browser) login(t *testing.T, email string) {
	t.Helper()

	var request struct {
		RequestID string `json:"requestId"`
	}
	status, raw := b.do(t, http.MethodPost, "/api/auth/request", map[string]string{"email": email}, &request)
	if status != 200 {
		t.Fatalf("request code: %d %s", status, raw)
	}

	status, raw = b.do(t, http.MethodPost, "/api/auth/verify",
		map[string]string{"requestId": request.RequestID, "code": "12345678"}, nil)
	if status != 200 {
		t.Fatalf("verify: %d %s", status, raw)
	}
}

func TestLoginCreatesAWorkspaceAndProtectsIt(t *testing.T) {
	h := newHarness(t)
	b := h.newBrowser(t)

	status, _ := b.do(t, http.MethodGet, "/api/tasks", nil, nil)
	if status != 401 {
		t.Fatalf("expected 401 before login, got %d", status)
	}

	b.login(t, "jane@example.com")

	var me struct {
		User map[string]any `json:"user"`
	}
	status, raw := b.do(t, http.MethodGet, "/api/me", nil, &me)
	if status != 200 {
		t.Fatalf("me: %d %s", status, raw)
	}
	if me.User["email"] != "jane@example.com" {
		t.Fatalf("wrong user: %v", me.User)
	}

	var dashboard struct {
		Name string `json:"name"`
	}
	status, raw = b.do(t, http.MethodGet, "/api/dashboard", nil, &dashboard)
	if status != 200 {
		t.Fatalf("dashboard: %d %s", status, raw)
	}
	if dashboard.Name != "Jane Doe" {
		t.Fatalf("expected name from provider, got %q", dashboard.Name)
	}
}

func TestCSRFIsRequiredForWrites(t *testing.T) {
	h := newHarness(t)
	b := h.newBrowser(t)

	payload, _ := json.Marshal(map[string]string{"email": "jane@example.com"})
	req, _ := http.NewRequest(http.MethodPost, h.base+"/api/auth/request", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without csrf, got %d", resp.StatusCode)
	}
}

func TestFullWorkLoopFromTaskToApproval(t *testing.T) {
	h := newHarness(t)
	b := h.newBrowser(t)
	b.login(t, "jane@example.com")

	var createdAgent struct {
		Agent  map[string]any `json:"agent"`
		APIKey string         `json:"apiKey"`
	}
	status, raw := b.do(t, http.MethodPost, "/api/agents", map[string]string{
		"name": "Ada", "role": "Researcher", "emoji": "🔎",
	}, &createdAgent)
	if status != 201 {
		t.Fatalf("create agent: %d %s", status, raw)
	}
	if !strings.HasPrefix(createdAgent.APIKey, "crew_") {
		t.Fatalf("missing helper key: %q", createdAgent.APIKey)
	}

	agentID, _ := createdAgent.Agent["id"].(string)

	var createdTask struct {
		Task map[string]any `json:"task"`
	}
	status, raw = b.do(t, http.MethodPost, "/api/tasks", map[string]string{
		"title": "Find three competitors", "details": "Look at pricing pages.", "agentId": agentID,
	}, &createdTask)
	if status != 201 {
		t.Fatalf("create task: %d %s", status, raw)
	}
	taskID, _ := createdTask.Task["id"].(string)

	// The helper talks to the API with its key, not the browser session.
	runner := &http.Client{Timeout: 10 * time.Second}
	call := func(path string, body any, out any) int {
		payload, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, h.base+path, bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+createdAgent.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := runner.Do(req)
		if err != nil {
			t.Fatalf("runner %s: %v", path, err)
		}
		defer resp.Body.Close()

		raw, _ := io.ReadAll(resp.Body)
		if out != nil {
			_ = json.Unmarshal(raw, out)
		}
		return resp.StatusCode
	}

	var claim struct {
		Task map[string]any `json:"task"`
	}
	if code := call("/api/runner/claim", nil, &claim); code != 200 {
		t.Fatalf("claim: %d", code)
	}
	if claim.Task == nil || claim.Task["id"] != taskID {
		t.Fatalf("helper did not get the task: %v", claim.Task)
	}

	if code := call("/api/runner/tasks/"+taskID+"/progress", map[string]string{"message": "Reading their sites."}, nil); code != 201 {
		t.Fatalf("progress: %d", code)
	}

	if code := call("/api/runner/tasks/"+taskID+"/complete", map[string]any{
		"summary": "Found three: A, B, C.", "needsReview": true,
	}, nil); code != 200 {
		t.Fatalf("complete: %d", code)
	}

	var task struct {
		Task struct {
			Status string `json:"status"`
		} `json:"task"`
	}
	status, raw = b.do(t, http.MethodGet, "/api/tasks/"+taskID, nil, &task)
	if status != 200 {
		t.Fatalf("get task: %d %s", status, raw)
	}
	if task.Task.Status != "review" {
		t.Fatalf("expected review, got %q", task.Task.Status)
	}

	status, raw = b.do(t, http.MethodPost, "/api/tasks/"+taskID+"/approve", nil, nil)
	if status != 200 {
		t.Fatalf("approve: %d %s", status, raw)
	}

	status, raw = b.do(t, http.MethodGet, "/api/tasks/"+taskID, nil, &task)
	if status != 200 || task.Task.Status != "done" {
		t.Fatalf("expected done after approval: %d %s", status, raw)
	}
}

func TestOnePersonCannotReadAnotherPersonsTask(t *testing.T) {
	h := newHarness(t)

	jane := h.newBrowser(t)
	jane.login(t, "jane@example.com")

	var created struct {
		Task map[string]any `json:"task"`
	}
	jane.do(t, http.MethodPost, "/api/tasks", map[string]string{"title": "Private"}, &created)
	taskID, _ := created.Task["id"].(string)

	sam := h.newBrowser(t)
	sam.login(t, "sam@example.com")

	status, _ := sam.do(t, http.MethodGet, "/api/tasks/"+taskID, nil, nil)
	if status != 404 {
		t.Fatalf("expected 404 across workspaces, got %d", status)
	}
}

func TestHelperKeyIsRejectedWhenWrong(t *testing.T) {
	h := newHarness(t)

	req, _ := http.NewRequest(http.MethodPost, h.base+"/api/runner/claim", nil)
	req.Header.Set("Authorization", "Bearer crew_not_a_real_key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogoutRecordsAndRevokes(t *testing.T) {
	h := newHarness(t)
	b := h.newBrowser(t)
	b.login(t, "jane@example.com")

	status, raw := b.do(t, http.MethodPost, "/api/auth/logout", map[string]string{"reason": "done_for_now"}, nil)
	if status != 200 {
		t.Fatalf("logout: %d %s", status, raw)
	}

	status, _ = b.do(t, http.MethodGet, "/api/me", nil, nil)
	if status != 401 {
		t.Fatalf("expected 401 after logout, got %d", status)
	}
}

func TestDeleteMyDataRemovesTheWorkspace(t *testing.T) {
	h := newHarness(t)
	b := h.newBrowser(t)
	b.login(t, "jane@example.com")

	b.do(t, http.MethodPost, "/api/tasks", map[string]string{"title": "Temp"}, nil)

	status, raw := b.do(t, http.MethodDelete, "/api/me", nil, nil)
	if status != 200 {
		t.Fatalf("delete me: %d %s", status, raw)
	}

	status, _ = b.do(t, http.MethodGet, "/api/me", nil, nil)
	if status != 401 {
		t.Fatalf("expected 401 after deleting data, got %d", status)
	}
}
