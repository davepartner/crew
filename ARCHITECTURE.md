# Architecture

Crew is intentionally small: one Go binary, one SQLite file, one external auth authority.
This document explains the decisions and the seams that make them reversible.

## Shape

```
┌─────────────────────────────┐        ┌──────────────────────────────┐
│  Browser (React + Vite)     │        │  Helper (crew-worker or any  │
│  - session cookie           │        │  process you point at the    │
│  - CSRF double-submit       │        │  runner API)                 │
└──────────────┬──────────────┘        └──────────────┬───────────────┘
               │ HTTPS                                 │ Bearer helper key
               ▼                                       ▼
        ┌──────────────────────────────────────────────────┐
        │  Crew (Go)                                       │
        │  httpapi → store (SQLite) → payauth client       │
        └──────────────┬───────────────────────────────────┘
                       │ server-to-server
                       ▼
        ┌──────────────────────────────────────────────────┐
        │  PayTrackify (Laravel) — identity + email OTP    │
        └──────────────────────────────────────────────────┘
```

## Why these choices

### Go + stdlib HTTP

The workload is mostly I/O: proxy auth calls, read/write a small database, stream nothing
fancy. Go gives a static binary, predictable memory, and no dependency on Node's runtime for
the control plane. The router is `http.ServeMux` with method patterns (Go 1.22+), so there is
no routing framework to keep current.

### SQLite with WAL

A single-node control plane for one person does not need a database server. WAL mode plus
`MaxOpenConns(1)` removes lock contention and makes behaviour predictable. The trade-off is
explicit: no horizontal scaling, no concurrent writers.

**The seam:** everything SQL lives in `internal/store`. Moving to Postgres means replacing
that package, turning the `WHERE user_id = ?` scoping into Postgres row-level security, and
adding a migration runner. The HTTP layer never sees `database/sql`.

### Identity stays in PayTrackify

Crew is a second product on the same account. Duplicating users or passwords would create two
sources of truth. Instead:

- Crew stores only the PayTrackify user id, email, and display name.
- The session token is opaque, held in an `HttpOnly` cookie, and validated against
  PayTrackify (cached for 60 seconds).
- Revocation is immediate in PayTrackify and takes at most the cache TTL to propagate.

The alternative — sharing a JWT signing key and validating locally — removes the round trip
but makes revocation harder. For a product where "sign out all devices" is a promise, the
60-second cache is the better trade.

### CSRF

Cookies are `SameSite=Lax`, which blocks most cross-site POSTs, but not all browsers and not
all flows (older Safari, some in-app browsers). Crew adds a double-submit CSRF cookie for every
unsafe method. The runner API uses bearer keys, so it is exempt — a token that isn't sent
automatically cannot be forged cross-site.

### Helper keys

Each helper gets `crew_` + 24 random bytes. Only `sha256(key)` is stored. The key is shown once,
at creation or rotation. The runner API:

- resolves the key to `(agent, user)` in one indexed query;
- touches `last_seen_at`;
- refuses any task write where `task.agent_id != agent.id`.

That last check is what stops one helper from writing into another's work, even inside the same
workspace.

### The task state machine

```
todo ──claim──▶ working ──complete(needsReview)──▶ review ──approve──▶ done
                   │                                  │
                   └────────complete──────────────────┴──send back──▶ todo
```

Only four states. Every transition is a single UPDATE with the expected previous state in the
`WHERE` clause, so two workers racing for one task cannot both win.

## Security posture

| Concern | Control |
|---|---|
| Session theft | `HttpOnly`, `Secure` in production, `SameSite=Lax`, server-side revocation |
| CSRF | Double-submit token, constant-time compare, required on every unsafe method |
| Helper key leak | Hashed at rest, scoped to one agent, rotatable, revocable by deletion |
| Cross-workspace reads | `user_id` in every query; integration test proves a 404 across users |
| XSS | React escapes by default; strict CSP (`script-src 'self'`, no inline scripts) |
| Clickjacking | `X-Frame-Options: DENY` + `frame-ancestors 'none'` |
| Enumeration | Auth responses are identical for known and unknown emails |
| Brute force | 8-digit codes, 5 attempts, per-email/IP limits at the provider, plus an edge limiter |

## Deliberate limits

- **One person per workspace.** Adding teams means a `memberships` table and scoping tasks by
  company, not user. The store's `userID` parameter is the seam.
- **No server-side agent execution.** Helpers run where you point them. Running them in-process
  would re-introduce the sandbox/App Store problems the rebuild set out to avoid.
- **No plugin runtime.** Plugins that download and execute code are the single biggest reason
  the original cannot ship inside a sandbox. If Crew ever gets extensions, they should be
  declarative or bundled.
- **No websockets.** Polling on the runner side and refetch-on-action in the UI are enough for
  this size. A websocket layer can be added behind the same store events.

## Testing strategy

- `internal/httpapi/api_test.go` runs the real router against a fake PayTrackify, a temporary
  SQLite file, and a cookie jar. It covers the full loop plus the failure paths people actually
  hit (CSRF, bad helper key, cross-user access, logout revocation).
- Laravel's `OtpAuthTest` covers the invariants the passwordless skill requires: no enumeration,
  hashing, single use, expiry, attempt cap, resend cooldown, rate limits.

The two suites meet at the HTTP contract, which is the only thing either side knows about the
other.
