package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type Member struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	IsOwner bool   `json:"is_owner"`
}

type Invite struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) ListMembers(ctx context.Context, companyID string) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.email,
		       CASE WHEN c.user_id = u.id THEN 'owner' ELSE m.role END
		FROM companies c
		JOIN users u ON u.id = c.user_id
		LEFT JOIN company_members m ON m.company_id = c.id
		WHERE c.id = ?
		UNION
		SELECT u.id, u.name, u.email, m.role
		FROM company_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.company_id = ?
		ORDER BY 4 ASC, 2 ASC`, companyID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []Member{}
	for rows.Next() {
		var member Member
		if err := rows.Scan(&member.UserID, &member.Name, &member.Email, &member.Role); err != nil {
			return nil, err
		}
		member.IsOwner = member.Role == "owner"
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) AddMember(ctx context.Context, companyID, userID, role string) error {
	if role != "editor" && role != "viewer" {
		role = "editor"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_members (company_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id, user_id) DO UPDATE SET role = excluded.role`,
		companyID, userID, role, Now())
	return err
}

func (s *Store) RemoveMember(ctx context.Context, companyID, userID string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM company_members WHERE company_id = ? AND user_id = ?`, companyID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, external_id, email, name, created_at FROM users WHERE lower(email) = lower(?)`, strings.TrimSpace(email))

	var user User
	var created string
	if err := row.Scan(&user.ID, &user.ExternalID, &user.Email, &user.Name, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	user.CreatedAt = ParseTime(created)
	return user, nil
}

func (s *Store) CreateInvite(ctx context.Context, companyID, email, role string) (Invite, error) {
	if role != "editor" && role != "viewer" {
		role = "editor"
	}

	invite := Invite{ID: NewID(), Email: strings.ToLower(strings.TrimSpace(email)), Role: role, CreatedAt: Now()}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_invites (id, company_id, email, role, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(company_id, email) DO UPDATE SET role = excluded.role`,
		invite.ID, companyID, invite.Email, invite.Role, invite.CreatedAt)
	return invite, err
}

func (s *Store) ListInvites(ctx context.Context, companyID string) ([]Invite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, role, created_at FROM company_invites
		WHERE company_id = ? ORDER BY created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invites := []Invite{}
	for rows.Next() {
		var invite Invite
		if err := rows.Scan(&invite.ID, &invite.Email, &invite.Role, &invite.CreatedAt); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *Store) DeleteInvite(ctx context.Context, companyID, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM company_invites WHERE company_id = ? AND id = ?`, companyID, id)
	return err
}

// AcceptInvites turns pending invitations for this email into memberships.
func (s *Store) AcceptInvites(ctx context.Context, email, userID string) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, company_id, role FROM company_invites WHERE lower(email) = lower(?)`, strings.TrimSpace(email))
	if err != nil {
		return 0, err
	}

	type invite struct{ id, companyID, role string }
	var invites []invite
	for rows.Next() {
		var item invite
		if err := rows.Scan(&item.id, &item.companyID, &item.role); err != nil {
			rows.Close()
			return 0, err
		}
		invites = append(invites, item)
	}
	rows.Close()

	count := 0
	for _, item := range invites {
		if err := s.AddMember(ctx, item.companyID, userID, item.role); err != nil {
			return count, err
		}
		_, _ = s.db.ExecContext(ctx, `DELETE FROM company_invites WHERE id = ?`, item.id)
		count++
	}
	return count, nil
}
