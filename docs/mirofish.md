# MiroFish

Left menu → **MiroFish**.

Crew's MiroFish-inspired page rehearses a business decision with a small
multi-agent simulation before the user commits.

Original project: [666ghj/MiroFish](https://github.com/666ghj/MiroFish)
(AGPL-3.0) — a swarm-intelligence prediction engine that builds a digital
world with Zep graph memory and OASIS social simulation.

Crew does **not** vendor or run MiroFish. It implements the same product idea in
Crew's own Go stack, using whichever model the user has already configured.

## How it works

1. Crew reads the company profile and its products.
2. It suggests questions based on those products.
3. The user asks one question and picks a size:
   - 8, 12, 18 or 24 agents
   - 2, 3 or 4 rounds
4. Before running, Crew calls `/api/mirofish/estimate` and shows:
   - estimated total tokens
   - estimated seconds (range)
   - number of model calls
   - estimated model cost
5. On run, Crew:
   - casts customer, competitor, partner and operator personas
   - runs reaction rounds in batches
   - synthesizes a report with scenarios, confidence, agent quotes, second-order
     effects, actions, dissent and unknowns
6. The run continues on the server. The page polls until it is done; history is
   saved per company.

## Suggested questions

Generated from the company and its products, for example:

- What will customers say if we raise the price of Product X by 15%?
- Which segment adopts Product X first?
- What happens if a competitor copies Product X?
- What is the biggest risk to the company over the next 12 months?
- What would make Product X fail in its first 90 days?

## Token and time estimates

The estimate is deliberately conservative:

- company context and question tokens
- one persona-generation call
- one batched reaction call per 8 agents per round
- one synthesis call
- ~38 output tokens/second and ~2.5s overhead per call

The estimate is shown **before** the run. Actual input/output tokens and elapsed
time are recorded after the run.

## Model

Uses the same model resolution as Ads Studio, Advise Me and Outreach:

- the included platform model, or
- the user's configured provider key.

No Zep account, OASIS install, Python service, or MiroFish API key is needed.

## Limits

- This is a small LLM-driven simulation, not MiroFish's full Zep/OASIS engine.
- It predicts reactions and scenarios, not facts.
- Agent personas are generated, not real people.
- Token/time estimates are approximate and can be exceeded.
- Runs are in-process; restarting Crew while a run is in flight may leave that
  run marked running until it is retried.

## API

- `GET /api/mirofish`
- `POST /api/mirofish/estimate`
- `POST /api/mirofish/runs`
- `GET /api/mirofish/runs/{id}`
