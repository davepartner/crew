package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is everything the server needs. Values come from the environment so
// the same binary runs locally, in Docker, and on a VPS without edits.
type Config struct {
	Addr            string
	DBPath          string
	PaytrackifyURL  string
	PaytrackifyTime time.Duration
	SessionCacheTTL time.Duration
	SecureCookies   bool
	TrustProxy      bool
	TaskLease       time.Duration
	WebDir          string
	Env             string
	ServiceName     string
}

func Load() Config {
	cfg := Config{
		Addr:            env("CREW_ADDR", "127.0.0.1:8787"),
		DBPath:          env("CREW_DB", "data/crew.db"),
		PaytrackifyURL:  strings.TrimRight(env("PAYTRACKIFY_URL", "http://127.0.0.1:8012"), "/"),
		PaytrackifyTime: envDuration("PAYTRACKIFY_TIMEOUT", 8*time.Second),
		SessionCacheTTL: envDuration("CREW_SESSION_CACHE_TTL", 60*time.Second),
		SecureCookies:   envBool("CREW_SECURE_COOKIES", false),
		TrustProxy:      envBool("CREW_TRUST_PROXY", false),
		TaskLease:       envDuration("CREW_TASK_LEASE", time.Hour),
		WebDir:          env("CREW_WEB_DIR", ""),
		Env:             env("CREW_ENV", "development"),
		ServiceName:     env("CREW_NAME", "Crew"),
	}

	if cfg.SecureCookies || cfg.Env == "production" {
		cfg.SecureCookies = true
	}

	return cfg
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return parsed
}
