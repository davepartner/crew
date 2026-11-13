import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Dashboard, type Goal, type Task } from "../api";
import { NewTaskModal } from "../components/NewTaskModal";
import { TaskRow, Timeline } from "../components/rows";
import { Link, navigate } from "../router";
import { Avatar, Button, EmptyState, Field, Input, Modal, Skeleton, useToast } from "../ui";
import { shortDate } from "../time";

export function Home() {
  const { push } = useToast();
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [newTaskOpen, setNewTaskOpen] = useState(false);

  const load = useCallback(() => {
    api
      .dashboard()
      .then((result) => {
        setData(result);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load your dashboard.");
      });
  }, []);

  useEffect(load, [load]);

  async function approve(task: Task) {
    try {
      await api.approve(task.id);
      push("Approved. Nice work.");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't approve that.", "error");
    }
  }

  if (error) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't load your dashboard</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={load}>
          Try again
        </Button>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="stack">
        <div className="skeleton" style={{ height: 34, width: 280 }} />
        <div className="card">
          <Skeleton lines={3} />
        </div>
        <div className="card">
          <Skeleton lines={4} />
        </div>
      </div>
    );
  }

  const totalOpen = data.counts.todo + data.counts.working + data.counts.review;
  const firstName = data.name.split(" ")[0];

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">
            {data.greeting}, {firstName}
          </h1>
          <p className="tiny muted" style={{ marginBottom: 4 }}>
            {data.company.name}
          </p>
          <p className="page-sub">
            {totalOpen === 0
              ? "Nothing is waiting on you. Add a task when you're ready."
              : data.counts.review > 0
                ? `${data.counts.review} ${data.counts.review === 1 ? "task needs" : "tasks need"} your OK.`
                : "Everything is moving. Here's the latest."}
          </p>
        </div>
        <Button variant={data.needsYou.length > 0 ? "default" : "primary"} onClick={() => setNewTaskOpen(true)}>
          Add a task
        </Button>
      </header>

      <GoalsSection goals={data.goals ?? []} onChanged={load} />

      <section className="section" aria-labelledby="counts">
        <div className="grid-3">
          <Stat label="To do" value={data.counts.todo} onClick={() => navigate("/tasks?status=todo")} />
          <Stat label="In progress" value={data.counts.working} onClick={() => navigate("/tasks?status=working")} />
          <Stat
            label="Needs your OK"
            value={data.counts.review}
            highlight={data.counts.review > 0}
            onClick={() => navigate("/tasks?status=review")}
          />
        </div>
      </section>

      {data.needsYou.length > 0 ? (
        <section className="section" aria-labelledby="needs-you">
          <div className="section-head">
            <h2 className="section-title" id="needs-you">
              Needs your OK
            </h2>
            <Link className="section-link" to="/tasks?status=review">
              See all
            </Link>
          </div>

          <div className="list">
            {data.needsYou.map((task) => (
              <div className="card card-tight" key={task.id}>
                <div className="row-between" style={{ alignItems: "flex-start" }}>
                  <div style={{ minWidth: 0 }}>
                    <Link to={`/tasks/${task.id}`} className="card-title">
                      {task.title}
                    </Link>
                    <div className="card-meta">
                      {task.agent ? `${task.agent.emoji} ${task.agent.name}` : "Unassigned"} · waiting for you
                    </div>
                  </div>
                  <div style={{ display: "flex", gap: 8, flex: "0 0 auto" }}>
                    <Button size="sm" onClick={() => navigate(`/tasks/${task.id}`)}>
                      Review
                    </Button>
                    <Button size="sm" variant="primary" onClick={() => approve(task)}>
                      Approve
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <section className="section" aria-labelledby="working">
        <div className="section-head">
          <h2 className="section-title" id="working">
            Your queue
          </h2>
          <Link className="section-link" to="/tasks">
            All tasks
          </Link>
        </div>

        {data.working.length === 0 && data.upNext.length === 0 ? (
          <EmptyState
            title={data.counts.review > 0 ? "Nothing else in the queue" : "No work in the queue"}
            action={
              <Button size="sm" onClick={() => setNewTaskOpen(true)}>
                Add a task
              </Button>
            }
          >
            {data.counts.review > 0
              ? "Everything else is waiting for your OK above."
              : "Add a task, pick a helper, and it will show up here."}
          </EmptyState>
        ) : (
          <div className="list">
            {data.working.map((task) => (
              <TaskRow key={task.id} task={task} />
            ))}
            {data.upNext.map((task) => (
              <TaskRow key={task.id} task={task} />
            ))}
          </div>
        )}
      </section>

      <div className="grid-2">
        <section className="section" aria-labelledby="helpers">
          <div className="section-head">
            <h2 className="section-title" id="helpers">
              Your helpers
            </h2>
            <Link className="section-link" to="/helpers">
              {data.agents.length > 0 ? "Manage" : "Add one"}
            </Link>
          </div>

          {data.agents.length === 0 ? (
            <EmptyState title="No helpers yet" action={<Button size="sm" onClick={() => navigate("/helpers")}>Add a helper</Button>}>
              A helper is an AI worker. Add one, then give it tasks.
            </EmptyState>
          ) : (
            <div className="list">
              {data.agents.slice(0, 4).map((agent) => (
                <Link key={agent.id} to={`/helpers/${agent.id}`} className="row">
                  <Avatar emoji={agent.emoji} />
                  <span className="row-main">
                    <span className="row-title">{agent.name}</span>
                    <span className="row-meta">{agent.role}</span>
                  </span>
                  <span className={`tiny status-${agent.status}`}>
                    {statusLabel(agent.status)}
                  </span>
                </Link>
              ))}
            </div>
          )}
        </section>

        <section className="section" aria-labelledby="recent">
          <div className="section-head">
            <h2 className="section-title" id="recent">
              Recent activity
            </h2>
            <Link className="section-link" to="/activity">
              See all
            </Link>
          </div>
          <div className="card">
            <Timeline items={data.activity.slice(0, 6)} />
          </div>
        </section>
      </div>

      <NewTaskModal open={newTaskOpen} onClose={() => setNewTaskOpen(false)} onCreated={load} />
    </>
  );
}

function Stat({
  label,
  value,
  onClick,
  highlight = false,
}: {
  label: string;
  value: number;
  onClick: () => void;
  highlight?: boolean;
}) {
  return (
    <button
      className="card"
      onClick={onClick}
      style={{
        textAlign: "left",
        cursor: "pointer",
        padding: "14px 16px",
        borderColor: highlight ? "color-mix(in srgb, var(--warn) 45%, var(--line))" : undefined,
        background: highlight ? "var(--warn-soft)" : undefined,
      }}
    >
      <div className="tiny muted" style={{ fontWeight: 600, letterSpacing: "0.02em" }}>
        {label.toUpperCase()}
      </div>
      <div style={{ fontSize: 24, fontWeight: 650, letterSpacing: "-0.03em", marginTop: 2 }}>{value}</div>
    </button>
  );
}

function statusLabel(status: string): string {
  switch (status) {
    case "working":
      return "Working";
    case "ready":
      return "Ready";
    case "offline":
      return "Offline";
    default:
      return "Not connected";
  }
}


function GoalsSection({ goals, onChanged }: { goals: Goal[]; onChanged: () => void }) {
  const { push } = useToast();
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [target, setTarget] = useState("4");
  const [dueDate, setDueDate] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function create(event: React.FormEvent) {
    event.preventDefault();
    if (title.trim() === "") {
      setError("Give the goal a title.");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      await api.createGoal({
        title: title.trim(),
        target: Math.max(1, Number(target) || 1),
        dueDate,
      });
      setOpen(false);
      setTitle("");
      setTarget("4");
      setDueDate("");
      push("Goal set.");
      onChanged();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save that goal.");
    } finally {
      setBusy(false);
    }
  }

  async function step(goal: Goal, delta: number) {
    try {
      await api.updateGoal(goal.id, { current: goal.current + delta });
      onChanged();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't update that goal.", "error");
    }
  }

  async function finish(goal: Goal) {
    try {
      await api.updateGoal(goal.id, { current: goal.target });
      push("Goal complete.");
      onChanged();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't update that goal.", "error");
    }
  }

  async function remove(goal: Goal) {
    if (!window.confirm(`Delete “${goal.title}”?`)) return;
    try {
      await api.deleteGoal(goal.id);
      onChanged();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't delete that goal.", "error");
    }
  }

  return (
    <section className="section" aria-labelledby="goals">
      <div className="section-head">
        <h2 className="section-title" id="goals">
          Goals
        </h2>
        <button className="section-link link-button" onClick={() => setOpen(true)}>
          Add a goal
        </button>
      </div>

      {goals.length === 0 ? (
        <div className="card card-tight row-between" style={{ gap: 16 }}>
          <div>
            <div style={{ fontWeight: 600 }}>What does a good month look like?</div>
            <p className="small muted" style={{ margin: "4px 0 0" }}>
              A number you can move — demos booked, posts shipped, invoices sent.
            </p>
          </div>
          <Button size="sm" onClick={() => setOpen(true)}>
            Set a goal
          </Button>
        </div>
      ) : (
        <div className="goal-grid">
          {goals.map((goal) => {
            const done = goal.current >= goal.target;
            const percent = Math.min(100, Math.round((goal.current / Math.max(1, goal.target)) * 100));
            return (
              <div className={`card card-tight ${done ? "goal-done" : ""}`} key={goal.id}>
                <div className="row-between" style={{ alignItems: "flex-start" }}>
                  <div style={{ minWidth: 0 }}>
                    <div style={{ fontWeight: 600 }}>{goal.title}</div>
                    <div className="tiny muted">
                      {goal.current} / {goal.target}
                      {goal.due_date ? ` · by ${shortDate(goal.due_date)}` : ""}
                    </div>
                  </div>
                  <button className="goal-remove" onClick={() => remove(goal)} aria-label={`Delete ${goal.title}`}>
                    ×
                  </button>
                </div>

                <div className="goal-progress" aria-hidden="true">
                  <span style={{ width: `${percent}%` }} />
                </div>

                <div className="row-between mt-8">
                  <span className="tiny muted">{done ? "Done" : `${percent}%`}</span>
                  <div style={{ display: "flex", gap: 6 }}>
                    {!done ? (
                      <>
                        <Button size="sm" variant="quiet" onClick={() => step(goal, 1)}>
                          +1
                        </Button>
                        <Button size="sm" onClick={() => finish(goal)}>
                          Finish
                        </Button>
                      </>
                    ) : (
                      <span className="badge badge-done">Complete</span>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      <Modal
        open={open}
        title="New goal"
        onClose={() => setOpen(false)}
        footer={
          <>
            <Button onClick={() => setOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="primary" onClick={create} loading={busy}>
              Set goal
            </Button>
          </>
        }
      >
        <form onSubmit={create}>
          <Field label="What are you aiming for?" htmlFor="goal-title">
            <Input
              id="goal-title"
              autoFocus
              maxLength={160}
              placeholder="e.g. Book 20 demos"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
            />
          </Field>

          <div className="form-grid">
            <Field label="Target" htmlFor="goal-target">
              <Input
                id="goal-target"
                type="number"
                min={1}
                max={100000}
                value={target}
                onChange={(event) => setTarget(event.target.value)}
              />
            </Field>
            <Field label="Due date" hint="Optional." htmlFor="goal-due">
              <Input id="goal-due" type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} />
            </Field>
          </div>

          {error ? <p className="form-error">{error}</p> : null}
        </form>
      </Modal>
    </section>
  );
}
