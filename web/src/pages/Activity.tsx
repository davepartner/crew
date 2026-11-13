import { useEffect, useState } from "react";
import { api, ApiError, type ActivityItem } from "../api";
import { Timeline } from "../components/rows";
import { Button, EmptyState, Skeleton } from "../ui";

export function ActivityPage() {
  const [items, setItems] = useState<ActivityItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .activity()
      .then((result) => setItems(result.activity))
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "We couldn't load activity."));
  }, []);

  return (
    <>
      <header className="page-head">
        <div>
          <h1 className="page-title">Activity</h1>
          <p className="page-sub">Everything that happened, newest first.</p>
        </div>
      </header>

      {error ? (
        <EmptyState title="We couldn't load activity">{error}</EmptyState>
      ) : items === null ? (
        <div className="card">
          <Skeleton lines={6} />
        </div>
      ) : items.length === 0 ? (
        <EmptyState title="Nothing here yet">Add a task and this page will start filling up.</EmptyState>
      ) : (
        <div className="card">
          <Timeline items={items} />
        </div>
      )}

      {error ? (
        <div className="mt-16">
          <Button onClick={() => window.location.reload()}>Refresh</Button>
        </div>
      ) : null}
    </>
  );
}
