import { useEffect, useMemo, useState } from "react";
import {
  api,
  ApiError,
  type CTIActor,
  type CTIFinding,
  type CTIReport,
  type SecurityAssessment,
  type SecurityPayload,
  type SecurityProvider,
} from "../api";
import { Button, Field, Input, Modal, Select, Skeleton, Tabs, Textarea, useToast } from "../ui";

const tabs = [
  { id: "overview", label: "Overview" },
  { id: "graph", label: "Threat graph" },
  { id: "indicators", label: "Indicators" },
  { id: "intel", label: "Intelligence" },
  { id: "assets", label: "Assets & sources" },
];

const indicatorTypes = ["domain", "ip", "url", "email", "md5", "sha1", "sha256"];

export function Security() {
  const { push } = useToast();
  const [data, setData] = useState<SecurityPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState("overview");
  const [assetOpen, setAssetOpen] = useState(false);
  const [indicatorOpen, setIndicatorOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);

  function load() {
    api
      .security()
      .then((result) => {
        setData(result.security);
        setError(null);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "We couldn't load the security workspace."))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function scan() {
    setScanning(true);
    setError(null);
    try {
      const result = await api.scanSecurity();
      setData(result.security);
      push("Scan complete. Review the findings and the new assessment.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The scan failed.");
      push("The scan failed. Check the assets and try again.", "error");
    } finally {
      setScanning(false);
    }
  }

  if (loading) {
    return (
      <div className="stack">
        <div className="skeleton" style={{ height: 38, width: 320 }} />
        <div className="grid-2">
          <div className="card"><Skeleton lines={4} /></div>
          <div className="card"><Skeleton lines={4} /></div>
        </div>
        <div className="card"><Skeleton lines={7} /></div>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="empty">
        <div className="empty-title">We couldn't load Cyber Security</div>
        <p className="small">{error}</p>
        <Button className="mt-16" onClick={load}>Try again</Button>
      </div>
    );
  }

  if (!data) return null;

  const analysis = data.assessment?.analysis;
  const severity = data.summary.severity;
  const threat = String(analysis?.threat_level ?? threatFromFindings(data.findings)).toLowerCase();
  const exposure = analysis?.exposure_score ?? exposureFromFindings(data.findings);

  return (
    <>
      <header className="page-head">
        <div>
          <p className="tiny muted" style={{ marginBottom: 6 }}>Company-scoped threat intelligence</p>
          <h1 className="page-title">Cyber Security</h1>
          <p className="page-sub">
            Assets, indicators, actors, reports and relationships — modelled on OpenCTI, scoped to this company.
          </p>
        </div>
        <div className="row" style={{ gap: 8 }}>
          <Button
            onClick={async () => {
              try {
                const blob = await api.securityStix();
                const url = URL.createObjectURL(blob);
                const link = document.createElement("a");
                link.href = url;
                link.download = "crew-threat-intel.json";
                link.click();
                URL.revokeObjectURL(url);
              } catch {
                push("We couldn't export the STIX bundle.", "error");
              }
            }}
          >
            Export STIX
          </Button>
          <Button variant="primary" onClick={scan} loading={scanning}>
            {scanning ? "Scanning assets and feeds…" : "Run scan"}
          </Button>
        </div>
      </header>

      <div className={`security-banner security-${threat}`}>
        <div>
          <span className="advise-label">Threat level</span>
          <strong>{threat}</strong>
        </div>
        <div>
          <span className="advise-label">Exposure</span>
          <strong>{exposure}/100</strong>
        </div>
        <div>
          <span className="advise-label">Assets</span>
          <strong>{data.summary.assets}</strong>
        </div>
        <div>
          <span className="advise-label">Active indicators</span>
          <strong>{data.summary.active_indicators}</strong>
        </div>
        <div>
          <span className="advise-label">High / critical</span>
          <strong>{(severity.high ?? 0) + (severity.critical ?? 0)}</strong>
        </div>
        <div className="security-scan-note">
          <span className="advise-label">Last scan</span>
          <strong>{data.scanned_at ? new Date(data.scanned_at).toLocaleString() : "Not yet"}</strong>
        </div>
      </div>

      <Tabs items={tabs} active={tab} onChange={setTab} />

      {scanning ? (
        <div className="security-scanning card">
          <span className="spinner" aria-label="Scanning" />
          <div>
            <strong>Scanning {data.summary.assets} assets</strong>
            <p className="small muted">DNS, email authentication, TLS, headers, URLhaus, ThreatFox and any connected feeds.</p>
          </div>
        </div>
      ) : null}

      {tab === "overview" ? (
        <Overview data={data} analysis={analysis} onScan={scan} scanning={scanning} />
      ) : null}
      {tab === "graph" ? <ThreatGraph data={data} /> : null}
      {tab === "indicators" ? (
        <Indicators data={data} onAdd={() => setIndicatorOpen(true)} onChanged={load} />
      ) : null}
      {tab === "intel" ? <Intelligence data={data} onAdd={() => setReportOpen(true)} /> : null}
      {tab === "assets" ? (
        <Assets data={data} onAdd={() => setAssetOpen(true)} onChanged={load} />
      ) : null}

      <AssetModal open={assetOpen} onClose={() => setAssetOpen(false)} onSaved={() => { setAssetOpen(false); load(); }} />
      <IndicatorModal open={indicatorOpen} onClose={() => setIndicatorOpen(false)} onSaved={() => { setIndicatorOpen(false); load(); }} />
      <ReportModal open={reportOpen} onClose={() => setReportOpen(false)} onSaved={() => { setReportOpen(false); load(); }} />
    </>
  );
}

function Overview({
  data,
  analysis,
  onScan,
  scanning,
}: {
  data: SecurityPayload;
  analysis: SecurityAssessment["analysis"] | undefined;
  onScan: () => void;
  scanning: boolean;
}) {
  return (
    <div className="stack">
      <div className="grid-2">
        <section className="card">
          <div className="row-between">
            <h2 className="section-title">Assessment</h2>
            {data.assessment ? <span className="tiny muted">{data.assessment.model}</span> : null}
          </div>
          {data.assessment ? (
            <>
              <p className="security-summary">{analysis?.summary}</p>
              <span className="advise-label">Business impact</span>
              <p className="small muted">{analysis?.business_impact}</p>
              <span className="advise-label">Next actions</span>
              <ol className="security-actions">
                {(analysis?.actions ?? []).map((action, index) => <li key={index}>{action}</li>)}
              </ol>
            </>
          ) : (
            <>
              <p className="small muted">
                No assessment yet. Run a scan to map assets, pull free and connected threat feeds, and ask the model
                to prioritize the findings.
              </p>
              <Button onClick={onScan} loading={scanning}>Run first scan</Button>
            </>
          )}
        </section>

        <section className="card">
          <h2 className="section-title mb-8">Findings</h2>
          {data.findings.length === 0 ? (
            <p className="small muted">No technical findings yet. Run a scan.</p>
          ) : (
            <div className="security-findings">
              {data.findings.slice(0, 6).map((finding) => (
                <Finding key={finding.id} finding={finding} />
              ))}
            </div>
          )}
        </section>
      </div>

      <section className="card">
        <div className="row-between">
          <h2 className="section-title">Risk register</h2>
          <span className="tiny muted">{data.summary.total_indicators} indicators</span>
        </div>
        {(analysis?.risks ?? []).length === 0 ? (
          <p className="small muted">Risks appear after an assessment.</p>
        ) : (
          <div className="security-risks">
            {(analysis?.risks ?? []).map((risk, index) => (
              <div className="security-risk" key={index}>
                <span className={`badge security-sev-${risk.severity}`}>{risk.severity}</span>
                <strong>{risk.title}</strong>
                <p className="small muted">{risk.why}</p>
                <p className="small"><strong>Fix:</strong> {risk.fix}</p>
              </div>
            ))}
          </div>
        )}
      </section>

      <ProviderBanner providers={data.providers} />
    </div>
  );
}

function Finding({ finding }: { finding: CTIFinding }) {
  return (
    <div className="security-finding">
      <span className={`badge security-sev-${finding.severity}`}>{finding.severity}</span>
      <div>
        <strong>{finding.title}</strong>
        <p className="small muted">{finding.detail}</p>
        <p className="tiny muted">{finding.fix}</p>
      </div>
    </div>
  );
}

function ThreatGraph({ data }: { data: SecurityPayload }) {
  const assets = data.assets.slice(0, 7);
  const indicators = data.indicators.slice(0, 9);
  const reports = data.reports.slice(0, 4);
  const actors = data.actors.slice(0, 3);

  const positions = useMemo(() => {
    const map = new Map<string, { x: number; y: number; type: string; label: string }>();
    const column = (items: { id: string; label: string }[], x: number, type: string) => {
      items.forEach((item, index) => {
        const gap = 330 / Math.max(items.length, 1);
        map.set(type + ":" + item.id, { x, y: 55 + index * gap, type, label: item.label });
      });
    };
    column(assets.map((item) => ({ id: item.id, label: item.value })), 90, "asset");
    column(indicators.map((item) => ({ id: item.id, label: item.value })), 620, "indicator");
    column(reports.map((item) => ({ id: item.id, label: item.title })), 355, "report");
    column(actors.map((item) => ({ id: item.id, label: item.name })), 355, "actor");
    return map;
  }, [assets, indicators, reports, actors]);

  const edges: { from: { x: number; y: number }; to: { x: number; y: number } }[] = [];
  for (const indicator of indicators) {
    for (const asset of assets) {
      if (indicator.value.toLowerCase().includes(asset.value.toLowerCase())) {
        const from = positions.get("asset:" + asset.id);
        const to = positions.get("indicator:" + indicator.id);
        if (from && to) edges.push({ from, to });
      }
    }
  }
  for (const report of reports) {
    const from = positions.get("report:" + report.id);
    if (!from) continue;
    for (const indicator of indicators.slice(0, 2)) {
      const to = positions.get("indicator:" + indicator.id);
      if (to) edges.push({ from, to });
    }
  }

  return (
    <section className="card">
      <div className="row-between">
        <div>
          <h2 className="section-title">Threat graph</h2>
          <p className="small muted">Assets, indicators, reports and actors. Lines show observed relationships.</p>
        </div>
        <div className="security-legend">
          <span><i className="dot asset" /> Asset</span>
          <span><i className="dot indicator" /> Indicator</span>
          <span><i className="dot report" /> Report</span>
          <span><i className="dot actor" /> Actor</span>
        </div>
      </div>

      {assets.length === 0 && indicators.length === 0 && reports.length === 0 ? (
        <p className="small muted mt-16">Add assets and run a scan to build the graph.</p>
      ) : (
        <div className="security-graph">
          <svg viewBox="0 0 720 390" role="img" aria-label="Threat intelligence graph">
            {edges.map((edge, index) => (
              <line key={index} x1={edge.from.x} y1={edge.from.y} x2={edge.to.x} y2={edge.to.y} className="security-edge" />
            ))}
            {Array.from(positions.entries()).map(([key, node]) => (
              <g key={key} transform={`translate(${node.x},${node.y})`}>
                <circle r="18" className={`security-node ${node.type}`} />
                <text y="34" textAnchor="middle" className="security-node-label">
                  {node.label.length > 24 ? node.label.slice(0, 23) + "…" : node.label}
                </text>
              </g>
            ))}
          </svg>
        </div>
      )}
    </section>
  );
}

function Indicators({
  data,
  onAdd,
  onChanged,
}: {
  data: SecurityPayload;
  onAdd: () => void;
  onChanged: () => void;
}) {
  const { push } = useToast();
  return (
    <section className="card">
      <div className="row-between">
        <div>
          <h2 className="section-title">Indicators</h2>
          <p className="small muted">IOCs with confidence, TLP and source. Mark false positives benign.</p>
        </div>
        <Button onClick={onAdd}>Add indicator</Button>
      </div>

      {data.indicators.length === 0 ? (
        <p className="small muted mt-16">No indicators yet. A scan pulls URLhaus and ThreatFox matches; connected feeds add more.</p>
      ) : (
        <div className="security-table">
          <div className="security-table-head">
            <span>Type</span><span>Value</span><span>Source</span><span>Confidence</span><span>TLP</span><span>Status</span><span />
          </div>
          {data.indicators.map((indicator) => (
            <div className="security-table-row" key={indicator.id}>
              <span className="tiny muted">{indicator.type}</span>
              <span className="security-ioc" title={indicator.value}>{indicator.value}</span>
              <span className="tiny muted">{indicator.source}</span>
              <span>{indicator.confidence}%</span>
              <span className={`badge tlp-${indicator.tlp}`}>{indicator.tlp}</span>
              <span className={`badge ${indicator.status === "active" ? "badge-todo" : "badge-done"}`}>{indicator.status}</span>
              <span className="row" style={{ gap: 4 }}>
                {indicator.status === "active" ? (
                  <button className="link-button tiny" onClick={async () => {
                    try { await api.setSecurityIndicatorStatus(indicator.id, "benign"); push("Marked benign."); onChanged(); }
                    catch { push("We couldn't update it.", "error"); }
                  }}>Benign</button>
                ) : null}
                <button className="link-button tiny" onClick={async () => {
                  try { await api.deleteSecurityIndicator(indicator.id); onChanged(); }
                  catch { push("We couldn't delete it.", "error"); }
                }}>Delete</button>
              </span>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function Intelligence({ data, onAdd }: { data: SecurityPayload; onAdd: () => void }) {
  return (
    <div className="stack">
      <section className="card">
        <div className="row-between">
          <div>
            <h2 className="section-title">Reports</h2>
            <p className="small muted">External reports and connected-feed observations, with source and confidence.</p>
          </div>
          <Button onClick={onAdd}>Add report</Button>
        </div>
        {data.reports.length === 0 ? (
          <p className="small muted mt-16">No reports yet. URLhaus and OTX create reports during a scan.</p>
        ) : (
          <div className="security-reports">
            {data.reports.map((report) => (
              <ReportCard key={report.id} report={report} />
            ))}
          </div>
        )}
      </section>

      <section className="card">
        <h2 className="section-title">Actors</h2>
        <p className="small muted">Attribution stays explicit; Crew never invents an actor.</p>
        {data.actors.length === 0 ? (
          <p className="small muted mt-16">No actor has been linked to this company.</p>
        ) : (
          <div className="security-reports">
            {data.actors.map((actor) => <ActorCard key={actor.id} actor={actor} />)}
          </div>
        )}
      </section>
    </div>
  );
}

function ReportCard({ report }: { report: CTIReport }) {
  return (
    <article className="security-report">
      <div className="row-between">
        <strong>{report.title}</strong>
        <span className={`badge tlp-${report.tlp}`}>{report.tlp}</span>
      </div>
      <p className="small muted">{report.summary}</p>
      <div className="row-between tiny muted">
        <span>{report.source} · {report.confidence}% confidence</span>
        {report.source_url ? <a href={report.source_url} target="_blank" rel="noreferrer noopener">Source ↗</a> : null}
      </div>
    </article>
  );
}

function ActorCard({ actor }: { actor: CTIActor }) {
  return (
    <article className="security-report">
      <div className="row-between">
        <strong>{actor.name}</strong>
        <span className="tiny muted">{actor.confidence}%</span>
      </div>
      <p className="small muted">{actor.description}</p>
      <div className="tiny muted">{[actor.motivation, actor.sophistication, actor.country].filter(Boolean).join(" · ")}</div>
    </article>
  );
}

function Assets({ data, onAdd, onChanged }: { data: SecurityPayload; onAdd: () => void; onChanged: () => void }) {
  const { push } = useToast();
  return (
    <div className="stack">
      <section className="card">
        <div className="row-between">
          <div>
            <h2 className="section-title">Monitored assets</h2>
            <p className="small muted">Domains, IPs, emails, URLs and brands. The scan only checks these.</p>
          </div>
          <Button onClick={onAdd}>Add asset</Button>
        </div>
        <div className="security-assets">
          {data.assets.map((asset) => (
            <div className="security-asset" key={asset.id}>
              <span className={`security-asset-type ${asset.type}`}>{asset.type}</span>
              <span style={{ minWidth: 0 }}>
                <strong className="security-ioc">{asset.value}</strong>
                <span className="tiny muted">{asset.label || asset.source} · {asset.criticality} criticality</span>
              </span>
              <button className="link-button tiny" onClick={async () => {
                try { await api.deleteSecurityAsset(asset.id); onChanged(); }
                catch { push("We couldn't delete that asset.", "error"); }
              }}>Remove</button>
            </div>
          ))}
        </div>
      </section>

      <section className="card">
        <h2 className="section-title">Threat feeds</h2>
        <p className="small muted mb-16">Free feeds work now. Add a key for deeper enrichment.</p>
        <div className="security-providers">
          {data.providers.map((provider) => (
            <ProviderCard key={provider.key} provider={provider} onSaved={onChanged} />
          ))}
        </div>
        <div className="security-free">
          <strong>Free feeds</strong>
          <span>URLhaus · ThreatFox · DNS · TLS · SPF/DMARC · security headers</span>
        </div>
      </section>
    </div>
  );
}

function ProviderCard({ provider, onSaved }: { provider: SecurityProvider; onSaved: () => void }) {
  const { push } = useToast();
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);

  async function save() {
    if (key.trim() === "") return;
    setBusy(true);
    try {
      await api.saveIntegration(provider.key, { api_key: key.trim() });
      setKey("");
      push(`${provider.name} connected.`);
      onSaved();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't save that key.", "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="security-provider">
      <div className="row-between">
        <strong>{provider.name}</strong>
        <span className={`badge ${provider.configured ? "badge-done" : "badge-todo"}`}>{provider.configured ? "Connected" : "Not connected"}</span>
      </div>
      <p className="tiny muted">
        <a href={provider.url} target="_blank" rel="noreferrer noopener">Get a key ↗</a>
        <span> · </span>
        <a href={provider.docs} target="_blank" rel="noreferrer noopener">Docs ↗</a>
      </p>
      <div className="row" style={{ gap: 6 }}>
        <Input
          type="password"
          placeholder={provider.configured ? "•••••••• (leave blank)" : "API key"}
          value={key}
          onChange={(event) => setKey(event.target.value)}
        />
        <Button size="sm" loading={busy} onClick={save}>Save</Button>
      </div>
    </div>
  );
}

function ProviderBanner({ providers }: { providers: SecurityProvider[] }) {
  const configured = providers.filter((provider) => provider.configured).length;
  if (configured > 0) return null;
  return (
    <div className="setup-callout">
      <div>
        <strong>Add a threat-intel key</strong>
        <p className="small muted" style={{ margin: "4px 0 0" }}>
          Free feeds run now. VirusTotal, AbuseIPDB, OTX and Shodan add domain, IP and reputation context.
        </p>
      </div>
      <a className="button button-small button-primary" href="/settings">Add keys</a>
    </div>
  );
}

function AssetModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [type, setType] = useState("domain");
  const [value, setValue] = useState("");
  const [label, setLabel] = useState("");
  const [criticality, setCriticality] = useState("high");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    setBusy(true);
    try {
      await api.addSecurityAsset({ type, value: value.trim(), label: label.trim(), criticality });
      push("Asset added.");
      setValue("");
      setLabel("");
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't add that asset.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Add monitored asset" onClose={onClose} footer={
      <>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button variant="primary" onClick={save} loading={busy} disabled={value.trim() === ""}>Add asset</Button>
      </>
    }>
      <div className="form-grid">
        <Field label="Type" htmlFor="cti-asset-type">
          <Select id="cti-asset-type" value={type} onChange={(event) => setType(event.target.value)}>
            {["domain", "ip", "email", "url", "brand"].map((item) => <option key={item} value={item}>{item}</option>)}
          </Select>
        </Field>
        <Field label="Criticality" htmlFor="cti-asset-criticality">
          <Select id="cti-asset-criticality" value={criticality} onChange={(event) => setCriticality(event.target.value)}>
            {["critical", "high", "medium", "low"].map((item) => <option key={item} value={item}>{item}</option>)}
          </Select>
        </Field>
      </div>
      <Field label="Value" htmlFor="cti-asset-value">
        <Input id="cti-asset-value" value={value} onChange={(event) => setValue(event.target.value)} placeholder="example.com" />
      </Field>
      <Field label="Label" htmlFor="cti-asset-label">
        <Input id="cti-asset-label" value={label} onChange={(event) => setLabel(event.target.value)} placeholder="Primary store" />
      </Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function IndicatorModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [type, setType] = useState("domain");
  const [value, setValue] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [confidence, setConfidence] = useState("70");
  const [tlp, setTLP] = useState("amber");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    setBusy(true);
    try {
      await api.addSecurityIndicator({
        type, value: value.trim(), name: name.trim(), description: description.trim(),
        confidence: Number(confidence) || 50, tlp, tags: [],
      });
      push("Indicator added.");
      setValue("");
      setName("");
      setDescription("");
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't add that indicator.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Add indicator" onClose={onClose} footer={
      <>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button variant="primary" onClick={save} loading={busy} disabled={value.trim() === ""}>Add indicator</Button>
      </>
    }>
      <div className="form-grid">
        <Field label="Type" htmlFor="cti-indicator-type">
          <Select id="cti-indicator-type" value={type} onChange={(event) => setType(event.target.value)}>
            {indicatorTypes.map((item) => <option key={item} value={item}>{item}</option>)}
          </Select>
        </Field>
        <Field label="Confidence" htmlFor="cti-indicator-confidence">
          <Input id="cti-indicator-confidence" type="number" min="1" max="100" value={confidence} onChange={(event) => setConfidence(event.target.value)} />
        </Field>
        <Field label="TLP" htmlFor="cti-indicator-tlp">
          <Select id="cti-indicator-tlp" value={tlp} onChange={(event) => setTLP(event.target.value)}>
            {["clear", "white", "green", "amber", "red"].map((item) => <option key={item} value={item}>{item}</option>)}
          </Select>
        </Field>
      </div>
      <Field label="Value" htmlFor="cti-indicator-value">
        <Input id="cti-indicator-value" value={value} onChange={(event) => setValue(event.target.value)} placeholder="malicious.example" />
      </Field>
      <Field label="Name" htmlFor="cti-indicator-name">
        <Input id="cti-indicator-name" value={name} onChange={(event) => setName(event.target.value)} />
      </Field>
      <Field label="Description" htmlFor="cti-indicator-description">
        <Textarea id="cti-indicator-description" rows={3} value={description} onChange={(event) => setDescription(event.target.value)} />
      </Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function ReportModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const { push } = useToast();
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [source, setSource] = useState("");
  const [confidence, setConfidence] = useState("60");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    setBusy(true);
    try {
      await api.addSecurityReport({ title, summary, source, confidence: Number(confidence) || 60, tlp: "amber" });
      push("Report added.");
      setTitle("");
      setSummary("");
      setSource("");
      onSaved();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't add that report.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Add intelligence report" onClose={onClose} footer={
      <>
        <Button onClick={onClose} disabled={busy}>Cancel</Button>
        <Button variant="primary" onClick={save} loading={busy} disabled={title.trim() === ""}>Add report</Button>
      </>
    }>
      <Field label="Title" htmlFor="cti-report-title">
        <Input id="cti-report-title" value={title} onChange={(event) => setTitle(event.target.value)} />
      </Field>
      <Field label="Summary" htmlFor="cti-report-summary">
        <Textarea id="cti-report-summary" rows={4} value={summary} onChange={(event) => setSummary(event.target.value)} />
      </Field>
      <div className="form-grid">
        <Field label="Source" htmlFor="cti-report-source">
          <Input id="cti-report-source" value={source} onChange={(event) => setSource(event.target.value)} />
        </Field>
        <Field label="Confidence" htmlFor="cti-report-confidence">
          <Input id="cti-report-confidence" type="number" min="1" max="100" value={confidence} onChange={(event) => setConfidence(event.target.value)} />
        </Field>
      </div>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function threatFromFindings(findings: CTIFinding[]): string {
  const critical = findings.filter((item) => item.severity === "critical").length;
  const high = findings.filter((item) => item.severity === "high").length;
  if (critical > 0) return "critical";
  if (high > 2) return "high";
  if (high > 0) return "elevated";
  return "guarded";
}

function exposureFromFindings(findings: CTIFinding[]): number {
  const weights: Record<string, number> = { critical: 30, high: 18, medium: 8, low: 3, info: 0 };
  return Math.min(100, findings.reduce((sum, finding) => sum + (weights[finding.severity] ?? 0), 0));
}
