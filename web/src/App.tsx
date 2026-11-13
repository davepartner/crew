import { useCallback, useEffect, useRef, useState } from "react";
import { api, rememberedCompany, setCompany, type Company, type Notification, type User } from "./api";
import { Login } from "./pages/Login";
import { Onboarding } from "./pages/Onboarding";
import { Home } from "./pages/Home";
import { Tasks } from "./pages/Tasks";
import { TaskDetail } from "./pages/TaskDetail";
import { Helpers } from "./pages/Helpers";
import { HelperDetail } from "./pages/HelperDetail";
import { Marketing } from "./pages/Marketing";
import { AdviseMe } from "./pages/AdviseMe";
import { Portfolio } from "./pages/Portfolio";
import { Routines } from "./pages/Routines";
import { ActivityPage } from "./pages/Activity";
import { Settings } from "./pages/Settings";
import { Privacy, Terms, Support, NotFound } from "./pages/Legal";
import { Button, Modal, Select, ToastProvider } from "./ui";
import { Link, navigate, usePath } from "./router";

export default function App() {
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [companies, setCompanies] = useState<Company[] | null>(null);
  const [company, setCompanyState] = useState<Company | null>(null);
  const [theme, setTheme] = useState(() => window.localStorage.getItem("crew-theme") ?? "system");
  const path = usePath();

  useEffect(() => {
    let cancelled = false;

    api
      .me()
      .then((result) => !cancelled && setUser(result.user))
      .catch(() => !cancelled && setUser(null));

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const onUnauthorized = () => {
      window.sessionStorage.setItem("crew:signed-out", "1");
      setUser(null);
      setCompanies(null);
      setCompanyState(null);
    };

    window.addEventListener("crew:unauthorized", onUnauthorized);
    return () => window.removeEventListener("crew:unauthorized", onUnauthorized);
  }, []);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");

    const apply = () => {
      const resolved = theme === "system" ? (media.matches ? "dark" : "light") : theme;
      document.documentElement.dataset.theme = resolved;
    };

    apply();
    media.addEventListener("change", apply);
    window.localStorage.setItem("crew-theme", theme);

    return () => media.removeEventListener("change", apply);
  }, [theme]);

  const loadCompanies = useCallback(async () => {
    const result = await api.companies();
    const saved = rememberedCompany();
    const active = result.companies.find((item) => item.id === saved) ?? result.companies[0] ?? null;

    setCompanies(result.companies);
    setCompanyState(active);
    setCompany(active?.id ?? "");
  }, []);

  useEffect(() => {
    if (!user) {
      setCompanies(null);
      return;
    }

    loadCompanies().catch(() => setCompanies([]));
  }, [user, loadCompanies]);

  function switchCompany(next: Company) {
    setCompany(next.id);
    setCompanyState(next);
    navigate("/");
  }

  function finishOnboarding(next: Company) {
    setCompany(next.id);
    loadCompanies()
      .then(() => navigate("/"))
      .catch(() => {
        setCompanies((current) => [...(current ?? []), next]);
        setCompanyState(next);
        navigate("/");
      });
  }

  if (user === undefined) return <Splash />;
  if (user === null) return <Login onSignedIn={setUser} />;
  if (companies === null) return <Splash />;

  if (path === "/advise" && companies.length === 0) {
    return (
      <ToastProvider>
        <AdviseMe />
      </ToastProvider>
    );
  }

  if (path === "/new-company" || companies.length === 0) {
    return (
      <ToastProvider>
        <Onboarding
          onDone={finishOnboarding}
          onCancel={companies.length > 0 ? () => navigate("/") : undefined}
        />
      </ToastProvider>
    );
  }

  if (!company) return <Splash />;

  return (
    <ToastProvider>
      <Shell
        user={user}
        company={company}
        companies={companies}
        theme={theme}
        onTheme={setTheme}
        onUser={setUser}
        onCompany={setCompanyState}
        onSwitchCompany={switchCompany}
        onSignedOut={() => {
          setUser(null);
          setCompanies(null);
          setCompanyState(null);
        }}
      />
    </ToastProvider>
  );
}

