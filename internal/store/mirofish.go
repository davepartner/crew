package store

import (
	"context"
	"database/sql"
	"time"
)

type MiroFishRun struct {
	ID             string    `json:"id"`
	Question       string    `json:"question"`
	Product        string    `json:"product"`
	Agents         int       `json:"agents"`
	Rounds         int       `json:"rounds"`
	Status         string    `json:"status"`
	EstimateTokens int       `json:"estimate_tokens"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	ElapsedMS      int64     `json:"elapsed_ms"`
	Result         string    `json:"-"`
	ResultJSON     any       `json:"result,omitempty"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    string    `json:"completed_at,omitempty"`
}

func (s *Store) CreateMiroFishRun(ctx context.Context, companyID, userID string, run MiroFishRun) (MiroFishRun, error) {
	run.ID = NewID()
	run.CreatedAt = time.Now().UTC()
	if run.Status == "" {
		run.Status = "queued"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mirofish_runs (id, company_id, user_id, question, product, agents, rounds, status,
		 estimate_tokens, input_tokens, output_tokens, elapsed_ms, result, error, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, companyID, userID, run.Question, run.Product, run.Agents, run.Rounds, run.Status,
		run.EstimateTokens, 0, 0, 0, "", "", run.CreatedAt.Format(TimeFormat), "")
	return run, err
}

func (s *Store) GetMiroFishRun(ctx context.Context, companyID, id string) (MiroFishRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, question, product, agents, rounds, status, estimate_tokens, input_tokens, output_tokens,
		       elapsed_ms, result, error, created_at, completed_at
		FROM mirofish_runs WHERE company_id = ? AND id = ?`, companyID, id)
	run, err := scanMiroFishRun(row)
	if err == sql.ErrNoRows {
		return MiroFishRun{}, ErrNotFound
	}
	return run, err
}

func (s *Store) ListMiroFishRuns(ctx context.Context, companyID string, limit int) ([]MiroFishRun, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, question, product, agents, rounds, status, estimate_tokens, input_tokens, output_tokens,
		       elapsed_ms, result, error, created_at, completed_at
		FROM mirofish_runs WHERE company_id = ? ORDER BY created_at DESC LIMIT ?`, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MiroFishRun{}
	for rows.Next() {
		run, err := scanMiroFishRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (s *Store) UpdateMiroFishRun(ctx context.Context, id string, run MiroFishRun) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE mirofish_runs SET status = ?, estimate_tokens = ?, input_tokens = ?, output_tokens = ?,
		       elapsed_ms = ?, result = ?, error = ?, completed_at = ? WHERE id = ?`,
		run.Status, run.EstimateTokens, run.InputTokens, run.OutputTokens, run.ElapsedMS,
		run.Result, run.Error, run.CompletedAt, id)
	return err
}

func scanMiroFishRun(row scanner) (MiroFishRun, error) {
	var run MiroFishRun
	var created string
	err := row.Scan(&run.ID, &run.Question, &run.Product, &run.Agents, &run.Rounds, &run.Status,
		&run.EstimateTokens, &run.InputTokens, &run.OutputTokens, &run.ElapsedMS, &run.Result,
		&run.Error, &created, &run.CompletedAt)
	if err != nil {
		return MiroFishRun{}, err
	}
	run.CreatedAt = ParseTime(created)
	return run, nil
}
