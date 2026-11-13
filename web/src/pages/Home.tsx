import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Dashboard, type Task } from "../api";
import { NewTaskModal } from "../components/NewTaskModal";
import { TaskRow, Timeline } from "../components/rows";
import { Link, navigate } from "../router";
import { Avatar, Button, EmptyState, Skeleton, useToast } from "../ui";

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
          <p className="page-sub">
            {totalOpen === 0
              ? "Nothing is waiting on you. Add a task when you're ready."
              : data.counts.review > 0
                ? `${data.counts.review} ${data.counts.review === 1 ? "task needs" : "tasks need"} your OK.`
                : "Everything is moving. Here's the latest."}
          </p>
        </div>
        <Button variant="primary" onClick={() => setNewTaskOpen(true)}>
          Add a task
        </Button>
      </header>

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
            title="No work in the queue"
            action={
              <Button variant="primary" size="sm" onClick={() => setNewTaskOpen(true)}>
                Add your first task
              </Button>
            }
          >
            Add a task, pick a helper, and it will show up here.
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