function Splash() {
  return (
    <div style={{ minHeight: "100dvh", display: "grid", placeItems: "center" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <span className="brand-mark" aria-hidden="true">
          C
        </span>
        <span className="brand-name">Crew</span>
        <span className="spinner" aria-label="Loading" />
      </div>
    </div>
  );
}

type NavItem = { to: string; label: string; icon: string; match: (path: string) => boolean };

const navItems: NavItem[] = [
  { to: "/", label: "Home", icon: "⌂", match: (path) => path === "/" },
  { to: "/advise", label: "Advise Me", icon: "◎", match: (path) => path === "/advise" },
  { to: "/portfolio", label: "Portfolio", icon: "▦", match: (path) => path === "/portfolio" },
  { to: "/tasks", label: "Tasks", icon: "☑", match: (path) => path.startsWith("/tasks") },
  { to: "/helpers", label: "Helpers", icon: "✦", match: (path) => path.startsWith("/helpers") },
  { to: "/marketing", label: "Marketing", icon: "◈", match: (path) => path === "/marketing" },
  { to: "/activity", label: "Activity", icon: "≡", match: (path) => path === "/activity" },
];

function Shell({
  user,
  company,
  companies,
  theme,
  onTheme,
  onUser,
  onCompany,
  onSwitchCompany,
  onSignedOut,
}: {
  user: User;
  company: Company;
  companies: Company[];
  theme: string;
  onTheme: (theme: string) => void;
  onUser: (user: User) => void;
  onCompany: (company: Company) => void;
  onSwitchCompany: (company: Company) => void;
  onSignedOut: () => void;
}) {
  const path = usePath();
  const [menuOpen, setMenuOpen] = useState(false);
  const [companyOpen, setCompanyOpen] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unread, setUnread] = useState(0);
  const [notifOpen, setNotifOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const companyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSidebarOpen(false);
    setMenuOpen(false);
    setCompanyOpen(false);
  }, [path]);

  useEffect(() => {
    if (!menuOpen) return;
    const onClick = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) setMenuOpen(false);
    };
    const onKey = (event: KeyboardEvent) => event.key === "Escape" && setMenuOpen(false);
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  useEffect(() => {
    let live = true;
    const load = () => {
      api
        .notifications()
        .then((result) => {
          if (!live) return;
          setNotifications(result.notifications);
          setUnread(result.unread);
        })
        .catch(() => {});
    };

    load();
    const timer = window.setInterval(load, 30000);
    return () => {
      live = false;
      window.clearInterval(timer);
    };
  }, [path, company.id]);

  useEffect(() => {
    if (!companyOpen) return;
    const onClick = (event: MouseEvent) => {
      if (companyRef.current && !companyRef.current.contains(event.target as Node)) setCompanyOpen(false);
    };
    const onKey = (event: KeyboardEvent) => event.key === "Escape" && setCompanyOpen(false);
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [companyOpen]);

  const initials = user.name
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <div className="shell">
      {sidebarOpen ? <div className="scrim" onClick={() => setSidebarOpen(false)} /> : null}

      <aside className={`sidebar ${sidebarOpen ? "open" : ""}`}>
        <Link to="/" className="brand">
          <span className="brand-mark" aria-hidden="true">
            C
          </span>
          <span className="brand-name">Crew</span>
        </Link>

        <div ref={companyRef} style={{ position: "relative", marginBottom: 10 }}>
          <button
            className="company-button"
            onClick={() => setCompanyOpen((open) => !open)}
            aria-haspopup="menu"
            aria-expanded={companyOpen}
          >
            <span className="company-avatar" aria-hidden="true">
              {company.name.slice(0, 1).toUpperCase()}
            </span>
            <span className="grow" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {company.name}
            </span>
            <span aria-hidden="true">⌄</span>
          </button>

          {companyOpen ? (
            <div className="card" style={{ position: "absolute", top: "calc(100% + 6px)", left: 0, right: 0, padding: 8, zIndex: 30, boxShadow: "var(--shadow-lg)" }} role="menu">
              <p className="tiny muted" style={{ padding: "4px 10px 8px" }}>
                Your companies
              </p>
              {companies.map((item) => (
                <button
                  key={item.id}
                  className="nav-item"
                  role="menuitem"
                  onClick={() => {
                    setCompanyOpen(false);
                    if (item.id !== company.id) onSwitchCompany(item);
                  }}
                >
                  <span className="company-avatar small" aria-hidden="true">
                    {item.name.slice(0, 1).toUpperCase()}
                  </span>
                  <span className="grow" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {item.name}
                    {item.is_product ? <span className="tiny muted"> · product</span> : null}
                  </span>
                  {item.id === company.id ? <span aria-hidden="true">✓</span> : null}
                </button>
              ))}
              <hr className="divider" style={{ margin: "6px 0" }} />
              <button
                className="nav-item"
                role="menuitem"
                onClick={() => {
                  setCompanyOpen(false);
                  navigate("/new-company");
                }}
              >
                <span className="icon" aria-hidden="true">
                  +
                </span>
                New company
              </button>
            </div>
          ) : null}
        </div>

        <nav className="nav" aria-label="Main">
          {navItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className="nav-item"
              aria-current={item.match(path) ? "page" : undefined}
            >
              <span className="icon" aria-hidden="true">
                {item.icon}
              </span>
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="nav-bottom">
          <button
            className="nav-item"
            onClick={() => setNotifOpen(true)}
            aria-label={unread > 0 ? `${unread} unread notifications` : "Notifications"}
          >
            <span className="icon" aria-hidden="true">
              ◔
            </span>
            <span className="grow">Notifications</span>
            {unread > 0 ? <span className="notif-count inline">{unread > 9 ? "9+" : unread}</span> : null}
          </button>

          <Link
            to="/settings"
            className="nav-item"
            aria-current={path.startsWith("/settings") ? "page" : undefined}
          >
            <span className="icon" aria-hidden="true">
              ⚙
            </span>
            Settings
          </Link>

          <div ref={menuRef} style={{ position: "relative" }}>
            {menuOpen ? (
              <div
                className="card"
                style={{
                  position: "absolute",
                  bottom: "calc(100% + 8px)",
                  left: 0,
                  right: 0,
                  padding: 8,
                  boxShadow: "var(--shadow-lg)",
                  zIndex: 30,
                }}
                role="menu"
              >
                <div style={{ padding: "8px 10px" }}>
                  <div style={{ fontWeight: 600 }}>{user.name}</div>
                  <div className="tiny muted">{user.email}</div>
                </div>
                <hr className="divider" style={{ margin: "8px 0" }} />
                <button className="nav-item" role="menuitem" onClick={() => navigate("/settings")}>
                  <span className="icon" aria-hidden="true">
                    ⚙
                  </span>
                  Settings
                </button>
                <button
                  className="nav-item"
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false);
                    setSignOutOpen(true);
                  }}
                >
                  <span className="icon" aria-hidden="true">
                    ⏻
                  </span>
                  Sign out
                </button>
              </div>
            ) : null}

            <button
              className="nav-item"
              onClick={() => setMenuOpen((open) => !open)}
              aria-haspopup="menu"
              aria-expanded={menuOpen}
            >
              <span className="avatar avatar-sm" aria-hidden="true">
                {initials || "?"}
              </span>
              <span className="grow" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {user.name}
              </span>
              <span aria-hidden="true">⌃</span>
            </button>
          </div>
        </div>
      </aside>

      <div className="main">
        <header className="topbar">
          <button className="btn btn-quiet btn-icon" onClick={() => setSidebarOpen(true)} aria-label="Open menu">
            ☰
          </button>
          <span className="brand-name">{company.name}</span>
          {company.is_product ? <span className="badge badge-working">Product</span> : null}
          {company.role === "viewer" ? <span className="badge badge-todo">View only</span> : null}

          <button
            className="btn btn-quiet btn-icon"
            onClick={() => setNotifOpen(true)}
            aria-label={unread > 0 ? `${unread} unread notifications` : "Notifications"}
            style={{ marginLeft: "auto", position: "relative" }}
          >
            ◔
            {unread > 0 ? <span className="notif-count">{unread > 9 ? "9+" : unread}</span> : null}
          </button>

          <Link to="/settings" aria-label="Settings">
            <span className="avatar avatar-sm" aria-hidden="true">
              {initials || "?"}
            </span>
          </Link>
        </header>

        <main className="content">
          <Route
            key={company.id}
            path={path}
            user={user}
            company={company}
            theme={theme}
            onTheme={onTheme}
            onUser={onUser}
            onCompany={onCompany}
          />
        </main>
      </div>

      <NotificationsModal
        open={notifOpen}
        items={notifications}
        unread={unread}
        onClose={() => setNotifOpen(false)}
        onRead={(id) => {
          setNotifications((current) =>
            current.map((entry) => (entry.id === id ? { ...entry, read_at: new Date().toISOString() } : entry)),
          );
          setUnread((count) => Math.max(0, count - 1));
          void api.readNotification(id).catch(() => {});
        }}
        onReadAll={() => {
          setNotifications((current) =>
            current.map((entry) => ({ ...entry, read_at: entry.read_at ?? new Date().toISOString() })),
          );
          setUnread(0);
          void api.readAllNotifications().catch(() => {});
        }}
      />

      <SignOutModal
        open={signOutOpen}
        onClose={() => setSignOutOpen(false)}
        onSignedOut={() => {
          setSignOutOpen(false);
          onSignedOut();
        }}
      />
    </div>
  );
}

