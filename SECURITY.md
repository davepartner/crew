# Security

## Reporting

Email **davepartner@users.noreply.github.com** with a description and, if possible, a reproduction.
Please do not open a public issue for a vulnerability. You will get an acknowledgement within a
few days; this is a small project, so please be patient.

## What Crew protects

- **Sign-in.** Passwords don't exist here. PayTrackify issues an 8-digit, single-use, 10-minute
  code, hashed at rest and rate-limited per email and per IP. Crew stores only an opaque session
  token, and only its SHA-256 hash exists in the identity database.
- **Sessions.** `HttpOnly`, `SameSite=Lax`, `Secure` in production, 30-day expiry, capped at ten
  live sessions, revocable per device.
- **Cross-site requests.** A double-submit CSRF token is required on every unsafe method.
  Helper traffic uses bearer keys and is exempt.
- **Tenancy.** Every query is scoped by `user_id` in SQL. A cross-workspace read returns 404, and
  the integration suite proves it.
- **Helper keys.** `crew_` + 24 random bytes, shown once, stored as SHA-256, scoped to one agent,
  rotatable.
- **Headers.** Strict CSP (`script-src 'self'`, no inline scripts), `X-Frame-Options: DENY`,
  `nosniff`, no-store on API responses, HSTS in production.
- **Retention.** Expired codes are purged hourly; revoked sessions after 30 days.

## What it does not protect

- **A helper runs code on your machine.** The worker executes the command you give it, with your
  permissions. Point it at a sandbox if the work is untrusted.
- **OTP is phishable.** A real-time relay can forward a code. Use passkeys if you need
  phishing resistance.
- **Anyone with your inbox can sign in.** That is the trade-off of passwordless.
- **SQLite is single-node.** Do not put the database on a network share.

## Deploying safely

1. `CREW_ENV=production` (forces secure cookies and HSTS).
2. TLS terminator in front; set `CREW_TRUST_PROXY=true` only when it sets `X-Forwarded-For`.
3. Set `TRUSTED_PROXIES` on the PayTrackify side to the Crew host.
4. Keep `AUTH_SERVICE_KEY` random and private; `/api/auth/introspect` is server-only.
5. Configure SPF, DKIM and DMARC for the sending domain.
