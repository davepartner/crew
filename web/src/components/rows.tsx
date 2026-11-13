import type { ActivityItem, Task } from "../api";
import { Link } from "../router";
import { relativeTime } from "../time";
import { Avatar, StatusBadge } from "../ui";

export function TaskRow({ task }: { task: Task }) {
  return (
    <Link to={`/tasks/${task.id}`} className="row">
      <Avatar emoji={task.agent?.emoji ?? "•"} size="sm" />
      <span className="row-main">
        <span className="row-title">{task.title}</span>
        <span className="row-meta">
          {task.agent ? `${task.agent.name} · ` : "Unassigned · "}
          {relativeTime(task.updated_at)}
        </span>
      </span>
      <StatusBadge status={task.status} />
    </Link>
  );
}

export function Timeline({ items }: { items: ActivityItem[] }) {
  if (items.length === 0) {
    return <p className="small muted">Nothing yet. Your activity will show up here.</p>;
  }

  return (
    <div className="timeline">
      {items.map((item, index) => (
        <div className="timeline-item" key={item.id}>
          <div className="timeline-rail" aria-hidden="true">
            <span className="timeline-dot" />
            {index < items.length - 1 ? <span className="timeline-line" /> : null}
          </div>
          <div className="timeline-body">
            <div className="timeline-text">{item.message}</div>
            <div className="timeline-time">{relativeTime(item.created_at)}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
