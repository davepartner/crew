# UI review — the 20 laws

Audited against `20-ui-ux-laws` (davepartner) on 13 November 2026, in Chromium at 1280×900
and 390×844, light and dark.

## Cognitive & psychological

| Law | Verdict | Where |
|---|---|---|
| **Hick's** | Pass | Sign-in exposes one field and one action. Helper form has 4 fields in one step. Settings splits into 5 sections. Nav has 4 items. |
| **Fitts's** | Pass | Every button, tab and nav item is ≥44 px tall; inputs 46 px; task rows 56 px. Primary actions sit top-right (desktop) or full-width (mobile). Destructive actions live in a separate card and open a confirm modal. |
| **Jakob's** | Pass | Sidebar navigation, avatars bottom-left/top-right, code entry with `autocomplete="one-time-code"`, standard selects and checkboxes. |
| **Miller's** | Pass | Nav 4, settings sections 5, statuses 4, helper roles 7 in a single select, task fields 3. |
| **Aesthetic-usability** | Pass | One type scale (12/13/15/18/23/26), two radii (9/14), warm neutral palette, every native control styled. |
| **Postel's** | Pass | Email is trimmed and lowercased; the code strips non-digits; the server accepts spaces. Errors keep the email and explain the next step. |
| **Tesler's** | Pass | The only technical string is the helper command, which is copied, not typed. Name is derived from the email for first-run users. |
| **Parkinson's** | Pass | Name pre-filled, role/emoji defaults, theme remembered, last-used email preserved after an error. |

## Visual perception

| Law | Verdict | Where |
|---|---|---|
| **Proximity** | Pass | 6 px label→input, 16 px between fields, 28–32 px between sections. |
| **Similarity** | Pass | One `Button`, `Input`, `Card`, `Badge` set; status colours are the only variation. |
| **Common region** | Pass | Every module sits in a bordered, radiused card; empty states share the same container. |
| **Closure** | Pass | Skeleton loaders on every async surface; the queue hints at more with "See all". |
| **Continuity** | Pass | Rows, timelines and tabs share one left edge per column. |
| **Peak-end** | Pass | Finishing an action produces a toast and a visible state change (status badge, activity entry). |

## Attention & feedback

| Law | Verdict | Where |
|---|---|---|
| **Serial position** | Pass | Home first, Settings last, Sign out last in the profile menu. |
| **Von Restorff** | Pass | Exactly one filled accent per region; "Add a task" steps back to a secondary button when a review is waiting. |
| **Occam's** | Pass | No decorative icons beyond the four nav glyphs and helper emoji. |
| **Doherty** | Pass | Buttons enter a loading state on submit; lists show skeletons; the verify step shows a spinner while the code is checked. |
| **Goal-gradient** | Pass | The sign-in flow shows a two-segment progress bar; Settings uses a persistent section bar. |
| **Paradox of choice** | Pass | No parallel plans or upsells. Role selection is a single defaulted select. |

## Universal standards

| Requirement | Verdict | Notes |
|---|---|---|
| Visible focus | Pass | Global `:focus-visible` ring, 2 px accent, 2 px offset. |
| Contrast ≥ 4.5:1 | Pass | Body ink 15:1, muted 5.1:1, faint raised to 4.6:1 in light and 4.5:1 in dark. |
| Reduced motion | Pass | All animation and transitions collapse under `prefers-reduced-motion`. |
| Responsive | Pass | 390 / 768 / 1280 breakpoints; sidebar becomes a drawer under 920 px. |
| Semantic + ARIA | Pass | `nav`, `main`, `form`, `label`, `fieldset`; `aria-current`, `aria-pressed`, `aria-live` for errors, `aria-modal` dialogs with a Tab trap, Esc closes. |
| No stack traces | Pass | Every error is a sentence with a next step. |

## Company onboarding

- Four steps, one decision each: business → details → team → model.
- The business picker is searchable and grouped by category; 66 presets, with "Something else"
  as an always-visible fallback.
- The team and first tasks are previewed before anything is created.
- The action bar is sticky, so Continue is reachable without scrolling a long list.
- The model step labels free providers and links to each key page.
- Consent for sharing business details with an AI provider is stated before creation.

## Marketing

- Two tabs: a searchable 53-skill library with category chips, and the teardown tool.
- Skill cards show category, name and a three-line description; the full text opens in a modal
  with a helper picker, capped at 8 skills per helper.
- Teardown form asks for one URL, an optional question, a playbook and a helper. It explains
  that only the page is read and unknowns are reported.
- If no model key exists, the form says so before the person tries; the run button is disabled.
- The report renders as pre-wrapped markdown in a scrollable panel with download and task links.

## Account surfaces

- Company: name, website, support email, audience, description, tone.
- Profile: name, email (read-only, explained), member since, appearance.
- Sign-in & devices: device, IP, last used, per-device sign-out.
- AI models: provider list, free flags, encrypted keys, test, replace, remove.
- Privacy & data: JSON export, link to the policy.
- Danger zone: delete workspace, gated by typing `delete`.
- Sign out: last item in the profile menu, confirmation dialog with an optional reason, recorded server-side; Esc and Cancel are safe.
- Legal and support are linked from sign-in and from the footer of both auth and app surfaces.

## Known deviations

- Inline prose links (e.g. "Terms" inside a sentence) are smaller than 44 px. They are text
  links inside copy, not controls; primary navigation and actions meet the target.
- The helper key uses `window.confirm` for rotation. It is a native, focus-trapped dialog and
  accessible, but a custom modal would match the rest of the product better.
- Toasts sit bottom-right and can cover the activity card for their four-second life. They are
  capped at three and never block a control.

## Trace

```
Decision: sign-in is a two-step flow with a single email field, then one 8-digit input
  → Law: Hick's Law (one decision per screen) + Goal-Gradient (two-segment progress bar)

Decision: primary CTA is full-width on mobile and top-right on desktop; all buttons ≥44 px
  → Law: Fitts's Law (large, predictable target) + Jakob's Law (standard placement)

Decision: status is always a coloured badge with a word, never colour alone
  → Law: Similarity (same role, same treatment) + accessibility (WCAG 1.4.1)

Decision: "Needs your OK" tasks lead the home screen and are the only place with a filled
  accent when one exists
  → Law: Von Restorff (single accent) + Serial Position (priority first)

Decision: empty states carry the next action ("Add your first task")
  → Law: Peak-End Rule (never end on a dead end) + Parkinson's Law (automate the next step)
```

## Evidence

Screenshots in [`docs/screenshots`](screenshots): `login.png`, `home.png`, `task.png`,
`mobile.png`. Dark mode was checked on the same screens with Chromium's
`prefers-color-scheme: dark`.
