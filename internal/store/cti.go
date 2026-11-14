package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type CTIAsset struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	Label       string    `json:"label"`
	Criticality string    `json:"criticality"`
	Source      string    `json:"source"`
	LastSeen    string    `json:"last_seen"`
	CreatedAt   time.Time `json:"created_at"`
}

type CTIIndicator struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Confidence  int       `json:"confidence"`
	TLP         string    `json:"tlp"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"source_url"`
	Status      string    `json:"status"`
	Tags        []string  `json:"tags"`
	FirstSeen   string    `json:"first_seen"`
	LastSeen    string    `json:"last_seen"`
	CreatedAt   time.Time `json:"created_at"`
}

type CTIActor struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Aliases        []string  `json:"aliases"`
	Motivation     string    `json:"motivation"`
	Sophistication string    `json:"sophistication"`
	Country        string    `json:"country"`
	Description    string    `json:"description"`
	Source         string    `json:"source"`
	Confidence     int       `json:"confidence"`
	CreatedAt      time.Time `json:"created_at"`
}

type CTIReport struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"source_url"`
	PublishedAt string    `json:"published_at"`
	Confidence  int       `json:"confidence"`
	TLP         string    `json:"tlp"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
}

type CTIRelationship struct {
	ID          string    `json:"id"`
	SourceType  string    `json:"source_type"`
	SourceID    string    `json:"source_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Relation    string    `json:"relation"`
	Confidence  int       `json:"confidence"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CTIFinding struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Evidence  string    `json:"evidence"`
	Fix       string    `json:"fix"`
	CreatedAt time.Time `json:"created_at"`
}

type CTIAssessment struct {
	Analysis  json.RawMessage `json:"analysis"`
	Model     string          `json:"model"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Store) EnsureCTIAsset(ctx context.Context, companyID, assetType, value, label, criticality, source string) (CTIAsset, error) {
	value = normalizeAsset(value)
	if value == "" {
		return CTIAsset{}, ErrNotFound
	}
	var existing CTIAsset
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, value, label, criticality, source, last_seen, created_at
		FROM cti_assets WHERE company_id = ? AND type = ? AND value = ?`,
		companyID, assetType, value).Scan(&existing.ID, &existing.Type, &existing.Value, &existing.Label,
		&existing.Criticality, &existing.Source, &existing.LastSeen, &existing.CreatedAt)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return CTIAsset{}, err
	}

	asset := CTIAsset{
		ID: NewID(), Type: assetType, Value: value, Label: label,
		Criticality: criticality, Source: source, LastSeen: Now(), CreatedAt: time.Now().UTC(),
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cti_assets (id, company_id, type, value, label, criticality, source, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		asset.ID, companyID, asset.Type, asset.Value, asset.Label, asset.Criticality, asset.Source,
		asset.LastSeen, asset.CreatedAt.Format(TimeFormat))
	return asset, err
}

