import { useEffect, useState } from "react";
import { api, ApiError, type Agent } from "../api";
import { Button, Field, Input, Modal, Select, Textarea, useToast } from "../ui";
import { roles } from "../roles";
const emojis = ["🔎", "✍️", "💻", "🎨", "🗺️", "🛟"];

export function NewHelperModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated?: (agent: Agent) => void;
}) {
  const { push } = useToast();
  const [name, setName] = useState("");
  const [role, setRole] = useState("Researcher");
  const [emoji, setEmoji] = useState("🔎");
  const [instructions, setInstructions] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<{ agent: Agent; apiKey: string } | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName("");
    setRole("Researcher");
    setEmoji("🔎");
    setInstructions("");
    setError(null);
    setCreated(null);
    setCopied(false);
  }, [open]);

  async function submit(event: React.FormEvent) {
    event.preventDefault();

    if (name.trim().length === 0) {
      setError("Give your helper a name.");
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
      });
      setCreated({ agent: result.agent, apiKey: result.apiKey });
      onCreated?.(result.agent);
      push("Helper added.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't add that helper. Please try again.");
    } finally {
      setBusy(false);
    }
  }

  const command = created
    ? `go run ./cmd/crew-worker --server ${window.location.origin} --key ${created.apiKey}`
    : "";

  async function copyCommand() {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  if (created) {
    return (
      <Modal
        open={open}
        title={`${created.agent.name} is ready`}
        onClose={onClose}
        footer={
          <>
            <Button onClick={copyCommand}>{copied ? "Copied" : "Copy command"}</Button>
            <Button variant="primary" onClick={onClose}>
              Done
            </Button>
          </>
        }
      >
        <p className="modal-body">
          One last step: give this key to the computer that will do the work. It is shown once, so copy it now.
        </p>

        <div className="key-box mb-8" aria-label="Helper key">
          <span>{created.apiKey}</span>
        </div>
        <p className="hint mb-16">Treat this like a password. You can make a new one later.</p>

        <label className="label" htmlFor="connect-command">
          Run this where the work happens
        </label>
        <pre className="code-block mt-8" id="connect-command">
          {command}
        </pre>

        <p className="tiny muted mt-8">Keep this window open until you've copied it.</p>
      </Modal>
    );
  }

  return (
    <Modal open={open} title="Add a helper" onClose={onClose}>
      <form onSubmit={submit}>
        <p className="modal-body">A helper is an AI worker you give tasks to. Two details and you're done.</p>

        {error ? (
          <div className="form-error" role="alert">
            <span aria-hidden="true">⚠</span>
            <span>{error}</span>
          </div>
        ) : null}

        <Field label="What's their name?" htmlFor="helper-name">
          <Input
            id="helper-name"
            autoFocus
            maxLength={40}
            placeholder="e.g. Ada"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </Field>

        <Field label="What do they do?" htmlFor="helper-role">
          <Select id="helper-role" value={role} onChange={(event) => setRole(event.target.value)}>
            {roles.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </Select>
        </Field>

        <fieldset className="field" style={{ border: 0, padding: 0, margin: "0 0 16px" }}>
          <legend className="label" style={{ marginBottom: 8 }}>
            Pick an icon
          </legend>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {emojis.map((item) => (
              <button
                key={item}
                type="button"
                className="btn btn-icon"
                aria-pressed={emoji === item}
                aria-label={`Icon ${item}`}
                onClick={() => setEmoji(item)}
                style={{
                  borderColor: emoji === item ? "var(--accent)" : "var(--line)",
                  background: emoji === item ? "var(--accent-soft)" : "var(--surface)",
                  fontSize: 18,
                }}
              >
                {item}
              </button>
            ))}
          </div>
        </fieldset>

        <Field
          label="Anything they should always know?"
          hint="Optional. Like a note you'd leave for a new teammate."
          htmlFor="helper-instructions"
        >
          <Textarea
            id="helper-instructions"
            rows={3}
            placeholder="e.g. Keep answers short. Always link your sources."
            value={instructions}
            onChange={(event) => setInstructions(event.target.value)}
          />
        </Field>

        <div className="modal-actions">
          <Button type="button" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            Add helper
          </Button>
        </div>
      </form>
    </Modal>
  );
}
