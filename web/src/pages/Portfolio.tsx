import { useEffect, useState } from "react";
import { api, ApiError, setCompany as rememberCompany, type PortfolioRow, type PortfolioTotal } from "../api";
import { Button, Select, Skeleton, useToast } from "../ui";

export function Portfolio() {
  const { push } = useToast();
  const [rows, setRows] = useState<PortfolioRow[] | null>(null);
  const [totals, setTotals] = useState<PortfolioTotal[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [base, setBase] = useState("USD");
  const [ratesAvailable, setRatesAvailable] = useState(false);
  const [fxKey, setFxKey] = useState(false);

  useEffect(() => {
    let live = true;
    setRows(null);
    api
      .portfolio(base)
      .then((result) => {
        if (!live) return;
        setRows(result.rows);
        setTotals(result.totals);
        setRatesAvailable(result.rates_available);
        setFxKey(result.fx_key);
      })
      .catch((err) => live && setError(err instanceof ApiError ? err.message : "We couldn't build the portfolio."))
      .finally(() => undefined);
    return () => {
      live = false;
    };
  }, [base]);

  function open(row: PortfolioRow) {
    rememberCompany(row.id);
    window.location.assign("/");
  }

  function format(currency: string, value: number) {
    const prefix = currency.split(" ")[0] || "";
    return `${prefix} ${Math.round(value).toLocaleString()}`;
  }

  if (rows === null && !error) {
    return (
      <div className="stack">
        <div className="skeleton" style={{ height: 36, width: 240 }} />
        <div className="grid-2">
          <div className="card"><Skeleton lines={4} /></div>
          <div className="card"><Skeleton lines={4} /></div>
        </div>
        <div className="card"><Skeleton lines={6} /></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't build the portfolio</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={() => window.location.reload()}>Try again</Button>
      </div>
    );
  }

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>All products</p>
          <h1 className="page-title">Portfolio</h1>
          <p className="page-sub">One view of every product: money, work and health.</p>
        </div>
        <div className="row" style={{ gap: 8 }}>
          <Select value={base} onChange={(event) => setBase(event.target.value)} aria-label="Report currency">
            {["USD", "NGN", "GBP", "EUR", "KES", "GHS", "ZAR", "CAD", "INR", "AED"].map((code) => (
              <option key={code} value={code}>{code}</option>
            ))}
          </Select>
          <Button
            onClick={() => {
              push("Create products in Settings → Company.");
              window.location.assign("/settings");
            }}
          >
            New product
          </Button>
        </div>
      </header>

      {!ratesAvailable ? (
        <div className="setup-callout">
          <div>
            <strong>Convert every product into {base}</strong>
            <p className="small muted" style={{ margin: "4px 0 0" }}>
              Add an ExchangeRate-API key to convert NGN, USD and other currencies into one total.
            </p>
          </div>
          <div className="row" style={{ gap: 8 }}>
            <a className="button button-small" href="https://www.exchangerate-api.com/" target="_blank" rel="noreferrer noopener">
              Get a key
            </a>
            <a className="button button-small button-primary" href="/settings">Add key</a>
          </div>
        </div>
      ) : (
        <p className="tiny muted" style={{ margin: "0 0 10px" }}>
          Converted to {base} at the latest rates. Original amounts are kept in the product records.{" "}
          {!fxKey ? (
            <a href="https://www.exchangerate-api.com/" target="_blank" rel="noreferrer noopener">
              Add a key for a supported SLA ↗
            </a>
          ) : null}
        </p>
      )}

      <div className="portfolio-totals">
        {totals.map((total) => (
          <section className="card portfolio-total" key={total.currency || "unknown"}>
            <span className="advise-label">{total.currency || "Currency"}</span>
            <strong>{format(total.currency, total.revenue)}</strong>
            <span className="tiny muted">revenue</span>
            <div className="portfolio-total-split">
              <span>Net <b className={total.net_profit < 0 ? "text-danger" : "text-ok"}>{format(total.currency, total.net_profit)}</b></span>
              <span>Cash <b>{format(total.currency, total.cash)}</b></span>
            </div>
          </section>
        ))}
      </div>

      <section className="card mt-16">
        <h2 className="section-title mb-8">Products</h2>
        <div className="portfolio-table">
          <div className="portfolio-head">
            <span>Product</span>
            <span>Stage</span>
            <span>Score</span>
            <span>Revenue</span>
            <span>Net</span>
            <span>Cash</span>
            <span>Work</span>
            <span />
          </div>
          {(rows ?? []).map((row) => (
            <div className="portfolio-row" key={row.id}>
              <span>
                <strong>{row.name}</strong>
                {row.is_product ? <span className="tiny muted"> · product</span> : <span className="tiny muted"> · company</span>}
              </span>
              <span className="tiny">{row.stage}</span>
              <span>{row.score}%</span>
              <span>{format(row.currency, row.revenue)}</span>
              <span className={row.net_profit < 0 ? "text-danger" : ""}>{format(row.currency, row.net_profit)}</span>
              <span>{format(row.currency, row.cash)}</span>
              <span className="tiny muted">{row.tasks} tasks · {row.agents} helpers</span>
              <Button size="sm" onClick={() => open(row)}>Open</Button>
            </div>
          ))}
        </div>
        {rows && rows.length <= 1 ? (
          <p className="small muted mt-16">
            You only have the parent company. Add products in Settings → Company to compare them here.
          </p>
        ) : null}
      </section>
    </>
  );
}
