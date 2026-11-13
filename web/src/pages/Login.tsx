import { useEffect, useMemo, useRef, useState } from "react";
import { api, ApiError, type User } from "../api";
import { Button, Field, Input } from "../ui";
import { clock } from "../time";

type Step = "email" | "code" | "name";

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function Login({ onSignedIn }: { onSignedIn: (user: User) => void }) {
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [requestId, setRequestId] = useState("");
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [name, setName] = useState("");
  const [signedInUser, setSignedInUser] = useState<User | null>(null);

  const codeRef = useRef<HTMLInputElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);

  const emailIsValid = emailPattern.test(email.trim());

  useEffect(() => {
    if (secondsLeft <= 0) return;
    const timer = window.setInterval(() => setSecondsLeft((value) => Math.max(0, value - 1)), 1000);
    return () => window.clearInterval(timer);
  }, [secondsLeft]);

  useEffect(() => {
    if (step === "code") codeRef.current?.focus();
    if (step === "name") nameRef.current?.focus();
  }, [step]);

  const canSubmitEmail = emailIsValid && !sending;
  const canSubmitCode = code.length === 8 && !verifying;

  const hint = useMemo(() => {
    if (step !== "email") return null;
    if (!email) return "We'll email you a code. No password to remember.";
    if (!emailIsValid) return "That doesn't look like an email yet.";
    return null;
  }, [email, emailIsValid, step]);

  async function requestCode(event: React.FormEvent) {
    event.preventDefault();
    if (!canSubmitEmail) return;

    setSending(true);
    setError(null);

    try {
      const result = await api.requestCode(email.trim().toLowerCase());
      setRequestId(result.requestId);
      setSecondsLeft(result.resendAfter ?? 30);
      setCode("");
      setStep("code");
    } catch (err) {
      setError(messageOf(err));
    } finally {
      setSending(false);
    }
  }

  async function verifyCode(event: React.FormEvent) {
    event.preventDefault();
    if (!canSubmitCode) return;

    setVerifying(true);
    setError(null);

    try {
      const result = await api.verifyCode(requestId, code);
      if (result.isNew) {
        setSignedInUser(result.user);
        setName(result.user.name);
        setStep("name");
      } else {
        onSignedIn(result.user);
      }
    } catch (err) {
      setError(messageOf(err));
      setCode("");
      codeRef.current?.focus();
    } finally {
      setVerifying(false);
    }
  }

  async function resend() {
    if (secondsLeft > 0 || !requestId) return;

    setError(null);
    setSending(true);

    try {
      const result = await api.resendCode(requestId);
      setSecondsLeft(result.resendAfter ?? 30);
      setCode("");
      codeRef.current?.focus();
    } catch (err) {
      const apiError = err as ApiError;
      if (apiError.retryAfter) setSecondsLeft(apiError.retryAfter);
      setError(messageOf(err));
    } finally {
      setSending(false);
    }
  }

  async function saveName(event: React.FormEvent) {
    event.preventDefault();
    if (!signedInUser) return;

    const clean = name.trim();
    setSending(true);

    try {
      if (clean.length > 0 && clean !== signedInUser.name) {
        const result = await api.updateMe(clean);
        onSignedIn(result.user);
      } else {
        onSignedIn(signedInUser);
      }
    } catch {
      // The name is optional; never block a signed-in person on it.
      onSignedIn(signedInUser);
    } finally {
      setSending(false);
    }
  }

  function startOver() {
    setStep("email");
    setCode("");
    setRequestId("");
    setError(null);
    setSecondsLeft(0);
  }

  return (
    <div className="auth">
      <aside className="auth-aside">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            C
          </span>
          <span className="brand-name">Crew</span>
        </div>

        <div>
          <h1 style={{ fontSize: 34, lineHeight: 1.15, letterSpacing: "-0.035em", fontWeight: 650 }}>
            Give your AI helpers work.
            <br />
            See it get done.
          </h1>
          <p className="muted mt-16" style={{ maxWidth: 380 }}>
            Crew is the calm place to hand off tasks, check the result, and keep your day moving. No jargon, no
            setup maze.
          </p>

          <div className="points">
            <Point icon="✉️" title="No passwords" text="A one-time code by email. That's it." />
            <Point icon="🧭" title="One clear next step" text="Every screen tells you what to do next." />
            <Point icon="✅" title="You stay in charge" text="Nothing ships until you say it's good." />
          </div>
        </div>

        <p className="tiny muted">Made for people, not for engineers.</p>
      </aside>

      <main className="auth-main">
        <div className="auth-card">
          <div className="brand mobile-only mb-16">
            <span className="brand-mark" aria-hidden="true">
              C
            </span>
            <span className="brand-name">Crew</span>
          </div>

          <div className="steps" aria-hidden="true">
            <span className="step on" />
            <span className={`step ${step === "code" || step === "name" ? "on" : ""}`} />
          </div>

          {step === "email" ? (
            <form onSubmit={requestCode} noValidate>
              <h1 className="auth-title">Sign in or create your account</h1>
              <p className="auth-sub">Enter your email. We'll send you a code — no password needed.</p>

              {error ? (
                <div className="form-error" role="alert">
                  <span aria-hidden="true">⚠</span>
                  <span>{error}</span>
                </div>
              ) : null}

              <Field label="Email address" hint={hint ?? undefined} htmlFor="email">
                <Input
                  id="email"
                  name="email"
                  type="email"
                  inputMode="email"
                  autoComplete="email"
                  autoFocus
                  placeholder="you@example.com"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  aria-invalid={email.length > 0 && !emailIsValid}
                />
              </Field>

              <Button type="submit" variant="primary" block loading={sending} disabled={!canSubmitEmail}>
                Continue
              </Button>

              <p className="auth-legal">
                New here? An account is created automatically the first time you sign in. By continuing you agree to
                our <a href="/terms">Terms</a> and <a href="/privacy">Privacy Policy</a>.
              </p>
            </form>
          ) : null}

          {step === "code" ? (
            <form onSubmit={verifyCode} noValidate>
              <h1 className="auth-title">Check your email</h1>
              <p className="auth-sub">
                We sent an 8-digit code to <strong>{email}</strong>.
              </p>

              <div
                className="card card-tight mb-16"
                style={{ background: "var(--accent-soft)", borderColor: "transparent" }}
                role="note"
              >
                <span className="small">
                  Can't see it? Check your <strong>spam or junk</strong> folder. It can take up to a minute.
                </span>
              </div>

              {error ? (
                <div className="form-error" role="alert" aria-live="assertive">
                  <span aria-hidden="true">⚠</span>
                  <span>{error}</span>
                </div>
              ) : null}

              <Field label="Your 8-digit code" htmlFor="code">
                <Input
                  id="code"
                  ref={codeRef}
                  className="code-input"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  pattern="[0-9]*"
                  maxLength={8}
                  placeholder="00000000"
                  value={code}
                  onChange={(event) => setCode(event.target.value.replace(/\D/g, "").slice(0, 8))}
                  aria-describedby="code-help"
                />
              </Field>

              <Button type="submit" variant="primary" block loading={verifying} disabled={!canSubmitCode}>
                Sign in
              </Button>

              <div className="row-between mt-16">
                <Button type="button" variant="quiet" size="sm" onClick={startOver}>
                  Use a different email
                </Button>
                <Button
                  type="button"
                  variant="quiet"
                  size="sm"
                  onClick={resend}
                  disabled={secondsLeft > 0 || sending}
                >
                  {secondsLeft > 0 ? `Send again in ${clock(secondsLeft)}` : "Send a new code"}
                </Button>
              </div>

              <p className="auth-legal" id="code-help">
                Trouble signing in? <a href="/support">Get help</a>.
              </p>
            </form>
          ) : null}

          {step === "name" ? (
            <form onSubmit={saveName} noValidate>
              <h1 className="auth-title">What should we call you?</h1>
              <p className="auth-sub">This is the name your helpers and teammates will see.</p>

              <Field label="Your name" htmlFor="name">
                <Input
                  id="name"
                  ref={nameRef}
                  autoComplete="name"
                  maxLength={60}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="Jane"
                />
              </Field>

              <Button type="submit" variant="primary" block loading={sending}>
                Start using Crew
              </Button>

              <div className="mt-16" style={{ textAlign: "center" }}>
                <Button
                  type="button"
                  variant="quiet"
                  size="sm"
                  onClick={() => signedInUser && onSignedIn(signedInUser)}
                >
                  Skip for now
                </Button>
              </div>
            </form>
          ) : null}

          <p className="auth-legal">
            <a href="/privacy">Privacy</a> · <a href="/terms">Terms</a> · <a href="/support">Support</a>
          </p>
        </div>
      </main>
    </div>
  );
}

function Point({ icon, title, text }: { icon: string; title: string; text: string }) {
  return (
    <div className="point">
      <span className="point-icon" aria-hidden="true">
        {icon}
      </span>
      <div>
        <div className="point-title">{title}</div>
        <div className="point-text">{text}</div>
      </div>
    </div>
  );
}

function messageOf(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return "Something went wrong. Please try again.";
}
