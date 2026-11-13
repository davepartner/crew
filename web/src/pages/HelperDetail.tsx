import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Agent, type Task } from "../api";
import { TaskRow } from "../components/rows";
import { Link, navigate } from "../router";
import { relativeTime } from "../time";
import { Avatar, Button, EmptyState, Field, Modal, Select, Textarea, useToast } from "../ui";
import { statusLabel } from "./Helpers";

const roles = ["Researcher", "Writer", "Coder", "Designer", "Planner", "Support", "Helper"];

export function HelperDetail({ id }: { id: string }) {
  const { push } = useToast();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [role, setRole] = useState("Researcher");
  const [instructions, setInstructions] = useState("");
  const [busy, setBusy] = useState(false);
  const [key, setKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [removeOpen, setRemoveOpen] = useState(false);

  const load = useCallback(() => {
    api
      .agent(id)
      .then((result) => {
        setAgent(result.agent);
        setRole(result.agent.role);
        setInstructions(result.agent.instructions);
        setTasks(result.tasks ?? []);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load that helper.");
      });
  }, [id]);

  useEffect(load, [load]);

  async function save() {
    setBusy(true);
    try {
      const result = await api.updateAgent(id, { role, instructions });
      setAgent(result.agent);
      push("Saved.");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function makeKey() {
    if (key && !window.confirm("Making a new key stops the old one from working. Continue?")) return;

    setBusy(true);
    try {
      const result = await api.rotateKey(id);
      setKey(result.apiKey);
      setCopied(false);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't make a key.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function copy() {
    if (!key) return;
    try {
      await navigator.clipboard.writeText(commandFor(key));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  async function remove() {
    setBusy(true);
    try {
      await api.deleteAgent(id);
      push("Helper removed.");
      navigate("/helpers");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't remove that helper.", "error");
      setBusy(false);
    }
  }

  if (error) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't find that helper</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={() => navigate("/helpers")}>
          Back to helpers
        </Button>
      </div>
    );
  }

  if (!agent) {
    return (
      <div className="card">
        <div className="skeleton" style={{ height: 22, width: "40%" }} />
      </div>
    );
  }

  return (
    <>
      <div className="mb-16">
        <Link to="/helpers" className="small muted">
          ← Back to helpers
        </Link>
      </div>

      <header className="page-head">
        <div style={{ display: "flex", gap: 14, alignItems: "center", minWidth: 0 }}>
          <Avatar emoji={agent.emoji} size="lg" />
          <div style={{ minWidth: 0 }}>
            <h1 className="page-title" style={{ fontSize: 23 }}>
              {agent.name}
            </h1>
            <p className="page-sub">
              {agent.role} · <span className={`status-${agent.status}`}>{statusLabel(agent.status)}</span>
              {agent.last_seen_at ? ` · seen ${relativeTime(agent.last_seen_at)}` : ""}
            </p>
          </div>
        </div>
      </header>

      <div className="grid-2" style={{ alignItems: "start" }}>
        <div className="stack">
          <section className="card">
            <h2 className="section-title mb-8">Connect this helper</h2>
            <p className="small muted mb-16">
              A helper does its work on your computer. Give that computer a key once, and it can pick up tasks.
            </p>

            <ol className="small" style={{ paddingLeft: 18, margin: "0 0 16px", color: "var(--ink-soft)" }}>
              <li className="mb-8">Make a key below.</li>
              <li className="mb-8">Run the one-line command on the computer that will do the work.</li>
              <li>Add a task and pick {agent.name}. It will start on its own.</li>
            </ol>

            {key ? (
              <>
                <label className="label" htmlFor="helper-key">
                  Your new key (copy it now)
                </label>
                <div className="key-box mt-8 mb-8" id="helper-key">
                  <span>{key}</span>
                </div>
                <pre className="code-block mt-8">{commandFor(key)}</pre>
                <div className="row-between mt-8">
                  <span className="tiny muted">Shown once. Treat it like a password.</span>
                  <Button size="sm" onClick={copy}>
                    {copied ? "Copied" : "Copy command"}
                  </Button>
                </div>
              </>
            ) : (
              <Button variant="primary" onClick={makeKey} loading={busy}>
                Make a connection key
              </Button>
            )}
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Work</h2>
            {tasks.length === 0 ? (
              <EmptyState title="Nothing assigned right now">Give this helper a task and it will show up here.</EmptyState>
            ) : (
              <div className="list">
                {tasks.map((task) => (
                  <TaskRow key={task.id} task={task} />
                ))}
              </div>
            )}
          </section>
        </div>

        <div className="stack">
          <section className="card">
            <h2 className="section-title mb-16">How this helper works</h2>

            <Field label="Their job" htmlFor="helper-role-edit">
              <Select id="helper-role-edit" value={role} onChange={(event) => setRole(event.target.value)}>
                {roles.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </Select>
            </Field>

            <Field
              label="Notes for this helper"
              hint="Like a note you'd leave for a new teammate. It reads this before every task."
              htmlFor="helper-instructions-edit"
            >
              <Textarea
                id="helper-instructions-edit"
                rows={5}
                value={instructions}
                onChange={(event) => setInstructions(event.target.value)}
                placeholder="e.g. Keep answers short. Always link your sources."
              />
            </Field>

            <Button variant="primary" onClick={save} loading={busy}>
              Save changes
            </Button>
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Remove this helper</h2>
            <p className="small muted mb-16">
              Its key stops working and it disappears from your team. Tasks it finished stay where they are.
            </p>
            <Button variant="quiet" size="sm" onClick={() => setRemoveOpen(true)}>
              Remove {agent.name}
            </Button>
          </section>
        </div>
      </div>

      <Modal
        open={removeOpen}
        title={`Remove ${agent.name}?`}
        onClose={() => setRemoveOpen(false)}
        footer={
          <>
            <Button onClick={() => setRemoveOpen(false)} disabled={busy}>
              Keep
            </Button>
            <Button variant="danger" onClick={remove} loading={busy}>
              Remove
            </Button>
          </>
        }
      >
        <p className="modal-body">
          This can't be undone. Any task still assigned to {agent.name} becomes unassigned.
        </p>
      </Modal>
    </>
  );
}

function commandFor(key: string): string {
  return `go run ./cmd/crew-worker --server ${window.location.origin} --key ${key}`;
}
