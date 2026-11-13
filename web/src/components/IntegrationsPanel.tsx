import { useEffect, useState } from "react";
import { api, ApiError, type Integration } from "../api";
import { Button, Field, Input, Skeleton, useToast } from "../ui";

type FieldSpec = { key: string; label: string; secret?: boolean; placeholder?: string; hint?: string };

const specs: Record<string, { title: string; blurb: string; fields: FieldSpec[]; testLabel?: string }> = {
  whatsapp: {
    title: "WhatsApp Cloud API",
    blurb: "Send approved text messages and reports to customers from your own WhatsApp Business number.",
    fields: [
      { key: "phone_number_id", label: "Phone number ID", placeholder: "1234567890" },
      { key: "token", label: "Permanent access token", secret: true },
      { key: "default_to", label: "Default recipient", placeholder: "2348012345678", hint: "Optional. Country code, no plus." },
    ],
    testLabel: "Test connection",
  },
  meta_ads: {
    title: "Meta Ads",
    blurb: "Read campaigns and spend from a Meta ad account. Read-only; Crew never changes budgets.",
    fields: [
      { key: "account_id", label: "Ad account ID", placeholder: "act_1234567890" },
      { key: "access_token", label: "Access token", secret: true },
    ],
    testLabel: "Read account",
  },
  tiktok_ads: {
    title: "TikTok Ads",
    blurb: "Read campaigns from a TikTok advertiser account. Read-only.",
    fields: [
      { key: "advertiser_id", label: "Advertiser ID", placeholder: "7000000000000000000" },
      { key: "access_token", label: "Access token", secret: true },
    ],
    testLabel: "Read account",
  },
  google_ads: {
    title: "Google Ads",
    blurb: "List the Google Ads accounts this developer token can access. Read-only.",
    fields: [
      { key: "developer_token", label: "Developer token", secret: true },
      { key: "client_id", label: "OAuth client ID" },
      { key: "client_secret", label: "OAuth client secret", secret: true },
      { key: "refresh_token", label: "Refresh token", secret: true },
      { key: "login_customer_id", label: "Login customer ID", placeholder: "123-456-7890", hint: "Optional manager account." },
    ],
    testLabel: "List accounts",
  },
  brave_search: {
    title: "Brave Search",
    blurb: "Arbitrary web search for the advisor. Without a key, Crew falls back to live industry news and Wikipedia.",
    fields: [{ key: "api_key", label: "Brave Search API key", secret: true }],
    testLabel: "Test key",
  },
};

