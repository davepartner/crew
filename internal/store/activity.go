package store

import (
	"context"
	"database/sql"
	"time"
)

type Activity struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Message   string    `json:"message"`
	TaskID    *string   `json:"task_id,omitempty"`
	AgentID   *string   `json:"agent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) AddActivity(ctx context.Context, userID, companyID, kind, message, taskID, agentID string) error {
	var task, agent any
	if taskID != "" {
		task = taskID
	}
	if agentID != "" {
		agent = agentID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activity (user_id, company_id, kind, message, task_id, agent_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, companyID, kind, message, task, agent, Now())

	return err
}

func (s *Store) ListActivity(ctx context.Context, companyID string, limit int) ([]Activity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, message, task_id, agent_id, created_at
		FROM activity WHERE company_id = ? ORDER BY id DESC LIMIT ?`, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Activity{}
	for rows.Next() {
		var item Activity
		var taskID, agentID sql.NullString
		var created string

		if err := rows.Scan(&item.ID, &item.Kind, &item.Message, &taskID, &agentID, &created); err != nil {
			return nil, err
		}

		if taskID.Valid {
			item.TaskID = &taskID.String
		}
		if agentID.Valid {
			item.AgentID = &agentID.String
		}
		item.CreatedAt = ParseTime(created)
		items = append(items, item)
	}

	return items, rows.Err()
}
