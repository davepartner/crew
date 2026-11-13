package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Routine struct {
	ID        string      `json:"id"`
	UserID    string      `json:"-"`
	CompanyID string      `json:"-"`
	Title     string      `json:"title"`
	Details   string      `json:"details"`
	AgentID   string      `json:"agent_id"`
	Agent     *AgentBrief `json:"agent"`
	Cadence   string      `json:"cadence"`
	Enabled   bool        `json:"enabled"`
	NextRunAt time.Time   `json:"next_run_at"`
	LastRunAt *time.Time  `json:"last_run_at"`
	CreatedAt time.Time   `json:"created_at"`
}

const (
	CadenceDaily    = "daily"
	CadenceWeekdays = "weekdays"
	CadenceWeekly   = "weekly"
	CadenceMonthly  = "monthly"
)

func ValidCadence(value string) bool {
	switch value {
	case CadenceDaily, CadenceWeekdays, CadenceWeekly, CadenceMonthly:
		return true
	default:
		return false
	}
}

// NextRun keeps the cadence honest: a missed run does not cause a burst of
// catch-up tasks, it simply moves to the next slot from now.
func NextRun(cadence string, from time.Time) time.Time {
	switch cadence {
	case CadenceWeekdays:
		next := from.AddDate(0, 0, 1)
		for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
			next = next.AddDate(0, 0, 1)
		}
		return next
	case CadenceWeekly:
		return from.AddDate(0, 0, 7)
	case CadenceMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}

func (s *Store) CreateRoutine(ctx context.Context, userID, companyID, title, details, agentID, cadence string) (Routine, error) {
	now := time.Now().UTC()
	routine := Routine{
		ID:        NewID(),
		UserID:    userID,
		CompanyID: companyID,
		Title:     title,
		Details:   details,
		AgentID:   agentID,
		Cadence:   cadence,
		Enabled:   true,
		NextRunAt: NextRun(cadence, now),
		CreatedAt: now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO routines (id, user_id, company_id, title, details, agent_id, cadence, enabled, next_run_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		routine.ID, userID, companyID, title, details, agentID, cadence,
		routine.NextRunAt.Format(TimeFormat), routine.CreatedAt.Format(TimeFormat))
	if err != nil {
		return Routine{}, err
	}

	return routine, nil
}

func (s *Store) ListRoutines(ctx context.Context, companyID string) ([]Routine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.user_id, r.company_id, r.title, r.details, r.agent_id, r.cadence,
		       r.enabled, r.next_run_at, r.last_run_at, r.created_at,
		       a.name, a.emoji, a.role
		FROM routines r
		LEFT JOIN agents a ON a.id = r.agent_id
		WHERE r.company_id = ?
		ORDER BY r.enabled DESC, r.next_run_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routines := []Routine{}
	for rows.Next() {
		routine, err := scanRoutine(rows)
		if err != nil {
			return nil, err
		}
		routines = append(routines, routine)
	}

	return routines, rows.Err()
}

func (s *Store) GetRoutine(ctx context.Context, companyID, id string) (Routine, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.user_id, r.company_id, r.title, r.details, r.agent_id, r.cadence,
		       r.enabled, r.next_run_at, r.last_run_at, r.created_at,
		       a.name, a.emoji, a.role
		FROM routines r
		LEFT JOIN agents a ON a.id = r.agent_id
		WHERE r.company_id = ? AND r.id = ?`, companyID, id)

	routine, err := scanRoutine(row)
	if isNotFound(err) {
		return Routine{}, ErrNotFound
	}
	return routine, err
}

type RoutinePatch struct {
	Title   *string
	Details *string
	AgentID *string
	Cadence *string
	Enabled *bool
}

func (s *Store) UpdateRoutine(ctx context.Context, companyID, id string, patch RoutinePatch) (Routine, error) {
	current, err := s.GetRoutine(ctx, companyID, id)
	if err != nil {
		return Routine{}, err
	}

	title, details, agentID, cadence, enabled := current.Title, current.Details, current.AgentID, current.Cadence, current.Enabled
	nextRun := current.NextRunAt

	if patch.Title != nil {
		title = *patch.Title
	}
	if patch.Details != nil {
		details = *patch.Details
	}
	if patch.AgentID != nil {
		agentID = *patch.AgentID
	}
	if patch.Cadence != nil && ValidCadence(*patch.Cadence) {
		cadence = *patch.Cadence
		nextRun = NextRun(cadence, time.Now().UTC())
	}
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE routines SET title = ?, details = ?, agent_id = ?, cadence = ?, enabled = ?, next_run_at = ?
		WHERE company_id = ? AND id = ?`,
		title, details, agentID, cadence, boolInt(enabled), nextRun.Format(TimeFormat), companyID, id)
	if err != nil {
		return Routine{}, err
	}

	return s.GetRoutine(ctx, companyID, id)
}

func (s *Store) DeleteRoutine(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM routines WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// RunDueRoutines creates one task per due routine and advances its next slot.
// The conditional update is the claim, so two schedulers cannot double-run.
func (s *Store) RunDueRoutines(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, company_id, title, details, agent_id, cadence, next_run_at
		FROM routines
		WHERE enabled = 1 AND next_run_at <= ?
		ORDER BY next_run_at ASC
		LIMIT 50`, now.Format(TimeFormat))
	if err != nil {
		return 0, err
	}

	type due struct {
		id, userID, companyID, title, details, agentID, cadence string
	}
	var items []due
	for rows.Next() {
		var item due
		var next string
		if err := rows.Scan(&item.id, &item.userID, &item.companyID, &item.title, &item.details,
			&item.agentID, &item.cadence, &next); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, item := range items {
		next := NextRun(item.cadence, now)

		res, err := s.db.ExecContext(ctx, `
			UPDATE routines SET next_run_at = ?, last_run_at = ?
			WHERE id = ? AND next_run_at <= ?`,
			next.Format(TimeFormat), now.Format(TimeFormat), item.id, now.Format(TimeFormat))
		if err != nil {
			return count, err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			continue
		}

		if _, err := s.CreateTask(ctx, item.userID, item.companyID, item.title, item.details, item.agentID); err != nil {
			if errors.Is(err, ErrTaskBudget) {
				_ = s.AddActivity(ctx, item.userID, item.companyID, "routine_skipped",
					"Routine skipped — monthly task budget reached", "", item.agentID)
				continue
			}
			return count, err
		}

		_ = s.AddActivity(ctx, item.userID, item.companyID, "routine_ran",
			"Routine ran: "+item.title, "", item.agentID)
		count++
	}

	return count, nil
}

func scanRoutine(row scanner) (Routine, error) {
	var routine Routine
	var enabled int
	var next, created string
	var last sql.NullString
	var agentName, agentEmoji, agentRole sql.NullString

	err := row.Scan(&routine.ID, &routine.UserID, &routine.CompanyID, &routine.Title, &routine.Details,
		&routine.AgentID, &routine.Cadence, &enabled, &next, &last, &created,
		&agentName, &agentEmoji, &agentRole)
	if err != nil {
		return Routine{}, err
	}

	routine.Enabled = enabled == 1
	routine.NextRunAt = ParseTime(next)
	routine.CreatedAt = ParseTime(created)
	if last.Valid {
		t := ParseTime(last.String)
		routine.LastRunAt = &t
	}
	if routine.AgentID != "" && agentName.Valid {
		routine.Agent = &AgentBrief{
			ID: routine.AgentID, Name: agentName.String, Emoji: agentEmoji.String, Role: agentRole.String,
		}
	}

	return routine, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
