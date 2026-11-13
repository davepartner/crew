package httpapi

import (
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/store"
	"github.com/davepartner/crew/internal/webui"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	auth    *payauth.Client
	cache   *payauth.Cache
	webFS   fs.FS
	limiter *limiter
}

func New(cfg config.Config, st *store.Store, client *payauth.Client) *Server {
	var webFS fs.FS
	if cfg.WebDir != "" {
		webFS = os.DirFS(cfg.WebDir)
	} else {
		webFS = webui.FS()
	}

	return &Server{
		cfg:     cfg,
		store:   st,
		auth:    client,
		cache:   payauth.NewCache(cfg.SessionCacheTTL),
		webFS:   webFS,
		limiter: newLimiter(12, time.Minute),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)

	var handler http.Handler = mux
	handler = s.withIdentity(handler)
	handler = s.withCSRF(handler)
	handler = s.withLogging(handler)
	handler = s.securityHeaders(handler)
	handler = withRecovery(handler)

	return handler
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/csrf", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	// Passwordless sign-in (delegated to PayTrackify).
	mux.HandleFunc("POST /api/auth/request", s.handleAuthRequest)
	mux.HandleFunc("POST /api/auth/resend", s.handleAuthResend)
	mux.HandleFunc("POST /api/auth/verify", s.handleAuthVerify)
	mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)

	// The signed-in person.
	mux.HandleFunc("GET /api/me", s.requireUser(s.handleMe))
	mux.HandleFunc("PATCH /api/me", s.requireUser(s.handleUpdateMe))
	mux.HandleFunc("DELETE /api/me", s.requireUser(s.handleDeleteMe))
	mux.HandleFunc("GET /api/me/export", s.requireUser(s.handleExport))
	mux.HandleFunc("GET /api/me/sessions", s.requireUser(s.handleSessions))
	mux.HandleFunc("DELETE /api/me/sessions/{id}", s.requireUser(s.handleRevokeSession))

	// Work.
	mux.HandleFunc("GET /api/dashboard", s.requireUser(s.handleDashboard))
	mux.HandleFunc("GET /api/activity", s.requireUser(s.handleActivity))

	mux.HandleFunc("GET /api/agents", s.requireUser(s.handleListAgents))
	mux.HandleFunc("POST /api/agents", s.requireUser(s.handleCreateAgent))
	mux.HandleFunc("GET /api/agents/{id}", s.requireUser(s.handleGetAgent))
	mux.HandleFunc("PATCH /api/agents/{id}", s.requireUser(s.handleUpdateAgent))
	mux.HandleFunc("DELETE /api/agents/{id}", s.requireUser(s.handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{id}/rotate-key", s.requireUser(s.handleRotateAgentKey))

	mux.HandleFunc("GET /api/tasks", s.requireUser(s.handleListTasks))
	mux.HandleFunc("POST /api/tasks", s.requireUser(s.handleCreateTask))
	mux.HandleFunc("GET /api/tasks/{id}", s.requireUser(s.handleGetTask))
	mux.HandleFunc("PATCH /api/tasks/{id}", s.requireUser(s.handleUpdateTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", s.requireUser(s.handleDeleteTask))
	mux.HandleFunc("POST /api/tasks/{id}/comments", s.requireUser(s.handleAddComment))
	mux.HandleFunc("POST /api/tasks/{id}/approve", s.requireUser(s.handleApproveTask))
	mux.HandleFunc("POST /api/tasks/{id}/send-back", s.requireUser(s.handleSendBack))

	// Helper (agent) runner API.
	mux.HandleFunc("POST /api/runner/claim", s.requireAgent(s.handleRunnerClaim))
	mux.HandleFunc("POST /api/runner/heartbeat", s.requireAgent(s.handleRunnerHeartbeat))
	mux.HandleFunc("POST /api/runner/tasks/{id}/progress", s.requireAgent(s.handleRunnerProgress))
	mux.HandleFunc("POST /api/runner/tasks/{id}/complete", s.requireAgent(s.handleRunnerComplete))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "That endpoint does not exist.")
	})

	mux.Handle("/", spaHandler{fsys: s.webFS})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	if err := s.store.Ping(r.Context()); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, map[string]any{
		"ok":     status == "ok",
		"status": status,
		"env":    s.cfg.Env,
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