function Route({
  path,
  user,
  company,
  theme,
  onTheme,
  onUser,
  onCompany,
}: {
  path: string;
  user: User;
  company: Company;
  theme: string;
  onTheme: (theme: string) => void;
  onUser: (user: User) => void;
  onCompany: (company: Company) => void;
}) {
  if (path === "/") return <Home />;
  if (path === "/tasks") return <Tasks />;
  if (path === "/helpers") return <Helpers />;
  if (path === "/marketing") return <Marketing />;
  if (path === "/advise") return <AdviseMe />;
  if (path === "/portfolio") return <Portfolio />;
  if (path === "/routines") return <Routines />;
  if (path === "/activity") return <ActivityPage />;
  if (path === "/settings")
    return (
      <Settings
        user={user}
        company={company}
        theme={theme}
        onTheme={onTheme}
        onUser={onUser}
        onCompany={onCompany}
      />
    );
  if (path === "/privacy") return <Privacy />;
  if (path === "/terms") return <Terms />;
  if (path === "/support") return <Support />;

  const task = path.match(/^\/tasks\/([^/]+)$/);
  if (task) return <TaskDetail id={task[1]} />;

  const helper = path.match(/^\/helpers\/([^/]+)$/);
  if (helper) return <HelperDetail id={helper[1]} />;

  return <NotFound />;
}

