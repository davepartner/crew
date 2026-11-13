package store

import (
	"context"
	"database/sql"
	"time"
)

type FinancialSnapshot struct {
	ID          string    `json:"id"`
	Period      string    `json:"period"`
	Revenue     float64   `json:"revenue"`
	COGS        float64   `json:"cogs"`
	Opex        float64   `json:"opex"`
	Cash        float64   `json:"cash"`
	Receivables float64   `json:"receivables"`
	Payables    float64   `json:"payables"`
	Source      string    `json:"source"`
	Audited     bool      `json:"audited"`
	AuditStatus string    `json:"audit_status"`
	AuditNote   string    `json:"audit_note"`
	Auditor     string    `json:"auditor"`
	AuditFileID string    `json:"audit_file_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Store) CreateFinancialSnapshot(ctx context.Context, companyID string, snapshot FinancialSnapshot) (FinancialSnapshot, error) {
	snapshot.ID = NewID()
	snapshot.AuditStatus = "self_reported"
	snapshot.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO financial_snapshots (id, company_id, period, revenue, cogs, opex, cash, receivables, payables, source, audited, audit_status, audit_note, auditor, audit_file_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.ID, companyID, snapshot.Period, snapshot.Revenue, snapshot.COGS, snapshot.Opex,
		snapshot.Cash, snapshot.Receivables, snapshot.Payables, snapshot.Source, boolInt(snapshot.Audited),
		"self_reported", "", "", "", snapshot.CreatedAt.Format(TimeFormat))
	if err != nil {
		return FinancialSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) ListFinancialSnapshots(ctx context.Context, companyID string) ([]FinancialSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, period, revenue, cogs, opex, cash, receivables, payables, source, audited,
		       audit_status, audit_note, auditor, audit_file_id, created_at
		FROM financial_snapshots WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []FinancialSnapshot{}
	for rows.Next() {
		snapshot, err := scanFinancialSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) LatestFinancialSnapshot(ctx context.Context, companyID string) (FinancialSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, period, revenue, cogs, opex, cash, receivables, payables, source, audited,
		       audit_status, audit_note, auditor, audit_file_id, created_at
		FROM financial_snapshots WHERE company_id = ? ORDER BY created_at DESC LIMIT 1`, companyID)
	snapshot, err := scanFinancialSnapshot(row)
	if isNotFound(err) {
		return FinancialSnapshot{}, ErrNotFound
	}
	return snapshot, err
}

func (s *Store) GetFinancialSnapshot(ctx context.Context, companyID, id string) (FinancialSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, period, revenue, cogs, opex, cash, receivables, payables, source, audited,
		       audit_status, audit_note, auditor, audit_file_id, created_at
		FROM financial_snapshots WHERE company_id = ? AND id = ?`, companyID, id)
	snapshot, err := scanFinancialSnapshot(row)
	if isNotFound(err) {
		return FinancialSnapshot{}, ErrNotFound
	}
	return snapshot, err
}

func (s *Store) SaveFinancialAudit(ctx context.Context, companyID, id, fileID, status, note, auditor string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE financial_snapshots
		SET audit_status = ?, audit_note = ?, auditor = ?, audit_file_id = ?
		WHERE company_id = ? AND id = ?`, status, note, auditor, fileID, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteFinancialSnapshot(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM financial_snapshots WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanFinancialSnapshot(row scanner) (FinancialSnapshot, error) {
	var snapshot FinancialSnapshot
	var audited int
	var created string
	err := row.Scan(&snapshot.ID, &snapshot.Period, &snapshot.Revenue, &snapshot.COGS, &snapshot.Opex,
		&snapshot.Cash, &snapshot.Receivables, &snapshot.Payables, &snapshot.Source, &audited,
		&snapshot.AuditStatus, &snapshot.AuditNote, &snapshot.Auditor, &snapshot.AuditFileID, &created)
	if err != nil {
		if err == sql.ErrNoRows {
			return FinancialSnapshot{}, ErrNotFound
		}
		return FinancialSnapshot{}, err
	}
	snapshot.Audited = audited == 1
	snapshot.CreatedAt = ParseTime(created)
	return snapshot, nil
}
