import { useEffect, useRef, useState } from "react";
import {
  api,
  ApiError,
  type MiroFishEstimate,
  type MiroFishPage,
  type MiroFishResult,
  type MiroFishRun,
} from "../api";
import { Button, Field, Select, Skeleton, Textarea } from "../ui";

export function MiroFish() {
  const [data, setData] = useState<MiroFishPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [question, setQuestion] = useState("");
  const [product, setProduct] = useState("");
  const [agents, setAgents] = useState(12);
  const [rounds, setRounds] = useState(3);
  const [estimate, setEstimate] = useState<MiroFishEstimate | null>(null);
  const [estimating, setEstimating] = useState(false);
  const [estimateError, setEstimateError] = useState<string | null>(null);
  const [active, setActive] = useState<MiroFishRun | null>(null);
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);
  const [elapsed, setElapsed] = useState(0);
  const startedAt = useRef<number | null>(null);

  useEffect(() => {
    api
      .mirofish()
      .then((result) => {
        setData(result.mirofish);
        if (result.mirofish.products[0]) setProduct(result.mirofish.products[0].name);
      })
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (question.trim() === "") {
      setEstimate(null);
      setEstimateError(null);
      return;
    }
    setEstimating(true);
    const timer = window.setTimeout(async () => {
      try {
        const result = await api.estimateMirofish(question.trim(), agents, rounds);
        setEstimate(result.estimate);
        setEstimateError(null);
      } catch (err) {
        setEstimate(null);
        setEstimateError(err instanceof ApiError ? err.message : "We couldn't estimate this run.");
      } finally {
        setEstimating(false);
      }
    }, 450);
    return () => window.clearTimeout(timer);
  }, [question, agents, rounds]);

  useEffect(() => {
    if (!running) return;
    const timer = window.setInterval(() => {
      if (startedAt.current) setElapsed(Math.floor((Date.now() - startedAt.current) / 1000));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [running]);

  useEffect(() => {
    if (!active || (active.status !== "queued" && active.status !== "running")) return;
    const timer = window.setInterval(async () => {
      try {
        const result = await api.mirofishRun(active.id);
        setActive(result.run);
        if (result.run.status === "done" || result.run.status === "failed") {
          setRunning(false);
          setData((current) =>
            current
              ? {
                  ...current,
                  runs: [result.run, ...current.runs.filter((item) => item.id !== result.run.id)].slice(0, 10),
                }
              : current,
          );
          if (result.run.status === "failed") setRunError(result.run.error ?? "The simulation failed.");
        }
      } catch {
        // keep polling; a transient failure should not lose a running run
      }
    }, 1600);
    return () => window.clearInterval(timer);
  }, [active]);

  async function start() {
    if (!question.trim() || !data?.model_ready) return;
    setRunning(true);
    setRunError(null);
    setActive(null);
    startedAt.current = Date.now();
    setElapsed(0);
    try {
      const result = await api.runMirofish({
        question: question.trim(),
        product,
        agents,
        rounds,
      });
      setActive(result.run);
    } catch (err) {
      setRunning(false);
      setRunError(err instanceof ApiError ? err.message : "The simulation couldn't start.");
    }
  }

  if (loading) {
    return (
      <div className="stack">
        <div className="skeleton" style={{ height: 40, width: 320 }} />
        <div className="mirofish-grid">
          <div className="card"><Skeleton lines={6} /></div>
          <div className="card"><Skeleton lines={7} /></div>
        </div>
      </div>
    );
  }
  if (!data) {
    return <div className="empty"><div className="empty-title">We couldn't load MiroFish</div><Button className="mt-16" onClick={() => window.location.reload()}>Retry</Button></div>;
  }

  const result = active?.result;
  const isWorking = active?.status === "queued" || active?.status === "running" || running;

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>Swarm prediction for {data.company.name}</p>
          <h1 className="page-title">MiroFish</h1>
          <p className="page-sub">Rehearse a decision with a room of simulated customers, competitors and operators.</p>
        </div>
        <div className="mirofish-meta">
          <span className="badge">MiroFish-inspired</span>
          <span className="tiny muted">{data.model || "No model configured"}</span>
        </div>
      </header>

      <div className="mirofish-grid">
        <section className="card mirofish-controls">
          <h2 className="section-title">What do you want to predict?</h2>
          <p className="small muted">One decision, one question. The agents see only your company context and this question.</p>

          <div className="mirofish-suggestions">
            {data.suggestions.map((suggestion) => (
              <button
                key={suggestion}
                className="chip chip-button"
                onClick={() => setQuestion(suggestion)}
              >
                {suggestion}
              </button>
            ))}
          </div>

          <Field label="Your question" htmlFor="mirofish-question">
            <Textarea
              id="mirofish-question"
              rows={5}
              value={question}
              placeholder="What will happen if we raise prices by 15% next quarter?"
              onChange={(event) => setQuestion(event.target.value)}
            />
          </Field>

          <div className="form-grid">
            <Field label="Product context" htmlFor="mirofish-product">
              <Select id="mirofish-product" value={product} onChange={(event) => setProduct(event.target.value)}>
                <option value="">Whole company</option>
                {data.products.map((item) => (
                  <option key={item.id} value={item.name}>{item.name}</option>
                ))}
              </Select>
            </Field>
            <Field label="Agents" htmlFor="mirofish-agents">
              <Select id="mirofish-agents" value={agents} onChange={(event) => setAgents(Number(event.target.value))}>
                <option value={8}>8 — quick</option>
                <option value={12}>12 — balanced</option>
                <option value={18}>18 — broader</option>
                <option value={24}>24 — deeper</option>
              </Select>
            </Field>
            <Field label="Rounds" htmlFor="mirofish-rounds">
              <Select id="mirofish-rounds" value={rounds} onChange={(event) => setRounds(Number(event.target.value))}>
                <option value={2}>2</option>
                <option value={3}>3</option>
                <option value={4}>4</option>
              </Select>
            </Field>
          </div>

          <div className="mirofish-estimate" aria-live="polite">
            <span className="advise-label">Before you run</span>
            {estimating ? (
              <div className="row" style={{ gap: 8 }}>
                <span className="spinner" aria-label="Estimating" />
                <span className="small muted">Estimating tokens and time…</span>
              </div>
            ) : estimate ? (
              <>
                <div className="mirofish-estimate-main">
                  <strong>{estimate.tokens.toLocaleString()} tokens</strong>
                  <span>·</span>
                  <strong>{estimate.seconds_min}–{estimate.seconds_max}s</strong>
                  <span className="tiny muted">estimated</span>
                </div>
                <div className="tiny muted">
                  {estimate.calls} model calls · {agents} agents × {rounds} rounds · est. ${estimate.cost_usd.toFixed(4)}
                </div>
              </>
            ) : estimateError ? (
              <div>
                <p className="form-error" style={{ margin: 0 }}>{estimateError}</p>
                <a className="tiny" href="/settings">Add a model in Settings → AI models</a>
              </div>
            ) : (
              <p className="small muted" style={{ margin: 0 }}>Ask a question to see the token and time estimate before anything runs.</p>
            )}
          </div>

          <div className="row-between">
            <span className="tiny muted">{data.model_ready ? "Uses your configured model" : "No model configured"}</span>
            <Button
              variant="primary"
              onClick={start}
              loading={running}
              disabled={!question.trim() || !data.model_ready || isWorking}
            >
              {isWorking ? "Simulating…" : "Run simulation"}
            </Button>
          </div>
        </section>

        <section className="card mirofish-output">
          {isWorking ? (
            <div className="mirofish-running">
              <span className="spinner" aria-label="Simulation running" />
              <h2 className="section-title">The room is deliberating…</h2>
              <p className="small muted">
                {active
                  ? `${active.agents} agents, ${active.rounds} rounds. ${estimate ? `Estimated ${estimate.seconds_min}–${estimate.seconds_max}s; ` : ""}${elapsed}s elapsed.`
                  : "Starting the simulation…"}
              </p>
              <div className="mirofish-progress"><span /></div>
              <p className="tiny muted">You can leave this page. The run continues and appears in history.</p>
            </div>
          ) : result ? (
            <MiroFishReport result={result} run={active} />
          ) : runError ? (
            <div className="empty" style={{ padding: "40px 12px" }}>
              <div className="empty-title">The simulation failed</div>
              <p className="small">{runError}</p>
            </div>
          ) : (
            <div className="mirofish-empty">
              <span className="mirofish-mark" aria-hidden="true">✦</span>
              <h2 className="section-title">Predict before you commit</h2>
              <p className="small muted">
                MiroFish casts a cast of agents from your company context, runs several rounds of reaction,
                then writes a scenario report with disagreements and unknowns.
              </p>
              <p className="tiny muted">Pick a suggested question or write your own. You will see tokens and time before running.</p>
            </div>
          )}
        </section>
      </div>

      <section className="card mt-16">
        <div className="row-between">
          <h2 className="section-title">Previous simulations</h2>
          <span className="tiny muted">{data.runs.length} recent</span>
        </div>
        {data.runs.length === 0 ? (
          <p className="small muted">No simulations yet.</p>
        ) : (
          <div className="mirofish-history">
            {data.runs.map((run) => (
              <button
                key={run.id}
                className="mirofish-history-row"
                onClick={async () => {
                  try {
                    const response = await api.mirofishRun(run.id);
                    setActive(response.run);
                    setQuestion(run.question);
                    setProduct(run.product);
                    setAgents(run.agents);
                    setRounds(run.rounds);
                    setRunning(response.run.status === "queued" || response.run.status === "running");
                    if (response.run.status === "queued" || response.run.status === "running") startedAt.current = Date.now();
                  } catch {
                    // leave history row in place
                  }
                }}
              >
                <span className="mirofish-history-question">{run.question}</span>
                <span className={`badge mirofish-status-${run.status}`}>{run.status}</span>
                <span className="tiny muted">
                  {run.agents} agents · {run.rounds} rounds · {run.input_tokens + run.output_tokens} tokens
                </span>
              </button>
            ))}
          </div>
        )}
      </section>
    </>
  );
}

function MiroFishReport({ result, run }: { result: MiroFishResult; run: MiroFishRun }) {
  const confidence = Math.max(0, Math.min(100, result.confidence ?? 0));
  return (
    <div className="mirofish-report">
      <span className="advise-label">Prediction</span>
      <h2 className="section-title">{result.headline || "Simulation result"}</h2>
      <p className="mirofish-prediction">{result.prediction}</p>
      <div className="mirofish-confidence">
        <div className="row-between tiny muted"><span>Confidence</span><span>{confidence}%</span></div>
        <div className="mirofish-confidence-track"><span style={{ width: `${confidence}%` }} /></div>
      </div>
      <div className="mirofish-run-meta">
        {run.agents} agents · {run.rounds} rounds · {run.input_tokens + run.output_tokens} tokens · {(run.elapsed_ms / 1000).toFixed(1)}s
      </div>

      {result.scenarios?.length ? (
        <div className="mirofish-scenarios">
          {result.scenarios.map((scenario) => (
            <article key={scenario.name} className="mirofish-scenario">
              <div className="row-between">
                <strong>{scenario.name}</strong>
                <span className="badge">{scenario.probability}%</span>
              </div>
              <p className="small muted">{scenario.description}</p>
            </article>
          ))}
        </div>
      ) : null}

      {result.agent_reactions?.length ? (
        <div className="mirofish-section">
          <span className="advise-label">What the room said</span>
          <div className="mirofish-quotes">
            {result.agent_reactions.map((reaction, index) => (
              <blockquote key={index}>
                “{reaction.quote}”
                <cite>{reaction.persona} · {reaction.stance}</cite>
              </blockquote>
            ))}
          </div>
        </div>
      ) : null}

      <div className="mirofish-two-col">
        <div>
          <span className="advise-label">Recommended actions</span>
          <ol>{result.recommended_actions?.map((item, index) => <li key={index}>{item}</li>)}</ol>
        </div>
        <div>
          <span className="advise-label">Second-order effects</span>
          <ul>{result.second_order_effects?.map((item, index) => <li key={index}>{item}</li>)}</ul>
        </div>
      </div>

      <div className="mirofish-two-col">
        <div>
          <span className="advise-label">Dissent</span>
          <ul>{result.dissent?.map((item, index) => <li key={index}>{item}</li>)}</ul>
        </div>
        <div>
          <span className="advise-label">Unknowns</span>
          <ul>{result.unknowns?.map((item, index) => <li key={index}>{item}</li>)}</ul>
        </div>
      </div>
    </div>
  );
}
