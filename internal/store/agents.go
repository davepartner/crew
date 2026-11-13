package store

import (
	"context"
	"database/sql"
	"time"
)

type Agent struct {
	ID            string     `json:"id"`
	UserID        string     `json:"-"`
	CompanyID     string     `json:"company_id,omitempty"`
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	Emoji         string     `json:"emoji"`
	Instructions  string     `json:"instructions"`
	ModelProvider string     `json:"model_provider"`
	ModelName     string     `json:"model_name"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (s *Store) CreateAgent(ctx context.Context, userID, companyID, name, role, emoji, instructions, keyHash string) (Agent, error) {
	agent := Agent{
		ID:           NewID(),
		UserID:       userID,
		CompanyID:    companyID,
		Name:         name,
		Role:         role,
		Emoji:        emoji,
		Instructions: instructions,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, user_id, company_id, name, role, emoji, instructions, api_key_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID, userID, companyID, agent.Name, agent.Role, agent.Emoji, agent.Instructions,
		keyHash, agent.CreatedAt.Format(TimeFormat),
	)
	if err != nil {
		return Agent{}, err
	}

	return agent, nil
}

func (s *Store) ListAgents(ctx context.Context, companyID string) ([]Agent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, company_id, name, role, emoji, instructions, model_provider, model_name, last_seen_at, created_at
		FROM agents WHERE company_id = ? ORDER BY created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, rows.Err()
}

func (s *Store) CountAgents(ctx context.Context, companyID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents WHERE company_id = ?`, companyID).Scan(&count)
	return count, err
}

func (s *Store) GetAgent(ctx context.Context, companyID, id string) (Agent, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, company_id, name, role, emoji, instructions, model_provider, model_name, last_seen_at, created_at
		FROM agents WHERE company_id = ? AND id = ?`, companyID, id)

	agent, err := scanAgent(row)
	if isNotFound(err) {
		return Agent{}, ErrNotFound
	}
	return agent, err
}

func (s *Store) UpdateAgent(ctx context.Context, companyID, id, name, role, emoji, instructions, modelProvider, modelName string) (Agent, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET name = ?, role = ?, emoji = ?, instructions = ?, model_provider = ?, model_name = ?
		WHERE company_id = ? AND id = ?`,
		name, role, emoji, instructions, modelProvider, modelName, companyID, id)
	if err != nil {
		return Agent{}, err
	}

	return s.GetAgent(ctx, companyID, id)
}

func (s *Store) DeleteAgent(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RotateAgentKey(ctx context.Context, companyID, id, keyHash string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE agents SET api_key_hash = ? WHERE company_id = ? AND id = ?`, keyHash, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// AgentForKeyHash resolves the bearer credential a helper presents to the
// runner API. The key itself is the scope.
func (s *Store) AgentForKeyHash(ctx context.Context, keyHash string) (Agent, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, company_id, name, role, emoji, instructions, model_provider, model_name, last_seen_at, created_at
		FROM agents WHERE api_key_hash = ?`, keyHash)

	agent, err := scanAgent(row)
	if isNotFound(err) {
		return Agent{}, ErrNotFound
	}
	return agent, err
}

func (s *Store) TouchAgent(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET last_seen_at = ? WHERE id = ?`, Now(), agentID)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(row scanner) (Agent, error) {
	var agent Agent
	var lastSeen sql.NullString
	var created string

	err := row.Scan(&agent.ID, &agent.UserID, &agent.CompanyID, &agent.Name, &agent.Role, &agent.Emoji,
		&agent.Instructions, &agent.ModelProvider, &agent.ModelName, &lastSeen, &created)
	if err != nil {
		return Agent{}, err
	}

	if lastSeen.Valid {
		t := ParseTime(lastSeen.String)
		agent.LastSeenAt = &t
	}
	agent.CreatedAt = ParseTime(created)

	return agent, nil
}
