import { useEffect, useState } from "react";
import { api, ApiError, type Company, type Provider, type Session, type User } from "../api";
import { ModelsPanel } from "../components/ModelsPanel";
import { relativeTime, shortDate } from "../time";
import { Button, Field, Input, Modal, Select, Tabs, Textarea, useToast } from "../ui";

const tones = ["Friendly", "Professional", "Playful", "Direct", "Warm"];

const tabs = [
  { id: "company", label: "Company" },
  { id: "you", label: "You" },
  { id: "models", label: "AI models" },
  { id: "privacy", label: "Privacy & data" },
  { id: "danger", label: "Danger zone" },
];

export function Settings({
  user,
  company,
  theme,
  onTheme,
  onUser,
  onCompany,
}: {
  user: User;
  company: Company;
  theme: string;
  onTheme: (theme: string) => void;
  onUser: (user: User) => void;
  onCompany: (company: Company) => void;
}) {
  const { push } = useToast();
  const [tab, setTab] = useState("company");
  const [name, setName] = useState(user.name);
  const [busy, setBusy] = useState(false);
  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");
  const [providers, setProviders] = useState<Provider[]>([]);

  const [companyName, setCompanyName] = useState(company.name);
  const [website, setWebsite] = useState(company.website);
  const [supportEmail, setSupportEmail] = useState(company.support_email);
  const [description, setDescription] = useState(company.description);
  const [audience, setAudience] = useState(company.audience);
  const [tone, setTone] = useState(company.tone || "Friendly");

  useEffect(() => {
    setCompanyName(company.name);
    setWebsite(company.website);
    setSupportEmail(company.support_email);
    setDescription(company.description);
    setAudience(company.audience);
    setTone(company.tone || "Friendly");
  }, [company]);

  useEffect(() => {
    if (tab !== "you" || sessions !== null) return;
    api
      .sessions()
      .then((result) => setSessions(result.sessions))
      .catch(() => setSessions([]));
  }, [tab, sessions]);

  useEffect(() => {
    if (tab !== "models" || providers.length > 0) return;
    api
      .providers()
      .then((result) => setProviders(result.providers))
      .catch(() => setProviders([]));
  }, [tab, providers.length]);

  async function saveCompany(event: React.FormEvent) {
    event.preventDefault();
    if (companyName.trim().length === 0) return;

    setBusy(true);
    try {
      const result = await api.updateCompany(company.id, {
        name: companyName.trim(),
        website: website.trim(),
        supportEmail: supportEmail.trim(),
        description: description.trim(),
        audience: audience.trim(),
        tone,
      });
      onCompany(result.company);
      push("Company saved.");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function saveName(event: React.FormEvent) {
    event.preventDefault();
    if (name.trim().length === 0) return;

    setBusy(true);
    try {
      const result = await api.updateMe(name.trim());
      onUser(result.user);
      push("Name saved.");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function revoke(session: Session) {
    setBusy(true);
    try {
      await api.revokeSession(session.id);
      push("That device is signed out.");
      const result = await api.sessions();
      setSessions(result.sessions);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't sign that device out.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function download() {
    try {
      const response = await fetch("/api/me/export", { credentials: "same-origin" });
      if (!response.ok) throw new Error("export failed");
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "crew-export.json";
      link.click();
      URL.revokeObjectURL(url);
      push("Your data is downloading.");
    } catch {
      push("We couldn't prepare your download. Please try again.", "error");
    }
  }

  async function deleteCompany() {
    setBusy(true);
    try {
      await api.deleteCompany(company.id);
      window.location.assign("/");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't delete this company.", "error");
      setBusy(false);
    }
  }

  async function deleteEverything() {
    setBusy(true);
    try {
      await api.deleteMe();
      window.location.href = "/";
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't delete your workspace.", "error");
      setBusy(false);
    }
  }

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Settings</h1>
          <p className="page-sub">{company.name} · your account, your models, your data.</p>
        </div>
      </header>

      <Tabs items={tabs} active={tab} onChange={setTab} />

      {tab === "company" ? (
        <section className="card" style={{ maxWidth: 680 }}>
          <h2 className="section-title mb-8">Company details</h2>
          <p className="small muted mb-16">Your helpers read this before every task. Keep it current.</p>

          <form onSubmit={saveCompany}>
            <div className="form-grid">
              <Field label="Company name" htmlFor="settings-company-name">
                <Input
                  id="settings-company-name"
                  value={companyName}
                  maxLength={80}
                  onChange={(event) => setCompanyName(event.target.value)}
                />
              </Field>
              <Field label="Website" htmlFor="settings-company-website">
                <Input id="settings-company-website" value={website} onChange={(event) => setWebsite(event.target.value)} />
              </Field>
              <Field label="Support email" htmlFor="settings-company-support">
                <Input
                  id="settings-company-support"
                  type="email"
                  value={supportEmail}
                  onChange={(event) => setSupportEmail(event.target.value)}
                />
              </Field>
              <Field label="Who do you serve?" htmlFor="settings-company-audience">
                <Input
                  id="settings-company-audience"
                  value={audience}
                  onChange={(event) => setAudience(event.target.value)}
                />
              </Field>
            </div>

            <Field label="What does the business do?" htmlFor="settings-company-description">
              <Textarea
                id="settings-company-description"
                rows={4}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </Field>

            <Field label="Tone" htmlFor="settings-company-tone">
              <Select id="settings-company-tone" value={tone} onChange={(event) => setTone(event.target.value)}>
                {tones.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </Select>
            </Field>

            <Button type="submit" variant="primary" loading={busy}>
              Save company
            </Button>
          </form>
        </section>
      ) : null}

      {tab === "you" ? (
        <div className="stack" style={{ maxWidth: 680 }}>
          <section className="card">
            <h2 className="section-title mb-16">Your profile</h2>
            <form onSubmit={saveName}>
              <Field label="Your name" htmlFor="profile-name">
                <Input id="profile-name" value={name} maxLength={60} onChange={(event) => setName(event.target.value)} />
              </Field>
              <Field label="Email" hint="This is how you sign in. It's managed by PayTrackify." htmlFor="profile-email">
                <Input id="profile-email" value={user.email} readOnly disabled />
              </Field>
              <p className="tiny muted mb-16">With Crew since {shortDate(user.created_at)}.</p>
              <Button type="submit" variant="primary" loading={busy}>
                Save name
              </Button>
            </form>
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Appearance</h2>
            <Field label="Theme" hint="Crew also follows your device's reduced-motion setting." htmlFor="theme">
              <Select id="theme" value={theme} onChange={(event) => onTheme(event.target.value)}>
                <option value="system">Match my device</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </Select>
            </Field>
          </section>

          <section className="card">
            <h2 className="section-title mb-8">Sign-in & devices</h2>
            <p className="small muted mb-16">
              You sign in with a code sent to your email — there is no password. Sign out any device you don't
              recognise.
            </p>

            {sessions === null ? (
              <div className="skeleton" style={{ height: 48 }} />
            ) : sessions.length === 0 ? (
              <p className="small muted">No other devices are signed in.</p>
            ) : (
              <div className="list">
                {sessions.map((session) => (
                  <div className="row" key={session.id} style={{ cursor: "default" }}>
                    <span className="avatar" aria-hidden="true">
                      🖥️
                    </span>
                    <span className="row-main">
                      <span className="row-title">{session.device}</span>
                      <span className="row-meta">
                        {session.ip_address} · last used {relativeTime(session.last_used_at)}
                      </span>
                    </span>
                    <Button size="sm" variant="quiet" onClick={() => revoke(session)} disabled={busy}>
                      Sign out
                    </Button>
                  </div>
                ))}
              </div>
            )}

            <p className="tiny muted mt-16">
              This browser is one of these devices. Signing it out here also signs you out.
            </p>
          </section>
        </div>
      ) : null}

      {tab === "models" ? (
        <section style={{ maxWidth: 900 }}>
          <ModelsPanel onConfigured={() => setProviders([])} />
        </section>
      ) : null}

      {tab === "privacy" ? (
        <section className="card" style={{ maxWidth: 620 }}>
          <h2 className="section-title mb-8">Privacy & data</h2>
          <p className="small muted mb-16">
            Crew stores your companies, tasks, helpers and activity. Your sign-in details live with PayTrackify. AI
            keys are encrypted at rest.
          </p>

          <div className="row-between mb-16">
            <div>
              <div style={{ fontWeight: 600 }}>Download my data</div>
              <div className="tiny muted">A JSON file with every company, task, note and activity entry.</div>
            </div>
            <Button onClick={download}>Download</Button>
          </div>

          <hr className="divider" />

          <p className="small muted">
            Read the <a href="/privacy">Privacy Policy</a> for what we store and why.
          </p>
        </section>
      ) : null}

      {tab === "danger" ? (
        <div className="stack" style={{ maxWidth: 620 }}>
          <section className="card danger-zone">
            <h2 className="section-title mb-8" style={{ color: "var(--danger)" }}>
              Delete this company
            </h2>
            <p className="small muted mb-16">
              Removes {company.name}, its helpers, tasks and notes. Your other companies and your account stay.
            </p>
            <Button variant="danger" onClick={deleteCompany} loading={busy}>
              Delete {company.name}
            </Button>
          </section>

          <section className="card danger-zone">
            <h2 className="section-title mb-8" style={{ color: "var(--danger)" }}>
              Delete my account data
            </h2>
            <p className="small muted mb-16">
              Erases every company, task and helper in Crew. Your PayTrackify account stays as it is. This can't be
              undone.
            </p>
            <Button variant="danger" onClick={() => setDeleteOpen(true)}>
              Delete everything
            </Button>
          </section>
        </div>
      ) : null}

      <Modal
        open={deleteOpen}
        title="Delete all Crew data?"
        onClose={() => setDeleteOpen(false)}
        footer={
          <>
            <Button onClick={() => setDeleteOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={deleteEverything}
              loading={busy}
              disabled={confirmText.trim().toLowerCase() !== "delete"}
            >
              Delete everything
            </Button>
          </>
        }
      >
        <p className="modal-body">
          Type <strong>delete</strong> to confirm. Every company, task, helper and note will be removed.
        </p>
        <Field label="Type delete to confirm" htmlFor="confirm-delete">
          <Input
            id="confirm-delete"
            value={confirmText}
            onChange={(event) => setConfirmText(event.target.value)}
            autoComplete="off"
          />
        </Field>
      </Modal>
    </>
  );
}