func (s *Store) ListCTIAssets(ctx context.Context, companyID string) ([]CTIAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, value, label, criticality, source, last_seen, created_at
		FROM cti_assets WHERE company_id = ? ORDER BY created_at`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIAsset{}
	for rows.Next() {
		var asset CTIAsset
		var created string
		if err := rows.Scan(&asset.ID, &asset.Type, &asset.Value, &asset.Label, &asset.Criticality,
			&asset.Source, &asset.LastSeen, &created); err != nil {
			return nil, err
		}
		asset.CreatedAt = ParseTime(created)
		out = append(out, asset)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCTIAsset(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM cti_assets WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpsertCTIIndicator(ctx context.Context, companyID string, indicator CTIIndicator) (CTIIndicator, error) {
	var existing CTIIndicator
	var existingTags string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, value, name, description, confidence, tlp, source, source_url, status, tags,
		       first_seen, last_seen, created_at
		FROM cti_indicators WHERE company_id = ? AND type = ? AND value = ?`,
		companyID, indicator.Type, indicator.Value).Scan(&existing.ID, &existing.Type, &existing.Value,
		&existing.Name, &existing.Description, &existing.Confidence, &existing.TLP, &existing.Source,
		&existing.SourceURL, &existing.Status, &existingTags, &existing.FirstSeen,
		&existing.LastSeen, &existing.CreatedAt)
	_ = json.Unmarshal([]byte(existingTags), &existing.Tags)
	if err == nil {
		_, err = s.db.ExecContext(ctx, `
			UPDATE cti_indicators SET confidence = MAX(confidence, ?), last_seen = ?, description = ? WHERE id = ?`,
			indicator.Confidence, Now(), indicator.Description, existing.ID)
		return existing, err
	}
	if err != sql.ErrNoRows {
		return CTIIndicator{}, err
	}

	indicator.ID = NewID()
	if indicator.Status == "" {
		indicator.Status = "active"
	}
	if indicator.TLP == "" {
		indicator.TLP = "amber"
	}
	if indicator.FirstSeen == "" {
		indicator.FirstSeen = Now()
	}
	indicator.LastSeen = Now()
	indicator.CreatedAt = time.Now().UTC()
	tags, _ := json.Marshal(indicator.Tags)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cti_indicators (id, company_id, type, value, name, description, confidence, tlp, source,
		 source_url, status, tags, first_seen, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		indicator.ID, companyID, indicator.Type, indicator.Value, indicator.Name, indicator.Description,
		indicator.Confidence, indicator.TLP, indicator.Source, indicator.SourceURL, indicator.Status,
		string(tags), indicator.FirstSeen, indicator.LastSeen, indicator.CreatedAt.Format(TimeFormat))
	return indicator, err
}

func (s *Store) ListCTIIndicators(ctx context.Context, companyID string) ([]CTIIndicator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, value, name, description, confidence, tlp, source, source_url, status, tags,
		       first_seen, last_seen, created_at
		FROM cti_indicators WHERE company_id = ? ORDER BY confidence DESC, created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIIndicator{}
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, indicator)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCTIIndicatorStatus(ctx context.Context, companyID, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE cti_indicators SET status = ? WHERE company_id = ? AND id = ?`, status, companyID, id)
	return err
}

func (s *Store) DeleteCTIIndicator(ctx context.Context, companyID, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cti_indicators WHERE company_id = ? AND id = ?`, companyID, id)
	return err
}

func (s *Store) CreateCTIReport(ctx context.Context, companyID string, report CTIReport) (CTIReport, error) {
	report.ID = NewID()
	report.CreatedAt = time.Now().UTC()
	if report.TLP == "" {
		report.TLP = "amber"
	}
	tags, _ := json.Marshal(report.Tags)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cti_reports (id, company_id, title, summary, source, source_url, published_at, confidence, tlp, tags, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, companyID, report.Title, report.Summary, report.Source, report.SourceURL,
		report.PublishedAt, report.Confidence, report.TLP, string(tags), report.CreatedAt.Format(TimeFormat))
	return report, err
}

func (s *Store) ListCTIReports(ctx context.Context, companyID string) ([]CTIReport, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, summary, source, source_url, published_at, confidence, tlp, tags, created_at
		FROM cti_reports WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIReport{}
	for rows.Next() {
		var report CTIReport
		var tags string
		var created string
		if err := rows.Scan(&report.ID, &report.Title, &report.Summary, &report.Source, &report.SourceURL,
			&report.PublishedAt, &report.Confidence, &report.TLP, &tags, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &report.Tags)
		report.CreatedAt = ParseTime(created)
		out = append(out, report)
	}
	return out, rows.Err()
}

func (s *Store) CreateCTIActor(ctx context.Context, companyID string, actor CTIActor) (CTIActor, error) {
	actor.ID = NewID()
	actor.CreatedAt = time.Now().UTC()
	aliases, _ := json.Marshal(actor.Aliases)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cti_actors (id, company_id, name, aliases, motivation, sophistication, country, description, source, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		actor.ID, companyID, actor.Name, string(aliases), actor.Motivation, actor.Sophistication,
		actor.Country, actor.Description, actor.Source, actor.Confidence, actor.CreatedAt.Format(TimeFormat))
	return actor, err
}

func (s *Store) ListCTIActors(ctx context.Context, companyID string) ([]CTIActor, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, aliases, motivation, sophistication, country, description, source, confidence, created_at
		FROM cti_actors WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIActor{}
	for rows.Next() {
		var actor CTIActor
		var aliases string
		var created string
		if err := rows.Scan(&actor.ID, &actor.Name, &aliases, &actor.Motivation, &actor.Sophistication,
			&actor.Country, &actor.Description, &actor.Source, &actor.Confidence, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(aliases), &actor.Aliases)
		actor.CreatedAt = ParseTime(created)
		out = append(out, actor)
	}
	return out, rows.Err()
}

