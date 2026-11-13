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

	agent, err := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Writer", "✍️", "", "hash-1")
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
	agent, _ := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Writer", "✍️", "", "hash-1")
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
