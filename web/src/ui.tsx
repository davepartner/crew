import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type ComponentPropsWithRef,
  type ReactNode,
} from "react";

/* ---------------- buttons ---------------- */

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "default" | "quiet" | "danger";
  size?: "md" | "sm";
  loading?: boolean;
  block?: boolean;
};

export function Button({
  variant = "default",
  size = "md",
  loading = false,
  block = false,
  className = "",
  children,
  disabled,
  ...rest
}: ButtonProps) {
  const classes = [
    "btn",
    variant === "primary" ? "btn-primary" : "",
    variant === "quiet" ? "btn-quiet" : "",
    variant === "danger" ? "btn-danger" : "",
    size === "sm" ? "btn-sm" : "",
    block ? "btn-block" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button className={classes} disabled={disabled || loading} {...rest}>
      {loading ? <span className="spinner" aria-hidden="true" /> : null}
      {children}
    </button>
  );
}

/* ---------------- fields ---------------- */

export function Field({
  label,
  hint,
  error,
  htmlFor,
  children,
}: {
  label: string;
  hint?: string;
  error?: string | null;
  htmlFor?: string;
  children: ReactNode;
}) {
  return (
    <div className="field">
      <label className="label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {hint && !error ? <span className="hint">{hint}</span> : null}
      {error ? (
        <span className="hint" style={{ color: "var(--danger)" }} role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}

export function Input(props: ComponentPropsWithRef<"input">) {
  return <input {...props} className={`input ${props.className ?? ""}`} />;
}

export function Textarea(props: ComponentPropsWithRef<"textarea">) {
  return <textarea {...props} className={`textarea ${props.className ?? ""}`} />;
}

export function Select(props: ComponentPropsWithRef<"select">) {
  return <select {...props} className={`select ${props.className ?? ""}`} />;
}

/* ---------------- display ---------------- */

export function Avatar({ emoji, name, size = "md" }: { emoji?: string; name?: string; size?: "sm" | "md" | "lg" }) {
  const label = name ? name.slice(0, 1).toUpperCase() : "?";
  return (
    <span className={`avatar ${size === "lg" ? "avatar-lg" : size === "sm" ? "avatar-sm" : ""}`} aria-hidden="true">
      {emoji ?? label}
    </span>
  );
}

const statusLabels: Record<string, string> = {
  todo: "To do",
  working: "In progress",
  review: "Needs your OK",
  done: "Done",
};

export function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`badge badge-${status}`}>
      <span className="dot" aria-hidden="true" />
      {statusLabels[status] ?? status}
    </span>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: {
  title: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="empty">
      <div className="empty-title">{title}</div>
      {children ? <div className="small">{children}</div> : null}
      {action ? <div className="mt-16">{action}</div> : null}
    </div>
  );
}

export function Skeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div className="stack" aria-hidden="true">
      {Array.from({ length: lines }).map((_, index) => (
        <div key={index} className="skeleton" style={{ width: `${88 - index * 12}%` }} />
      ))}
    </div>
  );
}

/* ---------------- modal ---------------- */

export function Modal({
  open,
  title,
  children,
  onClose,
  footer,
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose: () => void;
  footer?: ReactNode;
}) {
  const titleId = useId();
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;

    closeRef.current?.focus();

    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };

    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="overlay" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby={titleId}>
        <div className="modal-head">
          <h2 className="modal-title" id={titleId}>
            {title}
          </h2>
          <button ref={closeRef} className="btn btn-quiet btn-icon btn-sm" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>
        {children}
        {footer ? <div className="modal-actions">{footer}</div> : null}
      </div>
    </div>
  );
}

/* ---------------- toasts ---------------- */

type Toast = { id: number; message: string; kind: "info" | "error" };

const ToastContext = createContext<{ push: (message: string, kind?: "info" | "error") => void } | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const push = useCallback((message: string, kind: "info" | "error" = "info") => {
    const id = Date.now() + Math.random();
    setToasts((current) => [...current, { id, message, kind }]);
    window.setTimeout(() => {
      setToasts((current) => current.filter((toast) => toast.id !== id));
    }, 4200);
  }, []);

  const value = useMemo(() => ({ push }), [push]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toasts" aria-live="polite">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast ${toast.kind === "error" ? "toast-error" : ""}`}>
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) throw new Error("useToast must be used inside ToastProvider");
  return context;
}

/* ---------------- tabs ---------------- */

export function Tabs({
  items,
  active,
  onChange,
}: {
  items: { id: string; label: string }[];
  active: string;
  onChange: (id: string) => void;
}) {
  return (
    <div className="tabs" role="tablist">
      {items.map((item) => (
        <button
          key={item.id}
          role="tab"
          className="tab"
          aria-selected={active === item.id}
          onClick={() => onChange(item.id)}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}
