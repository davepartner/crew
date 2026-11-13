package httpapi

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/store"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

func (s *Server) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("auth:" + s.clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Too many tries from this device. Please wait a minute.")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if len(email) > 254 || !emailPattern.MatchString(email) {
		writeError(w, http.StatusUnprocessableEntity, "That email doesn't look right. Check it and try again.")
		return
	}

	requestID, resendAfter, err := s.auth.RequestCode(r.Context(), email, s.clientIP(r))
	if err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"requestId":   requestID,
		"resendAfter": resendAfter,
		"message":     "If that email works, a code is on its way.",
	})
}

func (s *Server) handleAuthResend(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("auth:" + s.clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Too many tries from this device. Please wait a minute.")
		return
	}

	var body struct {
		RequestID string `json:"requestId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resendAfter, err := s.auth.ResendCode(r.Context(), body.RequestID, s.clientIP(r))
	if err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"resendAfter": resendAfter,
		"message":     "A new code is on its way.",
	})
}

func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("auth:" + s.clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Too many tries from this device. Please wait a minute.")
		return
	}

	var body struct {
		RequestID string `json:"requestId"`
		Code      string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	code := digitsOnly(body.Code)
	if body.RequestID == "" || len(code) != 8 {
		writeError(w, http.StatusUnprocessableEntity, "Enter the 8-digit code from your email.")
		return
	}

	result, err := s.auth.VerifyCode(r.Context(), body.RequestID, code, s.clientIP(r))
	if err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	local, err := s.store.UpsertUser(r.Context(), result.User.ID, result.User.Email, result.User.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We signed you in but couldn't open your workspace. Please try again.")
		return
	}

	s.setSession(w, result.Token)
	s.cache.Set(result.Token, result.User)

	if result.IsNew {
		_ = s.store.AddActivity(r.Context(), local.ID, "welcome", "You joined Crew", "", "")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"user":  userPayload(local),
		"isNew": result.IsNew,
	})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	token := s.sessionToken(r)

	reason := "prefer_not_to_say"
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err == nil && body.Reason != "" {
		reason = body.Reason
	}

	if token != "" {
		// Signing out locally must always work, even if the provider is down.
		_ = s.auth.Logout(r.Context(), token, reason)
		s.cache.Delete(token)
	}

	s.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	local, err := s.store.GetUser(r.Context(), id.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Please sign in again.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"user": userPayload(local),
	})
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := trim(body.Name, 60)
	if len([]rune(name)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Please add a name.")
		return
	}

	updated, err := s.auth.UpdateProfile(r.Context(), id.Token, name)
	if err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	local, err := s.store.UpsertUser(r.Context(), updated.ID, updated.Email, updated.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that. Please try again.")
		return
	}

	s.cache.Set(id.Token, updated)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"user": userPayload(local),
	})
}

func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	if err := s.store.DeleteUserData(r.Context(), id.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't delete your workspace. Please try again.")
		return
	}

	_ = s.auth.Logout(r.Context(), id.Token, "prefer_not_to_say")
	s.cache.Delete(id.Token)
	s.clearSession(w)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	user, err := s.store.GetUser(r.Context(), id.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Please sign in again.")
		return
	}

	agents, _ := s.store.ListAgents(r.Context(), id.UserID)
	tasks, _ := s.store.ListTasks(r.Context(), id.UserID, "")
	activity, _ := s.store.ListActivity(r.Context(), id.UserID, 100)
	comments, _ := s.store.CommentsForUser(r.Context(), id.UserID)

	w.Header().Set("Content-Disposition", `attachment; filename="crew-export.json"`)

	writeJSON(w, http.StatusOK, map[string]any{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"user":        userPayload(user),
		"agents":      agents,
		"tasks":       tasks,
		"activity":    activity,
		"comments":    comments,
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	sessions, err := s.auth.Sessions(r.Context(), id.Token)
	if err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sessions": sessions})
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	sessionID := r.PathValue("id")

	if err := s.auth.RevokeSession(r.Context(), id.Token, sessionID); err != nil {
		message, status, retryAfter := authErrorMessage(err)
		writeJSON(w, status, errorBody{OK: false, Error: message, RetryAfter: retryAfter})
		return
	}

	// Drop the cached identity so the revocation takes effect on the very next
	// request. If this device revoked a different session, the next request
	// simply revalidates and carries on.
	s.cache.Delete(id.Token)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func userPayload(user store.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"created_at": user.CreatedAt,
	}
}

func digitsOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
