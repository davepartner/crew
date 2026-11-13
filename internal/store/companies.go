package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Company struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	BusinessKey      string    `json:"business_key"`
	Website          string    `json:"website"`
	SupportEmail     string    `json:"support_email"`
	Description      string    `json:"description"`
	Audience         string    `json:"audience"`
	Tone             string    `json:"tone"`
	MonthlyTaskLimit int       `json:"monthly_task_limit"`
	MonthlyBudgetUSD float64   `json:"monthly_budget_usd"`
	SpendUSD         float64   `json:"spend_usd"`
	Role             string    `json:"role"`
	CreatedAt        time.Time `json:"created_at"`
	AgentCount       int       `json:"agent_count"`
	TaskCount        int       `json:"task_count"`
}

type CompanyInput struct {
	Name             string
	BusinessKey      string
	Website          string
	SupportEmail     string
	Description      string
	Audience         string
	Tone             string
	MonthlyTaskLimit int
	MonthlyBudgetUSD float64
}

func (s *Store) CreateCompany(ctx context.Context, userID string, input CompanyInput) (Company, error) {
	company := Company{
		ID:               NewID(),
		Name:             input.Name,
		BusinessKey:      input.BusinessKey,
		Website:          input.Website,
		SupportEmail:     input.SupportEmail,
		Description:      input.Description,
		Audience:         input.Audience,
		Tone:             input.Tone,
		MonthlyTaskLimit: input.MonthlyTaskLimit,
		CreatedAt:        time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO companies (id, user_id, name, business_key, website, support_email, description, audience, tone, monthly_task_limit, monthly_budget_usd, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		company.ID, userID, company.Name, company.BusinessKey, company.Website,
		company.SupportEmail, company.Description, company.Audience, company.Tone,
		company.MonthlyTaskLimit, company.MonthlyBudgetUSD, company.CreatedAt.Format(TimeFormat))
	if err != nil {
		return Company{}, err
	}

	return company, nil
}

func (s *Store) ListCompanies(ctx context.Context, userID string) ([]Company, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.business_key, c.website, c.support_email, c.description,
		       c.audience, c.tone, c.monthly_task_limit, c.monthly_budget_usd,
		       (SELECT COALESCE(SUM(u.cost_usd), 0) FROM model_usage u
		        WHERE u.company_id = c.id
		          AND strftime('%Y-%m', u.created_at) = strftime('%Y-%m', 'now')),
		       c.created_at,
		       CASE WHEN c.user_id = ? THEN 'owner'
		            ELSE COALESCE((SELECT m.role FROM company_members m WHERE m.company_id = c.id AND m.user_id = ?), 'viewer') END,
		       (SELECT COUNT(*) FROM agents a WHERE a.company_id = c.id),
		       (SELECT COUNT(*) FROM tasks t WHERE t.company_id = c.id)
		FROM companies c
		WHERE c.user_id = ? OR EXISTS (SELECT 1 FROM company_members m WHERE m.company_id = c.id AND m.user_id = ?)
		ORDER BY c.created_at ASC`, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	companies := []Company{}
	for rows.Next() {
		company, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		companies = append(companies, company)
	}

	return companies, rows.Err()
}

func (s *Store) GetCompany(ctx context.Context, userID, id string) (Company, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.business_key, c.website, c.support_email, c.description,
		       c.audience, c.tone, c.monthly_task_limit, c.monthly_budget_usd,
		       (SELECT COALESCE(SUM(u.cost_usd), 0) FROM model_usage u
		        WHERE u.company_id = c.id
		          AND strftime('%Y-%m', u.created_at) = strftime('%Y-%m', 'now')),
		       c.created_at,
		       CASE WHEN c.user_id = ? THEN 'owner'
		            ELSE COALESCE((SELECT m.role FROM company_members m WHERE m.company_id = c.id AND m.user_id = ?), 'viewer') END,
		       (SELECT COUNT(*) FROM agents a WHERE a.company_id = c.id),
		       (SELECT COUNT(*) FROM tasks t WHERE t.company_id = c.id)
		FROM companies c
		WHERE c.id = ? AND (c.user_id = ? OR EXISTS (SELECT 1 FROM company_members m WHERE m.company_id = c.id AND m.user_id = ?))`,
		userID, userID, id, userID, userID)

	company, err := scanCompany(row)
	if isNotFound(err) {
		return Company{}, ErrNotFound
	}
	return company, err
}

func (s *Store) UpdateCompany(ctx context.Context, userID, id string, input CompanyInput) (Company, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE companies
		SET name = ?, business_key = ?, website = ?, support_email = ?, description = ?, audience = ?, tone = ?, monthly_task_limit = ?, monthly_budget_usd = ?
		WHERE user_id = ? AND id = ?`,
		input.Name, input.BusinessKey, input.Website, input.SupportEmail,
		input.Description, input.Audience, input.Tone, input.MonthlyTaskLimit, input.MonthlyBudgetUSD, userID, id)
	if err != nil {
		return Company{}, err
	}

	return s.GetCompany(ctx, userID, id)
}

func (s *Store) DeleteCompany(ctx context.Context, userID, id string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var exists string
		err := tx.QueryRowContext(ctx, `SELECT id FROM companies WHERE user_id = ? AND id = ?`, userID, id).Scan(&exists)
		if isNotFound(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			DELETE FROM comments WHERE task_id IN (SELECT id FROM tasks WHERE company_id = ?)`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM agents WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM activity WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM routines WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM goals WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM company_members WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM company_invites WHERE company_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM notifications WHERE company_id = ?`, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM companies WHERE id = ?`, id)
		return err
	})
}

func scanCompany(row scanner) (Company, error) {
	var company Company
	var created string

	err := row.Scan(&company.ID, &company.Name, &company.BusinessKey, &company.Website,
		&company.SupportEmail, &company.Description, &company.Audience, &company.Tone,
		&company.MonthlyTaskLimit, &company.MonthlyBudgetUSD, &company.SpendUSD, &created,
		&company.Role, &company.AgentCount, &company.TaskCount)
	if err != nil {
		return Company{}, err
	}

	company.CreatedAt = ParseTime(created)
	return company, nil
}

var ErrNoCompany = errors.New("no company")
