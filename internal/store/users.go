package store

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID         string    `json:"id"`
	ExternalID string    `json:"-"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) UpsertUser(ctx context.Context, externalID, email, name string) (User, error) {
	var row struct {
		id, externalID, email, name, createdAt string
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (id, external_id, email, name, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(external_id) DO UPDATE SET email = excluded.email, name = excluded.name
		RETURNING id, external_id, email, name, created_at`,
		NewID(), externalID, email, name, Now(),
	).Scan(&row.id, &row.externalID, &row.email, &row.name, &row.createdAt)

	if err != nil {
		return User{}, err
	}

	return User{
		ID:         row.id,
		ExternalID: row.externalID,
		Email:      row.email,
		Name:       row.name,
		CreatedAt:  ParseTime(row.createdAt),
	}, nil
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	var row struct {
		id, externalID, email, name, createdAt string
	}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, external_id, email, name, created_at FROM users WHERE id = ?`, id,
	).Scan(&row.id, &row.externalID, &row.email, &row.name, &row.createdAt)

	if isNotFound(err) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}

	return User{
		ID:         row.id,
		ExternalID: row.externalID,
		Email:      row.email,
		Name:       row.name,
		CreatedAt:  ParseTime(row.createdAt),
	}, nil
}

func (s *Store) UpdateUserName(ctx context.Context, id, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUserData removes everything Crew owns for a person. The sign-in account
// itself lives in PayTrackify and is intentionally left alone.
func (s *Store) DeleteUserData(ctx context.Context, id string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM activity WHERE user_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM agents WHERE user_id = ?`, id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
		return err
	})
}
