# PayTrackify app monitor

PayTrackify hosts two pages for Crew.

## Admin

`/admin/app-monitor` (admin only)

- users, companies, helpers and tasks
- 7-day active users, model calls, tokens and estimated AI spend
- 30-day charts for signups, tasks, completions and spend
- recent signups and top companies
- app announcements: create, publish/unpublish, delete

The page reads Crew through `GET /internal/stats` and `/internal/user`, protected
by `CREW_ADMIN_TOKEN`. Responses are cached for 30 seconds. If Crew is down the
page still renders with an "unavailable" notice.

## Signed-in users

`/app` (auth)

- their own Crew usage: companies, helpers, tasks, spend (matched by email)
- app-wide totals and charts
- Windows and macOS downloads
- iPhone/iPad and Android install steps
- published announcements

`GET /api/app/announcements` (token-protected) is how Crew pulls the same
announcements into its Home banner.

## Environment

PayTrackify `.env`:

```dotenv
CREW_STATS_URL=http://127.0.0.1:8790
CREW_ADMIN_TOKEN=<same value as Crew>
CREW_APP_URL=https://crew.paytrackify.com
```

Crew `.env`:

```dotenv
CREW_ADMIN_TOKEN=<same value as PayTrackify>
```

The token is also the key for the announcement feed. Rotate it in both places
and restart both services.

## Downloads

Files live in `public/downloads/crew/`:

- `crew-windows.zip` — `crew.exe`, `Start Crew.bat`, README
- `crew-macos.zip` — Apple silicon and Intel binaries, `Start Crew.command`, README

The desktop packages run Crew locally and open the browser; data stays in
`crew.db` beside the binary. iOS and Android use the web app's Add to Home
Screen flow.

## Admin flag

Only `users.is_admin` can reach `/admin/app-monitor`. The owner accounts are
already flagged; grant others with:

```sql
UPDATE users SET is_admin = 1 WHERE email = 'person@example.com';
```
