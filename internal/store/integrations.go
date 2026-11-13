package store

import (
	"context"
	"database/sql"
	"time"
)

type Integration struct {
	Kind      string    `json:"kind"`
	Config    string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) SaveIntegration(ctx context.Context, companyID, kind, encryptedConfig string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_integrations (company_id, kind, config_cipher, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id, kind) DO UPDATE SET config_cipher = excluded.config_cipher, created_at = excluded.created_at`,
		companyID, kind, encryptedConfig, Now())
	return err
}

func (s *Store) GetIntegration(ctx context.Context, companyID, kind string) (Integration, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT kind, config_cipher, created_at FROM company_integrations WHERE company_id = ? AND kind = ?`,
		companyID, kind)

	var integration Integration
	var created string
	if err := row.Scan(&integration.Kind, &integration.Config, &created); err != nil {
		if err == sql.ErrNoRows {
			return Integration{}, ErrNotFound
		}
		return Integration{}, err
	}
	integration.CreatedAt = ParseTime(created)
	return integration, nil
}

func (s *Store) ListIntegrations(ctx context.Context, companyID string) ([]Integration, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, config_cipher, created_at FROM company_integrations WHERE company_id = ? ORDER BY kind`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Integration{}
	for rows.Next() {
		var integration Integration
		var created string
		if err := rows.Scan(&integration.Kind, &integration.Config, &created); err != nil {
			return nil, err
		}
		integration.CreatedAt = ParseTime(created)
		out = append(out, integration)
	}
	return out, rows.Err()
}

func (s *Store) DeleteIntegration(ctx context.Context, companyID, kind string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM company_integrations WHERE company_id = ? AND kind = ?`, companyID, kind)
	return err
}
