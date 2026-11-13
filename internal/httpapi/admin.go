package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) requireAdminToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.AdminToken == "" {
			http.NotFound(w, r)
			return
		}

		provided := strings.TrimSpace(r.Header.Get("X-Crew-Token"))
		if provided == "" {
			provided = strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.cfg.AdminToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "That token isn't valid.")
			return
		}

		next(w, r)
	}
}

func (s *Server) handleInternalStats(w http.ResponseWriter, r *http.Request) {
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			days = parsed
		}
	}

	stats, err := s.store.AdminStats(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not read stats.")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleInternalUser(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	if email == "" {
		writeError(w, http.StatusUnprocessableEntity, "email is required")
		return
	}

	usage, err := s.store.UserUsage(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not read user stats.")
		return
	}

	writeJSON(w, http.StatusOK, usage)
}
