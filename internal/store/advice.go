package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type CompanyAdvice struct {
	Analysis  json.RawMessage `json:"analysis"`
	Model     string          `json:"model"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Store) GetCompanyAdvice(ctx context.Context, companyID string) (CompanyAdvice, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT analysis, model, created_at FROM company_advice WHERE company_id = ?`, companyID)

	var advice CompanyAdvice
	var analysis, created string
	if err := row.Scan(&analysis, &advice.Model, &created); err != nil {
		if err == sql.ErrNoRows {
			return CompanyAdvice{}, ErrNotFound
		}
		return CompanyAdvice{}, err
	}
	advice.Analysis = json.RawMessage(analysis)
	advice.CreatedAt = ParseTime(created)
	return advice, nil
}

func (s *Store) SaveCompanyAdvice(ctx context.Context, companyID string, analysis []byte, model string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_advice (company_id, analysis, model, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id) DO UPDATE SET analysis = excluded.analysis, model = excluded.model, created_at = excluded.created_at`,
		companyID, string(analysis), model, Now())
	return err
}

func (s *Store) DeleteCompanyAdvice(ctx context.Context, companyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM company_advice WHERE company_id = ?`, companyID)
	return err
}