export function IntegrationsPanel() {
  const { push } = useToast();
  const [state, setState] = useState<{ integrations: Integration[]; email: { available: boolean; from: string } } | null>(null);
  const [values, setValues] = useState<Record<string, Record<string, string>>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, string>>({});
  const [emailTo, setEmailTo] = useState("");

  function load() {
    api
      .integrations()
      .then((result) => {
        setState(result);
        const next: Record<string, Record<string, string>> = {};
        for (const item of result.integrations) {
          next[item.kind] = { ...item.details };
        }
        setValues(next);
      })
      .catch(() => setState({ integrations: [], email: { available: false, from: "" } }));
  }

  useEffect(load, []);

  async function save(kind: string) {
    setBusy(kind);
    try {
      const result = await api.saveIntegration(kind, values[kind] ?? {});
      setState((current) =>
        current
          ? {
              ...current,
              integrations: current.integrations.map((item) =>
                item.kind === kind ? result.integration : item,
              ),
            }
          : current,
      );
      push(`${specs[kind].title} saved.`);
      setResults((current) => ({ ...current, [kind]: "Saved." }));
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that integration.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function test(kind: string) {
    setBusy(kind);
    setResults((current) => ({ ...current, [kind]: "Testing…" }));
    try {
      const result = await api.testIntegration(kind);
      setResults((current) => ({ ...current, [kind]: JSON.stringify(result.detail, null, 2) }));
    } catch (err) {
      setResults((current) => ({
        ...current,
        [kind]: err instanceof ApiError ? err.message : "The test failed.",
      }));
    } finally {
      setBusy(null);
    }
  }

  async function disconnect(kind: string) {
    if (!window.confirm(`Disconnect ${specs[kind].title}?`)) return;
    setBusy(kind);
    try {
      await api.deleteIntegration(kind);
      setState((current) =>
        current
          ? {
              ...current,
              integrations: current.integrations.map((item) =>
                item.kind === kind ? { ...item, configured: false, details: {} } : item,
              ),
            }
          : current,
      );
      setValues((current) => ({ ...current, [kind]: {} }));
      setResults((current) => ({ ...current, [kind]: "Disconnected." }));
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't disconnect it.", "error");
    } finally {
      setBusy(null);
    }
  }

  if (!state) {
    return (
      <section className="card">
        <Skeleton lines={5} />
      </section>
    );
  }

  return (
    <div className="stack">
      <section className="card">
        <h2 className="section-title mb-8">Email</h2>
        <p className="small muted mb-16">
          Crew sends plain-text email from <code>{state.email.from || "the server"}</code>. Use it for reports,
          proposals and follow-ups.
        </p>
        <div className="row">
          <Input
            value={emailTo}
            placeholder="client@example.com"
            onChange={(event) => setEmailTo(event.target.value)}
            style={{ flex: 1 }}
          />
          <Button
            loading={busy === "email"}
            disabled={!emailTo.includes("@")}
            onClick={async () => {
              setBusy("email");
              try {
                await api.sendEmail({
                  to: emailTo,
                  subject: "Test from Crew",
                  body: "Crew can send email from your workspace.",
                });
                push("Test email sent.");
              } catch (err) {
                push(err instanceof ApiError ? err.message : "The email couldn't be sent.", "error");
              } finally {
                setBusy(null);
              }
            }}
          >
            Send test
          </Button>
        </div>
        <p className="tiny muted mt-8">
          A test only proves the mail server accepts the message; check the inbox as well.
        </p>
      </section>

      <div className="grid-2">
        {Object.entries(specs).map(([kind, spec]) => {
          const integration = state.integrations.find((item) => item.kind === kind);
          const configured = Boolean(integration?.configured);
          const result = results[kind];

          return (
            <section className="card" key={kind}>
              <div className="row-between" style={{ alignItems: "flex-start" }}>
                <div>
                  <h2 className="section-title" style={{ marginBottom: 4 }}>{spec.title}</h2>
                  <p className="small muted" style={{ margin: 0 }}>{spec.blurb}</p>
                </div>
                <span className={`badge ${configured ? "badge-done" : "badge-todo"}`}>
                  {configured ? "Connected" : "Not connected"}
                </span>
              </div>

              <div className="mt-16">
                {spec.fields.map((field) => (
                  <Field key={field.key} label={field.label} hint={field.hint} htmlFor={`${kind}-${field.key}`}>
                    <Input
                      id={`${kind}-${field.key}`}
                      type={field.secret ? "password" : "text"}
                      placeholder={configured && field.secret ? "•••••••• (leave blank to keep)" : field.placeholder}
                      value={values[kind]?.[field.key] ?? ""}
                      onChange={(event) =>
                        setValues((current) => ({
                          ...current,
                          [kind]: { ...(current[kind] ?? {}), [field.key]: event.target.value },
                        }))
                      }
                    />
                  </Field>
                ))}
              </div>

              <div className="row mt-8" style={{ flexWrap: "wrap" }}>
                <Button size="sm" variant="primary" loading={busy === kind} onClick={() => save(kind)}>
                  Save
                </Button>
                {configured ? (
                  <>
                    <Button size="sm" loading={busy === kind} onClick={() => test(kind)}>
                      {spec.testLabel ?? "Test"}
                    </Button>
                    <Button size="sm" variant="quiet" disabled={busy === kind} onClick={() => disconnect(kind)}>
                      Disconnect
                    </Button>
                  </>
                ) : null}
              </div>

              {result ? (
                <pre className="integration-result">{result}</pre>
              ) : null}
            </section>
          );
        })}
      </div>
    </div>
  );
}
