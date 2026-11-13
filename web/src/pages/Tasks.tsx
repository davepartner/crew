import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Task, type TaskStatus } from "../api";
import { NewTaskModal } from "../components/NewTaskModal";
import { TaskRow } from "../components/rows";
import { Button, EmptyState, Skeleton } from "../ui";

const filters: { id: "" | TaskStatus; label: string }[] = [
  { id: "", label: "All" },
  { id: "todo", label: "To do" },
  { id: "working", label: "In progress" },
  { id: "review", label: "Needs your OK" },
  { id: "done", label: "Done" },
];

export function Tasks() {
  const [filter, setFilter] = useState<string>(readFilter());
  const [tasks, setTasks] = useState<Task[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [newTaskOpen, setNewTaskOpen] = useState(false);

  const load = useCallback(() => {
    setTasks(null);
    api
      .tasks(filter || undefined)
      .then((result) => {
        setTasks(result.tasks);
        setError(null);
      })
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "We couldn't load your tasks.");
      });
  }, [filter]);

  useEffect(load, [load]);

  useEffect(() => {
    const sync = () => setFilter(readFilter());
    window.addEventListener("popstate", sync);
    return () => window.removeEventListener("popstate", sync);
  }, []);

  function choose(next: string) {
    setFilter(next);
    const url = next ? `/tasks?status=${next}` : "/tasks";
    window.history.replaceState({}, "", url);
  }

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Tasks</h1>
          <p className="page-sub">Everything you've asked for, and where it stands.</p>
        </div>
        <Button variant="primary" onClick={() => setNewTaskOpen(true)}>
          Add a task
        </Button>
      </header>

      <div className="tabs" role="group" aria-label="Filter tasks">
        {filters.map((item) => (
          <button
            key={item.id || "all"}
            className="tab"
            aria-pressed={filter === item.id}
            onClick={() => choose(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>

      {error ? (
        <div className="empty">
          <div className="empty-title">We couldn't load your tasks</div>
          <p className="small">{error}</p>
          <Button className="mt-16" onClick={load}>
            Try again
          </Button>
        </div>
      ) : tasks === null ? (
        <div className="card">
          <Skeleton lines={5} />
        </div>
      ) : tasks.length === 0 ? (
        <EmptyState
          title={filter ? "Nothing here right now" : "No tasks yet"}
          action={
            <Button variant="primary" size="sm" onClick={() => setNewTaskOpen(true)}>
              Add a task
            </Button>
          }
        >
          {filter
            ? "Try another filter, or add a new task."
            : "Add your first task and choose who should do it."}
        </EmptyState>
      ) : (
        <div className="list">
          {tasks.map((task) => (
            <TaskRow key={task.id} task={task} />
          ))}
        </div>
      )}

      <NewTaskModal
        open={newTaskOpen}
        onClose={() => setNewTaskOpen(false)}
        onCreated={(task) => {
          load();
          window.setTimeout(() => {
            window.history.pushState({}, "", `/tasks/${task.id}`);
            window.dispatchEvent(new PopStateEvent("popstate"));
          }, 250);
        }}
      />
    </>
  );
}

function readFilter(): string {
  const params = new URLSearchParams(window.location.search);
  const status = params.get("status") ?? "";
  return filters.some((item) => item.id === status) ? status : "";
}