func (s *Store) CreateCTIRelationship(ctx context.Context, companyID string, rel CTIRelationship) (CTIRelationship, error) {
	rel.ID = NewID()
	rel.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cti_relationships (id, company_id, source_type, source_id, target_type, target_id, relation, confidence, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rel.ID, companyID, rel.SourceType, rel.SourceID, rel.TargetType, rel.TargetID, rel.Relation,
		rel.Confidence, rel.Description, rel.CreatedAt.Format(TimeFormat))
	return rel, err
}

func (s *Store) ListCTIRelationships(ctx context.Context, companyID string) ([]CTIRelationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_type, source_id, target_type, target_id, relation, confidence, description, created_at
		FROM cti_relationships WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIRelationship{}
	for rows.Next() {
		var rel CTIRelationship
		var created string
		if err := rows.Scan(&rel.ID, &rel.SourceType, &rel.SourceID, &rel.TargetType, &rel.TargetID,
			&rel.Relation, &rel.Confidence, &rel.Description, &created); err != nil {
			return nil, err
		}
		rel.CreatedAt = ParseTime(created)
		out = append(out, rel)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceCTIFindings(ctx context.Context, companyID string, findings []CTIFinding) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM cti_findings WHERE company_id = ?`, companyID); err != nil {
			return err
		}
		for _, finding := range findings {
			finding.ID = NewID()
			finding.CreatedAt = time.Now().UTC()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO cti_findings (id, company_id, category, severity, title, detail, evidence, fix, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				finding.ID, companyID, finding.Category, finding.Severity, finding.Title,
				finding.Detail, finding.Evidence, finding.Fix, finding.CreatedAt.Format(TimeFormat)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListCTIFindings(ctx context.Context, companyID string) ([]CTIFinding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category, severity, title, detail, evidence, fix, created_at
		FROM cti_findings WHERE company_id = ? ORDER BY
		  CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
		  created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CTIFinding{}
	for rows.Next() {
		var finding CTIFinding
		var created string
		if err := rows.Scan(&finding.ID, &finding.Category, &finding.Severity, &finding.Title,
			&finding.Detail, &finding.Evidence, &finding.Fix, &created); err != nil {
			return nil, err
		}
		finding.CreatedAt = ParseTime(created)
		out = append(out, finding)
	}
	return out, rows.Err()
}

func (s *Store) GetCTIAssessment(ctx context.Context, companyID string) (CTIAssessment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT analysis, model, created_at FROM cti_assessments WHERE company_id = ?`, companyID)
	var assessment CTIAssessment
	var analysis, created string
	if err := row.Scan(&analysis, &assessment.Model, &created); err != nil {
		if err == sql.ErrNoRows {
			return CTIAssessment{}, ErrNotFound
		}
		return CTIAssessment{}, err
	}
	assessment.Analysis = json.RawMessage(analysis)
	assessment.CreatedAt = ParseTime(created)
	return assessment, nil
}

func (s *Store) SaveCTIAssessment(ctx context.Context, companyID string, analysis []byte, model string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cti_assessments (company_id, analysis, model, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id) DO UPDATE SET analysis = excluded.analysis, model = excluded.model, created_at = excluded.created_at`,
		companyID, string(analysis), model, Now())
	return err
}

func scanIndicator(row scanner) (CTIIndicator, error) {
	var indicator CTIIndicator
	var tags, created string
	if err := row.Scan(&indicator.ID, &indicator.Type, &indicator.Value, &indicator.Name, &indicator.Description,
		&indicator.Confidence, &indicator.TLP, &indicator.Source, &indicator.SourceURL, &indicator.Status,
		&tags, &indicator.FirstSeen, &indicator.LastSeen, &created); err != nil {
		return CTIIndicator{}, err
	}
	_ = json.Unmarshal([]byte(tags), &indicator.Tags)
	indicator.CreatedAt = ParseTime(created)
	return indicator, nil
}

func normalizeAsset(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
