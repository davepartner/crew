import { useEffect, useMemo, useState } from "react";
import { api, ApiError, type Agent, type MarketingReport, type Playbook, type Skill } from "../api";
import { Link, navigate } from "../router";
import { Button, Field, Input, Modal, Select, Tabs, Textarea, useToast } from "../ui";

const tabs = [
  { id: "skills", label: "Skills" },
  { id: "reverse", label: "Reverse-engineer" },
];

export function Marketing() {
  const [tab, setTab] = useState("skills");

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Marketing</h1>
          <p className="page-sub">
            A library of marketing playbooks, and a teardown tool that explains how another company grows.
          </p>
        </div>
      </header>

      <Tabs items={tabs} active={tab} onChange={setTab} />

      {tab === "skills" ? <SkillsLibrary /> : <ReverseEngineer />}
    </>
  );
}

function SkillsLibrary() {
  const { push } = useToast();
  const [skills, setSkills] = useState<Skill[] | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("All");
  const [open, setOpen] = useState<Skill | null>(null);
  const [body, setBody] = useState<string | null>(null);
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .skills()
      .then((result) => setSkills(result.skills))
      .catch(() => setSkills([]));
    api
      .agents()
      .then((result) => setAgents(result.agents))
      .catch(() => setAgents([]));
  }, []);

  useEffect(() => {
    if (!open) return;
    setBody(null);
    api
      .skill(open.key)
      .then((result) => setBody(result.body))
      .catch(() => setBody("We couldn't load this skill."));
  }, [open]);

  const categories = useMemo(() => {
    const set = new Set((skills ?? []).map((skill) => skill.category));
    return ["All", ...Array.from(set).sort()];
  }, [skills]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return (skills ?? []).filter((skill) => {
      if (category !== "All" && skill.category !== category) return false;
      if (!needle) return true;
      return (
        skill.name.toLowerCase().includes(needle) ||
        skill.description.toLowerCase().includes(needle) ||
        skill.category.toLowerCase().includes(needle)
      );
    });
  }, [skills, query, category]);

  async function attach() {
    if (!open || !target) return;

    const agent = agents.find((item) => item.id === target);
    if (!agent) return;

    if (agent.skills?.includes(open.key)) {
      push(`${agent.name} already has ${open.name}.`, "error");
      return;
    }
    if ((agent.skills?.length ?? 0) >= 8) {
      push(`${agent.name} already has 8 skills. Remove one first.`, "error");
      return;
    }

    setBusy(true);
    try {
      const result = await api.updateAgent(agent.id, { skills: [...(agent.skills ?? []), open.key] });
      setAgents((current) => current.map((item) => (item.id === agent.id ? result.agent : item)));
      push(`${open.name} added to ${agent.name}.`);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't add that skill.", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section>
      <div className="marketing-toolbar">
        <Input
          placeholder="Search skills…"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label="Search skills"
          style={{ maxWidth: 320 }}
        />
        <div className="chip-row" role="group" aria-label="Skill categories">
          {categories.map((item) => (
            <button
              key={item}
              className={`chip chip-button ${category === item ? "chip-on" : ""}`}
              onClick={() => setCategory(item)}
              aria-pressed={category === item}
            >
              {item}
            </button>
          ))}
        </div>
      </div>

      {skills === null ? (
        <div className="skeleton" style={{ height: 160 }} />
      ) : filtered.length === 0 ? (
        <div className="empty">
          <div className="empty-title">No skills match</div>
          <p className="small">Try “pricing”, “email” or “SEO”.</p>
        </div>
      ) : (
        <div className="skill-grid">
          {filtered.map((skill) => (
            <button key={skill.key} className="skill-card" onClick={() => setOpen(skill)}>
              <span className="skill-cat">{skill.category}</span>
              <span className="skill-name">{skill.name}</span>
              <span className="skill-desc">{skill.description}</span>
            </button>
          ))}
        </div>
      )}

      <p className="tiny muted mt-16">
        {skills?.length ?? 0} skills from{" "}
        <a href="https://github.com/coreyhaines31/marketingskills" target="_blank" rel="noreferrer noopener">
          Corey Haines' marketingskills
        </a>{" "}
        (MIT). Attach up to 8 to a helper; they become part of its briefing.
      </p>

      <Modal
        open={open !== null}
        title={open?.name ?? ""}
        onClose={() => setOpen(null)}
        width={760}
        footer={
          <>
            <Button onClick={() => setOpen(null)}>Close</Button>
            <Button variant="primary" onClick={attach} loading={busy} disabled={!target}>
              Add to helper
            </Button>
          </>
        }
      >
        {open ? (
          <>
            <p className="tiny muted mb-8">
              {open.category} · {open.chars.toLocaleString()} characters ·{" "}
              <a href={open.source_url} target="_blank" rel="noreferrer noopener">
                source ↗
              </a>
            </p>
            <p className="small mb-16">{open.description}</p>

            <Field label="Add to a helper" htmlFor="skill-target">
              <Select id="skill-target" value={target} onChange={(event) => setTarget(event.target.value)}>
                <option value="">Choose a helper…</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.emoji} {agent.name} · {agent.role}
                  </option>
                ))}
              </Select>
            </Field>

            <div className="skill-body">
              {body === null ? <span className="spinner" aria-label="Loading" /> : <pre>{body}</pre>}
            </div>
          </>
        ) : null}
      </Modal>
    </section>
  );
}

