import { useEffect, useRef, useState } from "react";
import {
  api,
  ApiError,
  type OutreachMessage,
  type OutreachPayload,
  type OutreachProspect,
  type OutreachSettings,
} from "../api";
import { Button, Field, Input, Modal, Skeleton, Tabs, Textarea, useToast } from "../ui";

const tabs = [
  { id: "pipeline", label: "Pipeline" },
  { id: "campaigns", label: "Campaigns" },
  { id: "outbox", label: "Outbox" },
  { id: "replies", label: "Replies" },
  { id: "settings", label: "Settings" },
];

export function Outreach() {
  const { push } = useToast();
  const [data, setData] = useState<OutreachPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState("pipeline");
  const [prospectOpen, setProspectOpen] = useState(false);
  const [discoverOpen, setDiscoverOpen] = useState(false);
  const [campaignOpen, setCampaignOpen] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const importInput = useRef<HTMLInputElement>(null);

  function load() {
    api
      .outreach()
      .then((result) => {
        setData(result.outreach);
        setError(null);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "We couldn't load outreach."))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function scan(prospect: OutreachProspect) {
    setBusy("scan:" + prospect.id);
    try {
      await api.scanOutreachProspect(prospect.id);
      push("Website scanned. Signals and contacts updated.");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "The scan failed.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function draft(prospect: OutreachProspect) {
    if (!data) return;
    const campaign = data.campaigns.find((item) => item.status !== "paused");
    if (!campaign) {
      push("Create a campaign first.", "error");
      return;
    }
    const contact = (prospect.contacts ?? []).find((item) => item.email);
    if (!contact) {
      push("Add a contact email first.", "error");
      return;
    }
    setBusy("draft:" + prospect.id);
    try {
      const result = await api.draftOutreach(campaign.id, prospect.id, contact.id, 1);
      push(result.model === "template" ? "Draft created from the template." : "AI draft created.");
      setTab("outbox");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "The draft failed.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function send(message: OutreachMessage) {
    setBusy("send:" + message.id);
    try {
      const result = await api.sendOutreachMessage(message.id);
      push(result.sent ? "Sent." : "Nothing sent; check the daily limit or suppression list.", result.sent ? "info" : "error");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "The message wasn't sent.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function batchSend() {
    setBusy("batch");
    try {
      const result = await api.sendOutreachBatch();
      push(`Sent ${result.sent}; skipped ${result.skipped}.`);
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "The batch didn't send.", "error");
    } finally {
      setBusy(null);
    }
  }

  if (loading) {
    return (
      <div className="stack">
        <div className="skeleton" style={{ height: 38, width: 280 }} />
        <div className="grid-2"><div className="card"><Skeleton lines={4} /></div><div className="card"><Skeleton lines={4} /></div></div>
        <div className="card"><Skeleton lines={6} /></div>
      </div>
    );
  }
  if (error && !data) {
    return <div className="empty"><div className="empty-title">We couldn't load Outreach</div><p className="small">{error}</p><Button className="mt-16" onClick={load}>Try again</Button></div>;
  }
  if (!data) return null;

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>Sellbird-inspired outreach</p>
          <h1 className="page-title">Outreach</h1>
          <p className="page-sub">Find prospects, read buying signals, draft personal email, approve, send and track replies.</p>
        </div>
        <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
          <input
            ref={importInput}
            type="file"
            accept=".csv,text/csv"
            hidden
            onChange={async (event) => {
              const file = event.target.files?.[0];
              if (!file) return;
              setBusy("import");
              try {
                const result = await api.importOutreach(file);
                push(`Imported ${result.imported} prospects.`);
                load();
              } catch (err) {
                push(err instanceof ApiError ? err.message : "The CSV couldn't be imported.", "error");
              } finally {
                setBusy(null);
                if (importInput.current) importInput.current.value = "";
              }
            }}
          />
          <Button
            onClick={async () => {
              try {
                const blob = await api.outreachExport();
                const url = URL.createObjectURL(blob);
                const link = document.createElement("a");
                link.href = url;
                link.download = "crew-outreach.csv";
                link.click();
                URL.revokeObjectURL(url);
              } catch {
                push("We couldn't export the CSV.", "error");
              }
            }}
          >
            Export CSV
          </Button>
          <Button onClick={() => importInput.current?.click()} loading={busy === "import"}>Import CSV</Button>
          <Button onClick={() => setDiscoverOpen(true)}>Discover</Button>
          <Button variant="primary" onClick={() => setProspectOpen(true)}>New prospect</Button>
        </div>
      </header>

      <div className="outreach-stats">
        <Metric label="Prospects" value={data.stats.prospects ?? data.prospects.length} />
        <Metric label="Drafts" value={data.stats.draft ?? 0} />
        <Metric label="Approved" value={data.stats.approved ?? 0} />
        <Metric label="Sent" value={data.stats.sent ?? 0} />
        <Metric label="Replied" value={data.stats.replied ?? 0} />
      </div>

      <Tabs items={tabs} active={tab} onChange={setTab} />

      {tab === "pipeline" ? (
        <Pipeline data={data} busy={busy} onScan={scan} onDraft={draft} onAdd={() => setProspectOpen(true)} />
      ) : null}
      {tab === "campaigns" ? (
        <Campaigns data={data} onAdd={() => setCampaignOpen(true)} onChanged={load} />
      ) : null}
      {tab === "outbox" ? (
        <Outbox data={data} busy={busy} onSend={send} onBatch={batchSend} onChanged={load} />
      ) : null}
      {tab === "replies" ? <Replies data={data} /> : null}
      {tab === "settings" ? <OutreachSettingsPanel data={data} onChanged={load} /> : null}

      <ProspectModal open={prospectOpen} onClose={() => setProspectOpen(false)} onSaved={() => { setProspectOpen(false); load(); }} />
      <DiscoverModal open={discoverOpen} onClose={() => setDiscoverOpen(false)} onSaved={() => { setDiscoverOpen(false); load(); }} />
      <CampaignModal open={campaignOpen} onClose={() => setCampaignOpen(false)} onSaved={() => { setCampaignOpen(false); load(); }} />
    </>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return <div className="card outreach-metric"><span className="tiny muted">{label}</span><strong>{value}</strong></div>;
}

function Pipeline({
  data,
  busy,
  onScan,
  onDraft,
  onAdd,
}: {
  data: OutreachPayload;
  busy: string | null;
  onScan: (prospect: OutreachProspect) => void;
  onDraft: (prospect: OutreachProspect) => void;
  onAdd: () => void;
}) {
  return (
    <section className="card">
      <div className="row-between">
        <div>
          <h2 className="section-title">Prospects</h2>
          <p className="small muted">Stage, score, observed signals and the contact Crew can reach.</p>
        </div>
        <Button onClick={onAdd}>Add prospect</Button>
      </div>
      {data.prospects.length === 0 ? (
        <div className="empty" style={{ padding: "30px 0" }}>
          <div className="empty-title">No prospects yet</div>
          <p className="small">Import Sellbird’s CSV, discover local businesses, or add one manually.</p>
        </div>
      ) : (
        <div className="outreach-prospects">
          {data.prospects.map((prospect) => {
            const contact = (prospect.contacts ?? []).find((item) => item.email) ?? prospect.contacts?.[0];
            return (
              <article className="outreach-prospect" key={prospect.id}>
                <div className="row-between" style={{ alignItems: "flex-start" }}>
                  <div style={{ minWidth: 0 }}>
                    <strong>{prospect.name}</strong>
                    <div className="tiny muted">{prospect.industry || "industry unknown"} · {prospect.location || "location unknown"} · {prospect.source}</div>
                  </div>
                  <span className={`badge outreach-stage-${prospect.stage}`}>{prospect.stage}</span>
                </div>
                <div className="outreach-score-row">
                  <span className="outreach-score">{prospect.score}<span className="tiny muted">/100</span></span>
                  <div className="outreach-signal-chips">
                    {(prospect.signals ?? []).filter((signal) => signal.found).slice(0, 4).map((signal) => (
                      <span className="chip" key={signal.key}>{signal.label}</span>
                    ))}
                  </div>
                </div>
                <div className="row-between tiny">
                  <span className="muted">{contact ? `${contact.email} · ${contact.email_status}` : "No contact email"}</span>
                  <span className="row" style={{ gap: 5 }}>
                    <Button size="sm" variant="quiet" loading={busy === "scan:" + prospect.id} onClick={() => onScan(prospect)}>Scan</Button>
                    <Button size="sm" loading={busy === "draft:" + prospect.id} onClick={() => onDraft(prospect)}>Draft</Button>
                  </span>
                </div>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}

function Campaigns({ data, onAdd, onChanged }: { data: OutreachPayload; onAdd: () => void; onChanged: () => void }) {
  const { push } = useToast();
  return (
    <section className="card">
      <div className="row-between">
        <div>
          <h2 className="section-title">Campaigns</h2>
          <p className="small muted">Offer, audience, tone and a short sequence. Nothing sends without approval.</p>
        </div>
        <Button onClick={onAdd}>New campaign</Button>
      </div>
      {data.campaigns.length === 0 ? (
        <p className="small muted mt-16">No campaign yet. Create one before drafting.</p>
      ) : (
        <div className="outreach-campaigns">
          {data.campaigns.map((campaign) => (
            <article className="outreach-campaign" key={campaign.id}>
              <div className="row-between">
                <strong>{campaign.name}</strong>
                <span className="badge">{campaign.status}</span>
              </div>
              <p className="small muted">{campaign.offer}</p>
              <div className="tiny muted">Audience: {campaign.audience || "not set"} · Daily limit: {campaign.daily_limit}</div>
              <div className="outreach-steps">
                {(campaign.steps ?? []).map((step) => (
                  <div className="outreach-step" key={step.id}>
                    <span>Step {step.position} · day {step.delay_days}</span>
                    <strong>{step.subject || "(no subject)"}</strong>
                  </div>
                ))}
              </div>
              <div className="row mt-8">
                <Button
                  size="sm"
                  onClick={async () => {
                    try {
                      await api.updateOutreachCampaign(campaign.id, { status: campaign.status === "active" ? "paused" : "active", name: campaign.name, offer: campaign.offer, audience: campaign.audience, tone: campaign.tone, daily_limit: campaign.daily_limit, autopilot: campaign.autopilot });
                      push(campaign.status === "active" ? "Campaign paused." : "Campaign active.");
                      onChanged();
                    } catch {
                      push("We couldn't update that campaign.", "error");
                    }
                  }}
                >
                  {campaign.status === "active" ? "Pause" : "Activate"}
                </Button>
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function Outbox({ data, busy, onSend, onBatch, onChanged }: { data: OutreachPayload; busy: string | null; onSend: (message: OutreachMessage) => void; onBatch: () => void; onChanged: () => void }) {
  const { push } = useToast();
  const messages = (data.messages ?? []).filter((message) => message.status !== "replied" && message.status !== "optout");
  return (
    <section className="card">
      <div className="row-between">
        <div>
          <h2 className="section-title">Outbox</h2>
          <p className="small muted">Draft → approve → send. SMTP is {data.send_ready ? "ready" : "not configured"}.</p>
        </div>
        <Button loading={busy === "batch"} onClick={onBatch}>Send approved</Button>
      </div>
      {messages.length === 0 ? <p className="small muted mt-16">No drafts. Open a prospect and press Draft.</p> : (
        <div className="outreach-outbox">
          {messages.map((message) => (
            <article className="outreach-message" key={message.id}>
              <div className="row-between">
                <strong>{message.subject}</strong>
                <span className={`badge outreach-msg-${message.status}`}>{message.status}</span>
              </div>
              <div className="tiny muted">To {message.to} · step {message.step} · {new Date(message.created_at).toLocaleDateString()}</div>
              <p className="small outreach-body">{message.body}</p>
              <div className="row" style={{ gap: 6 }}>
                {message.status === "draft" ? (
                  <Button size="sm" onClick={async () => {
                    try { await api.outreachMessageStatus(message.id, "approved"); push("Approved."); onChanged(); }
                    catch { push("We couldn't approve it.", "error"); }
                  }}>Approve</Button>
                ) : null}
                {message.status === "approved" ? (
                  <Button size="sm" loading={busy === "send:" + message.id} onClick={() => onSend(message)}>Send</Button>
                ) : null}
                {message.status === "sent" ? (
                  <>
                    <Button size="sm" variant="quiet" onClick={async () => {
                      const text = window.prompt("Paste the reply");
                      if (!text) return;
                      try { await api.outreachMessageStatus(message.id, "replied", text); push("Reply logged."); onChanged(); }
                      catch { push("We couldn't log that reply.", "error"); }
                    }}>Log reply</Button>
                    <Button size="sm" variant="quiet" onClick={async () => {
                      if (!window.confirm("Add this address to the opt-out list?")) return;
                      try { await api.outreachMessageStatus(message.id, "optout"); push("Opted out."); onChanged(); }
                      catch { push("We couldn't record the opt-out.", "error"); }
                    }}>Opt out</Button>
                  </>
                ) : null}
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function Replies({ data }: { data: OutreachPayload }) {
  const replies = (data.messages ?? []).filter((message) => message.status === "replied");
  return (
    <section className="card">
      <h2 className="section-title">Replies</h2>
      <p className="small muted">Log replies from the Outbox. Opt-outs are suppressed immediately.</p>
      {replies.length === 0 ? <p className="small muted mt-16">No replies logged.</p> : (
        <div className="stack mt-16">
          {replies.map((message) => (
            <article className="outreach-message" key={message.id}>
              <strong>{message.subject}</strong>
              <div className="tiny muted">{message.to}</div>
              <p className="small">{message.reply_text}</p>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function OutreachSettingsPanel({ data, onChanged }: { data: OutreachPayload; onChanged: () => void }) {
  const { push } = useToast();
  const [settings, setSettings] = useState<OutreachSettings>(data.settings);
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      await api.saveOutreachSettings(settings);
      push("Outreach settings saved.");
      onChanged();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save the settings.", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="stack">
      <section className="card">
        <h2 className="section-title mb-8">Sending</h2>
        <p className="small muted mb-16">Server SMTP: {data.from_address || "not configured"}. Every send is capped by your daily limit.</p>
        <form onSubmit={save}>
          <div className="form-grid">
            <Field label="From name" htmlFor="outreach-from-name">
              <Input id="outreach-from-name" value={settings.from_name} onChange={(event) => setSettings({ ...settings, from_name: event.target.value })} />
            </Field>
            <Field label="From email" htmlFor="outreach-from-email">
              <Input id="outreach-from-email" value={settings.from_email} onChange={(event) => setSettings({ ...settings, from_email: event.target.value })} />
            </Field>
            <Field label="Daily limit" htmlFor="outreach-daily">
              <Input id="outreach-daily" type="number" min="1" max="200" value={settings.daily_limit} onChange={(event) => setSettings({ ...settings, daily_limit: Number(event.target.value) || 20 })} />
            </Field>
          </div>
          <Field label="Signature" htmlFor="outreach-signature">
            <Textarea id="outreach-signature" rows={3} value={settings.signature} onChange={(event) => setSettings({ ...settings, signature: event.target.value })} />
          </Field>
          <label className="checkbox-row">
            <input type="checkbox" checked={settings.autopilot} onChange={(event) => setSettings({ ...settings, autopilot: event.target.checked })} />
            <span>Autopilot may send approved drafts (still subject to the daily limit)</span>
          </label>
          <Button type="submit" loading={busy}>Save settings</Button>
        </form>
      </section>

      <section className="card">
        <h2 className="section-title mb-8">Suppression list</h2>
        <p className="small muted mb-16">Opt-outs and do-not-contact addresses are never sent to.</p>
        <div className="row" style={{ gap: 6 }}>
          <Input placeholder="email@example.com" value={email} onChange={(event) => setEmail(event.target.value)} />
          <Input placeholder="Reason" value={reason} onChange={(event) => setReason(event.target.value)} />
          <Button onClick={async () => {
            try {
              await api.addOutreachSuppression(email, reason || "manual");
              setEmail(""); setReason(""); push("Suppressed."); onChanged();
            } catch { push("We couldn't add that address.", "error"); }
          }}>Add</Button>
        </div>
        <div className="mt-16">
          {data.suppressions.map((item) => (
            <div className="outreach-suppression" key={item.id}>{item.email}<span className="tiny muted">{item.reason}</span></div>
          ))}
        </div>
      </section>

      <section className="card">
        <h2 className="section-title">How this compares</h2>
        <div className="outreach-compare">
          <div><strong>Sellbird</strong><span>Full local pipeline, approval-first, OSM discovery, email verification, replies, AGPL.</span></div>
          <div><strong>Warmbly</strong><span>Cold email + mailbox warmup, multi-step sequences, unified inbox, Go/Apache-2.0.</span></div>
          <div><strong>Mautic</strong><span>Marketing automation and campaigns, not a 1:1 outbound pipeline, PHP.</span></div>
          <div><strong>Crew</strong><span>Company-scoped prospects, signals, AI drafts, approval outbox, SMTP sending, suppression and CSV interop.</span></div>
        </div>
      </section>
    </div>
  );
}

function ProspectModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [form, setForm] = useState({ name: "", website: "", industry: "", location: "", notes: "", email: "", contact: "", role: "" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const set = (key: keyof typeof form, value: string) => setForm((current) => ({ ...current, [key]: value }));

  return (
    <Modal open={open} title="New prospect" onClose={onClose} footer={<><Button onClick={onClose} disabled={busy}>Cancel</Button><Button variant="primary" loading={busy} disabled={!form.name} onClick={async () => {
      setBusy(true);
      try { await api.addOutreachProspect(form); push("Prospect added."); onSaved(); }
      catch (err) { setError(err instanceof ApiError ? err.message : "We couldn't add that prospect."); }
      finally { setBusy(false); }
    }}>Add prospect</Button></>}>
      <div className="form-grid">
        <Field label="Company" htmlFor="op-name"><Input id="op-name" value={form.name} onChange={(e) => set("name", e.target.value)} /></Field>
        <Field label="Website" htmlFor="op-website"><Input id="op-website" value={form.website} onChange={(e) => set("website", e.target.value)} /></Field>
        <Field label="Industry" htmlFor="op-industry"><Input id="op-industry" value={form.industry} onChange={(e) => set("industry", e.target.value)} /></Field>
        <Field label="Location" htmlFor="op-location"><Input id="op-location" value={form.location} onChange={(e) => set("location", e.target.value)} /></Field>
      </div>
      <div className="form-grid">
        <Field label="Contact name" htmlFor="op-contact"><Input id="op-contact" value={form.contact} onChange={(e) => set("contact", e.target.value)} /></Field>
        <Field label="Role" htmlFor="op-role"><Input id="op-role" value={form.role} onChange={(e) => set("role", e.target.value)} /></Field>
        <Field label="Email" htmlFor="op-email"><Input id="op-email" value={form.email} onChange={(e) => set("email", e.target.value)} /></Field>
      </div>
      <Field label="Notes" htmlFor="op-notes"><Textarea id="op-notes" rows={3} value={form.notes} onChange={(e) => set("notes", e.target.value)} /></Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function DiscoverModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [query, setQuery] = useState("roofers");
  const [location, setLocation] = useState("Lagos");
  const [limit, setLimit] = useState("15");
  const [provider, setProvider] = useState("free");
  const [configured, setConfigured] = useState<Record<string, boolean>>({});
  const [values, setValues] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const providers: Record<string, { label: string; keyUrl?: string; fields: { key: string; label: string; secret?: boolean }[] }> = {
    free: { label: "Free — OpenStreetMap + web", fields: [] },
    serper: { label: "Serper (paid)", keyUrl: "https://serper.dev/", fields: [{ key: "api_key", label: "Serper API key", secret: true }] },
    tinyfish: { label: "TinyFish (free tier / paid)", keyUrl: "https://agent.tinyfish.ai/api-keys", fields: [{ key: "api_key", label: "TinyFish API key", secret: true }] },
    dataforseo: { label: "DataForSEO (paid)", keyUrl: "https://dataforseo.com/", fields: [{ key: "login", label: "API login" }, { key: "password", label: "API password", secret: true }] },
    scavio: { label: "Scavio (paid)", keyUrl: "https://dashboard.scavio.dev/", fields: [{ key: "api_key", label: "Scavio API key", secret: true }] },
  };

  useEffect(() => {
    if (!open) return;
    api.integrations()
      .then((result) => {
        const next: Record<string, boolean> = {};
        for (const item of result.integrations) next[item.kind] = item.configured;
        setConfigured(next);
      })
      .catch(() => setConfigured({}));
  }, [open]);

  const spec = providers[provider];
  const ready = provider === "free" || configured[provider];

  async function saveKey() {
    setBusy(true);
    setError(null);
    try {
      await api.saveIntegration(provider, values);
      setConfigured((current) => ({ ...current, [provider]: true }));
      setValues({});
      push(`${spec.label} key saved.`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save that key.");
    } finally {
      setBusy(false);
    }
  }

  async function discover() {
    setBusy(true);
    setError(null);
    try {
      const result = await api.discoverOutreach(query, location, Number(limit) || 15, provider);
      push(`Found ${result.found}; added ${result.prospects.length}.`);
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Discovery failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Discover businesses" onClose={onClose} footer={<><Button onClick={onClose} disabled={busy}>Cancel</Button><Button variant="primary" loading={busy} disabled={!ready} onClick={discover}>Discover</Button></>}>
      <p className="small muted mb-16">
        Free discovery uses OpenStreetMap and the built-in web search. Paid providers are optional and use their own credits.
      </p>
      <div className="form-grid">
        <Field label="Look for" htmlFor="od-query"><Input id="od-query" value={query} onChange={(e) => setQuery(e.target.value)} /></Field>
        <Field label="City or region" htmlFor="od-location"><Input id="od-location" value={location} onChange={(e) => setLocation(e.target.value)} /></Field>
        <Field label="Limit" htmlFor="od-limit"><Input id="od-limit" type="number" min="1" max="30" value={limit} onChange={(e) => setLimit(e.target.value)} /></Field>
      </div>
      <Field label="Data source" htmlFor="od-provider">
        <select id="od-provider" className="input" value={provider} onChange={(e) => { setProvider(e.target.value); setError(null); }}>
          {Object.entries(providers).map(([key, value]) => <option key={key} value={key}>{value.label}</option>)}
        </select>
      </Field>

      {provider !== "free" && !ready ? (
        <div className="setup-callout" style={{ marginBottom: 0 }}>
          <div style={{ flex: 1 }}>
            <strong>Add the {spec.label} key</strong>
            <p className="small muted" style={{ margin: "4px 0 0" }}>You control the provider account and its spend.</p>
            <div className="row" style={{ gap: 6, marginTop: 8, flexWrap: "wrap" }}>
              {spec.fields.map((field) => (
                <Input
                  key={field.key}
                  type={field.secret ? "password" : "text"}
                  placeholder={field.label}
                  value={values[field.key] ?? ""}
                  onChange={(e) => setValues((current) => ({ ...current, [field.key]: e.target.value }))}
                />
              ))}
              <Button size="sm" loading={busy} onClick={saveKey}>Save key</Button>
              {spec.keyUrl ? <a className="button button-small" href={spec.keyUrl} target="_blank" rel="noreferrer noopener">Get a key ↗</a> : null}
            </div>
          </div>
        </div>
      ) : null}

      {provider !== "free" && ready ? (
        <p className="tiny muted">
          {spec.label} connected. This search may consume provider credits.{" "}
          {spec.keyUrl ? <a href={spec.keyUrl} target="_blank" rel="noreferrer noopener">Provider dashboard ↗</a> : null}
        </p>
      ) : null}
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function CampaignModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [name, setName] = useState("");
  const [offer, setOffer] = useState("");
  const [audience, setAudience] = useState("");
  const [tone, setTone] = useState("Direct and human");
  const [stepOne, setStepOne] = useState("");
  const [stepTwo, setStepTwo] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return (
    <Modal open={open} title="New outreach campaign" onClose={onClose} footer={<><Button onClick={onClose} disabled={busy}>Cancel</Button><Button variant="primary" loading={busy} disabled={!name || !offer} onClick={async () => {
      setBusy(true);
      try {
        await api.createOutreachCampaign({
          name, offer, audience, tone, status: "active", daily_limit: 20, autopilot: false,
          steps: [
            { position: 1, delay_days: 0, subject: "Quick question about {{company}}", body: stepOne || "Lead with the observed signal and one clear question." },
            { position: 2, delay_days: 3, subject: "Re: Quick question about {{company}}", body: stepTwo || "One short follow-up with a new angle." },
          ],
        });
        push("Campaign created."); onSaved();
      } catch (err) { setError(err instanceof ApiError ? err.message : "We couldn't create that campaign."); }
      finally { setBusy(false); }
    }}>Create campaign</Button></>}>
      <div className="form-grid">
        <Field label="Campaign name" htmlFor="oc-name"><Input id="oc-name" value={name} onChange={(e) => setName(e.target.value)} /></Field>
        <Field label="Tone" htmlFor="oc-tone"><Input id="oc-tone" value={tone} onChange={(e) => setTone(e.target.value)} /></Field>
      </div>
      <Field label="Offer" htmlFor="oc-offer"><Textarea id="oc-offer" rows={3} value={offer} onChange={(e) => setOffer(e.target.value)} placeholder="What outcome do you sell?" /></Field>
      <Field label="Audience" htmlFor="oc-audience"><Textarea id="oc-audience" rows={2} value={audience} onChange={(e) => setAudience(e.target.value)} /></Field>
      <Field label="Step 1 instruction" htmlFor="oc-step1"><Textarea id="oc-step1" rows={2} value={stepOne} onChange={(e) => setStepOne(e.target.value)} /></Field>
      <Field label="Step 2 instruction" htmlFor="oc-step2"><Textarea id="oc-step2" rows={2} value={stepTwo} onChange={(e) => setStepTwo(e.target.value)} /></Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}
