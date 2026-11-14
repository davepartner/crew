# Advise Me

Left menu → **Advise Me**.

## No company

A short form asks for the company name, country, business type, website, support
email, audience and description. Creating it also builds the starter team. The
page then switches to the company study.

## Several companies

A picker lists each company with country, helper count, task count and your role.
Choosing one makes it active and studies it.

## The study

The breakdown follows the stage model from **Foreman** (MIT): idea, validation,
early traction, growth, scale. The questions and frameworks are informed by
**biz-lens** (MIT): Business Model Canvas, Porter, Ansoff, unit economics and
pre-mortems, applied in plain language.

Ten sections:

1. Foundation
2. Customer & research
3. Offer & pricing
4. Brand & website
5. Acquisition
6. Sales & follow-up
7. Delivery & operations
8. Money, legal & country
9. Team & systems
10. Growth

Each section has:

- a done / partial / not-started state
- a checklist with evidence and the next action
- practical actions
- one recommended helper you can create in place
- a country note where relevant

The score is the percentage of checks that pass based on the company's actual
profile, helpers, tasks, routines, goals, members, campaigns and model setup.

## Deep study

After the checklist loads, the page asks DeepSeek (or the user's model) for:

- a one-sentence diagnosis
- three constraints
- five actions for the next seven days
- three opportunities and three risks

It is cached until refreshed. Spinners cover the fetch and the study.

## Country advice

Profiles exist for Nigeria, Kenya, Ghana, South Africa, the UK, the US, Canada,
India and the UAE, with a generic fallback. They cover registration, tax,
payments, ad rules and data protection. They are general guidance, not legal,
tax or investment advice.

## Files and financials

**Company files** accepts up to 20 files (10 MB each). Text, CSV and JSON are
read directly. PDFs are parsed with `pdftotext`, scanned PDFs are rendered and
OCR'd, and images are OCR'd with Tesseract. Files can be downloaded or removed.
The host needs `pdftotext`, `pdftoppm` and `tesseract` installed (on AlmaLinux:
`dnf install -y poppler-utils tesseract`).

**Financial model** accepts one month of revenue, cost of sales, operating
costs, cash, receivables and payables. Crew computes gross profit and margin,
net profit, runway, break-even sales and annualised revenue. Mark a snapshot
**audited** only when the numbers come from audited statements; otherwise the
card says management accounts or estimate.

Adding a file, a financial snapshot or changing the company profile clears the
cached deep study so the next run uses the new evidence.

**Verify against the statement** compares the declared revenue, cost of sales
and operating costs with an uploaded statement. The result is `verified`,
`discrepancies` or `evidence_attached`. This is evidence checking, not a
statutory audit; the dialog links to the ICAN directory for a licensed auditor.
If no text comes back from a scan, the file card prompts for an OCR.space key.

## Products

A company can own multiple products. Each product is a full workspace with its
own helpers, tasks, routines, goals, marketing, Ads Studio and Advise Me.
See `docs/products.md`.

## Floating chat

The advisor reads the company profile, stage, score, actions, helpers, recent
tasks and campaigns, plus:

- the company's own website
- Wikipedia summaries for the country and business type
- Wikipedia search results for the question
- live Google News results for the industry (or Brave Search when a key is connected)
- a live market quote when the company has a public ticker

Answers cite those sources. Built-in SearXNG provides arbitrary web results
without a third-party key; Brave is optional. A public ticker adds a live market
quote. The chat is told to say “I don't know” when the data does not support an
answer. Every message is budgeted and recorded like other model calls.

## Sending

An **Email report** or **WhatsApp** button sends the study. Email uses the
server SMTP relay. WhatsApp uses a connected WhatsApp Cloud API number and only
works inside Meta's 24-hour customer window for free-form text. See
`docs/integrations.md`.

## Sources

- [Foreman](https://github.com/fatihguner/foreman) — stage model and diagnostic structure
- [biz-lens](https://github.com/sidtheone/biz-lens) — framework selection and research-first advice

Both are MIT. See `THIRD_PARTY_NOTICES.md`.
