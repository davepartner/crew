package store

import (
	"context"
	"time"
)

type UsageInput struct {
	UserID           string
	CompanyID        string
	AgentID          string
	TaskID           string
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	CostUSD          float64
}

func (s *Store) RecordUsage(ctx context.Context, input UsageInput) error {
	if input.UserID == "" || input.CompanyID == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO model_usage (id, user_id, company_id, agent_id, task_id, provider, model,
			prompt_tokens, completion_tokens, cost_usd, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		NewID(), input.UserID, input.CompanyID, input.AgentID, input.TaskID, input.Provider, input.Model,
		input.PromptTokens, input.CompletionTokens, input.CostUSD, Now())
	return err
}

func (s *Store) AgentSpend(ctx context.Context, companyID, agentID string, since time.Time) (float64, error) {
	var total float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0) FROM model_usage
		WHERE company_id = ? AND agent_id = ? AND created_at >= ?`,
		companyID, agentID, since.Format(TimeFormat)).Scan(&total)
	return total, err
}

func (s *Store) CompanySpend(ctx context.Context, companyID string, since time.Time) (float64, error) {
	var total float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0) FROM model_usage
		WHERE company_id = ? AND created_at >= ?`,
		companyID, since.Format(TimeFormat)).Scan(&total)
	return total, err
}

func MonthStart() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}
