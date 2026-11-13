import { useEffect, useMemo, useRef, useState } from "react";
import {
  api,
  ApiError,
  setCompany as rememberCompany,
  type AdviceAgent,
  type AdviceReport,
  type AdviceSection,
  type Agent,
  type ChatSource,
  type Company,
  type CompanyAdvice,
  type CompanyFile,
  type FinancialModel,
  type BusinessPreset,
} from "../api";
import { Link, navigate } from "../router";
import { Button, Field, Input, Modal, Select, Skeleton, Textarea, useToast } from "../ui";

const countries = [
  "Nigeria",
  "Kenya",
  "Ghana",
  "South Africa",
  "United Kingdom",
  "United States",
  "Canada",
  "India",
  "United Arab Emirates",
];

export function AdviseMe() {
  const { push } = useToast();
  const [companies, setCompanies] = useState<Company[] | null>(null);
  const [company, setActiveCompany] = useState<Company | null>(null);
  const [report, setReport] = useState<AdviceReport | null>(null);
  const [advice, setAdvice] = useState<CompanyAdvice | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [studying, setStudying] = useState(false);
  const [studyError, setStudyError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [agentRec, setAgentRec] = useState<AdviceAgent | null>(null);
  const [share, setShare] = useState<"email" | "whatsapp" | null>(null);

  useEffect(() => {
    api
      .companies()
      .then((result) => {
        setCompanies(result.companies);
        if (result.companies.length === 1) {
          setActiveCompany(result.companies[0]);
        }
      })
      .catch(() => setCompanies([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!company) return;
    let live = true;
    setLoading(true);
    setReport(null);
    setAdvice(null);
    setStudyError(null);

    Promise.all([api.advice(), api.agents()])
      .then(([result, agentResult]) => {
        if (!live) return;
        setReport(result.report);
        setAdvice(result.advice ?? null);
        setAgents(agentResult.agents);
        setError(null);
      })
      .catch((err) => {
        if (!live) return;
        setError(err instanceof ApiError ? err.message : "We couldn't study this company.");
      })
      .finally(() => live && setLoading(false));

    return () => {
      live = false;
    };
  }, [company]);

  useEffect(() => {
    if (!company || !report || advice || studying || !report.has_model) return;
    void study(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [company, report, advice]);

  async function study(automatic = false) {
    if (!company) return;
    setStudying(true);
    setStudyError(null);
    try {
      const result = await api.studyCompany();
      setAdvice(result.advice);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setStudyError("Add a model key to unlock the deep study.");
      } else {
        setStudyError(err instanceof ApiError ? err.message : "We couldn't finish the study.");
        if (!automatic) push("The study failed. Try again.", "error");
      }
    } finally {
      setStudying(false);
    }
  }

  function pickCompany(next: Company) {
    rememberCompany(next.id);
    setActiveCompany(next);
  }

  if (loading && companies === null) {
    return <PageSkeleton />;
  }

  if (companies && companies.length === 0) {
    return <CreateCompany onCreated={(next) => {
      rememberCompany(next.id);
      window.location.assign("/advise");
    }} />;
  }

  if (companies && companies.length > 1 && !company) {
    return (
      <CompanyPicker
        companies={companies}
        onPick={pickCompany}
        onCreate={() => navigate("/new-company")}
      />
    );
  }

  if (error) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't study this company</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={() => window.location.reload()}>
          Try again
        </Button>
      </div>
    );
  }

  if (loading || !report || !company) {
    return <PageSkeleton />;
  }

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>
            {company.name} · {report.country.name}
          </p>
          <h1 className="page-title">Advise Me</h1>
          <p className="page-sub">{report.stage_label}</p>
        </div>
        <div className="advise-score-card">
          <div className="advise-score">{report.score}%</div>
          <div className="tiny muted">
            {report.done} of {report.total} foundations done
          </div>
        </div>
      </header>

      <div className="advise-progress" aria-hidden="true">
        <span style={{ width: `${report.score}%` }} />
      </div>

      <div className="advise-share-row">
        <span className="tiny muted">Share this study</span>
        <Button size="sm" onClick={() => setShare("email")}>Email report</Button>
        <Button size="sm" onClick={() => setShare("whatsapp")}>WhatsApp</Button>
      </div>

      <section className="card advise-next mt-16">
        <div className="row-between" style={{ alignItems: "flex-start" }}>
          <div>
            <h2 className="section-title mb-8">Your next 7 days</h2>
            <p className="small muted">The first unchecked action in each area, in order.</p>
          </div>
          <span className="badge badge-todo">{report.stage}</span>
        </div>
        <ol className="advise-next-list">
          {report.next.map((item, index) => (
            <li key={index}>{item}</li>
          ))}
        </ol>
      </section>

      <CompanyData currency={report.country.currency} />

      <StudyCard
        advice={advice}
        studying={studying}
        error={studyError}
        hasModel={report.has_model}
        onStudy={() => study(false)}
      />

      <div className="advise-grid">
        {report.sections.map((section) => (
          <SectionCard
            key={section.key}
            section={section}
            agents={agents}
            onAgent={setAgentRec}
          />
        ))}
      </div>

      <section className="card mt-16">
        <h2 className="section-title mb-8">{report.country.name} essentials</h2>
        <p className="small muted" style={{ marginTop: -4 }}>
          General guidance only — not legal, tax or investment advice.
        </p>
        <div className="advise-country">
          <div>
            <span className="advise-label">Registration</span>
            <ul>{report.country.registration.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
          <div>
            <span className="advise-label">Tax</span>
            <ul>{report.country.tax.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
          <div>
            <span className="advise-label">Payments</span>
            <ul>{report.country.payments.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
          <div>
            <span className="advise-label">Customers & ads</span>
            <ul>{report.country.ads.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
        </div>
        <p className="tiny muted mt-16">
          General guidance, not legal, tax or investment advice. Confirm with a local professional.
        </p>
      </section>

      <AgentModal
        rec={agentRec}
        company={company}
        onClose={() => setAgentRec(null)}
        onCreated={(agent) => {
          setAgentRec(null);
          navigate(`/helpers/${agent.id}`);
        }}
      />

      <ShareModal
        kind={share}
        company={company}
        report={report}
        advice={advice}
        onClose={() => setShare(null)}
      />

      <AdviceChat company={company} enabled={report.has_model} />
    </>
  );
}

function PageSkeleton() {
  return (
    <div className="stack">
      <div className="skeleton" style={{ height: 36, width: 260 }} />
      <div className="skeleton" style={{ height: 14, width: 420 }} />
      <div className="card">
        <Skeleton lines={4} />
      </div>
      <div className="grid-2">
        <div className="card"><Skeleton lines={5} /></div>
        <div className="card"><Skeleton lines={5} /></div>
      </div>
      <div className="grid-2">
        <div className="card"><Skeleton lines={5} /></div>
        <div className="card"><Skeleton lines={5} /></div>
      </div>
    </div>
  );
}

function CompanyPicker({
  companies,
  onPick,
  onCreate,
}: {
  companies: Company[];
  onPick: (company: Company) => void;
  onCreate: () => void;
}) {
  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Advise Me</h1>
          <p className="page-sub">Choose the company you want studied.</p>
        </div>
        <Button onClick={onCreate}>New company</Button>
      </header>

      <div className="company-picker">
        {companies.map((item) => (
          <button key={item.id} className="company-card" onClick={() => onPick(item)}>
            <span className="company-avatar" aria-hidden="true">
              {item.name.slice(0, 1).toUpperCase()}
            </span>
            <span style={{ minWidth: 0 }}>
              <span className="company-card-name">{item.name}</span>
              <span className="company-card-meta">
                {item.is_product ? "Product · " : ""}{item.country || "Country not set"} · {item.agent_count} helpers · {item.task_count} tasks
              </span>
            </span>
            <span className="tiny muted">{item.role === "viewer" ? "View only" : "Open"}</span>
          </button>
        ))}
      </div>
    </>
  );
}

function CreateCompany({ onCreated }: { onCreated: (company: Company) => void }) {
  const [presets, setPresets] = useState<BusinessPreset[] | null>(null);
  const [name, setName] = useState("");
  const [country, setCountry] = useState("Nigeria");
  const [businessKey, setBusinessKey] = useState("custom");
  const [website, setWebsite] = useState("");
  const [supportEmail, setSupportEmail] = useState("");
  const [description, setDescription] = useState("");
  const [audience, setAudience] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .presets()
      .then((result) => setPresets(result.businesses))
      .catch(() => setPresets([]));
  }, []);

  const groups = useMemo(() => {
    const map = new Map<string, BusinessPreset[]>();
    for (const item of presets ?? []) {
      const list = map.get(item.category) ?? [];
      list.push(item);
      map.set(item.category, list);
    }
    return Array.from(map.entries());
  }, [presets]);

  async function create(event: React.FormEvent) {
    event.preventDefault();
    if (name.trim() === "") {
      setError("Give your company a name.");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      const result = await api.createCompany({
        name: name.trim(),
        country,
        businessKey,
        website: website.trim(),
        supportEmail: supportEmail.trim(),
        description: description.trim(),
        audience: audience.trim(),
        createTeam: true,
      });
      onCreated(result.company);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't create that company.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ maxWidth: 760, margin: "0 auto" }}>
      <header className="page-head">
        <div>
          <h1 className="page-title">Advise Me</h1>
          <p className="page-sub">Create your company and get a country-aware plan for starting and growing it.</p>
        </div>
      </header>

      <form className="card" onSubmit={create}>
        <h2 className="section-title mb-8">Start your company</h2>
        <p className="small muted mb-16">
          We'll build a starter team, then study the business and show what is done and what is missing.
        </p>

        <div className="form-grid">
          <Field label="Company name" htmlFor="advise-company-name">
            <Input
              id="advise-company-name"
              autoFocus
              value={name}
              maxLength={80}
              placeholder="e.g. Northstar Studio"
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          <Field label="Country" htmlFor="advise-country">
            <Select id="advise-country" value={country} onChange={(event) => setCountry(event.target.value)}>
              {countries.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field label="What type of business?" htmlFor="advise-business">
          <Select
            id="advise-business"
            value={businessKey}
            onChange={(event) => {
              setBusinessKey(event.target.value);
              const found = (presets ?? []).find((item) => item.key === event.target.value);
              if (found && description === "") setDescription(found.blurb);
            }}
          >
            <option value="custom">Something else</option>
            {groups.map(([category, items]) => (
              <optgroup key={category} label={category}>
                {items.map((item) => (
                  <option key={item.key} value={item.key}>
                    {item.emoji} {item.name}
                  </option>
                ))}
              </optgroup>
            ))}
          </Select>
        </Field>

        <div className="form-grid">
          <Field label="Website" hint="Optional." htmlFor="advise-website">
            <Input id="advise-website" value={website} onChange={(event) => setWebsite(event.target.value)} placeholder="yourcompany.com" />
          </Field>
          <Field label="Support email" hint="Optional." htmlFor="advise-support">
            <Input id="advise-support" value={supportEmail} onChange={(event) => setSupportEmail(event.target.value)} placeholder="support@…" />
          </Field>
          <Field label="Who do you serve?" htmlFor="advise-audience">
            <Input id="advise-audience" value={audience} onChange={(event) => setAudience(event.target.value)} />
          </Field>
        </div>

        <Field label="What does the business do?" htmlFor="advise-description">
          <Textarea
            id="advise-description"
            rows={3}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="We help…"
          />
        </Field>

        {error ? <p className="form-error">{error}</p> : null}

        <Button type="submit" variant="primary" loading={busy}>
          Create company & study it
        </Button>
      </form>
    </div>
  );
}

function StudyCard({
  advice,
  studying,
  error,
  hasModel,
  onStudy,
}: {
  advice: CompanyAdvice | null;
  studying: boolean;
  error: string | null;
  hasModel: boolean;
  onStudy: () => void;
}) {
  if (studying) {
    return (
      <section className="card advise-study mt-16">
        <div className="row-between">
          <div>
            <h2 className="section-title mb-8">Studying your company…</h2>
            <p className="small muted">Reading the profile, helpers, tasks, routines, goals and country rules.</p>
          </div>
          <span className="spinner" aria-label="Studying company" />
        </div>
        <Skeleton lines={4} />
      </section>
    );
  }

  if (!advice || !advice.analysis) {
    return (
      <section className="card advise-study mt-16">
        <div className="row-between" style={{ alignItems: "flex-start" }}>
          <div>
            <h2 className="section-title mb-8">Deep study</h2>
            <p className="small muted">
              {hasModel
                ? "The advisor will find the three constraints holding this company back."
                : "Add a model key in Settings → AI models to unlock the deep study."}
            </p>
          </div>
          {hasModel ? <Button onClick={onStudy}>Study my company</Button> : <Link to="/settings">Add model</Link>}
        </div>
        {error ? <p className="form-error mt-16">{error}</p> : null}
      </section>
    );
  }

  return (
    <section className="card advise-study mt-16">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <p className="advise-label">Deep study</p>
          <h2 className="section-title" style={{ marginTop: 4 }}>
            {advice.analysis?.headline}
          </h2>
          <p className="tiny muted">{advice.model} · {new Date(advice.created_at).toLocaleDateString()}</p>
        </div>
        <Button size="sm" onClick={onStudy}>
          Refresh study
        </Button>
      </div>

      <div className="advise-constraints">
        {(advice.analysis?.constraints ?? []).map((item, index) => (
          <div className="advise-constraint" key={index}>
            <strong>{item.title}</strong>
            <p>{item.why}</p>
            <p className="small"><strong>Fix:</strong> {item.fix}</p>
          </div>
        ))}
      </div>

      <div className="advise-study-grid">
        <div>
          <span className="advise-label">Next 7 days</span>
          <ol>{advice.analysis?.next_7_days?.map((item, i) => <li key={i}>{item}</li>)}</ol>
        </div>
        <div>
          <span className="advise-label">Opportunities</span>
          <ul>{advice.analysis?.opportunities?.map((item, i) => <li key={i}>{item}</li>)}</ul>
        </div>
        <div>
          <span className="advise-label">Risks</span>
          <ul>{advice.analysis?.risks?.map((item, i) => <li key={i}>{item}</li>)}</ul>
        </div>
      </div>
      {error ? <p className="form-error mt-16">{error}</p> : null}
    </section>
  );
}

function CompanyData({ currency }: { currency: string }) {
  return (
    <div className="advise-data-grid">
      <FilesCard />
      <FinancialCard currency={currency} />
    </div>
  );
}

function FilesCard() {
  const [files, setFiles] = useState<CompanyFile[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api
      .files()
      .then((result) => setFiles(result.files))
      .catch(() => setFiles([]));
  }, []);

  async function upload(file: File) {
    setBusy(true);
    setError(null);
    try {
      const result = await api.uploadFile(file);
      setFiles((current) => [result.file, ...(current ?? [])]);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't upload that file.");
    } finally {
      setBusy(false);
      if (input.current) input.current.value = "";
    }
  }

  return (
    <section className="card advise-data-card">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <p className="advise-label">Company files</p>
          <h2 className="section-title" style={{ marginTop: 3 }}>Give the advisor evidence</h2>
          <p className="small muted">Price lists, customer lists, plans, receipts, CSV exports.</p>
        </div>
        <span className="badge badge-todo">{files ? files.length : "…"}/20</span>
      </div>

      <input
        ref={input}
        type="file"
        hidden
        accept=".txt,.csv,.json,.md,.pdf,.png,.jpg,.jpeg,.webp,.doc,.docx,.xls,.xlsx"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) void upload(file);
        }}
      />
      <Button size="sm" onClick={() => input.current?.click()} loading={busy}>
        Upload a file
      </Button>
      <p className="tiny muted">Text, CSV and JSON are read by the study. PDFs and images are stored but not parsed yet.</p>

      {files === null ? <Skeleton lines={2} /> : null}
      {files && files.length === 0 ? <p className="small muted">No files yet.</p> : null}
      {files && files.length > 0 ? (
        <div className="advise-file-list">
          {files.map((file) => (
            <div className="advise-file" key={file.id}>
              <span style={{ minWidth: 0 }}>
                <span className="advise-file-name">{file.name}</span>
                <span className="tiny muted">{Math.ceil(file.size / 1024)} KB · {file.mime || "file"}</span>
              </span>
              <button
                className="link-button tiny"
                onClick={async () => {
                  try {
                    const blob = await api.downloadFile(file.id);
                    const url = URL.createObjectURL(blob);
                    const link = document.createElement("a");
                    link.href = url;
                    link.download = file.name;
                    link.click();
                    URL.revokeObjectURL(url);
                  } catch {
                    setError("We couldn't download that file.");
                  }
                }}
              >
                Download
              </button>
              <button
                className="link-button"
                onClick={async () => {
                  try {
                    await api.deleteFile(file.id);
                    setFiles((current) => (current ?? []).filter((item) => item.id !== file.id));
                  } catch {
                    setError("We couldn't remove that file.");
                  }
                }}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      ) : null}
      {error ? <p className="form-error">{error}</p> : null}
    </section>
  );
}

function FinancialCard({ currency }: { currency: string }) {
  const { push } = useToast();
  const [model, setModel] = useState<FinancialModel | null>(null);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({
    period: "This month",
    revenue: "",
    cogs: "",
    opex: "",
    cash: "",
    receivables: "",
    payables: "",
    source: "",
    audited: false,
  });

  useEffect(() => {
    api
      .financials()
      .then((result) => setModel(result.model))
      .catch(() => setModel(null))
      .finally(() => setLoading(false));
  }, []);

  function update(key: keyof typeof form, value: string | boolean) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await api.saveFinancials({
        period: form.period,
        revenue: Number(form.revenue) || 0,
        cogs: Number(form.cogs) || 0,
        opex: Number(form.opex) || 0,
        cash: Number(form.cash) || 0,
        receivables: Number(form.receivables) || 0,
        payables: Number(form.payables) || 0,
        source: form.source,
        audited: form.audited,
      });
      setModel(result.model);
      setOpen(false);
      push("Financial model updated.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save those numbers.");
    } finally {
      setBusy(false);
    }
  }

  const format = (value: number) => `${currency.split(" ")[0]} ${Math.round(value).toLocaleString()}`;

  return (
    <section className="card advise-data-card">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <p className="advise-label">Financial model</p>
          <h2 className="section-title" style={{ marginTop: 3 }}>Numbers the study can use</h2>
          <p className="small muted">Revenue, margin, burn and runway. Mark audited accounts when you have them.</p>
        </div>
        {model ? <span className={`badge ${model.quality === "Audited" ? "badge-done" : "badge-working"}`}>{model.quality}</span> : null}
      </div>

      {loading ? <Skeleton lines={3} /> : null}
      {!loading && !model ? (
        <>
          <p className="small muted">No financials yet. Add one month and the deep study gets sharper.</p>
          <Button size="sm" onClick={() => setOpen(true)}>Add numbers</Button>
        </>
      ) : null}

      {model ? (
        <>
          <div className="advise-finance-grid">
            <Metric label="Revenue" value={format(model.revenue)} />
            <Metric label="Gross margin" value={`${Math.round(model.gross_margin * 100)}%`} />
            <Metric label="Net profit" value={format(model.net_profit)} tone={model.net_profit < 0 ? "bad" : "good"} />
            <Metric label="Cash" value={format(model.cash)} />
            <Metric label="Runway" value={model.runway_months > 0 ? `${model.runway_months.toFixed(1)} months` : "Profitable"} />
            <Metric label="Break-even" value={format(model.break_even_sales)} />
          </div>
          <div className="row-between">
            <span className="tiny muted">{model.period} · {model.source || "management numbers"}</span>
            <Button size="sm" onClick={() => setOpen(true)}>Update</Button>
          </div>
        </>
      ) : null}

      <Modal
        open={open}
        title={model ? "Update the financial model" : "Add a financial snapshot"}
        onClose={() => setOpen(false)}
        footer={
          <>
            <Button onClick={() => setOpen(false)} disabled={busy}>Cancel</Button>
            <Button variant="primary" onClick={save} loading={busy}>Save model</Button>
          </>
        }
      >
        <p className="small muted mb-16">
          One month is enough. Mark “audited” only if these numbers come from audited statements.
        </p>
        <div className="form-grid">
          <Field label="Period" htmlFor="fin-period">
            <Input id="fin-period" value={form.period} onChange={(event) => update("period", event.target.value)} />
          </Field>
          <Field label="Revenue" htmlFor="fin-revenue">
            <Input id="fin-revenue" type="number" min="0" value={form.revenue} onChange={(event) => update("revenue", event.target.value)} />
          </Field>
          <Field label="Cost of sales" htmlFor="fin-cogs">
            <Input id="fin-cogs" type="number" min="0" value={form.cogs} onChange={(event) => update("cogs", event.target.value)} />
          </Field>
          <Field label="Operating costs" htmlFor="fin-opex">
            <Input id="fin-opex" type="number" min="0" value={form.opex} onChange={(event) => update("opex", event.target.value)} />
          </Field>
          <Field label="Cash" htmlFor="fin-cash">
            <Input id="fin-cash" type="number" min="0" value={form.cash} onChange={(event) => update("cash", event.target.value)} />
          </Field>
          <Field label="Receivables" htmlFor="fin-receivables">
            <Input id="fin-receivables" type="number" min="0" value={form.receivables} onChange={(event) => update("receivables", event.target.value)} />
          </Field>
          <Field label="Payables" htmlFor="fin-payables">
            <Input id="fin-payables" type="number" min="0" value={form.payables} onChange={(event) => update("payables", event.target.value)} />
          </Field>
        </div>
        <Field label="Source" hint="e.g. management accounts, accountant, audited statements." htmlFor="fin-source">
          <Input id="fin-source" value={form.source} onChange={(event) => update("source", event.target.value)} />
        </Field>
        <label className="checkbox-row">
          <input type="checkbox" checked={form.audited} onChange={(event) => update("audited", event.target.checked)} />
          <span>These numbers are from audited statements</span>
        </label>
        {error ? <p className="form-error">{error}</p> : null}
      </Modal>
    </section>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone?: "good" | "bad" }) {
  return (
    <div className="advise-metric">
      <span className="tiny muted">{label}</span>
      <strong className={tone === "bad" ? "text-danger" : tone === "good" ? "text-ok" : ""}>{value}</strong>
    </div>
  );
}

function SectionCard({
  section,
  agents,
  onAgent,
}: {
  section: AdviceSection;
  agents: Agent[];
  onAgent: (agent: AdviceAgent) => void;
}) {
  return (
    <section className={`advise-section advise-${section.status}`}>
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <span className="advise-label">{section.status === "done" ? "Done" : section.status === "partial" ? "In progress" : "Not started"}</span>
          <h2 className="section-title" style={{ marginTop: 3 }}>{section.title}</h2>
          <p className="small muted">{section.summary}</p>
        </div>
        <span className="advise-percent">{section.progress}%</span>
      </div>

      <div className="advise-checks">
        {section.checks.map((check) => (
          <div className={`advise-check ${check.done ? "done" : ""}`} key={check.label}>
            <span className="advise-check-mark" aria-hidden="true">
              {check.done ? "✓" : "○"}
            </span>
            <span>
              <span className="advise-check-label">{check.label}</span>
              {!check.done ? <span className="advise-check-action">{check.action}</span> : null}
            </span>
          </div>
        ))}
      </div>

      {section.actions.length > 0 ? (
        <div className="advise-actions">
          <span className="advise-label">Do this</span>
          <ul>{section.actions.map((item, i) => <li key={i}>{item}</li>)}</ul>
        </div>
      ) : null}

      {section.country_note ? <p className="advise-country-note">{section.country_note}</p> : null}

      <div className="advise-agents">
        {section.agents.map((rec) => {
          const existing = agents.find(
            (agent) => agent.role === rec.role || agent.name.toLowerCase() === rec.name.toLowerCase(),
          );
          return (
            <Button
              key={rec.name}
              size="sm"
              onClick={() => (existing ? navigate(`/helpers/${existing.id}`) : onAgent(rec))}
            >
              {existing ? `Open ${existing.name}` : `Create ${rec.name}`}
            </Button>
          );
        })}
      </div>
    </section>
  );
}

function ShareModal({
  kind,
  company,
  report,
  advice,
  onClose,
}: {
  kind: "email" | "whatsapp" | null;
  company: Company;
  report: AdviceReport;
  advice: CompanyAdvice | null;
  onClose: () => void;
}) {
  const { push } = useToast();
  const [to, setTo] = useState("");
  const [subject, setSubject] = useState(`Advice for ${company.name}`);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!kind) return;
    const lines = [
      `${company.name} — ${report.score}% foundation score`,
      report.stage_label,
      "",
      advice?.analysis?.headline ?? "Next 7 days:",
      ...(advice?.analysis?.next_7_days ?? report.next).map((item, index) => `${index + 1}. ${item}`),
      "",
      "General guidance, not legal, tax or investment advice.",
    ];
    setBody(lines.join("\n"));
    setSubject(`Advice for ${company.name}`);
    setError(null);
  }, [kind, company.name, report, advice]);

  async function send() {
    setBusy(true);
    setError(null);
    try {
      if (kind === "email") {
        await api.sendEmail({ to, subject, body });
        push("Report emailed.");
      } else {
        await api.sendWhatsApp({ to, message: body });
        push("WhatsApp message sent.");
      }
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't send that message.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={kind !== null}
      title={kind === "email" ? "Email this report" : "Send on WhatsApp"}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>Cancel</Button>
          <Button variant="primary" onClick={send} loading={busy} disabled={to.trim() === "" || body.trim() === ""}>
            Send
          </Button>
        </>
      }
    >
      {kind === "whatsapp" ? (
        <p className="small muted mb-16">
          Connect WhatsApp in Settings → Integrations. WhatsApp only allows free-form text inside the 24-hour customer window.
        </p>
      ) : null}
      <Field label={kind === "email" ? "Recipient email" : "Recipient number"} htmlFor="share-to">
        <Input
          id="share-to"
          value={to}
          placeholder={kind === "email" ? "client@example.com" : "2348012345678"}
          onChange={(event) => setTo(event.target.value)}
        />
      </Field>
      {kind === "email" ? (
        <Field label="Subject" htmlFor="share-subject">
          <Input id="share-subject" value={subject} onChange={(event) => setSubject(event.target.value)} />
        </Field>
      ) : null}
      <Field label="Message" htmlFor="share-body">
        <Textarea id="share-body" rows={10} value={body} onChange={(event) => setBody(event.target.value)} />
      </Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function richText(text: string) {
  return text.split(/(\*\*[^*]+\*\*)/g).map((part, index) => {
    if (part.startsWith("**") && part.endsWith("**") && part.length > 4) {
      return <strong key={index}>{part.slice(2, -2)}</strong>;
    }
    return part;
  });
}

function AgentModal({
  rec,
  company,
  onClose,
  onCreated,
}: {
  rec: AdviceAgent | null;
  company: Company;
  onClose: () => void;
  onCreated: (agent: Agent) => void;
}) {
  const { push } = useToast();
  const [name, setName] = useState("");
  const [role, setRole] = useState("Helper");
  const [emoji, setEmoji] = useState("🤖");
  const [instructions, setInstructions] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!rec) return;
    setName(rec.name);
    setRole(rec.role);
    setEmoji(rec.emoji);
    setInstructions(rec.instructions.replace(/\{\{company\}\}/g, company.name));
    setError(null);
  }, [rec, company.name]);

  async function create() {
    if (!rec) return;
    if (name.trim() === "") {
      setError("Give the helper a name.");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      const result = await api.createAgent({
        name: name.trim(),
        role,
        emoji,
        instructions: instructions.trim(),
        skills: rec.skills,
      });
      push(`${name} is ready.`);
      onCreated(result.agent);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't create that helper.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={rec !== null}
      title={`Create ${rec?.name ?? "helper"}`}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>Cancel</Button>
          <Button variant="primary" onClick={create} loading={busy}>Create helper</Button>
        </>
      }
    >
      <p className="small muted mb-16">This helper is suggested by the study. You can edit it before creating.</p>
      <div className="form-grid">
        <Field label="Name" htmlFor="advise-agent-name">
          <Input id="advise-agent-name" value={name} maxLength={40} onChange={(event) => setName(event.target.value)} />
        </Field>
        <Field label="Role" htmlFor="advise-agent-role">
          <Input id="advise-agent-role" value={role} maxLength={40} onChange={(event) => setRole(event.target.value)} />
        </Field>
        <Field label="Emoji" htmlFor="advise-agent-emoji">
          <Input id="advise-agent-emoji" value={emoji} maxLength={4} onChange={(event) => setEmoji(event.target.value)} />
        </Field>
      </div>
      <Field label="Instructions" htmlFor="advise-agent-instructions">
        <Textarea id="advise-agent-instructions" rows={5} value={instructions} onChange={(event) => setInstructions(event.target.value)} />
      </Field>
      {rec?.skills?.length ? <p className="tiny muted">Skills: {rec.skills.join(", ")}</p> : null}
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function AdviceChat({ company, enabled }: { company: Company; enabled: boolean }) {
  const { push } = useToast();
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [turns, setTurns] = useState<{ role: "user" | "assistant"; content: string; sources?: ChatSource[] }[]>([]);
  const bottom = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottom.current?.scrollIntoView({ behavior: "smooth" });
  }, [turns, busy, open]);

  async function send() {
    if (message.trim() === "" || busy) return;
    const question = message.trim();
    const history = turns.map((turn) => ({ role: turn.role, content: turn.content }));
    setTurns((current) => [...current, { role: "user", content: question }]);
    setMessage("");
    setBusy(true);
    setError(null);

    try {
      const result = await api.adviceChat(question, history);
      setTurns((current) => [...current, { role: "assistant", content: result.answer, sources: result.sources }]);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The advisor couldn't answer.");
      push("The advisor couldn't answer. Try again.", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={`advisor-chat ${open ? "open" : ""}`}>
      {open ? (
        <div className="advisor-panel">
          <div className="advisor-head">
            <div>
              <strong>Ask about {company.name}</strong>
              <p className="tiny muted">Company data + web sources</p>
            </div>
            <button className="advisor-close" onClick={() => setOpen(false)} aria-label="Close advisor">
              ×
            </button>
          </div>

          <div className="advisor-messages">
            {turns.length === 0 ? (
              <div className="advisor-empty">
                <p>Ask anything about this company.</p>
                <div className="advisor-suggestions">
                  {["What should I do first?", "Who is my best customer?", "Why are sales slow?"].map((item) => (
                    <button key={item} className="chip chip-button" onClick={() => setMessage(item)}>
                      {item}
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            {turns.map((turn, index) => (
              <div key={index} className={`advisor-turn ${turn.role}`}>
                <p>{richText(turn.content)}</p>
                {turn.sources?.length ? (
                  <div className="advisor-sources">
                    {turn.sources.map((source) => (
                      <a key={source.url} href={source.url} target="_blank" rel="noreferrer noopener">
                        {source.title}
                      </a>
                    ))}
                  </div>
                ) : null}
              </div>
            ))}

            {busy ? (
              <div className="advisor-turn assistant">
                <span className="spinner" aria-label="Thinking" />
                <span className="tiny muted">Reading your company…</span>
              </div>
            ) : null}
            {error ? <p className="form-error">{error}</p> : null}
            <div ref={bottom} />
          </div>

          <div className="advisor-input">
            <Textarea
              rows={2}
              value={message}
              placeholder={enabled ? "Ask a question…" : "Add a model key to chat"}
              disabled={!enabled || busy}
              onChange={(event) => setMessage(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  void send();
                }
              }}
            />
            <Button variant="primary" onClick={send} loading={busy} disabled={!enabled || message.trim() === ""}>
              Send
            </Button>
          </div>
        </div>
      ) : null}

      <button className="advisor-fab" onClick={() => setOpen((value) => !value)} aria-label="Ask the advisor">
        {open ? "×" : "✦"}
      </button>
    </div>
  );
}