const signOutReasons = [
  { value: "done_for_now", label: "I'm done for now" },
  { value: "switching_account", label: "Switching account" },
  { value: "privacy", label: "Privacy" },
  { value: "trouble_signing_in", label: "Trouble signing in" },
  { value: "prefer_not_to_say", label: "Prefer not to say" },
];

function SignOutModal({
  open,
  onClose,
  onSignedOut,
}: {
  open: boolean;
  onClose: () => void;
  onSignedOut: () => void;
}) {
  const [reason, setReason] = useState("done_for_now");
  const [busy, setBusy] = useState(false);

  async function signOut() {
    setBusy(true);
    try {
      await api.logout(reason);
    } catch {
      // Even if the network fails, clear the screen: the cookie is cleared
      // server-side on the next successful call.
    } finally {
      setBusy(false);
      onSignedOut();
    }
  }

  return (
    <Modal
      open={open}
      title="Sign out of Crew?"
      onClose={onClose}
      footer={
        <>
          <Button data-autofocus onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button variant="primary" onClick={signOut} loading={busy}>
            Sign out
          </Button>
        </>
      }
    >
      <p className="modal-body">You can sign back in with a code at any time. Nothing is deleted.</p>
      <label className="label" htmlFor="signout-reason">
        Why are you signing out? (optional)
      </label>
      <Select
        id="signout-reason"
        value={reason}
        onChange={(event) => setReason(event.target.value)}
        style={{ marginTop: 6 }}
      >
        {signOutReasons.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </Select>
    </Modal>
  );
}


function NotificationsModal({
  open,
  items,
  unread,
  onClose,
  onRead,
  onReadAll,
}: {
  open: boolean;
  items: Notification[];
  unread: number;
  onClose: () => void;
  onRead: (id: string) => void;
  onReadAll: () => void;
}) {
  return (
    <Modal
      open={open}
      title="Notifications"
      onClose={onClose}
      footer={
        <>
          {unread > 0 ? <Button onClick={onReadAll}>Mark all read</Button> : null}
          <Button variant="primary" onClick={onClose}>
            Close
          </Button>
        </>
      }
    >
      {items.length === 0 ? (
        <p className="modal-body">Nothing yet. Reviews and marketing teardowns show up here.</p>
      ) : (
        <div className="list">
          {items.map((item) => (
            <button
              key={item.id}
              className={`notif-item ${item.read_at ? "" : "unread"}`}
              onClick={() => {
                onRead(item.id);
                onClose();
                if (item.task_id) navigate(`/tasks/${item.task_id}`);
              }}
            >
              <span className="notif-title">{item.title}</span>
              {item.body ? <span className="notif-body">{item.body}</span> : null}
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
}
