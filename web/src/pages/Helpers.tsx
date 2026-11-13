import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Agent } from "../api";
import { NewHelperModal } from "../components/NewHelperModal";
import { Link } from "../router";
import { Avatar, Button, EmptyState, Skeleton } from "../ui";

export function Helpers() {
  const [agents, setAgents] = useState<Agent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [addOpen, setAddOpen] = useState(false);

  const load = useCallback(() => {
    api
      .agents()
      .then((result) => {
        setAgents(result.agents);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load your helpers.");
      });
  }, []);

  useEffect(load, [load]);

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Helpers</h1>
          <p className="page-sub">The AI workers on your team. Add one, connect it, give it tasks.</p>
        </div>
        <Button variant="primary" onClick={() => setAddOpen(true)}>
          Add a helper
        </Button>
      </header>

      {error ? (
        <div className="empty">
          <div className="empty-title">We couldn't load your helpers</div>
          <p className="small">{error}</p>
          <Button className="mt-16" onClick={load}>
            Try again
          </Button>
        </div>
      ) : agents === null ? (
        <div className="card">
          <Skeleton lines={4} />
        </div>
      ) : agents.length === 0 ? (
        <EmptyState
          title="No helpers yet"
          action={
            <Button variant="primary" size="sm" onClick={() => setAddOpen(true)}>
              Add your first helper
            </Button>
          }
        >
          A helper is an AI worker with a name and a job. You stay in charge of what it ships.
        </EmptyState>
      ) : (
        <div className="grid-2">
          {agents.map((agent) => (
            <Link key={agent.id} to={`/helpers/${agent.id}`} className="card card-link helper-card">
              <div className="helper-top">
                <Avatar emoji={agent.emoji} size="lg" />
                <div style={{ minWidth: 0 }}>
                  <div className="helper-name">{agent.name}</div>
                  <div className="helper-role">{agent.role}</div>
                </div>
              </div>

              <div className="row-between">
                <span className={`tiny status-${agent.status}`}>{statusLabel(agent.status)}</span>
                <span className="tiny muted">
                  {agent.open_tasks === 0
                    ? "No open tasks"
                    : `${agent.open_tasks} open ${agent.open_tasks === 1 ? "task" : "tasks"}`}
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}

      <NewHelperModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onCreated={() => load()}
      />
    </>
  );
}

export function statusLabel(status: string): string {
  switch (status) {
    case "working":
      return "Working now";
    case "ready":
      return "Connected";
    case "offline":
      return "Offline";
    default:
      return "Not connected yet";
  }
}
