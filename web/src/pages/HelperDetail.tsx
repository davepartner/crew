import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Agent, type Provider, type Skill, type Task } from "../api";
import { NewTaskModal } from "../components/NewTaskModal";
import { TaskRow } from "../components/rows";
import { Link, navigate } from "../router";
import { money, relativeTime } from "../time";
import { Avatar, Button, EmptyState, Field, Input, Modal, Select, Textarea, useToast } from "../ui";
import { statusLabel } from "./Helpers";
import { roles } from "../roles";

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
  const [taskOpen, setTaskOpen] = useState(false);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [modelProvider, setModelProvider] = useState("");
  const [modelName, setModelName] = useState("");
  const [modelBusy, setModelBusy] = useState(false);
  const [catalog, setCatalog] = useState<Skill[]>([]);
  const [skillsOpen, setSkillsOpen] = useState(false);
  const [draftSkills, setDraftSkills] = useState<string[]>([]);
  const [skillQuery, setSkillQuery] = useState("");
  const [skillBusy, setSkillBusy] = useState(false);
  const [budget, setBudget] = useState("0");
  const [spend, setSpend] = useState(0);
  const [budgetBusy, setBudgetBusy] = useState(false);
  const [canDelegate, setCanDelegate] = useState(true);

  const load = useCallback(() => {
    api
      .agent(id)
      .then((result) => {
        setAgent(result.agent);
        setRole(result.agent.role);
        setInstructions(result.agent.instructions);
        setModelProvider(result.agent.model_provider ?? "");
        setModelName(result.agent.model_name ?? "");
        setBudget(String(result.agent.monthly_budget_usd ?? 0));
        setCanDelegate(result.agent.can_delegate ?? true);
        setSpend(result.spend_usd ?? 0);
        setTasks(result.tasks ?? []);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load that helper.");
      });
  }, [id]);

  useEffect(load, [load]);

  useEffect(() => {
    api
      .providers()
      .then((result) => setProviders(result.providers.filter((provider) => provider.configured)))
      .catch(() => setProviders([]));

    api
      .skills()
      .then((result) => setCatalog(result.skills))
      .catch(() => setCatalog([]));
  }, []);

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
        <Button variant="primary" onClick={() => setTaskOpen(true)}>
          Give {agent.name} a task
        </Button>
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
              <Button onClick={makeKey} loading={busy}>
                Make a connection key
              </Button>
            )}
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Work</h2>
            {tasks.length === 0 ? (
              <EmptyState
                title="Nothing assigned right now"
                action={
                  <Button size="sm" onClick={() => setTaskOpen(true)}>
                    Give {agent.name} a task
                  </Button>
                }
              >
                Give this helper a task and it will show up here.
              </EmptyState>
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

            <Button onClick={save} loading={busy}>
              Save changes
            </Button>
          </section>

          <section className="card">
            <div className="row-between" style={{ alignItems: "flex-start" }}>
              <div>
                <h2 className="section-title" style={{ marginBottom: 4 }}>
                  Skills
                </h2>
                <p className="small muted" style={{ marginBottom: 0 }}>
                  Playbooks this helper reads before every task.
                </p>
              </div>
              <Button
                size="sm"
                onClick={() => {
                  setDraftSkills(agent.skills ?? []);
                  setSkillQuery("");
                  setSkillsOpen(true);
                }}
              >
                Choose skills
              </Button>
            </div>

            <div className="chip-row mt-16">
              {(agent.skills ?? []).length === 0 ? (
                <p className="small muted" style={{ margin: 0 }}>
                  No skills yet. Pick a few from the marketing library.
                </p>
              ) : (
                (agent.skills ?? []).map((key) => (
                  <span className="chip" key={key}>
                    {catalog.find((skill) => skill.key === key)?.name ?? key}
                  </span>
                ))
              )}
            </div>
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Model</h2>
            <p className="small muted mb-16">
              Which AI model {agent.name} should use. Add keys in{" "}
              <Link to="/settings">Settings → AI models</Link>.
            </p>

            {providers.length === 0 ? (
              <p className="small muted">
                No model keys yet. Add one and this helper can think for itself.
              </p>
            ) : (
              <>
                <Field label="Provider" htmlFor="agent-provider">
                  <Select
                    id="agent-provider"
                    value={modelProvider}
                    onChange={(event) => {
                      const value = event.target.value;
                      setModelProvider(value);
                      const provider = providers.find((item) => item.key === value);
                      setModelName(provider?.default_model ?? "");
                    }}
                  >
                    <option value="">Use the worker's default</option>
                    {providers.map((provider) => (
                      <option key={provider.key} value={provider.key}>
                        {provider.name}
                        {provider.platform ? " (included)" : provider.free ? " (free)" : ""}
                      </option>
                    ))}
                  </Select>
                </Field>

                {modelProvider ? (
                  <Field label="Model" htmlFor="agent-model">
                    <Select
                      id="agent-model"
                      value={modelName}
                      onChange={(event) => setModelName(event.target.value)}
                    >
                      {(providers.find((item) => item.key === modelProvider)?.models ?? []).map((model) => (
                        <option key={model} value={model}>
                          {model}
                        </option>
                      ))}
                    </Select>
                  </Field>
                ) : null}

                <Button
                  onClick={async () => {
                    setModelBusy(true);
                    try {
                      const result = await api.updateAgent(id, { modelProvider, modelName });
                      setAgent(result.agent);
                      push("Model saved.");
                    } catch (err) {
                      push(err instanceof ApiError ? err.message : "We couldn't save that.", "error");
                    } finally {
                      setModelBusy(false);
                    }
                  }}
                  loading={modelBusy}
                >
                  Save model
                </Button>
              </>
            )}
          </section>

          <section className="card">
            <h2 className="section-title mb-8">AI budget</h2>
            <p className="small muted mb-16">
              A monthly cap for this helper's model calls. 0 means no limit. Estimates use public list prices.
            </p>
            <p className="small mb-16">
              Spent this month: <strong>{money(spend)}</strong>
              {agent.monthly_budget_usd > 0 ? ` of ${money(agent.monthly_budget_usd)}` : ""}
            </p>
            <label className="checkbox-row mb-16">
              <input
                type="checkbox"
                checked={canDelegate}
                onChange={async (event) => {
                  const next = event.target.checked;
                  setCanDelegate(next);
                  try {
                    const result = await api.updateAgent(id, { canDelegate: next });
                    setAgent(result.agent);
                  } catch (err) {
                    setCanDelegate(!next);
                    push(err instanceof ApiError ? err.message : "We couldn't change that.", "error");
                  }
                }}
              />
              <span>Allow {agent.name} to hand work to another helper</span>
            </label>

            <div style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
              <div style={{ flex: 1 }}>
                <Field label="Monthly budget (USD)" htmlFor="agent-budget">
                  <Input
                    id="agent-budget"
                    type="number"
                    min={0}
                    step="0.01"
                    value={budget}
                    onChange={(event) => setBudget(event.target.value)}
                  />
                </Field>
              </div>
              <Button
                onClick={async () => {
                  setBudgetBusy(true);
                  try {
                    const result = await api.updateAgent(id, {
                      monthlyBudgetUsd: Math.max(0, Number(budget) || 0),
                    });
                    setAgent(result.agent);
                    push("Budget saved.");
                  } catch (err) {
                    push(err instanceof ApiError ? err.message : "We couldn't save that budget.", "error");
                  } finally {
                    setBudgetBusy(false);
                  }
                }}
                loading={budgetBusy}
              >
                Save budget
              </Button>
            </div>
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Remove this helper</h2>
            <p className="small muted mb-16">
              Its key stops working and it disappears from your team. Tasks it finished stay where they are.
            </p>
            <Button
              variant="quiet"
              size="sm"
              style={{ color: "var(--danger)" }}
              onClick={() => setRemoveOpen(true)}
            >
              Remove {agent.name}
            </Button>
          </section>
        </div>
      </div>

      <NewTaskModal
        open={taskOpen}
        defaultAgentId={agent.id}
        onClose={() => setTaskOpen(false)}
        onCreated={(task) => {
          load();
          navigate(`/tasks/${task.id}`);
        }}
      />

      <Modal
        open={skillsOpen}
        title={`Skills for ${agent.name}`}
        onClose={() => setSkillsOpen(false)}
        footer={
          <>
            <Button onClick={() => setSkillsOpen(false)} disabled={skillBusy}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={skillBusy}
              onClick={async () => {
                setSkillBusy(true);
                try {
                  const result = await api.updateAgent(id, { skills: draftSkills });
                  setAgent(result.agent);
                  setSkillsOpen(false);
                  push("Skills saved.");
                } catch (err) {
                  push(err instanceof ApiError ? err.message : "We couldn't save those skills.", "error");
                } finally {
                  setSkillBusy(false);
                }
              }}
            >
              Save skills
            </Button>
          </>
        }
      >
        <p className="small muted mb-16">
          Choose up to 8. Each one is added to the helper's briefing, within a safe size limit. From the{" "}
          <Link to="/marketing">marketing library</Link>.
        </p>

        <Input
          placeholder="Search skills…"
          value={skillQuery}
          onChange={(event) => setSkillQuery(event.target.value)}
          aria-label="Search skills"
        />
        <p className="tiny muted mt-8 mb-8">{draftSkills.length} of 8 selected</p>

        <div className="skill-picker">
          {catalog
            .filter((skill) => {
              const needle = skillQuery.trim().toLowerCase();
              if (!needle) return true;
              return (
                skill.name.toLowerCase().includes(needle) ||
                skill.category.toLowerCase().includes(needle) ||
                skill.description.toLowerCase().includes(needle)
              );
            })
            .map((skill) => {
              const checked = draftSkills.includes(skill.key);
              return (
                <label className="skill-option" key={skill.key}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => {
                      if (checked) {
                        setDraftSkills(draftSkills.filter((item) => item !== skill.key));
                      } else if (draftSkills.length >= 8) {
                        push("Choose at most 8 skills.", "error");
                      } else {
                        setDraftSkills([...draftSkills, skill.key]);
                      }
                    }}
                  />
                  <span>
                    <span className="skill-option-name">
                      {skill.name} <span className="tiny muted">· {skill.category}</span>
                    </span>
                    <span className="skill-option-desc">{skill.description}</span>
                  </span>
                </label>
              );
            })}
        </div>
      </Modal>

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
