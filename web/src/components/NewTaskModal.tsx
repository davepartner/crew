import { useEffect, useState } from "react";
import { api, ApiError, type Agent, type Task } from "../api";
import { Button, Field, Input, Modal, Select, Textarea, useToast } from "../ui";

export function NewTaskModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated?: (task: Task) => void;
}) {
  const { push } = useToast();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [title, setTitle] = useState("");
  const [details, setDetails] = useState("");
  const [agentId, setAgentId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setTitle("");
    setDetails("");
    setAgentId("");
    setError(null);

    api
      .agents()
      .then((result) => setAgents(result.agents))
      .catch(() => setAgents([]));
  }, [open]);

  async function submit(event: React.FormEvent) {
    event.preventDefault();

    if (title.trim().length === 0) {
      setError("Give the task a short title.");
      return;
    }

    setBusy(true);
    setError(null);

    try {
      const result = await api.createTask({ title: title.trim(), details: details.trim(), agentId });
      push(agentId ? "Task added and handed over." : "Task saved. Assign it whenever you're ready.");
      onCreated?.(result.task);
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't save that task. Please try again.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Add a task" onClose={onClose}>
      <form onSubmit={submit}>
        <p className="modal-body">Say what needs doing. Keep it short — you can add detail below.</p>

        {error ? (
          <div className="form-error" role="alert">
            <span aria-hidden="true">⚠</span>
            <span>{error}</span>
          </div>
        ) : null}

        <Field label="What needs doing?" htmlFor="task-title">
          <Input
            id="task-title"
            autoFocus
            maxLength={200}
            placeholder="e.g. Find three competitors and their prices"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
        </Field>

        <Field label="Any details?" hint="Optional. Links, context, or what “done” looks like." htmlFor="task-details">
          <Textarea
            id="task-details"
            rows={4}
            placeholder="Add anything that will help your helper do this well."
            value={details}
            onChange={(event) => setDetails(event.target.value)}
          />
        </Field>

        <Field
          label="Who should do it?"
          hint={agents.length === 0 ? "You have no helpers yet. Save the task and add one when you're ready." : undefined}
          htmlFor="task-agent"
        >
          <Select id="task-agent" value={agentId} onChange={(event) => setAgentId(event.target.value)}>
            <option value="">I'll decide later</option>
            {agents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.emoji} {agent.name} — {agent.role}
              </option>
            ))}
          </Select>
        </Field>

        <div className="modal-actions">
          <Button type="button" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            Add task
          </Button>
        </div>
      </form>
    </Modal>
  );
}
