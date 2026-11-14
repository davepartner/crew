package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type OutreachSignal struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Found  bool   `json:"found"`
	Detail string `json:"detail"`
}

type OutreachProspect struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Website       string            `json:"website"`
	Domain        string            `json:"domain"`
	Industry      string            `json:"industry"`
	Location      string            `json:"location"`
	Source        string            `json:"source"`
	Stage         string            `json:"stage"`
	Score         int               `json:"score"`
	Signals       []OutreachSignal  `json:"signals"`
	Notes         string            `json:"notes"`
	LastContacted string            `json:"last_contacted"`
	CreatedAt     time.Time         `json:"created_at"`
	Contacts      []OutreachContact `json:"contacts,omitempty"`
}

type OutreachContact struct {
	ID          string    `json:"id"`
	ProspectID  string    `json:"prospect_id"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	EmailStatus string    `json:"email_status"`
	Source      string    `json:"source"`
	IsPrimary   bool      `json:"is_primary"`
	CreatedAt   time.Time `json:"created_at"`
}

type OutreachCampaign struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Offer      string         `json:"offer"`
	Audience   string         `json:"audience"`
	Tone       string         `json:"tone"`
	Status     string         `json:"status"`
	DailyLimit int            `json:"daily_limit"`
	Autopilot  bool           `json:"autopilot"`
	CreatedAt  time.Time      `json:"created_at"`
	Steps      []OutreachStep `json:"steps,omitempty"`
}

type OutreachStep struct {
	ID        string `json:"id"`
	Position  int    `json:"position"`
	DelayDays int    `json:"delay_days"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

type OutreachMessage struct {
	ID          string    `json:"id"`
	CampaignID  string    `json:"campaign_id"`
	ProspectID  string    `json:"prospect_id"`
	ContactID   string    `json:"contact_id"`
	Step        int       `json:"step"`
	To          string    `json:"to"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	ScheduledAt string    `json:"scheduled_at"`
	SentAt      string    `json:"sent_at"`
	ReplyText   string    `json:"reply_text"`
	CreatedAt   time.Time `json:"created_at"`
}

type OutreachSuppression struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type OutreachSettings struct {
	FromName   string    `json:"from_name"`
	FromEmail  string    `json:"from_email"`
	DailyLimit int       `json:"daily_limit"`
	Autopilot  bool      `json:"autopilot"`
	Signature  string    `json:"signature"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Store) CreateOutreachProspect(ctx context.Context, companyID string, prospect OutreachProspect) (OutreachProspect, error) {
	prospect.ID = NewID()
	prospect.CreatedAt = time.Now().UTC()
	if prospect.Stage == "" {
		prospect.Stage = "new"
	}
	if prospect.Source == "" {
		prospect.Source = "manual"
	}
	if prospect.Signals == nil {
		prospect.Signals = []OutreachSignal{}
	}
	signals, _ := json.Marshal(prospect.Signals)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_prospects (id, company_id, name, website, domain, industry, location, source, stage, score, signals, notes, last_contacted, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		prospect.ID, companyID, prospect.Name, prospect.Website, prospect.Domain, prospect.Industry,
		prospect.Location, prospect.Source, prospect.Stage, prospect.Score, string(signals),
		prospect.Notes, prospect.LastContacted, prospect.CreatedAt.Format(TimeFormat))
	return prospect, err
}

func (s *Store) ListOutreachProspects(ctx context.Context, companyID string) ([]OutreachProspect, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, website, domain, industry, location, source, stage, score, signals, notes, last_contacted, created_at
		FROM outreach_prospects WHERE company_id = ? ORDER BY score DESC, created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []OutreachProspect{}
	for rows.Next() {
		var prospect OutreachProspect
		var signals, created string
		if err := rows.Scan(&prospect.ID, &prospect.Name, &prospect.Website, &prospect.Domain,
			&prospect.Industry, &prospect.Location, &prospect.Source, &prospect.Stage,
			&prospect.Score, &signals, &prospect.Notes, &prospect.LastContacted, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(signals), &prospect.Signals)
		prospect.CreatedAt = ParseTime(created)
		out = append(out, prospect)
	}
	return out, rows.Err()
}

func (s *Store) GetOutreachProspect(ctx context.Context, companyID, id string) (OutreachProspect, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, website, domain, industry, location, source, stage, score, signals, notes, last_contacted, created_at
		FROM outreach_prospects WHERE company_id = ? AND id = ?`, companyID, id)
	var prospect OutreachProspect
	var signals, created string
	if err := row.Scan(&prospect.ID, &prospect.Name, &prospect.Website, &prospect.Domain,
		&prospect.Industry, &prospect.Location, &prospect.Source, &prospect.Stage,
		&prospect.Score, &signals, &prospect.Notes, &prospect.LastContacted, &created); err != nil {
		if err == sql.ErrNoRows {
			return OutreachProspect{}, ErrNotFound
		}
		return OutreachProspect{}, err
	}
	_ = json.Unmarshal([]byte(signals), &prospect.Signals)
	prospect.CreatedAt = ParseTime(created)
	return prospect, nil
}

func (s *Store) UpdateOutreachProspect(ctx context.Context, companyID, id string, prospect OutreachProspect) error {
	signals, _ := json.Marshal(prospect.Signals)
	_, err := s.db.ExecContext(ctx, `
		UPDATE outreach_prospects
		SET name = ?, website = ?, domain = ?, industry = ?, location = ?, source = ?, stage = ?,
		    score = ?, signals = ?, notes = ?, last_contacted = ?
		WHERE company_id = ? AND id = ?`,
		prospect.Name, prospect.Website, prospect.Domain, prospect.Industry, prospect.Location,
		prospect.Source, prospect.Stage, prospect.Score, string(signals), prospect.Notes,
		prospect.LastContacted, companyID, id)
	return err
}

func (s *Store) AddOutreachContact(ctx context.Context, companyID, prospectID string, contact OutreachContact) (OutreachContact, error) {
	contact.ID = NewID()
	contact.ProspectID = prospectID
	contact.CreatedAt = time.Now().UTC()
	if contact.EmailStatus == "" {
		contact.EmailStatus = "guess"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_contacts (id, company_id, prospect_id, name, role, email, email_status, source, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		contact.ID, companyID, prospectID, contact.Name, contact.Role, contact.Email,
		contact.EmailStatus, contact.Source, boolInt(contact.IsPrimary), contact.CreatedAt.Format(TimeFormat))
	return contact, err
}

func (s *Store) ListOutreachContacts(ctx context.Context, companyID, prospectID string) ([]OutreachContact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, prospect_id, name, role, email, email_status, source, is_primary, created_at
		FROM outreach_contacts WHERE company_id = ? AND prospect_id = ? ORDER BY is_primary DESC, created_at`, companyID, prospectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []OutreachContact{}
	for rows.Next() {
		var contact OutreachContact
		var primary int
		var created string
		if err := rows.Scan(&contact.ID, &contact.ProspectID, &contact.Name, &contact.Role,
			&contact.Email, &contact.EmailStatus, &contact.Source, &primary, &created); err != nil {
			return nil, err
		}
		contact.IsPrimary = primary == 1
		contact.CreatedAt = ParseTime(created)
		out = append(out, contact)
	}
	return out, rows.Err()
}

func (s *Store) CreateOutreachCampaign(ctx context.Context, companyID string, campaign OutreachCampaign) (OutreachCampaign, error) {
	campaign.ID = NewID()
	campaign.CreatedAt = time.Now().UTC()
	if campaign.Status == "" {
		campaign.Status = "draft"
	}
	if campaign.DailyLimit <= 0 {
		campaign.DailyLimit = 20
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_campaigns (id, company_id, name, offer, audience, tone, status, daily_limit, autopilot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaign.ID, companyID, campaign.Name, campaign.Offer, campaign.Audience, campaign.Tone,
		campaign.Status, campaign.DailyLimit, boolInt(campaign.Autopilot), campaign.CreatedAt.Format(TimeFormat))
	return campaign, err
}

func (s *Store) ListOutreachCampaigns(ctx context.Context, companyID string) ([]OutreachCampaign, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, offer, audience, tone, status, daily_limit, autopilot, created_at
		FROM outreach_campaigns WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []OutreachCampaign{}
	for rows.Next() {
		var campaign OutreachCampaign
		var autopilot int
		var created string
		if err := rows.Scan(&campaign.ID, &campaign.Name, &campaign.Offer, &campaign.Audience,
			&campaign.Tone, &campaign.Status, &campaign.DailyLimit, &autopilot, &created); err != nil {
			return nil, err
		}
		campaign.Autopilot = autopilot == 1
		campaign.CreatedAt = ParseTime(created)
		out = append(out, campaign)
	}
	return out, rows.Err()
}

func (s *Store) GetOutreachCampaign(ctx context.Context, companyID, id string) (OutreachCampaign, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, offer, audience, tone, status, daily_limit, autopilot, created_at
		FROM outreach_campaigns WHERE company_id = ? AND id = ?`, companyID, id)
	var campaign OutreachCampaign
	var autopilot int
	var created string
	if err := row.Scan(&campaign.ID, &campaign.Name, &campaign.Offer, &campaign.Audience,
		&campaign.Tone, &campaign.Status, &campaign.DailyLimit, &autopilot, &created); err != nil {
		if err == sql.ErrNoRows {
			return OutreachCampaign{}, ErrNotFound
		}
		return OutreachCampaign{}, err
	}
	campaign.Autopilot = autopilot == 1
	campaign.CreatedAt = ParseTime(created)
	return campaign, nil
}

func (s *Store) UpdateOutreachCampaign(ctx context.Context, companyID, id string, campaign OutreachCampaign) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE outreach_campaigns SET name = ?, offer = ?, audience = ?, tone = ?, status = ?, daily_limit = ?, autopilot = ?
		WHERE company_id = ? AND id = ?`,
		campaign.Name, campaign.Offer, campaign.Audience, campaign.Tone, campaign.Status,
		campaign.DailyLimit, boolInt(campaign.Autopilot), companyID, id)
	return err
}

func (s *Store) ReplaceOutreachSteps(ctx context.Context, companyID, campaignID string, steps []OutreachStep) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM outreach_steps WHERE company_id = ? AND campaign_id = ?`, companyID, campaignID); err != nil {
			return err
		}
		for index, step := range steps {
			if step.Position <= 0 {
				step.Position = index + 1
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO outreach_steps (id, company_id, campaign_id, position, delay_days, subject, body, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				NewID(), companyID, campaignID, step.Position, step.DelayDays, step.Subject, step.Body, Now()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListOutreachSteps(ctx context.Context, companyID, campaignID string) ([]OutreachStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, position, delay_days, subject, body FROM outreach_steps
		WHERE company_id = ? AND campaign_id = ? ORDER BY position`, companyID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutreachStep{}
	for rows.Next() {
		var step OutreachStep
		if err := rows.Scan(&step.ID, &step.Position, &step.DelayDays, &step.Subject, &step.Body); err != nil {
			return nil, err
		}
		out = append(out, step)
	}
	return out, rows.Err()
}

