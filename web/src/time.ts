const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function relativeTime(iso: string | null | undefined): string {
  if (!iso) return "never";

  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "just now";

  const seconds = Math.round((then - Date.now()) / 1000);
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["week", 60 * 60 * 24 * 7],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
    ["second", 1],
  ];

  for (const [unit, size] of units) {
    if (Math.abs(seconds) >= size || unit === "second") {
      return formatter.format(Math.round(seconds / size), unit);
    }
  }

  return "just now";
}

export function shortDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function clock(seconds: number): string {
  const safe = Math.max(0, seconds);
  return `0:${String(safe).padStart(2, "0")}`;
}
