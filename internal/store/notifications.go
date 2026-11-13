package store

import (
	"context"
	"database/sql"
	"time"
)

type Notification struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	TaskID    string     `json:"task_id"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s *Store) NotifyCompany(ctx context.Context, companyID, kind, title, body, taskID string) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id FROM companies WHERE id = ?
		UNION
		SELECT user_id FROM company_members WHERE company_id = ?`, companyID, companyID)
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

	for _, userID := range users {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO notifications (id, user_id, company_id, kind, title, body, task_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			NewID(), userID, companyID, kind, title, body, taskID, Now()); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) ListNotifications(ctx context.Context, userID, companyID string, limit int) ([]Notification, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, title, body, task_id, read_at, created_at
		FROM notifications
		WHERE user_id = ? AND company_id = ?
		ORDER BY created_at DESC LIMIT ?`, userID, companyID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []Notification{}
	for rows.Next() {
		var item Notification
		var read sql.NullString
		var created string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Body, &item.TaskID, &read, &created); err != nil {
			return nil, 0, err
		}
		if read.Valid {
			t := ParseTime(read.String)
			item.ReadAt = &t
		}
		item.CreatedAt = ParseTime(created)
		items = append(items, item)
	}

	var unread int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = ? AND company_id = ? AND read_at IS NULL`,
		userID, companyID).Scan(&unread)

	return items, unread, rows.Err()
}

func (s *Store) MarkNotification(ctx context.Context, userID, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL`,
		Now(), id, userID)
	return err
}

func (s *Store) MarkAllNotifications(ctx context.Context, userID, companyID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ? WHERE user_id = ? AND company_id = ? AND read_at IS NULL`,
		Now(), userID, companyID)
	return err
}