function ReverseEngineer() {
  const { push } = useToast();
  const [playbooks, setPlaybooks] = useState<Playbook[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [url, setUrl] = useState("");
  const [focus, setFocus] = useState("");
  const [playbook, setPlaybook] = useState("funnel-monetization");
  const [agent, setAgent] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<MarketingReport | null>(null);

  useEffect(() => {
    api
      .skills()
      .then((response) => {
        setPlaybooks(response.playbooks);
        if (!response.playbooks.some((item) => item.key === "funnel-monetization") && response.playbooks[0]) {
          setPlaybook(response.playbooks[0].key);
        }
      })
      .catch(() => setPlaybooks([]));

    api
      .agents()
      .then((response) => {
        setAgents(response.agents);
        const marketing =
          response.agents.find((item) => item.role === "Marketing" || item.role === "Audience") ??
          response.agents[0];
        if (marketing) setAgent(marketing.id);
      })
      .catch(() => setAgents([]));

    api
      .providers()
      .then((response) => setConfigured(response.providers.some((provider) => provider.configured && provider.key !== "ollama")))
      .catch(() => setConfigured(false));
  }, []);

  async function analyze(event: React.FormEvent) {
    event.preventDefault();
    if (url.trim() === "") {
      setError("Paste the company's website first.");
      return;
    }

    setBusy(true);
    setError(null);
    setResult(null);

    try {
      const response = await api.analyzeMarketing({
        url: url.trim(),
        focus: focus.trim(),
        playbook,
        agentId: agent,
      });
      setResult(response);
      push("Teardown ready.");
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError("Add an AI model key in Settings → AI models, then run the teardown.");
      } else {
        setError(err instanceof ApiError ? err.message : "We couldn't finish that analysis.");
      }
    } finally {
      setBusy(false);
    }
  }

  function download() {
    if (!result) return;
    const blob = new Blob([result.report], { type: "text/markdown" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = "marketing-teardown.md";
    link.click();
    URL.revokeObjectURL(link.href);
  }

  return (
    <div className="marketing-layout">
      <form className="card" onSubmit={analyze} style={{ maxWidth: 520 }}>
        <h2 className="section-title mb-8">Analyse a company</h2>
        <p className="small muted mb-16">
          Paste a public website. Crew reads the page, applies the analyst harness, and writes a teardown you can
          act on. It can only see what is on the page — anything else is reported as unknown.
        </p>

        {configured === false ? (
          <div className="callout mb-16">
            <strong>No model key yet.</strong> Add one in <Link to="/settings">Settings → AI models</Link> to run
            teardowns.
          </div>
        ) : null}

        <Field label="Company website" htmlFor="marketing-url">
          <Input
            id="marketing-url"
            autoFocus
            placeholder="e.g. linear.app"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
          />
        </Field>

        <Field label="What do you want to learn?" hint="Optional." htmlFor="marketing-focus">
          <Textarea
            id="marketing-focus"
            rows={3}
            placeholder="e.g. how do they turn free users into paid teams?"
            value={focus}
            onChange={(event) => setFocus(event.target.value)}
          />
        </Field>

        <div className="form-grid">
          <Field label="Playbook" htmlFor="marketing-playbook">
            <Select id="marketing-playbook" value={playbook} onChange={(event) => setPlaybook(event.target.value)}>
              {playbooks.map((item) => (
                <option key={item.key} value={item.key}>
                  {item.name.replace(/^Playbook:\s*/, "")}
                </option>
              ))}
            </Select>
          </Field>

          <Field label="Give the report to" htmlFor="marketing-agent">
            <Select id="marketing-agent" value={agent} onChange={(event) => setAgent(event.target.value)}>
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

        <Button type="submit" variant="primary" loading={busy} disabled={configured === false}>
          {busy ? "Reading and analysing…" : "Run teardown"}
        </Button>
        {busy ? <p className="tiny muted mt-8">This usually takes 20–60 seconds.</p> : null}
      </form>

      <div style={{ minWidth: 0 }}>
        {result ? (
          <article className="card">
            <div className="row-between" style={{ alignItems: "flex-start" }}>
              <div style={{ minWidth: 0 }}>
                <h2 className="section-title" style={{ marginBottom: 4 }}>
                  {result.evidence.title || result.evidence.url}
                </h2>
                <p className="tiny muted">
                  {result.evidence.url} · {result.model} · {result.evidence.characters.toLocaleString()} characters read
                  {result.evidence.truncated ? " (truncated)" : ""}
                </p>
              </div>
              <div style={{ display: "flex", gap: 6 }}>
                <Button size="sm" onClick={download}>
                  Download .md
                </Button>
                {result.task ? (
                  <Button size="sm" onClick={() => navigate(`/tasks/${result.task!.id}`)}>
                    Open task
                  </Button>
                ) : null}
              </div>
            </div>

            <pre className="report-text mt-16">{result.report}</pre>

            <p className="tiny muted mt-16">
              Harness from{" "}
              <a href="https://github.com/dansugc/reverse-engineer-marketing" target="_blank" rel="noreferrer noopener">
                Reverse Engineer Marketing
              </a>{" "}
              (MIT). Saved as a task for your review.
            </p>
          </article>
        ) : (
          <div className="card" style={{ height: "100%", display: "grid", placeItems: "center", minHeight: 320 }}>
            <div style={{ textAlign: "center", maxWidth: 380 }}>
              <div style={{ fontSize: 28 }} aria-hidden="true">
                🔎
              </div>
              <h2 className="section-title mt-8">The report appears here</h2>
              <p className="small muted">
                It follows the evidence: positioning, offer, funnel, channels and the playbook to copy. Unknowns are
                listed, not guessed.
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