func (s *Store) CreateOutreachMessage(ctx context.Context, companyID string, message OutreachMessage) (OutreachMessage, error) {
	message.ID = NewID()
	message.CreatedAt = time.Now().UTC()
	if message.Status == "" {
		message.Status = "draft"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_messages (id, company_id, campaign_id, prospect_id, contact_id, step, to_email, subject, body, status, scheduled_at, sent_at, reply_text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.ID, companyID, message.CampaignID, message.ProspectID, message.ContactID, message.Step,
		message.To, message.Subject, message.Body, message.Status, message.ScheduledAt, message.SentAt,
		message.ReplyText, message.CreatedAt.Format(TimeFormat))
	return message, err
}

func (s *Store) ListOutreachMessages(ctx context.Context, companyID string) ([]OutreachMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, campaign_id, prospect_id, contact_id, step, to_email, subject, body, status, scheduled_at, sent_at, reply_text, created_at
		FROM outreach_messages WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutreachMessage{}
	for rows.Next() {
		message, err := scanOutreachMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, message)
	}
	return out, rows.Err()
}

func (s *Store) GetOutreachMessage(ctx context.Context, companyID, id string) (OutreachMessage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, campaign_id, prospect_id, contact_id, step, to_email, subject, body, status, scheduled_at, sent_at, reply_text, created_at
		FROM outreach_messages WHERE company_id = ? AND id = ?`, companyID, id)
	message, err := scanOutreachMessage(row)
	if err == sql.ErrNoRows {
		return OutreachMessage{}, ErrNotFound
	}
	return message, err
}

func (s *Store) UpdateOutreachMessageStatus(ctx context.Context, companyID, id, status, sentAt, replyText string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE outreach_messages SET status = ?, sent_at = CASE WHEN ? != '' THEN ? ELSE sent_at END,
		       reply_text = CASE WHEN ? != '' THEN ? ELSE reply_text END
		WHERE company_id = ? AND id = ?`,
		status, sentAt, sentAt, replyText, replyText, companyID, id)
	return err
}

