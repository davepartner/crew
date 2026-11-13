import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Agent, type Routine } from "../api";
import { Link } from "../router";
import { relativeTime } from "../time";
import { Button, EmptyState, Field, Input, Modal, Select, Textarea, useToast } from "../ui";

const cadences = [
  { value: "daily", label: "Every day" },
  { value: "weekdays", label: "Weekdays" },
  { value: "weekly", label: "Every week" },
  { value: "monthly", label: "Every month" },
];

function cadenceLabel(value: string) {
  return cadences.find((item) => item.value === value)?.label ?? value;
}

export function Routines() {
  const { push } = useToast();
  const [routines, setRoutines] = useState<Routine[] | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [details, setDetails] = useState("");
  const [cadence, setCadence] = useState("weekly");
  const [agent, setAgent] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    api
      .routines()
      .then((result) => setRoutines(result.routines))
      .catch(() => setRoutines([]));
    api
      .agents()
      .then((result) => setAgents(result.agents))
      .catch(() => setAgents([]));
  }, []);

  useEffect(load, [load]);

  async function create(event: React.FormEvent) {
    event.preventDefault();
    if (title.trim() === "") {
      setError("Give the routine a title.");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      await api.createRoutine({ title: title.trim(), details: details.trim(), cadence, agentId: agent });
      push("Routine scheduled.");
      setOpen(false);
      setTitle("");
      setDetails("");
      setCadence("weekly");
      setAgent("");
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save that routine.");
    } finally {
      setBusy(false);
    }
  }

  async function toggle(routine: Routine) {
    try {
      await api.updateRoutine(routine.id, { enabled: !routine.enabled });
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't change that routine.", "error");
    }
  }

  async function remove(routine: Routine) {
    if (!window.confirm(`Delete “${routine.title}”? Tasks it already created stay.`)) return;
    try {
      await api.deleteRoutine(routine.id);
      push("Routine deleted.");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't delete that routine.", "error");
    }
  }

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>
            <Link to="/tasks">← Tasks</Link>
          </p>
          <h1 className="page-title">Routines</h1>
          <p className="page-sub">Work that repeats on a schedule. Crew creates the task; a helper picks it up.</p>
        </div>
        <Button variant="primary" onClick={() => setOpen(true)}>
          New routine
        </Button>
      </header>

      {routines === null ? (
        <div className="skeleton" style={{ height: 120 }} />
      ) : routines.length === 0 ? (
        <EmptyState
          title="No routines yet"
          action={
            <Button variant="primary" size="sm" onClick={() => setOpen(true)}>
              Create the first routine
            </Button>
          }
        >
          A weekly review, a Monday pipeline check, a month-end report — anything you'd otherwise forget.
        </EmptyState>
      ) : (
        <div className="list">
          {routines.map((routine) => (
            <div className="routine-row" key={routine.id}>
              <div style={{ minWidth: 0 }}>
                <div className="row-title">{routine.title}</div>
                <div className="row-meta">
                  {cadenceLabel(routine.cadence)} · {routine.agent ? routine.agent.name : "No helper"} ·{" "}
                  {routine.enabled ? relativeTime(routine.next_run_at) : "paused"}
                </div>
                {routine.details ? (
                  <p className="small muted" style={{ margin: "6px 0 0" }}>
                    {routine.details}
                  </p>
                ) : null}
              </div>

              <div style={{ display: "flex", alignItems: "center", gap: 10, flexShrink: 0 }}>
                <label className="switch">
                  <input type="checkbox" checked={routine.enabled} onChange={() => toggle(routine)} />
                  <span>{routine.enabled ? "On" : "Paused"}</span>
                </label>
                <Button size="sm" variant="quiet" onClick={() => remove(routine)}>
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <p className="tiny muted mt-16">
        A routine runs once per slot, even if the server was down. Its task counts toward the company's monthly task
        budget.
      </p>

      <Modal
        open={open}
        title="New routine"
        onClose={() => setOpen(false)}
        footer={
          <>
            <Button onClick={() => setOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="primary" onClick={create} loading={busy}>
              Schedule routine
            </Button>
          </>
        }
      >
        <form onSubmit={create}>
          <Field label="What should happen?" htmlFor="routine-title">
            <Input
              id="routine-title"
              autoFocus
              maxLength={200}
              placeholder="e.g. Review the week's tasks"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
            />
          </Field>

          <Field label="Details for the helper" hint="Optional." htmlFor="routine-details">
            <Textarea
              id="routine-details"
              rows={3}
              placeholder="What to check, and what the result should look like."
              value={details}
              onChange={(event) => setDetails(event.target.value)}
            />
          </Field>

          <div className="form-grid">
            <Field label="How often" htmlFor="routine-cadence">
              <Select id="routine-cadence" value={cadence} onChange={(event) => setCadence(event.target.value)}>
                {cadences.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </Select>
            </Field>

            <Field label="Give it to" htmlFor="routine-agent">
              <Select id="routine-agent" value={agent} onChange={(event) => setAgent(event.target.value)}>
                <option value="">No helper</option>
                {agents.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.emoji} {item.name}
                  </option>
                ))}
              </Select>
            </Field>
          </div>

          {error ? <p className="form-error">{error}</p> : null}
        </form>
      </Modal>
    </>
  );
}
