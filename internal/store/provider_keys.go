package store

import (
	"context"
	"time"
)

type ProviderKey struct {
	Provider  string    `json:"provider"`
	Last4     string    `json:"last4"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) UpsertProviderKey(ctx context.Context, userID, provider, cipherText, last4 string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_keys (id, user_id, provider, key_cipher, last4, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			key_cipher = excluded.key_cipher,
			last4 = excluded.last4,
			created_at = excluded.created_at`,
		NewID(), userID, provider, cipherText, last4, Now())
	return err
}

func (s *Store) ListProviderKeys(ctx context.Context, userID string) ([]ProviderKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider, last4, created_at FROM provider_keys
		WHERE user_id = ? ORDER BY provider ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []ProviderKey{}
	for rows.Next() {
		var key ProviderKey
		var created string
		if err := rows.Scan(&key.Provider, &key.Last4, &created); err != nil {
			return nil, err
		}
		key.CreatedAt = ParseTime(created)
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

func (s *Store) ProviderKeyCipher(ctx context.Context, userID, provider string) (string, error) {
	var cipherText string
	err := s.db.QueryRowContext(ctx, `
		SELECT key_cipher FROM provider_keys WHERE user_id = ? AND provider = ?`,
		userID, provider).Scan(&cipherText)
	if isNotFound(err) {
		return "", ErrNotFound
	}
	return cipherText, err
}

func (s *Store) DeleteProviderKey(ctx context.Context, userID, provider string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM provider_keys WHERE user_id = ? AND provider = ?`, userID, provider)
	return err
}
