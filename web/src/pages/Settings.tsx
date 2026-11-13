import { useEffect, useState } from "react";
import { api, ApiError, type Session, type User } from "../api";
import { relativeTime, shortDate } from "../time";
import { Button, Field, Input, Modal, Select, Tabs, useToast } from "../ui";

const tabs = [
  { id: "profile", label: "Your profile" },
  { id: "signin", label: "Sign-in & devices" },
  { id: "preferences", label: "Preferences" },
  { id: "privacy", label: "Privacy & data" },
  { id: "danger", label: "Danger zone" },
];

export function Settings({
  user,
  theme,
  onTheme,
}: {
  user: User;
  theme: string;
  onTheme: (theme: string) => void;
}) {
  const { push } = useToast();
  const [tab, setTab] = useState("profile");
  const [name, setName] = useState(user.name);
  const [busy, setBusy] = useState(false);
  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");

  useEffect(() => {
    if (tab !== "signin" || sessions !== null) return;
    api
      .sessions()
      .then((result) => setSessions(result.sessions))
      .catch(() => setSessions([]));
  }, [tab, sessions]);

  async function saveName(event: React.FormEvent) {
    event.preventDefault();
    if (name.trim().length === 0) return;

    setBusy(true);
    try {
      await api.updateMe(name.trim());
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
      setSessions((current) => (current ?? []).filter((item) => item.id !== session.id));
      push("That device is signed out.");
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
          <p className="page-sub">Your account, your devices, and your data.</p>
        </div>
      </header>

      <Tabs items={tabs} active={tab} onChange={setTab} />

      {tab === "profile" ? (
        <section className="card" style={{ maxWidth: 560 }}>
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
              Save changes
            </Button>
          </form>
        </section>
      ) : null}

      {tab === "signin" ? (
        <section className="card">
          <h2 className="section-title mb-8">Sign-in & devices</h2>
          <p className="small muted mb-16">
            You sign in with a code sent to your email — there is no password to change. Sign out any device you
            don't recognise.
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
      ) : null}

      {tab === "preferences" ? (
        <section className="card" style={{ maxWidth: 560 }}>
          <h2 className="section-title mb-16">Preferences</h2>

          <Field label="Appearance" hint="Crew also follows your system setting for reduced motion." htmlFor="theme">
            <Select id="theme" value={theme} onChange={(event) => onTheme(event.target.value)}>
              <option value="system">Match my device</option>
              <option value="light">Light</option>
              <option value="dark">Dark</option>
            </Select>
          </Field>

          <p className="tiny muted">
            That's it for now. We add settings only when they earn their place.
          </p>
        </section>
      ) : null}

      {tab === "privacy" ? (
        <section className="card" style={{ maxWidth: 620 }}>
          <h2 className="section-title mb-8">Privacy & data</h2>
          <p className="small muted mb-16">
            Crew stores your tasks, helpers, and activity. Your sign-in details live with PayTrackify. You can take
            your data with you at any time.
          </p>

          <div className="row-between mb-16">
            <div>
              <div style={{ fontWeight: 600 }}>Download my data</div>
              <div className="tiny muted">A JSON file with your tasks, helpers, and activity.</div>
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
        <section className="card danger-zone" style={{ maxWidth: 620 }}>
          <h2 className="section-title mb-8" style={{ color: "var(--danger)" }}>
            Delete my workspace
          </h2>
          <p className="small muted mb-16">
            This erases your tasks, helpers, notes, and activity in Crew. Your PayTrackify account and its data stay
            as they are. This can't be undone.
          </p>

          <Button variant="danger" onClick={() => setDeleteOpen(true)}>
            Delete my workspace
          </Button>
        </section>
      ) : null}

      <Modal
        open={deleteOpen}
        title="Delete your Crew workspace?"
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
          Type <strong>delete</strong> to confirm. Your tasks, helpers, and notes will be removed immediately.
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
