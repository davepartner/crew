package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestClaimRequeuesStaleWork(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	user, err := st.UpsertUser(ctx, "ext-1", "jane@example.com", "Jane")
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	company, err := st.CreateCompany(ctx, user.ID, CompanyInput{Name: "Test Co", BusinessKey: "custom"})
	if err != nil {
		t.Fatalf("company: %v", err)
	}

	agent, err := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Writer", "✍️", "", nil, "hash-1")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}

	task, err := st.CreateTask(ctx, user.ID, company.ID, "Do the thing", "", agent.ID)
	if err != nil {
		t.Fatalf("task: %v", err)
	}

	claimed, err := st.ClaimNextTask(ctx, company.ID, agent.ID, time.Hour)
	if err != nil || claimed.ID != task.ID {
		t.Fatalf("first claim: %v %v", claimed, err)
	}

	// A live lease must not be stolen.
	if _, err := st.ClaimNextTask(ctx, company.ID, agent.ID, time.Hour); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound while the lease is fresh, got %v", err)
	}

	// Simulate a helper that crashed two hours ago.
	old := time.Now().UTC().Add(-2 * time.Hour).Format(TimeFormat)
	if _, err := st.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, old, task.ID); err != nil {
		t.Fatalf("age task: %v", err)
	}

	again, err := st.ClaimNextTask(ctx, company.ID, agent.ID, time.Hour)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if again.ID != task.ID {
		t.Fatalf("expected the stale task back, got %s", again.ID)
	}
}

func TestTouchTaskExtendsTheLease(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	user, _ := st.UpsertUser(ctx, "ext-1", "jane@example.com", "Jane")
	company, _ := st.CreateCompany(ctx, user.ID, CompanyInput{Name: "Test Co", BusinessKey: "custom"})
	agent, _ := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Writer", "✍️", "", nil, "hash-1")
	task, _ := st.CreateTask(ctx, user.ID, company.ID, "Do the thing", "", agent.ID)

	old := time.Now().UTC().Add(-30 * time.Minute).Format(TimeFormat)
	if _, err := st.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, old, task.ID); err != nil {
		t.Fatalf("age task: %v", err)
	}

	if err := st.TouchTask(ctx, company.ID, task.ID); err != nil {
		t.Fatalf("touch: %v", err)
	}

	var updated string
	if err := st.db.QueryRowContext(ctx, `SELECT updated_at FROM tasks WHERE id = ?`, task.ID).Scan(&updated); err != nil {
		t.Fatalf("read: %v", err)
	}
	if updated == old {
		t.Fatal("touch did not move updated_at")
	}
}

func TestRoutineRunsOnceAndAdvances(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	user, err := st.UpsertUser(ctx, "external-routine", "routine@example.com", "Routine")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	company, err := st.CreateCompany(ctx, user.ID, CompanyInput{Name: "Routine Co", BusinessKey: "custom"})
	if err != nil {
		t.Fatalf("company: %v", err)
	}
	agent, err := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Operations", "🧭", "", nil, "hash-r")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}

	routine, err := st.CreateRoutine(ctx, user.ID, company.ID, "Morning check", "Look at orders", agent.ID, CadenceDaily)
	if err != nil {
		t.Fatalf("routine: %v", err)
	}
	if !routine.NextRunAt.After(time.Now()) {
		t.Fatalf("next run should be in the future: %v", routine.NextRunAt)
	}

	future := time.Now().Add(48 * time.Hour)
	count, err := st.RunDueRoutines(ctx, future)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 run, got %d", count)
	}

	tasks, err := st.ListTasks(ctx, company.ID, StatusTodo)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks: %v %d", err, len(tasks))
	}
	if tasks[0].Title != "Morning check" || tasks[0].AgentID == nil || *tasks[0].AgentID != agent.ID {
		t.Fatalf("routine task wrong: %+v", tasks[0])
	}

	updated, err := st.GetRoutine(ctx, company.ID, routine.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !updated.NextRunAt.After(future) {
		t.Fatalf("next run did not advance: %v", updated.NextRunAt)
	}

	if again, err := st.RunDueRoutines(ctx, future); err != nil || again != 0 {
		t.Fatalf("second run should do nothing: %d %v", again, err)
	}
}
