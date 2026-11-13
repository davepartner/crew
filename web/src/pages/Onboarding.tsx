import { useEffect, useMemo, useState } from "react";
import { navigate } from "../router";
import { api, ApiError, type BusinessPreset, type Company, type PresetDetail } from "../api";
import { ModelsPanel } from "../components/ModelsPanel";
import { Button, Field, Input, Select, Textarea, useToast } from "../ui";

const tones = ["Friendly", "Professional", "Playful", "Direct", "Warm"];

const countries = ["Nigeria", "Kenya", "Ghana", "South Africa", "United Kingdom", "United States", "Canada", "India", "United Arab Emirates"];

export function Onboarding({
  onDone,
  onCancel,
}: {
  onDone: (company: Company) => void;
  onCancel?: () => void;
}) {
  const { push } = useToast();
  const [step, setStep] = useState(1);
  const [businesses, setBusinesses] = useState<BusinessPreset[] | null>(null);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<BusinessPreset | null>(null);
  const [detail, setDetail] = useState<PresetDetail | null>(null);
  const [name, setName] = useState("");
  const [website, setWebsite] = useState("");
  const [supportEmail, setSupportEmail] = useState("");
  const [description, setDescription] = useState("");
  const [audience, setAudience] = useState("");
  const [tone, setTone] = useState("Friendly");
  const [country, setCountry] = useState("Nigeria");
  const [createTeam, setCreateTeam] = useState(true);
  const [created, setCreated] = useState<Company | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .presets()
      .then((result) => setBusinesses(result.businesses))
      .catch(() => setBusinesses([]));
  }, []);

  useEffect(() => {
    if (!selected) return;
    setDetail(null);
    api
      .preset(selected.key)
      .then((result) => setDetail(result))
      .catch(() => setDetail(null));
  }, [selected]);

  const filtered = useMemo(() => {
    const list = businesses ?? [];
    const needle = query.trim().toLowerCase();
    if (!needle) return list;
    return list.filter(
      (item) =>
        item.name.toLowerCase().includes(needle) ||
        item.blurb.toLowerCase().includes(needle) ||
        item.category.toLowerCase().includes(needle),
    );
  }, [businesses, query]);

  const grouped = useMemo(() => {
    const map = new Map<string, BusinessPreset[]>();
    for (const item of filtered) {
      const list = map.get(item.category) ?? [];
      list.push(item);
      map.set(item.category, list);
    }
    return Array.from(map.entries());
  }, [filtered]);

  function choose(item: BusinessPreset) {
    setSelected(item);
    setError(null);
    if (!name) {
      setName(item.key === "custom" ? "" : "My " + item.name.toLowerCase());
    }
    if (!description) {
      setDescription(item.key === "custom" ? "" : item.blurb);
    }
  }

  async function createCompany() {
    if (!selected) return;
    if (name.trim().length < 1) {
      setError("Give your company a name.");
      setStep(2);
      return;
    }

    setBusy(true);
    setError(null);

    try {
      const result = await api.createCompany({
        name: name.trim(),
        businessKey: selected.key,
        website: website.trim(),
        country,
        supportEmail: supportEmail.trim(),
        description: description.trim(),
        audience: audience.trim(),
        tone,
        createTeam,
      });
      setCreated(result.company);
      push("Company created. Your team is ready.");
      setStep(4);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't create that company.");
    } finally {
      setBusy(false);
    }
  }

  const progress = created ? 4 : step;

  return (
    <div className="onboarding">
      <header className="onboarding-top">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            C
          </span>
          <span className="brand-name">Crew</span>
        </div>
        {onCancel ? (
          <Button variant="quiet" size="sm" onClick={onCancel}>
            Cancel
          </Button>
        ) : null}
      </header>

      <div className="onboarding-body">
        <div className="steps mb-16" aria-hidden="true">
          {[1, 2, 3, 4].map((item) => (
            <span key={item} className={`step ${progress >= item ? "on" : ""}`} />
          ))}
        </div>
        <p className="tiny muted mb-8">Step {progress} of 4</p>

        {step === 1 ? (
          <>
            <h1 className="page-title">What business do you want to run?</h1>
            <p className="page-sub mb-16">
              Pick the closest match. We'll set up a team of AI helpers and their first tasks. You can change
              everything later.
            </p>
            <p className="small muted mb-24">
              Not sure where to start?{" "}
              <button className="link-button" onClick={() => navigate("/advise")}>
                Let the advisor study your business first.
              </button>
            </p>

            <Field label="Search businesses" htmlFor="business-search">
              <Input
                id="business-search"
                autoFocus
                placeholder="Try “shop”, “agency”, “clinic”, “saas”…"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
            </Field>

            <div className="business-list">
              {businesses === null ? (
                <div className="skeleton" style={{ height: 120 }} />
              ) : grouped.length === 0 ? (
                <div className="empty">
                  <div className="empty-title">No match</div>
                  <p className="small">
                    Choose{" "}
                    <button
                      className="link-button"
                      onClick={() =>
                        choose({
                          key: "custom",
                          name: "Something else",
                          category: "Custom",
                          emoji: "✨",
                          blurb: "Describe it in your own words.",
                          agents: 3,
                          tasks: 3,
                          integrations: [],
                        })
                      }
                    >
                      Something else
                    </button>{" "}
                    and tell us about it.
                  </p>
                </div>
              ) : (
                grouped.map(([category, items]) => (
                  <section key={category} className="business-group">
                    <h2 className="section-title mb-8">{category}</h2>
                    <div className="business-grid">
                      {items.map((item) => (
                        <button
                          key={item.key}
                          className={`business-card ${selected?.key === item.key ? "selected" : ""}`}
                          onClick={() => choose(item)}
                          aria-pressed={selected?.key === item.key}
                        >
                          <span className="business-emoji" aria-hidden="true">
                            {item.emoji}
                          </span>
                          <span style={{ minWidth: 0 }}>
                            <span className="business-name">{item.name}</span>
                            <span className="business-blurb">{item.blurb}</span>
                          </span>
                        </button>
                      ))}
                    </div>
                  </section>
                ))
              )}

              <section className="business-group">
                <h2 className="section-title mb-8">Not on the list?</h2>
                <button
                  className={`business-card ${selected?.key === "custom" ? "selected" : ""}`}
                  onClick={() =>
                    choose({
                      key: "custom",
                      name: "Something else",
                      category: "Custom",
                      emoji: "✨",
                      blurb: "Describe it in your own words.",
                      agents: 3,
                      tasks: 3,
                      integrations: [],
                    })
                  }
                  aria-pressed={selected?.key === "custom"}
                >
                  <span className="business-emoji" aria-hidden="true">
                    ✨
                  </span>
                  <span>
                    <span className="business-name">Something else</span>
                    <span className="business-blurb">Tell us what it does and we'll build a sensible team.</span>
                  </span>
                </button>
              </section>
            </div>

            {error ? <p className="form-error mt-16">{error}</p> : null}

            <div className="onboarding-actions">
              <Button
                variant="primary"
                disabled={!selected}
                onClick={() => {
                  if (!selected) return;
                  setError(null);
                  setStep(2);
                }}
              >
                Continue
              </Button>
            </div>
          </>
        ) : null}

        {step === 2 ? (
          <>
            <h1 className="page-title">Tell us about {selected?.name ?? "the business"}</h1>
            <p className="page-sub mb-24">
              This is the context your helpers read before every task. Only fill what you know.
            </p>

            {error ? <p className="form-error">{error}</p> : null}

            <div className="form-grid">
              <Field label="Company name" htmlFor="company-name">
                <Input
                  id="company-name"
                  autoFocus
                  maxLength={80}
                  placeholder="e.g. Northstar Studio"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </Field>

              <Field label="Website" hint="Optional." htmlFor="company-website">
                <Input
                  id="company-website"
                  placeholder="yourcompany.com"
                  value={website}
                  onChange={(event) => setWebsite(event.target.value)}
                />
              </Field>

              <Field label="Support email" hint="Where customers write to you. Optional." htmlFor="company-support">
                <Input
                  id="company-support"
                  type="email"
                  placeholder="support@yourcompany.com"
                  value={supportEmail}
                  onChange={(event) => setSupportEmail(event.target.value)}
                />
              </Field>

              <Field label="Country" htmlFor="company-country">
                <Select id="company-country" value={country} onChange={(event) => setCountry(event.target.value)}>
                  {countries.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </Select>
              </Field>

              <Field label="Who do you serve?" hint="Optional, but it sharpens the writing." htmlFor="company-audience">
                <Input
                  id="company-audience"
                  placeholder="e.g. small clinics in Lagos"
                  value={audience}
                  onChange={(event) => setAudience(event.target.value)}
                />
              </Field>
            </div>

            <Field label="What does the business do?" hint="A few sentences is plenty." htmlFor="company-description">
              <Textarea
                id="company-description"
                rows={4}
                placeholder="We sell… We help… What makes us different is…"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </Field>

            <Field label="How should your helpers sound?" htmlFor="company-tone">
              <Select id="company-tone" value={tone} onChange={(event) => setTone(event.target.value)}>
                {tones.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </Select>
            </Field>

            <div className="onboarding-actions">
              <Button onClick={() => setStep(1)}>Back</Button>
              <Button
                variant="primary"
                onClick={() => {
                  if (name.trim().length < 1) {
                    setError("Give your company a name.");
                    return;
                  }
                  setError(null);
                  setStep(3);
                }}
              >
                Continue
              </Button>
            </div>
          </>
        ) : null}

        {step === 3 ? (
          <>
            <h1 className="page-title">Your starting team</h1>
            <p className="page-sub mb-24">
              Based on {selected?.name}, we'll create these helpers and their first tasks. You can
              rename or reconfigure any of them.
            </p>

            {detail === null ? (
              <div className="skeleton" style={{ height: 140 }} />
            ) : (
              <>
                <div className="grid-2 mb-16">
                  {detail.agents.map((agent) => (
                    <div className="card card-tight" key={agent.role}>
                      <div className="row-between" style={{ justifyContent: "flex-start", gap: 10 }}>
                        <span className="avatar" aria-hidden="true">
                          {agent.emoji}
                        </span>
                        <div>
                          <div style={{ fontWeight: 600 }}>{agent.name}</div>
                          <div className="tiny muted">{agent.role}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                <div className="card mb-16">
                  <h2 className="section-title mb-8">First tasks</h2>
                  <ul className="task-preview">
                    {detail.tasks.map((task) => (
                      <li key={task.title}>
                        <span>{task.title}</span>
                        <span className="tiny muted">{task.agent}</span>
                      </li>
                    ))}
                  </ul>
                </div>

                {detail.integrations.length > 0 ? (
                  <p className="small muted mb-16">
                    Works well with: {detail.integrations.join(", ")}.
                  </p>
                ) : null}
              </>
            )}

            <label className="checkbox-row mb-16">
              <input
                type="checkbox"
                checked={createTeam}
                onChange={(event) => setCreateTeam(event.target.checked)}
              />
              <span>Create the helpers and starter tasks for me</span>
            </label>

            <p className="tiny muted mb-16">
              By creating a company you agree to our <a href="/terms">Terms</a> and{" "}
              <a href="/privacy">Privacy Policy</a>. Business details and task text are shared with any AI provider
              you connect.
            </p>

            {error ? <p className="form-error">{error}</p> : null}

            <div className="onboarding-actions">
              <Button onClick={() => setStep(2)}>Back</Button>
              <Button variant="primary" onClick={createCompany} loading={busy}>
                Create company
              </Button>
            </div>
          </>
        ) : null}

        {step === 4 && created ? (
          <>
            <h1 className="page-title">Give {created.name} a brain</h1>
            <p className="page-sub mb-24">
              Add one AI model key so your helpers can think. Pick a free provider to start — you can change this
              later in Settings.
            </p>

            <ModelsPanel />

            <div className="onboarding-actions">
              <Button variant="primary" onClick={() => onDone(created)}>
                Finish
              </Button>
              <Button variant="quiet" onClick={() => onDone(created)}>
                I'll do this later
              </Button>
            </div>
          </>
        ) : null}
      </div>
    </div>
  );
}
