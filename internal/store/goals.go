package store

import (
	"context"
	"database/sql"
	"time"
)

type Goal struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Target      int        `json:"target"`
	Current     int        `json:"current"`
	DueDate     string     `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

func (s *Store) CreateGoal(ctx context.Context, userID, companyID, title string, target int, dueDate string) (Goal, error) {
	if target < 1 {
		target = 1
	}

	goal := Goal{
		ID:        NewID(),
		Title:     title,
		Target:    target,
		Current:   0,
		DueDate:   dueDate,
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO goals (id, user_id, company_id, title, target, current, due_date, created_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)`,
		goal.ID, userID, companyID, goal.Title, goal.Target, goal.DueDate, goal.CreatedAt.Format(TimeFormat))
	if err != nil {
		return Goal{}, err
	}

	return goal, nil
}

func (s *Store) ListGoals(ctx context.Context, companyID string) ([]Goal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, target, current, due_date, created_at, completed_at
		FROM goals WHERE company_id = ?
		ORDER BY completed_at IS NOT NULL, due_date = '', due_date ASC, created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := []Goal{}
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func (s *Store) GetGoal(ctx context.Context, companyID, id string) (Goal, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, target, current, due_date, created_at, completed_at
		FROM goals WHERE company_id = ? AND id = ?`, companyID, id)

	goal, err := scanGoal(row)
	if isNotFound(err) {
		return Goal{}, ErrNotFound
	}
	return goal, err
}

type GoalPatch struct {
	Title   *string
	Target  *int
	Current *int
	DueDate *string
}

func (s *Store) UpdateGoal(ctx context.Context, companyID, id string, patch GoalPatch) (Goal, error) {
	current, err := s.GetGoal(ctx, companyID, id)
	if err != nil {
		return Goal{}, err
	}

	title, target, progress, due := current.Title, current.Target, current.Current, current.DueDate
	if patch.Title != nil {
		title = *patch.Title
	}
	if patch.Target != nil && *patch.Target > 0 {
		target = *patch.Target
	}
	if patch.Current != nil {
		progress = *patch.Current
	}
	if patch.DueDate != nil {
		due = *patch.DueDate
	}
	if progress < 0 {
		progress = 0
	}
	if progress > target {
		progress = target
	}

	var completed any
	if progress >= target {
		completed = Now()
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE goals SET title = ?, target = ?, current = ?, due_date = ?, completed_at = ?
		WHERE company_id = ? AND id = ?`,
		title, target, progress, due, completed, companyID, id)
	if err != nil {
		return Goal{}, err
	}

	return s.GetGoal(ctx, companyID, id)
}

func (s *Store) DeleteGoal(ctx context.Context, companyID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM goals WHERE company_id = ? AND id = ?`, companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanGoal(row scanner) (Goal, error) {
	var goal Goal
	var created string
	var completed sql.NullString

	err := row.Scan(&goal.ID, &goal.Title, &goal.Target, &goal.Current, &goal.DueDate, &created, &completed)
	if err != nil {
		return Goal{}, err
	}

	goal.CreatedAt = ParseTime(created)
	if completed.Valid {
		t := ParseTime(completed.String)
		goal.CompletedAt = &t
	}

	return goal, nil
}
