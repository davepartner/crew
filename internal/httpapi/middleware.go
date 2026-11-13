package httpapi

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ctxKey string

const identityKey ctxKey = "identity"

type identity struct {
	Token   string
	Subject string // PayTrackify user id
	Email   string
	Name    string
	UserID  string // local Crew user id
}

func (s *Server) withIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Helper calls authenticate with an agent key, never a browser cookie.
		// Resolving a session for them would call the identity provider on
		// every poll and is meaningless.
		if strings.HasPrefix(r.URL.Path, "/api/runner/") {
			next.ServeHTTP(w, r)
			return
		}

		token := s.sessionToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, ok := s.cache.Get(token)
		if !ok {
			fetched, err := s.auth.Me(r.Context(), token)
			if err != nil {
				if isUnauthorized(err) {
					s.clearSession(w)
					next.ServeHTTP(w, r)
					return
				}

				slog.Warn("session validation failed", "err", err)
				if strings.HasPrefix(r.URL.Path, "/api/") {
					writeError(w, http.StatusServiceUnavailable, "We couldn't check your sign-in just now. Please refresh in a moment.")
					return
				}
			} else {
				user = fetched
				s.cache.Set(token, user)
			}
		}

		if user.ID != "" {
			local, err := s.store.UpsertUser(r.Context(), user.ID, user.Email, user.Name)
			if err != nil {
				slog.Error("upsert user", "err", err)
			} else {
				ctx := context.WithValue(r.Context(), identityKey, &identity{
					Token:   token,
					Subject: user.ID,
					Email:   user.Email,
					Name:    user.Name,
					UserID:  local.ID,
				})
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) sessionToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func identityFrom(r *http.Request) *identity {
	value, _ := r.Context().Value(identityKey).(*identity)
	return value
}

func (s *Server) requireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if identityFrom(r) == nil {
			writeError(w, http.StatusUnauthorized, "Please sign in to continue.")
			return
		}
		next(w, r)
	}
}

func (s *Server) sessionCookieValue(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Server) setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (s *Server) withCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Helpers authenticate with a bearer key, never a cookie, so CSRF does
		// not apply and they should not receive a browser cookie.
		if strings.HasPrefix(r.URL.Path, "/api/runner/") {
			next.ServeHTTP(w, r)
			return
		}

		csrf, err := r.Cookie(csrfCookie)
		if err != nil || csrf.Value == "" {
			csrf = &http.Cookie{
				Name:     csrfCookie,
				Value:    randomToken(32),
				Path:     "/",
				HttpOnly: false,
				Secure:   s.cfg.SecureCookies,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   30 * 24 * 60 * 60,
			}
			http.SetCookie(w, csrf)
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		provided := r.Header.Get("X-CSRF-Token")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(csrf.Value)) != 1 {
			writeError(w, http.StatusForbidden, "That request was blocked for safety. Please refresh the page and try again.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		if s.cfg.SecureCookies {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// Every API response is personal and short-lived.
			w.Header().Set("Cache-Control", "no-store")
		}
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "value", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// limiter is a deliberately small fixed-window limiter for the public auth
// endpoints. PayTrackify enforces the durable limits; this is the cheap shield
// in front of it, so a broken client cannot hammer the provider.
type limiter struct {
	mu      sync.Mutex
	buckets map[string]window
	limit   int
	period  time.Duration
}

type window struct {
	count int
	reset time.Time
}

func newLimiter(limit int, period time.Duration) *limiter {
	return &limiter{buckets: make(map[string]window), limit: limit, period: period}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, ok := l.buckets[key]
	if !ok || now.After(bucket.reset) {
		l.buckets[key] = window{count: 1, reset: now.Add(l.period)}
		return true
	}

	if bucket.count >= l.limit {
		return false
	}

	bucket.count++
	l.buckets[key] = bucket

	if len(l.buckets) > 10000 {
		for k, v := range l.buckets {
			if now.After(v.reset) {
				delete(l.buckets, k)
			}
		}
	}

	return true
}

// clientIP is the address the identity provider should rate-limit. Only trust
// X-Forwarded-For when the operator has explicitly said a proxy is in front.
func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
			if first != "" {
				return first
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
