package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type CompanyFile struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"-"`
	Name      string    `json:"name"`
	Mime      string    `json:"mime"`
	Size      int64     `json:"size"`
	Path      string    `json:"-"`
	Text      string    `json:"-"`
	HasText   bool      `json:"has_text"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) CreateCompanyFile(ctx context.Context, userID, companyID, name, mime, path, text string, size int64) (CompanyFile, error) {
	file := CompanyFile{
		ID: NewID(), CompanyID: companyID, Name: name, Mime: mime, Size: size,
		Path: path, Text: text, HasText: strings.TrimSpace(text) != "", CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_files (id, user_id, company_id, name, mime, size, path, text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.ID, userID, companyID, file.Name, file.Mime, file.Size, file.Path, file.Text,
		file.CreatedAt.Format(TimeFormat))
	if err != nil {
		return CompanyFile{}, err
	}
	return file, nil
}

func (s *Store) ListCompanyFiles(ctx context.Context, companyID string) ([]CompanyFile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, mime, size, path, text, created_at
		FROM company_files WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []CompanyFile{}
	for rows.Next() {
		var file CompanyFile
		var created string
		if err := rows.Scan(&file.ID, &file.Name, &file.Mime, &file.Size, &file.Path, &file.Text, &created); err != nil {
			return nil, err
		}
		file.CompanyID = companyID
		file.HasText = strings.TrimSpace(file.Text) != ""
		file.CreatedAt = ParseTime(created)
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Store) GetCompanyFile(ctx context.Context, companyID, id string) (CompanyFile, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, mime, size, path, text, created_at
		FROM company_files WHERE company_id = ? AND id = ?`, companyID, id)

	var file CompanyFile
	var created string
	if err := row.Scan(&file.ID, &file.Name, &file.Mime, &file.Size, &file.Path, &file.Text, &created); err != nil {
		if err == sql.ErrNoRows {
			return CompanyFile{}, ErrNotFound
		}
		return CompanyFile{}, err
	}
	file.CompanyID = companyID
	file.HasText = strings.TrimSpace(file.Text) != ""
	file.CreatedAt = ParseTime(created)
	return file, nil
}

func (s *Store) DeleteCompanyFile(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM company_files WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountCompanyFiles(ctx context.Context, companyID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM company_files WHERE company_id = ?`, companyID).Scan(&count)
	return count, err
}
