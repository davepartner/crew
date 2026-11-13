package store

import (
	"context"
	"encoding/json"
	"time"
)

type AdCampaign struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Brief     json.RawMessage `json:"brief"`
	Result    json.RawMessage `json:"result"`
	Model     string          `json:"model"`
	CostUSD   float64         `json:"cost_usd"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Store) CreateAdCampaign(ctx context.Context, userID, companyID, title string, brief, result []byte, model string, cost float64) (AdCampaign, error) {
	campaign := AdCampaign{
		ID:        NewID(),
		Title:     title,
		Brief:     brief,
		Result:    result,
		Model:     model,
		CostUSD:   cost,
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ad_campaigns (id, user_id, company_id, title, brief, result, model, cost_usd, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaign.ID, userID, companyID, campaign.Title, string(brief), string(result),
		campaign.Model, campaign.CostUSD, campaign.CreatedAt.Format(TimeFormat))
	if err != nil {
		return AdCampaign{}, err
	}

	return campaign, nil
}

func (s *Store) ListAdCampaigns(ctx context.Context, companyID string, limit int) ([]AdCampaign, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, brief, result, model, cost_usd, created_at
		FROM ad_campaigns WHERE company_id = ?
		ORDER BY created_at DESC LIMIT ?`, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := []AdCampaign{}
	for rows.Next() {
		campaign, err := scanAdCampaign(rows)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func (s *Store) GetAdCampaign(ctx context.Context, companyID, id string) (AdCampaign, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, brief, result, model, cost_usd, created_at
		FROM ad_campaigns WHERE company_id = ? AND id = ?`, companyID, id)

	campaign, err := scanAdCampaign(row)
	if isNotFound(err) {
		return AdCampaign{}, ErrNotFound
	}
	return campaign, err
}

func (s *Store) DeleteAdCampaign(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ad_campaigns WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAdCampaign(row scanner) (AdCampaign, error) {
	var campaign AdCampaign
	var brief, result, created string

	err := row.Scan(&campaign.ID, &campaign.Title, &brief, &result, &campaign.Model, &campaign.CostUSD, &created)
	if err != nil {
		return AdCampaign{}, err
	}

	campaign.Brief = json.RawMessage(brief)
	campaign.Result = json.RawMessage(result)
	campaign.CreatedAt = ParseTime(created)
	return campaign, nil
}
