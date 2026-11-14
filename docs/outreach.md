# Outreach

Left menu → **Outreach**.

Crew's outreach workspace follows the strongest ideas from open-source
outbound tools, adapted to a company-scoped web app with approval-first sending.

## The comparison

Star counts are from GitHub at the time of integration (November 2026).

| Project | Stars | License | Stack | Best at | Not a fit for |
|---|---:|---|---|---|---|
| [Sellbird](https://github.com/ethanplusai/sellbird) | 5.4k | AGPL-3.0 | Python, local CLI, SQLite | End-to-end 1:1 outbound: OSM discovery, 29 website signals, email verification, personal drafts, replies, approval-first | Multi-tenant hosted web, no mailbox warmup |
| [Warmbly](https://github.com/warmbly/warmbly) | 1.0k | Apache-2.0 | Go, TypeScript, Postgres, Redis | Cold email at scale, mailbox warmup, unified inbox, CRM, deliverability | Heavy infrastructure and setup |
| [Mautic](https://github.com/mautic/mautic) | 11k | Other (check repo) | PHP, MySQL | Marketing automation, segments, forms, landing pages, campaigns | 1:1 cold prospecting and signals |
| [Twenty](https://github.com/twentyhq/twenty) | 61k | Other (check repo) | TypeScript, Postgres | CRM and sales data | Prospecting, sequencing, sending |
| [Chatwoot](https://github.com/chatwoot/chatwoot) | 40k | Other (check repo) | Ruby, Vue | Support inbox and conversations | Outbound discovery and sequences |
| [Listmonk](https://github.com/knadh/listmonk) | 24k | AGPL-3.0 | Go, Postgres | Newsletter and list sending | Personal cold outbound |
| [Plunk](https://github.com/useplunk/plunk) | 5.6k | AGPL-3.0 | TypeScript | Email infrastructure and campaigns | Sales pipeline and replies |
| [Erxes](https://github.com/erxes/erxes) | 4.1k | Other (check repo) | TypeScript | All-in-one marketing, sales and support | Focused outbound pipeline |

## What Crew adopted

From **Sellbird**:

- pipeline stages: prospect → signal → contact → verify → draft → approve → send → reply
- free OpenStreetMap discovery
- website buying signals instead of a scraped people database
- email status labels and a suppression gate
- approval-first outbox
- Sellbird-compatible CSV import/export (`email`, `first_name`, `last_name`,
  `company_name`, `title`, `website`, `industry`, `personalization`,
  `email_status`, `score`, `source`, `status`)
- no invented proof or urgency in drafts

From **Warmbly**:

- explicit daily sending caps
- sequence steps with delay days
- suppression and opt-out handling
- deliverability mindset: SPF/DMARC, mailbox identity, low volume

From **Mautic/Twenty/Chatwoot**:

- staged prospects, contact roles and a reply view
- not their full CRM or support suites

## What Crew adds

- company and product scoping
- existing company profile, brand tone and AI budget
- DeepSeek drafts with a deterministic template fallback
- STIX/OpenCTI-adjacent security context is separate
- server SMTP sending with a daily limit
- CSV interoperability rather than copying AGPL code

## Pipeline

1. **Discover** — free OpenStreetMap query by trade and location, with built-in
   web search as fallback, or import a Sellbird CSV. Optional paid providers:
   Serper, TinyFish, DataForSEO and Scavio. Add each key from
   Settings → Integrations or directly in the Discover dialog; each provider
   bills its own account.
2. **Scan** — fetch the prospect's site and record buying signals: no booking,
   no chat, no schema, ads pixel, hiring, no fresh blog, mobile viewport, public
   email.
3. **Contact** — extract a public email, label it `mx_valid`, `risky` or
   `invalid`, and keep a primary contact.
4. **Draft** — write one short, signal-based email with the configured model.
5. **Approve** — nothing sends automatically by default.
6. **Send** — SMTP, daily cap, suppression check, `From` identity and signature.
7. **Reply / opt-out** — log the reply or suppress the address immediately.
8. **Export** — Sellbird/Instantly/Smartlead-compatible CSV.

## Limits

- No LinkedIn or people-database scraping.
- No mailbox warmup pool.
- No inbound IMAP sync; replies are logged manually.
- OpenStreetMap coverage is best for businesses with a physical location.
- Paid discovery spends the user's provider credits; Crew never buys credits or
  hides the provider bill.
- No automated follow-up scheduler yet; sequence steps are stored and step 1 is
  drafted.
- Email verification is MX/syntax/role based, not SMTP mailbox probing.

Sellbird is AGPL-3.0; Warmbly is Apache-2.0. Crew reimplements the workflow and
uses a compatible CSV shape. No Sellbird code is vendored.
