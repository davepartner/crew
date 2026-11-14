package httpapi

import (
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/secret"
	"github.com/davepartner/crew/internal/store"
	"github.com/davepartner/crew/internal/webui"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	auth    *payauth.Client
	cache   *payauth.Cache
	cipher  *secret.Cipher
	webFS   fs.FS
	limiter *limiter
}

func New(cfg config.Config, st *store.Store, client *payauth.Client, cipher *secret.Cipher) *Server {
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
		cipher:  cipher,
		webFS:   webFS,
		limiter: newLimiter(12, time.Minute),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)

	var handler http.Handler = mux
	handler = s.withCompany(handler)
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

	// Companies and onboarding.
	mux.HandleFunc("GET /api/companies", s.requireUser(s.handleListCompanies))
	mux.HandleFunc("POST /api/companies", s.requireUser(s.handleCreateCompany))
	mux.HandleFunc("GET /api/companies/{id}", s.requireUser(s.handleGetCompany))
	mux.HandleFunc("PATCH /api/companies/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUpdateCompany))))
	mux.HandleFunc("DELETE /api/companies/{id}", s.requireUser(s.requireCompany(s.requireRole("owner", s.handleDeleteCompany))))

	mux.HandleFunc("GET /api/integrations", s.requireUser(s.requireCompany(s.handleListIntegrations)))
	mux.HandleFunc("PUT /api/integrations/{kind}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSaveIntegration))))
	mux.HandleFunc("DELETE /api/integrations/{kind}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteIntegration))))
	mux.HandleFunc("POST /api/integrations/{kind}/test", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleTestIntegration))))
	mux.HandleFunc("GET /api/integrations/{kind}/overview", s.requireUser(s.requireCompany(s.handleIntegrationOverview)))
	mux.HandleFunc("POST /api/send/email", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSendEmail))))
	mux.HandleFunc("POST /api/send/whatsapp", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSendWhatsApp))))

	mux.HandleFunc("GET /api/financials", s.requireUser(s.requireCompany(s.handleFinancials)))
	mux.HandleFunc("POST /api/financials", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateFinancials))))
	mux.HandleFunc("DELETE /api/financials/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteFinancials))))
	mux.HandleFunc("POST /api/financials/{id}/audit", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleAuditFinancials))))

	mux.HandleFunc("GET /api/files", s.requireUser(s.requireCompany(s.handleListFiles)))
	mux.HandleFunc("POST /api/files", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUploadFile))))
	mux.HandleFunc("GET /api/files/{id}/download", s.requireUser(s.requireCompany(s.handleDownloadFile)))
	mux.HandleFunc("DELETE /api/files/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteFile))))

	mux.HandleFunc("GET /api/security", s.requireUser(s.requireCompany(s.handleSecurity)))
	mux.HandleFunc("POST /api/security/scan", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSecurityScan))))
	mux.HandleFunc("GET /api/security/stix", s.requireUser(s.requireCompany(s.handleSecurityStix)))
	mux.HandleFunc("POST /api/security/assets", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSecurityAsset))))
	mux.HandleFunc("DELETE /api/security/assets/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteSecurityAsset))))
	mux.HandleFunc("POST /api/security/indicators", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSecurityIndicator))))
	mux.HandleFunc("POST /api/security/indicators/{id}/status", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSecurityIndicatorStatus))))
	mux.HandleFunc("DELETE /api/security/indicators/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteSecurityIndicator))))
	mux.HandleFunc("POST /api/security/reports", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSecurityReport))))

	mux.HandleFunc("GET /api/research/status", s.requireUser(s.requireCompany(s.handleResearchStatus)))

	mux.HandleFunc("GET /api/adaccounts", s.requireUser(s.requireCompany(s.handleAdAccounts)))
	mux.HandleFunc("POST /api/adaccounts/{platform}/campaigns", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateAdCampaign))))
	mux.HandleFunc("POST /api/adaccounts/{platform}/campaigns/{id}/status", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleAdCampaignStatus))))
	mux.HandleFunc("POST /api/adaccounts/{platform}/campaigns/{id}/budget", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleAdCampaignBudget))))
	mux.HandleFunc("POST /api/adaccounts/meta/campaigns/{id}/delivery", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleMetaDelivery))))

	mux.HandleFunc("GET /api/portfolio", s.requireUser(s.requireCompany(s.handlePortfolio)))

	mux.HandleFunc("GET /api/products", s.requireUser(s.requireCompany(s.handleListProducts)))
	mux.HandleFunc("POST /api/products", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateProduct))))
	mux.HandleFunc("DELETE /api/products/{id}", s.requireUser(s.requireCompany(s.requireRole("owner", s.handleDeleteProduct))))

	mux.HandleFunc("GET /api/members", s.requireUser(s.requireCompany(s.handleListMembers)))
	mux.HandleFunc("POST /api/members", s.requireUser(s.requireCompany(s.requireRole("owner", s.handleAddMember))))
	mux.HandleFunc("DELETE /api/members/{id}", s.requireUser(s.requireCompany(s.requireRole("owner", s.handleRemoveMember))))
	mux.HandleFunc("DELETE /api/invites/{id}", s.requireUser(s.requireCompany(s.requireRole("owner", s.handleDeleteInvite))))
	mux.HandleFunc("GET /api/presets", s.requireUser(s.handlePresets))
	mux.HandleFunc("GET /api/presets/{key}", s.requireUser(s.handlePreset))

	// AI providers and keys.
	mux.HandleFunc("GET /api/providers", s.requireUser(s.handleProviders))
	mux.HandleFunc("PUT /api/providers/{key}", s.requireUser(s.handleSetProviderKey))
	mux.HandleFunc("DELETE /api/providers/{key}", s.requireUser(s.handleDeleteProviderKey))
	mux.HandleFunc("POST /api/providers/{key}/test", s.requireUser(s.handleTestProvider))

	// Marketing library and lab.
	mux.HandleFunc("GET /api/skills", s.requireUser(s.handleSkills))
	mux.HandleFunc("POST /api/ads/generate", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleGenerateAds))))
	mux.HandleFunc("GET /api/ads/campaigns", s.requireUser(s.requireCompany(s.handleListAdCampaigns)))
	mux.HandleFunc("GET /api/ads/campaigns/{id}", s.requireUser(s.requireCompany(s.handleGetAdCampaign)))
	mux.HandleFunc("DELETE /api/ads/campaigns/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteAdCampaign))))
	mux.HandleFunc("GET /api/skills/{key}", s.requireUser(s.handleSkill))
	mux.HandleFunc("POST /api/marketing/analyze", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleMarketingAnalyze))))

	// Work, scoped to the active company.
	mux.HandleFunc("GET /api/advice", s.requireUser(s.requireCompany(s.handleAdvice)))
	mux.HandleFunc("POST /api/advice/study", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleStudyCompany))))
	mux.HandleFunc("POST /api/advice/chat", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleAdviceChat))))

	mux.HandleFunc("GET /api/dashboard", s.requireUser(s.requireCompany(s.handleDashboard)))
	mux.HandleFunc("GET /api/announcements", s.requireUser(s.handleAnnouncements))
	mux.HandleFunc("GET /api/notifications", s.requireUser(s.requireCompany(s.handleNotifications)))
	mux.HandleFunc("POST /api/notifications/{id}/read", s.requireUser(s.requireCompany(s.handleReadNotification)))
	mux.HandleFunc("POST /api/notifications/read-all", s.requireUser(s.requireCompany(s.handleReadAllNotifications)))
	mux.HandleFunc("GET /api/activity", s.requireUser(s.requireCompany(s.handleActivity)))

	mux.HandleFunc("GET /api/agents", s.requireUser(s.requireCompany(s.handleListAgents)))
	mux.HandleFunc("POST /api/agents", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateAgent))))
	mux.HandleFunc("GET /api/agents/{id}", s.requireUser(s.requireCompany(s.handleGetAgent)))
	mux.HandleFunc("PATCH /api/agents/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUpdateAgent))))
	mux.HandleFunc("DELETE /api/agents/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteAgent))))
	mux.HandleFunc("POST /api/agents/{id}/rotate-key", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleRotateAgentKey))))

	mux.HandleFunc("GET /api/goals", s.requireUser(s.requireCompany(s.handleListGoals)))
	mux.HandleFunc("POST /api/goals", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateGoal))))
	mux.HandleFunc("PATCH /api/goals/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUpdateGoal))))
	mux.HandleFunc("DELETE /api/goals/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteGoal))))

	mux.HandleFunc("GET /api/routines", s.requireUser(s.requireCompany(s.handleListRoutines)))
	mux.HandleFunc("POST /api/routines", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateRoutine))))
	mux.HandleFunc("PATCH /api/routines/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUpdateRoutine))))
	mux.HandleFunc("DELETE /api/routines/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteRoutine))))

	mux.HandleFunc("GET /api/tasks", s.requireUser(s.requireCompany(s.handleListTasks)))
	mux.HandleFunc("POST /api/tasks", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleCreateTask))))
	mux.HandleFunc("GET /api/tasks/{id}", s.requireUser(s.requireCompany(s.handleGetTask)))
	mux.HandleFunc("PATCH /api/tasks/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleUpdateTask))))
	mux.HandleFunc("DELETE /api/tasks/{id}", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleDeleteTask))))
	mux.HandleFunc("POST /api/tasks/{id}/comments", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleAddComment))))
	mux.HandleFunc("POST /api/tasks/{id}/approve", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleApproveTask))))
	mux.HandleFunc("POST /api/tasks/{id}/send-back", s.requireUser(s.requireCompany(s.requireRole("editor", s.handleSendBack))))

	// Helper (agent) runner API.
	mux.HandleFunc("POST /api/runner/claim", s.requireAgent(s.handleRunnerClaim))
	mux.HandleFunc("POST /api/runner/heartbeat", s.requireAgent(s.handleRunnerHeartbeat))
	mux.HandleFunc("GET /api/runner/model", s.requireAgent(s.handleRunnerModel))
	mux.HandleFunc("POST /api/runner/ai", s.requireAgent(s.handleRunnerAI))
	mux.HandleFunc("POST /api/runner/delegate", s.requireAgent(s.handleRunnerDelegate))
	mux.HandleFunc("POST /api/runner/tasks/{id}/progress", s.requireAgent(s.handleRunnerProgress))
	mux.HandleFunc("POST /api/runner/tasks/{id}/complete", s.requireAgent(s.handleRunnerComplete))

	mux.HandleFunc("GET /internal/stats", s.requireAdminToken(s.handleInternalStats))
	mux.HandleFunc("GET /internal/user", s.requireAdminToken(s.handleInternalUser))

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
