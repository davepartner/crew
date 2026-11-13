import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Provider } from "../api";
import { Button, Input, useToast } from "../ui";

export function ModelsPanel({ onConfigured }: { onConfigured?: (provider: Provider) => void }) {
  const { push } = useToast();
  const [providers, setProviders] = useState<Provider[] | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [keyValue, setKeyValue] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState<{ provider: string; text: string; ok: boolean } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    api
      .providers()
      .then((result) => setProviders(result.providers))
      .catch(() => setProviders([]));
  }, []);

  useEffect(load, [load]);

  async function save(provider: Provider, value?: string) {
    const key = (value ?? keyValue).trim();
    if (key.length < 8) {
      setError("Paste the full key from " + provider.name + ".");
      return;
    }

    setBusy(provider.key);
    setError(null);

    try {
      await api.setProviderKey(provider.key, key);
      setEditing(null);
      setKeyValue("");
      setMessage({ provider: provider.key, text: "Key saved and encrypted.", ok: true });
      push(provider.name + " connected.");
      load();
      onConfigured?.({ ...provider, configured: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save that key.");
    } finally {
      setBusy(null);
    }
  }

  async function test(provider: Provider) {
    setBusy(provider.key);
    setError(null);

    try {
      const result = await api.testProvider(provider.key);
      setMessage({ provider: provider.key, text: result.message, ok: result.ok });
    } catch (err) {
      setMessage({
        provider: provider.key,
        text: err instanceof ApiError ? err.message : "We couldn't test that key.",
        ok: false,
      });
    } finally {
      setBusy(null);
    }
  }

  async function remove(provider: Provider) {
    if (!window.confirm("Remove the " + provider.name + " key? Helpers using it will stop until you add it again.")) {
      return;
    }

    setBusy(provider.key);
    try {
      await api.deleteProviderKey(provider.key);
      setMessage(null);
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't remove that key.");
    } finally {
      setBusy(null);
    }
  }

  if (providers === null) {
    return <div className="skeleton" style={{ height: 80 }} aria-hidden="true" />;
  }

  const sorted = [...providers].sort((a, b) => Number(b.free) - Number(a.free));

  return (
    <div className="stack">
      <p className="small muted">
        Keys are encrypted on the server and never shown again. Helpers read them through their own key, not the
        browser. You can add one key per provider and use it across all your companies.
      </p>

      {error ? (
        <div className="form-error" role="alert">
          <span aria-hidden="true">⚠</span>
          <span>{error}</span>
        </div>
      ) : null}

      <div className="grid-2">
        {sorted.map((provider) => (
          <div className="card" key={provider.key}>
            <div className="row-between" style={{ alignItems: "flex-start" }}>
              <div style={{ minWidth: 0 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                  <span style={{ fontWeight: 650 }}>{provider.name}</span>
                  {provider.platform ? <span className="badge badge-done">Included</span> : null}
                  {provider.free ? <span className="badge badge-done">Free</span> : null}
                  {provider.configured && !provider.platform ? (
                    <span className="badge badge-working">Connected</span>
                  ) : null}
                </div>
                <p className="tiny muted mt-8">{provider.free_note}</p>
              </div>
              <a className="small" href={provider.keys_url} target="_blank" rel="noreferrer noopener">
                Get a key ↗
              </a>
            </div>

            <div className="mt-8" style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
              {provider.models.slice(0, 3).map((model) => (
                <span key={model} className="chip">
                  {model}
                </span>
              ))}
            </div>

            {provider.platform ? (
              <p className="tiny muted mt-16">
                Included with Crew. No key needed, and usage is tracked against your AI budget.
              </p>
            ) : null}

            {provider.configured && !provider.platform && editing !== provider.key ? (
              <div className="row-between mt-16" style={{ gap: 8 }}>
                <span className="tiny muted">Key ending {provider.last4}</span>
                <div style={{ display: "flex", gap: 6 }}>
                  <Button size="sm" onClick={() => test(provider)} loading={busy === provider.key}>
                    Test
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => {
                      setEditing(provider.key);
                      setKeyValue("");
                      setError(null);
                    }}
                  >
                    Replace
                  </Button>
                  <Button size="sm" variant="quiet" onClick={() => remove(provider)} disabled={busy === provider.key}>
                    Remove
                  </Button>
                </div>
              </div>
            ) : null}

            {editing === provider.key ? (
              <div className="mt-16">
                <label className="label" htmlFor={`key-${provider.key}`}>
                  Paste your {provider.name} key
                </label>
                <div style={{ display: "flex", gap: 8, marginTop: 6 }}>
                  <Input
                    id={`key-${provider.key}`}
                    type="password"
                    autoComplete="off"
                    autoFocus
                    placeholder={provider.key === "ollama" ? "not needed for local models" : "sk-…"}
                    value={keyValue}
                    onChange={(event) => setKeyValue(event.target.value)}
                  />
                  <Button variant="primary" onClick={() => save(provider)} loading={busy === provider.key}>
                    Save
                  </Button>
                  <Button
                    onClick={() => {
                      setEditing(null);
                      setKeyValue("");
                      setError(null);
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : null}

            {!provider.configured && !provider.platform && editing !== provider.key ? (
              <div className="mt-16">
                <Button
                  size="sm"
                  onClick={() => {
                    if (provider.key === "ollama") {
                      void save(provider, "ollama-local");
                      return;
                    }
                    setEditing(provider.key);
                    setKeyValue("");
                    setError(null);
                  }}
                  loading={busy === provider.key}
                >
                  {provider.key === "ollama" ? "Use local models" : "Add key"}
                </Button>
              </div>
            ) : null}

            {message?.provider === provider.key ? (
              <p className="tiny mt-8" style={{ color: message.ok ? "var(--ok)" : "var(--danger)" }}>
                {message.text}
              </p>
            ) : null}
          </div>
        ))}
      </div>

      <p className="tiny muted">
        Your business details and task text are sent to the provider you connect. Read our{" "}
        <a href="/privacy">Privacy Policy</a> for what is stored.
      </p>
    </div>
  );
}