func (s *Store) CountOutreachSentToday(ctx context.Context, companyID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM outreach_messages
		WHERE company_id = ? AND status = 'sent' AND date(sent_at) = date('now')`, companyID).Scan(&count)
	return count, err
}

func (s *Store) AddOutreachSuppression(ctx context.Context, companyID, email, reason string) (OutreachSuppression, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	suppression := OutreachSuppression{ID: NewID(), Email: email, Reason: reason, CreatedAt: time.Now().UTC()}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_suppressions (id, company_id, email, reason, created_at)
		VALUES (?, ?, ?, ?, ?)`, suppression.ID, companyID, email, reason, suppression.CreatedAt.Format(TimeFormat))
	return suppression, err
}

func (s *Store) ListOutreachSuppressions(ctx context.Context, companyID string) ([]OutreachSuppression, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, reason, created_at FROM outreach_suppressions WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutreachSuppression{}
	for rows.Next() {
		var suppression OutreachSuppression
		var created string
		if err := rows.Scan(&suppression.ID, &suppression.Email, &suppression.Reason, &created); err != nil {
			return nil, err
		}
		suppression.CreatedAt = ParseTime(created)
		out = append(out, suppression)
	}
	return out, rows.Err()
}

func (s *Store) IsOutreachSuppressed(ctx context.Context, companyID, email string) bool {
	var count int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM outreach_suppressions WHERE company_id = ? AND lower(email) = lower(?)`,
		companyID, strings.TrimSpace(email)).Scan(&count)
	return count > 0
}

func (s *Store) GetOutreachSettings(ctx context.Context, companyID string) (OutreachSettings, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT from_name, from_email, daily_limit, autopilot, signature, updated_at
		FROM outreach_settings WHERE company_id = ?`, companyID)
	var settings OutreachSettings
	var autopilot int
	var updated string
	if err := row.Scan(&settings.FromName, &settings.FromEmail, &settings.DailyLimit,
		&autopilot, &settings.Signature, &updated); err != nil {
		if err == sql.ErrNoRows {
			return OutreachSettings{DailyLimit: 20}, nil
		}
		return OutreachSettings{}, err
	}
	settings.Autopilot = autopilot == 1
	settings.UpdatedAt = ParseTime(updated)
	return settings, nil
}

