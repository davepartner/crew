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
payments, ad rules and data protection. They are general guidance, not legal or
tax advice.

## Floating chat

The advisor reads the company profile, stage, score, actions, helpers, recent
tasks and campaigns, plus:

- the company's own website
- Wikipedia summaries for the country and business type

Answers cite those sources. It is told to say “I don't know” when the data does
not support an answer. Every message is budgeted and recorded like other model
calls.

## Sources

- [Foreman](https://github.com/fatihguner/foreman) — stage model and diagnostic structure
- [biz-lens](https://github.com/sidtheone/biz-lens) — framework selection and research-first advice

Both are MIT. See `THIRD_PARTY_NOTICES.md`.
