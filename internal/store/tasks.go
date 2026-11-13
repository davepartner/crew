package store

import (
	"context"
	"database/sql"
	"time"
)

const (
	StatusTodo    = "todo"
	StatusWorking = "working"
	StatusReview  = "review"
	StatusDone    = "done"
)

type AgentBrief struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Role  string `json:"role"`
}

type Comment struct {
	ID         string    `json:"id"`
	AuthorType string    `json:"author_type"`
	AuthorName string    `json:"author_name"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

type Task struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Details     string      `json:"details"`
	Status      string      `json:"status"`
	AgentID     *string     `json:"agent_id"`
	Agent       *AgentBrief `json:"agent"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	CompletedAt *time.Time  `json:"completed_at"`
	Comments    []Comment   `json:"comments,omitempty"`
}

type TaskPatch struct {
	Title   *string
	Details *string
	Status  *string
	AgentID *string
	Clear   bool // when true, AgentID is set to NULL
}

func (s *Store) CreateTask(ctx context.Context, userID, title, details, agentID string) (Task, error) {
	now := Now()
	id := NewID()

	var agent any
	if agentID != "" {
		agent = agentID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, user_id, title, details, status, agent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, title, details, StatusTodo, agent, now, now)
	if err != nil {
		return Task{}, err
	}

	return s.GetTask(ctx, userID, id)
}

func (s *Store) ListTasks(ctx context.Context, userID, status string) ([]Task, error) {
	query := `
		SELECT t.id, t.title, t.details, t.status, t.agent_id, t.created_at, t.updated_at, t.completed_at,
		       a.name, a.emoji, a.role
		FROM tasks t
		LEFT JOIN agents a ON a.id = t.agent_id
		WHERE t.user_id = ?`
	args := []any{userID}

	if status != "" {
		query += ` AND t.status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY
		CASE t.status WHEN 'review' THEN 0 WHEN 'working' THEN 1 WHEN 'todo' THEN 2 ELSE 3 END,
		t.updated_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (s *Store) GetTask(ctx context.Context, userID, id string) (Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.title, t.details, t.status, t.agent_id, t.created_at, t.updated_at, t.completed_at,
		       a.name, a.emoji, a.role
		FROM tasks t
		LEFT JOIN agents a ON a.id = t.agent_id
		WHERE t.user_id = ? AND t.id = ?`, userID, id)

	task, err := scanTask(row)
	if isNotFound(err) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}

	task.Comments, err = s.listComments(ctx, task.ID)
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func (s *Store) UpdateTask(ctx context.Context, userID, id string, patch TaskPatch) (Task, error) {
	current, err := s.GetTask(ctx, userID, id)
	if err != nil {
		return Task{}, err
	}

	title, details, status, agentID := current.Title, current.Details, current.Status, current.AgentID

	if patch.Title != nil {
		title = *patch.Title
	}
	if patch.Details != nil {
		details = *patch.Details
	}
	if patch.Status != nil {
		status = *patch.Status
	}
	if patch.Clear {
		agentID = nil
	} else if patch.AgentID != nil {
		value := *patch.AgentID
		agentID = &value
	}

	var completed any
	if status == StatusDone {
		completed = Now()
	}

	var agent any
	if agentID != nil {
		agent = *agentID
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE tasks SET title = ?, details = ?, status = ?, agent_id = ?, updated_at = ?, completed_at = ?
		WHERE user_id = ? AND id = ?`,
		title, details, status, agent, Now(), completed, userID, id)
	if err != nil {
		return Task{}, err
	}

	return s.GetTask(ctx, userID, id)
}

func (s *Store) SetTaskStatus(ctx context.Context, userID, id, status string) (Task, error) {
	return s.UpdateTask(ctx, userID, id, TaskPatch{Status: &status})
}

func (s *Store) DeleteTask(ctx context.Context, userID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AddComment(ctx context.Context, taskID, authorType, authorName, body string) (Comment, error) {
	comment := Comment{
		ID:         NewID(),
		AuthorType: authorType,
		AuthorName: authorName,
		Body:       body,
		CreatedAt:  time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (id, task_id, author_type, author_name, body, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		comment.ID, taskID, comment.AuthorType, comment.AuthorName, comment.Body,
		comment.CreatedAt.Format(TimeFormat))
	if err != nil {
		return Comment{}, err
	}

	return comment, nil
}

// ClaimNextTask hands the oldest waiting task to a helper. The conditional
// UPDATE is what makes the checkout atomic: two workers racing for the same
// row cannot both win.
func (s *Store) ClaimNextTask(ctx context.Context, userID, agentID string) (*Task, error) {
	var taskID string

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM tasks
			WHERE user_id = ? AND agent_id = ? AND status = ?
			ORDER BY created_at ASC LIMIT 1`, userID, agentID, StatusTodo).Scan(&taskID)
		if isNotFound(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks SET status = ?, updated_at = ?
			WHERE id = ? AND status = ?`, StatusWorking, Now(), taskID, StatusTodo)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrNotFound
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	task, err := s.GetTask(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *Store) StatusCounts(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM tasks WHERE user_id = ? GROUP BY status`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{
		StatusTodo:    0,
		StatusWorking: 0,
		StatusReview:  0,
		StatusDone:    0,
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

func (s *Store) OpenTasksForAgent(ctx context.Context, agentID string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.title, t.details, t.status, t.agent_id, t.created_at, t.updated_at, t.completed_at,
		       a.name, a.emoji, a.role
		FROM tasks t
		LEFT JOIN agents a ON a.id = t.agent_id
		WHERE t.agent_id = ? AND t.status IN (?, ?)
		ORDER BY t.updated_at DESC`, agentID, StatusTodo, StatusWorking)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (s *Store) listComments(ctx context.Context, taskID string) ([]Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_type, author_name, body, created_at
		FROM comments WHERE task_id = ? ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		var created string
		if err := rows.Scan(&c.ID, &c.AuthorType, &c.AuthorName, &c.Body, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = ParseTime(created)
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

func scanTask(row scanner) (Task, error) {
	var task Task
	var agentID, agentName, agentEmoji, agentRole sql.NullString
	var created, updated string
	var completed sql.NullString

	err := row.Scan(&task.ID, &task.Title, &task.Details, &task.Status, &agentID,
		&created, &updated, &completed, &agentName, &agentEmoji, &agentRole)
	if err != nil {
		return Task{}, err
	}

	task.CreatedAt = ParseTime(created)
	task.UpdatedAt = ParseTime(updated)
	if completed.Valid {
		t := ParseTime(completed.String)
		task.CompletedAt = &t
	}
	if agentID.Valid {
		id := agentID.String
		task.AgentID = &id

		if agentName.Valid {
			task.Agent = &AgentBrief{
				ID:    id,
				Name:  agentName.String,
				Emoji: agentEmoji.String,
				Role:  agentRole.String,
			}
		}
	}

	return task, nil
}
