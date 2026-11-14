package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const TimeFormat = "2006-01-02T15:04:05.000000000Z"

var ErrNotFound = errors.New("not found")

// Store owns the SQLite database. Every method takes the owner's id so a bug in
// one handler can never read another person's rows: tenancy is enforced in the
// query, not in the template.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}

	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite is a single-writer database; one connection keeps behaviour
	// predictable and avoids "database is locked" under concurrent writes.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			external_id TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			role TEXT NOT NULL,
			emoji TEXT NOT NULL,
			instructions TEXT NOT NULL DEFAULT '',
			api_key_hash TEXT NOT NULL UNIQUE,
			last_seen_at TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_user ON agents(user_id)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'todo',
			agent_id TEXT REFERENCES agents(id) ON DELETE SET NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			completed_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON tasks(user_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_agent ON tasks(agent_id)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			author_type TEXT NOT NULL,
			author_name TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_task ON comments(task_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS activity (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			message TEXT NOT NULL,
			task_id TEXT,
			agent_id TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_user ON activity(user_id, id DESC)`,
		`CREATE TABLE IF NOT EXISTS companies (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			business_key TEXT NOT NULL DEFAULT 'custom',
			website TEXT NOT NULL DEFAULT '',
			support_email TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			audience TEXT NOT NULL DEFAULT '',
			tone TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_companies_user ON companies(user_id)`,
		`CREATE TABLE IF NOT EXISTS model_usage (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			company_id TEXT NOT NULL,
			agent_id TEXT NOT NULL DEFAULT '',
			task_id TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			cost_usd REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_company ON model_usage(company_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_agent ON model_usage(agent_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS goals (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			company_id TEXT NOT NULL,
			title TEXT NOT NULL,
			target INTEGER NOT NULL DEFAULT 1,
			current INTEGER NOT NULL DEFAULT 0,
			due_date TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			completed_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goals_company ON goals(company_id, completed_at)`,
		`CREATE TABLE IF NOT EXISTS company_files (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			company_id TEXT NOT NULL,
			name TEXT NOT NULL,
			mime TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			path TEXT NOT NULL,
			text TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_company_files ON company_files(company_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS financial_snapshots (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			period TEXT NOT NULL DEFAULT '',
			revenue REAL NOT NULL DEFAULT 0,
			cogs REAL NOT NULL DEFAULT 0,
			opex REAL NOT NULL DEFAULT 0,
			cash REAL NOT NULL DEFAULT 0,
			receivables REAL NOT NULL DEFAULT 0,
			payables REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT '',
			audited INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_financials_company ON financial_snapshots(company_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS company_integrations (
			company_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			config_cipher TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (company_id, kind)
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_prospects (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			name TEXT NOT NULL,
			website TEXT NOT NULL DEFAULT '',
			domain TEXT NOT NULL DEFAULT '',
			industry TEXT NOT NULL DEFAULT '',
			location TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'manual',
			stage TEXT NOT NULL DEFAULT 'new',
			score INTEGER NOT NULL DEFAULT 0,
			signals TEXT NOT NULL DEFAULT '[]',
			notes TEXT NOT NULL DEFAULT '',
			last_contacted TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_contacts (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			prospect_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			email_status TEXT NOT NULL DEFAULT 'guess',
			source TEXT NOT NULL DEFAULT '',
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_campaigns (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			name TEXT NOT NULL,
			offer TEXT NOT NULL DEFAULT '',
			audience TEXT NOT NULL DEFAULT '',
			tone TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			daily_limit INTEGER NOT NULL DEFAULT 20,
			autopilot INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_steps (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			campaign_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			delay_days INTEGER NOT NULL DEFAULT 0,
			subject TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_messages (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			campaign_id TEXT NOT NULL,
			prospect_id TEXT NOT NULL,
			contact_id TEXT NOT NULL,
			step INTEGER NOT NULL DEFAULT 1,
			to_email TEXT NOT NULL DEFAULT '',
			subject TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			scheduled_at TEXT NOT NULL DEFAULT '',
			sent_at TEXT NOT NULL DEFAULT '',
			reply_text TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_suppressions (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			email TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS outreach_settings (
			company_id TEXT PRIMARY KEY,
			from_name TEXT NOT NULL DEFAULT '',
			from_email TEXT NOT NULL DEFAULT '',
			daily_limit INTEGER NOT NULL DEFAULT 20,
			autopilot INTEGER NOT NULL DEFAULT 0,
			signature TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cti_assets (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			type TEXT NOT NULL,
			value TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			criticality TEXT NOT NULL DEFAULT 'medium',
			source TEXT NOT NULL DEFAULT 'manual',
			last_seen TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(company_id, type, value)
		)`,
		`CREATE TABLE IF NOT EXISTS cti_indicators (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			type TEXT NOT NULL,
			value TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			confidence INTEGER NOT NULL DEFAULT 50,
			tlp TEXT NOT NULL DEFAULT 'amber',
			source TEXT NOT NULL DEFAULT 'manual',
			source_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			tags TEXT NOT NULL DEFAULT '[]',
			first_seen TEXT NOT NULL,
			last_seen TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(company_id, type, value)
		)`,
		`CREATE TABLE IF NOT EXISTS cti_actors (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			name TEXT NOT NULL,
			aliases TEXT NOT NULL DEFAULT '[]',
			motivation TEXT NOT NULL DEFAULT '',
			sophistication TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			confidence INTEGER NOT NULL DEFAULT 50,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cti_reports (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			title TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			published_at TEXT NOT NULL DEFAULT '',
			confidence INTEGER NOT NULL DEFAULT 50,
			tlp TEXT NOT NULL DEFAULT 'amber',
			tags TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cti_relationships (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relation TEXT NOT NULL,
			confidence INTEGER NOT NULL DEFAULT 50,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cti_findings (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			category TEXT NOT NULL,
			severity TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			evidence TEXT NOT NULL DEFAULT '',
			fix TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cti_assessments (
			company_id TEXT PRIMARY KEY,
			analysis TEXT NOT NULL,
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS company_advice (
			company_id TEXT PRIMARY KEY,
			analysis TEXT NOT NULL,
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ad_campaigns (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			company_id TEXT NOT NULL,
			title TEXT NOT NULL,
			brief TEXT NOT NULL,
			result TEXT NOT NULL,
			model TEXT NOT NULL DEFAULT '',
			cost_usd REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ad_campaigns_company ON ad_campaigns(company_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			company_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			task_id TEXT NOT NULL DEFAULT '',
			read_at TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, company_id, read_at, created_at)`,
		`CREATE TABLE IF NOT EXISTS company_members (
			company_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'editor',
			created_at TEXT NOT NULL,
			PRIMARY KEY (company_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS company_invites (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			email TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'editor',
			created_at TEXT NOT NULL,
			UNIQUE (company_id, email)
		)`,
		`CREATE TABLE IF NOT EXISTS routines (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			company_id TEXT NOT NULL,
			title TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			agent_id TEXT,
			cadence TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			next_run_at TEXT NOT NULL,
			last_run_at TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_routines_due ON routines(enabled, next_run_at)`,
		`CREATE TABLE IF NOT EXISTS provider_keys (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			key_cipher TEXT NOT NULL,
			last4 TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(user_id, provider)
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	columns := []struct {
		table, column, definition string
	}{
		{"agents", "company_id", "TEXT NOT NULL DEFAULT ''"},
		{"agents", "model_provider", "TEXT NOT NULL DEFAULT ''"},
		{"agents", "model_name", "TEXT NOT NULL DEFAULT ''"},
		{"agents", "skills", "TEXT NOT NULL DEFAULT '[]'"},
		{"agents", "monthly_budget_usd", "REAL NOT NULL DEFAULT 0"},
		{"agents", "can_delegate", "INTEGER NOT NULL DEFAULT 1"},
		{"tasks", "company_id", "TEXT NOT NULL DEFAULT ''"},
		{"activity", "company_id", "TEXT NOT NULL DEFAULT ''"},
		{"companies", "monthly_task_limit", "INTEGER NOT NULL DEFAULT 0"},
		{"companies", "monthly_budget_usd", "REAL NOT NULL DEFAULT 0"},
		{"companies", "country", "TEXT NOT NULL DEFAULT 'Nigeria'"},
		{"companies", "ticker", "TEXT NOT NULL DEFAULT ''"},
		{"financial_snapshots", "audit_status", "TEXT NOT NULL DEFAULT 'self_reported'"},
		{"financial_snapshots", "audit_note", "TEXT NOT NULL DEFAULT ''"},
		{"financial_snapshots", "auditor", "TEXT NOT NULL DEFAULT ''"},
		{"financial_snapshots", "audit_file_id", "TEXT NOT NULL DEFAULT ''"},
		{"companies", "parent_company_id", "TEXT NOT NULL DEFAULT ''"},
		{"companies", "is_product", "INTEGER NOT NULL DEFAULT 0"},
	}

	for _, column := range columns {
		if err := s.ensureColumn(ctx, column.table, column.column, column.definition); err != nil {
			return err
		}
	}

	return s.ensureDefaultCompanies(ctx)
}

func (s *Store) ensureColumn(ctx context.Context, table, column, definition string) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int
		var name, kind string
		var notNull, pk int
		var defaultValue any

		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if found {
		return nil
	}

	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

// ensureDefaultCompanies preserves data from before companies existed: anyone
// with agents or tasks gets a "My company" workspace and their rows are linked
// to it. New users start with no company so onboarding runs.
func (s *Store) ensureDefaultCompanies(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT user_id FROM agents WHERE company_id = ''
		UNION
		SELECT DISTINCT user_id FROM tasks WHERE company_id = ''`)
	if err != nil {
		return err
	}

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return err
		}
		users = append(users, userID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, userID := range users {
		var companyID string
		err := s.db.QueryRowContext(ctx, `SELECT id FROM companies WHERE user_id = ? ORDER BY created_at LIMIT 1`, userID).Scan(&companyID)
		if err != nil && !isNotFound(err) {
			return err
		}

		if companyID == "" {
			companyID = NewID()
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO companies (id, user_id, name, business_key, created_at)
				VALUES (?, ?, ?, ?, ?)`, companyID, userID, "My company", "custom", Now()); err != nil {
				return err
			}
		}

		for _, stmt := range []string{
			`UPDATE agents SET company_id = ? WHERE user_id = ? AND company_id = ''`,
			`UPDATE tasks SET company_id = ? WHERE user_id = ? AND company_id = ''`,
			`UPDATE activity SET company_id = ? WHERE user_id = ? AND company_id = ''`,
		} {
			if _, err := s.db.ExecContext(ctx, stmt, companyID, userID); err != nil {
				return err
			}
		}
	}

	return nil
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func Now() string { return time.Now().UTC().Format(TimeFormat) }

func ParseTime(value string) time.Time {
	t, err := time.Parse(TimeFormat, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *Store) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func normalizeIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