func (s *Store) SaveOutreachSettings(ctx context.Context, companyID string, settings OutreachSettings) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outreach_settings (company_id, from_name, from_email, daily_limit, autopilot, signature, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(company_id) DO UPDATE SET from_name = excluded.from_name, from_email = excluded.from_email,
		  daily_limit = excluded.daily_limit, autopilot = excluded.autopilot, signature = excluded.signature,
		  updated_at = excluded.updated_at`,
		companyID, settings.FromName, settings.FromEmail, settings.DailyLimit, boolInt(settings.Autopilot),
		settings.Signature, Now())
	return err
}

func (s *Store) OutreachStats(ctx context.Context, companyID string) (map[string]int, error) {
	stats := map[string]int{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM outreach_messages WHERE company_id = ? GROUP BY status`, companyID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return stats, err
		}
		stats[status] = count
	}
	var prospects int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outreach_prospects WHERE company_id = ?`, companyID).Scan(&prospects)
	stats["prospects"] = prospects
	return stats, rows.Err()
}

func scanOutreachMessage(row scanner) (OutreachMessage, error) {
	var message OutreachMessage
	var created string
	if err := row.Scan(&message.ID, &message.CampaignID, &message.ProspectID, &message.ContactID,
		&message.Step, &message.To, &message.Subject, &message.Body, &message.Status,
		&message.ScheduledAt, &message.SentAt, &message.ReplyText, &created); err != nil {
		return OutreachMessage{}, err
	}
	message.CreatedAt = ParseTime(created)
	return message, nil
}
