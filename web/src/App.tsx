import { useEffect, useRef, useState } from "react";
import { api, ApiError, type User } from "./api";
import { Login } from "./pages/Login";
import { Home } from "./pages/Home";
import { Tasks } from "./pages/Tasks";
import { TaskDetail } from "./pages/TaskDetail";
import { Helpers } from "./pages/Helpers";
import { HelperDetail } from "./pages/HelperDetail";
import { ActivityPage } from "./pages/Activity";
import { Settings } from "./pages/Settings";
import { Privacy, Terms, Support, NotFound } from "./pages/Legal";
import { Button, Modal, Select, ToastProvider } from "./ui";
import { Link, navigate, usePath } from "./router";

export default function App() {
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [theme, setTheme] = useState(() => window.localStorage.getItem("crew-theme") ?? "system");

  useEffect(() => {
    let cancelled = false;

    api
      .me()
      .then((result) => !cancelled && setUser(result.user))
      .catch((error: unknown) => {
        if (cancelled) return;
        if (error instanceof ApiError && error.status >= 500) {
          // Keep the login screen usable; the next request retries.
        }
        setUser(null);
      });

    return () => {
      cancelled = true;
    };
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

  if (user === undefined) return <Splash />;
  if (user === null) return <Login onSignedIn={setUser} />;

  return (
    <ToastProvider>
      <Shell user={user} theme={theme} onTheme={setTheme} onSignedOut={() => setUser(null)} />
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
  { to: "/tasks", label: "Tasks", icon: "☑", match: (path) => path.startsWith("/tasks") },
  { to: "/helpers", label: "Helpers", icon: "✦", match: (path) => path.startsWith("/helpers") },
  { to: "/activity", label: "Activity", icon: "≡", match: (path) => path === "/activity" },
];

function Shell({
  user,
  theme,
  onTheme,
  onSignedOut,
}: {
  user: User;
  theme: string;
  onTheme: (theme: string) => void;
  onSignedOut: () => void;
}) {
  const path = usePath();
  const [menuOpen, setMenuOpen] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSidebarOpen(false);
    setMenuOpen(false);
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
          <span className="brand-name">Crew</span>
          <Link to="/settings" aria-label="Settings">
            <span className="avatar avatar-sm" aria-hidden="true">
              {initials || "?"}
            </span>
          </Link>
        </header>

        <main className="content">
          <Route path={path} user={user} theme={theme} onTheme={onTheme} />
        </main>
      </div>

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
  theme,
  onTheme,
}: {
  path: string;
  user: User;
  theme: string;
  onTheme: (theme: string) => void;
}) {
  if (path === "/") return <Home />;
  if (path === "/tasks") return <Tasks />;
  if (path === "/helpers") return <Helpers />;
  if (path === "/activity") return <ActivityPage />;
  if (path === "/settings") return <Settings user={user} theme={theme} onTheme={onTheme} />;
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
