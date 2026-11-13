import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Agent, type Comment, type Task, type TaskStatus } from "../api";
import { Link, navigate } from "../router";
import { relativeTime } from "../time";
import { Button, Field, Input, Modal, Select, StatusBadge, Textarea, useToast } from "../ui";

const statusOptions: { value: TaskStatus; label: string }[] = [
  { value: "todo", label: "To do" },
  { value: "working", label: "In progress" },
  { value: "review", label: "Needs your OK" },
  { value: "done", label: "Done" },
];

export function TaskDetail({ id }: { id: string }) {
  const { push } = useToast();
  const [task, setTask] = useState<Task | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [draftDetails, setDraftDetails] = useState("");
  const [editingTitle, setEditingTitle] = useState(false);
  const [draftTitle, setDraftTitle] = useState("");
  const [note, setNote] = useState("");
  const [sendBackOpen, setSendBackOpen] = useState(false);
  const [sendBackNote, setSendBackNote] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api
      .task(id)
      .then((result) => {
        setTask(result.task);
        setDraftDetails(result.task.details);
        setDraftTitle(result.task.title);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load that task.");
      });

    api
      .agents()
      .then((result) => setAgents(result.agents))
      .catch(() => setAgents([]));
  }, [id]);

  useEffect(load, [load]);

  async function patch(input: Record<string, unknown>, message: string) {
    setBusy(true);
    try {
      const result = await api.updateTask(id, input);
      setTask(result.task);
      push(message);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function postNote(event: React.FormEvent) {
    event.preventDefault();
    if (note.trim().length === 0) return;

    try {
      const result = await api.comment(id, note.trim());
      setTask((current) =>
        current ? { ...current, comments: [...(current.comments ?? []), result.comment] } : current,
      );
      setNote("");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't post that note.", "error");
    }
  }

  async function approve() {
    setBusy(true);
    try {
      const result = await api.approve(id);
      setTask(result.task);
      push("Approved. Nice work.");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't approve that.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function sendBack() {
    setBusy(true);
    try {
      const result = await api.sendBack(id, sendBackNote.trim());
      setTask(result.task);
      setSendBackOpen(false);
      setSendBackNote("");
      push("Sent back with your note.");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't send that back.", "error");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    try {
      await api.deleteTask(id);
      push("Task deleted.");
      navigate("/tasks");
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't delete that.", "error");
      setBusy(false);
    }
  }

  if (error) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't find that task</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={() => navigate("/tasks")}>
          Back to tasks
        </Button>
      </div>
    );
  }

  if (!task) {
    return (
      <div className="card">
        <div className="skeleton" style={{ height: 22, width: "50%" }} />
      </div>
    );
  }

  return (
    <>
      <div className="mb-16">
        <Link to="/tasks" className="small muted">
          ← Back to tasks
        </Link>
      </div>

      <header className="page-head" style={{ marginBottom: 20 }}>
        <div style={{ minWidth: 0, flex: 1 }}>
          {editingTitle ? (
            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <Input
                aria-label="Task title"
                value={draftTitle}
                maxLength={200}
                autoFocus
                onChange={(event) => setDraftTitle(event.target.value)}
                style={{ fontSize: 18, fontWeight: 600, maxWidth: 460 }}
              />
              <Button
                size="sm"
                variant="primary"
                loading={busy}
                onClick={async () => {
                  if (draftTitle.trim().length === 0) return;
                  await patch({ title: draftTitle.trim() }, "Renamed.");
                  setEditingTitle(false);
                }}
              >
                Save
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  setEditingTitle(false);
                  setDraftTitle(task.title);
                }}
              >
                Cancel
              </Button>
            </div>
          ) : (
            <div className="row-between" style={{ justifyContent: "flex-start", gap: 10 }}>
              <h1 className="page-title" style={{ fontSize: 23 }}>
                {task.title}
              </h1>
              <Button size="sm" variant="quiet" onClick={() => setEditingTitle(true)}>
                Rename
              </Button>
            </div>
          )}
          <p className="page-sub">
            Added {relativeTime(task.created_at)}
            {task.completed_at ? ` · finished ${relativeTime(task.completed_at)}` : ""}
          </p>
        </div>
        <StatusBadge status={task.status} />
      </header>

      <div className="grid-2" style={{ alignItems: "start" }}>
        <div className="stack">
          <section className="card">
            <div className="row-between mb-8">
              <h2 className="section-title">What needs doing</h2>
              {!editing ? (
                <Button size="sm" variant="quiet" onClick={() => setEditing(true)}>
                  Edit
                </Button>
              ) : null}
            </div>

            {editing ? (
              <>
                <Textarea
                  rows={6}
                  value={draftDetails}
                  onChange={(event) => setDraftDetails(event.target.value)}
                  aria-label="Task details"
                />
                <div className="row-between mt-16">
                  <Button size="sm" onClick={() => { setEditing(false); setDraftDetails(task.details); }}>
                    Cancel
                  </Button>
                  <Button
                    size="sm"
                    variant="primary"
                    loading={busy}
                    onClick={async () => {
                      await patch({ details: draftDetails }, "Saved.");
                      setEditing(false);
                    }}
                  >
                    Save
                  </Button>
                </div>
              </>
            ) : (
              <p className="small" style={{ whiteSpace: "pre-wrap", color: task.details ? "var(--ink-soft)" : "var(--muted)" }}>
                {task.details || "No extra detail yet."}
              </p>
            )}
          </section>

          {task.status === "review" ? (
            <section className="card" style={{ background: "var(--warn-soft)", borderColor: "transparent" }}>
              <h2 className="section-title mb-8">This is ready for your OK</h2>
              <p className="small mb-16">Read the result below. Approve it, or send it back with a note.</p>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                <Button variant="primary" onClick={approve} loading={busy}>
                  Approve
                </Button>
                <Button onClick={() => setSendBackOpen(true)}>Send back</Button>
              </div>
            </section>
          ) : null}

          <section className="card">
            <h2 className="section-title mb-8">Notes</h2>

            <div className="timeline">
              {(task.comments ?? []).length === 0 ? (
                <p className="small muted">No notes yet. Anything you add here stays with the task.</p>
              ) : (
                (task.comments ?? []).map((comment) => (
                  <CommentRow key={comment.id} comment={comment} />
                ))
              )}
            </div>

            <form onSubmit={postNote} className="mt-16">
              <label className="label" htmlFor="note">
                Add a note
              </label>
              <Textarea
                id="note"
                rows={3}
                placeholder="Ask a question, give context, or say what to change."
                value={note}
                onChange={(event) => setNote(event.target.value)}
                style={{ marginTop: 6 }}
              />
              <div className="row-between mt-8">
                <span className="tiny muted">Your helper sees this.</span>
                <Button type="submit" size="sm" disabled={note.trim().length === 0}>
                  Post note
                </Button>
              </div>
            </form>
          </section>
        </div>

        <div className="stack">
          <section className="card">
            <h2 className="section-title mb-16">Details</h2>

            <Field label="Who's doing it" htmlFor="assign">
              <Select
                id="assign"
                value={task.agent_id ?? ""}
                disabled={busy}
                onChange={(event) => {
                  const value = event.target.value;
                  if (value === "") {
                    patch({ unassign: true }, "Unassigned.");
                  } else {
                    patch({ agentId: value }, "Assigned.");
                  }
                }}
              >
                <option value="">Nobody yet</option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.emoji} {agent.name}
                  </option>
                ))}
              </Select>
            </Field>

            <Field label="Where it stands" htmlFor="status">
              <Select
                id="status"
                value={task.status}
                disabled={busy}
                onChange={(event) => patch({ status: event.target.value }, "Updated.")}
              >
                {statusOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </Field>

            <hr className="divider" />

            <Button
              variant="quiet"
              size="sm"
              style={{ color: "var(--danger)" }}
              onClick={() => setDeleteOpen(true)}
            >
              Delete this task
            </Button>
          </section>
        </div>
      </div>

      <Modal
        open={sendBackOpen}
        title="Send it back"
        onClose={() => setSendBackOpen(false)}
        footer={
          <>
            <Button onClick={() => setSendBackOpen(false)} disabled={busy}>
              Cancel
            </Button>
            <Button variant="primary" onClick={sendBack} loading={busy}>
              Send back
            </Button>
          </>
        }
      >
        <p className="modal-body">Tell your helper what to change. A clear note saves a round trip.</p>
        <Field label="What should change?" htmlFor="send-back-note">
          <Textarea
            id="send-back-note"
            rows={4}
            autoFocus
            placeholder="e.g. Add prices, and use our own product as the comparison."
            value={sendBackNote}
            onChange={(event) => setSendBackNote(event.target.value)}
          />
        </Field>
      </Modal>

      <Modal
        open={deleteOpen}
        title="Delete this task?"
        onClose={() => setDeleteOpen(false)}
        footer={
          <>
            <Button onClick={() => setDeleteOpen(false)} disabled={busy}>
              Keep it
            </Button>
            <Button variant="danger" onClick={remove} loading={busy}>
              Delete
            </Button>
          </>
        }
      >
        <p className="modal-body">This removes the task and all its notes. It can't be undone.</p>
      </Modal>
    </>
  );
}

function CommentRow({ comment }: { comment: Comment }) {
  const isHelper = comment.author_type === "helper";

  return (
    <div className="timeline-item">
      <div className="timeline-rail" aria-hidden="true">
        <span className="timeline-dot" style={{ background: isHelper ? "var(--accent)" : "var(--line-strong)" }} />
      </div>
      <div className="timeline-body">
        <div className="row-between" style={{ gap: 8 }}>
          <span className="small" style={{ fontWeight: 600 }}>
            {comment.author_name}
          </span>
          <span className="timeline-time">{relativeTime(comment.created_at)}</span>
        </div>
        <div className="small" style={{ whiteSpace: "pre-wrap", color: "var(--ink-soft)" }}>
          {comment.body}
        </div>
      </div>
    </div>
  );
}
