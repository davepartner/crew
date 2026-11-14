import { useEffect, useState } from "react";
import { api, ApiError, type AdAccount, type AdCampaign, type AdConcept, type AdResult, type AdsBrief, type Agent } from "../api";
import { navigate } from "../router";
import { money } from "../time";
import { Button, Field, Input, Modal, Select, Skeleton, Textarea, useToast } from "../ui";

const platforms = [
  { key: "meta", label: "Meta", hint: "Facebook + Instagram" },
  { key: "tiktok", label: "TikTok", hint: "Native short video" },
  { key: "youtube", label: "YouTube", hint: "Bumper, in-stream, Shorts" },
];

const objectives = [
  { value: "sales", label: "Sales" },
  { value: "leads", label: "Leads" },
  { value: "traffic", label: "Traffic" },
  { value: "awareness", label: "Awareness" },
];

const tones = ["Direct and human", "Friendly", "Premium", "Playful", "Technical", "Bold"];

const emptyBrief: AdsBrief = {
  title: "",
  product: "",
  audience: "",
  problem: "",
  proof: "",
  price: "",
  url: "",
  objective: "sales",
  tone: "Direct and human",
  extra: "",
  platforms: ["meta", "tiktok", "youtube"],
};

export function AdsStudio() {
  const { push } = useToast();
  const [brief, setBrief] = useState<AdsBrief>(emptyBrief);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [campaign, setCampaign] = useState<AdCampaign | null>(null);
  const [history, setHistory] = useState<AdCampaign[]>([]);
  const [platformTab, setPlatformTab] = useState("meta");
  const [agents, setAgents] = useState<Agent[]>([]);

  useEffect(() => {
    api
      .adCampaigns()
      .then((result) => setHistory(result.campaigns))
      .catch(() => setHistory([]));
    api
      .agents()
      .then((result) => setAgents(result.agents))
      .catch(() => setAgents([]));
  }, []);

  function update<K extends keyof AdsBrief>(key: K, value: AdsBrief[K]) {
    setBrief((current) => ({ ...current, [key]: value }));
  }

  function togglePlatform(key: string) {
    setBrief((current) => {
      const has = current.platforms.includes(key);
      const next = has ? current.platforms.filter((item) => item !== key) : [...current.platforms, key];
      return { ...current, platforms: next.length ? next : [key] };
    });
  }

  async function generate(event: React.FormEvent) {
    event.preventDefault();
    if (brief.product.trim() === "" || brief.audience.trim() === "") {
      setError("Tell us what you sell and who it is for.");
      return;
    }

    setBusy(true);
    setError(null);
    setCampaign(null);

    try {
      const result = await api.generateAds(brief);
      setCampaign(result.campaign);
      setHistory((current) => [result.campaign, ...current].slice(0, 20));
      setPlatformTab(brief.platforms[0] ?? "meta");
      push("Campaign ready.");
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError("Add a model key in Settings → AI models, then try again.");
      } else {
        setError(err instanceof ApiError ? err.message : "We couldn't write that campaign.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function openCampaign(id: string) {
    try {
      const result = await api.adCampaign(id);
      setCampaign(result.campaign);
      setBrief(result.campaign.brief);
      setPlatformTab((result.campaign.brief.platforms ?? ["meta"])[0]);
      setError(null);
    } catch {
      push("We couldn't load that campaign.", "error");
    }
  }

  async function createTask() {
    if (!campaign) return;

    const marketing =
      agents.find((agent) => agent.role === "Marketing" || agent.role === "Audience" || agent.role === "Editorial") ??
      agents[0];

    try {
      const task = await api.createTask({
        title: "Produce ad campaign: " + campaign.title,
        details: toMarkdown(campaign.result),
        agentId: marketing?.id ?? "",
      });
      push("Campaign sent to " + (marketing?.name ?? "the team") + ".");
      navigate(`/tasks/${task.task.id}`);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't create that task.", "error");
    }
  }

  function exportCSV() {
    if (!campaign) return;
    const rows: string[][] = [
      ["platform", "concept", "hook", "primary_text", "headline", "description", "cta", "caption", "script"],
    ];

    for (const key of ["meta", "tiktok", "youtube"] as const) {
      for (const concept of campaign.result.platforms[key]?.concepts ?? []) {
        const script = (concept.script ?? [])
          .map((beat) => `${beat.time}: ${beat.line ?? beat.beat ?? ""} ${beat.visual ?? ""}`.trim())
          .join(" | ");
        rows.push([
          key,
          concept.name,
          concept.hook,
          concept.primary_text ?? "",
          concept.headline ?? "",
          concept.description ?? "",
          concept.cta ?? "",
          concept.caption ?? "",
          script || (concept.script_beats ?? []).join(" | "),
        ]);
      }
    }

    const csv = rows
      .map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(","))
      .join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = "crew-ads-" + campaign.title.toLowerCase().replace(/[^a-z0-9]+/g, "-") + ".csv";
    link.click();
    URL.revokeObjectURL(link.href);
  }

  return (
    <>
      <AdAccountsPanel />
      <div className="marketing-layout">
      <form className="card" onSubmit={generate} style={{ maxWidth: 560 }}>
        <h2 className="section-title mb-8">New ad campaign</h2>
        <p className="small muted mb-16">
          One brief becomes platform-native Meta, TikTok and YouTube concepts, scripts and a test plan. Everything is
          grounded in what you type here — no invented proof.
        </p>

        <Field label="What are you selling?" htmlFor="ads-product">
          <Textarea
            id="ads-product"
            rows={3}
            placeholder="e.g. A bookkeeping service for small clinics. We reconcile Xero and chase receipts."
            value={brief.product}
            onChange={(event) => update("product", event.target.value)}
          />
        </Field>

        <Field label="Who is it for?" htmlFor="ads-audience">
          <Textarea
            id="ads-audience"
            rows={2}
            placeholder="e.g. Clinic owners with 2–15 staff who do their own books at night."
            value={brief.audience}
            onChange={(event) => update("audience", event.target.value)}
          />
        </Field>

        <Field label="What problem does it solve?" hint="Optional, but it sharpens the hook." htmlFor="ads-problem">
          <Textarea
            id="ads-problem"
            rows={2}
            placeholder="e.g. They lose weekends to receipts and still miss deductions."
            value={brief.problem}
            onChange={(event) => update("problem", event.target.value)}
          />
        </Field>

        <Field
          label="Proof you can defend"
          hint="Real reviews, numbers, awards or demo facts. Leave blank rather than inventing."
          htmlFor="ads-proof"
        >
          <Textarea
            id="ads-proof"
            rows={2}
            placeholder="e.g. 42 clinics, average 6 hours saved a month, Xero certified."
            value={brief.proof}
            onChange={(event) => update("proof", event.target.value)}
          />
        </Field>

        <div className="form-grid">
          <Field label="Price or offer" htmlFor="ads-price">
            <Input
              id="ads-price"
              placeholder="e.g. from $249/month"
              value={brief.price}
              onChange={(event) => update("price", event.target.value)}
            />
          </Field>
          <Field label="Destination URL" htmlFor="ads-url">
            <Input
              id="ads-url"
              placeholder="https://…"
              value={brief.url}
              onChange={(event) => update("url", event.target.value)}
            />
          </Field>
          <Field label="Objective" htmlFor="ads-objective">
            <Select
              id="ads-objective"
              value={brief.objective}
              onChange={(event) => update("objective", event.target.value)}
            >
              {objectives.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Tone" htmlFor="ads-tone">
            <Select id="ads-tone" value={brief.tone} onChange={(event) => update("tone", event.target.value)}>
              {tones.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field label="Platforms" htmlFor="ads-platforms">
          <div className="ad-platforms" id="ads-platforms">
            {platforms.map((item) => (
              <label className={`ad-platform ${brief.platforms.includes(item.key) ? "on" : ""}`} key={item.key}>
                <input
                  type="checkbox"
                  checked={brief.platforms.includes(item.key)}
                  onChange={() => togglePlatform(item.key)}
                />
                <span>
                  <strong>{item.label}</strong>
                  <small>{item.hint}</small>
                </span>
              </label>
            ))}
          </div>
        </Field>

        <Field label="Anything else?" hint="Optional. Competitors, must-say lines, things to avoid." htmlFor="ads-extra">
          <Textarea
            id="ads-extra"
            rows={2}
            value={brief.extra}
            onChange={(event) => update("extra", event.target.value)}
          />
        </Field>

        {error ? <p className="form-error">{error}</p> : null}

        <div className="row-between" style={{ gap: 10 }}>
          <Button type="submit" variant="primary" loading={busy}>
            {busy ? "Writing the campaign…" : "Generate campaign"}
          </Button>
          <span className="tiny muted">{busy ? "30–90 seconds" : "3 platforms · 9 concepts"}</span>
        </div>

        {history.length > 0 ? (
          <div className="mt-24">
            <p className="tiny muted mb-8">Recent campaigns</p>
            <div className="ad-history">
              {history.slice(0, 6).map((item) => (
                <button key={item.id} className="link-button small" onClick={() => openCampaign(item.id)}>
                  {item.title}
                </button>
              ))}
            </div>
          </div>
        ) : null}
      </form>

      <div style={{ minWidth: 0 }}>
        {campaign ? (
          <CampaignView
            campaign={campaign}
            platformTab={platformTab}
            onPlatform={setPlatformTab}
            onCreateTask={createTask}
            onExport={exportCSV}
          />
        ) : (
          <div className="card" style={{ minHeight: 360, display: "grid", placeItems: "center" }}>
            <div style={{ textAlign: "center", maxWidth: 420 }}>
              <div style={{ fontSize: 30 }} aria-hidden="true">
                ◈
              </div>
              <h2 className="section-title mt-8">The campaign appears here</h2>
              <p className="small muted">
                Three materially different concepts per platform, hooks, timed scripts, copy fields, a test plan and a
                compliance checklist.
              </p>
            </div>
          </div>
        )}
      </div>
      </div>
    </>
  );
}

function AdAccountsPanel() {
  const { push } = useToast();
  const [accounts, setAccounts] = useState<AdAccount[] | null>(null);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [budget, setBudget] = useState("20");
  const [budgets, setBudgets] = useState<Record<string, string>>({});
  const [delivery, setDelivery] = useState<{ platform: string; campaign: { id: string; name: string } } | null>(null);

  function load() {
    api.adAccounts()
      .then((result) => setAccounts(result.accounts))
      .catch(() => setAccounts([]));
  }

  useEffect(load, []);

  async function create(platform: string) {
    if (name.trim() === "" || Number(budget) <= 0) return;
    setBusy(platform + ":create");
    try {
      await api.createAdCampaign(platform, name.trim(), Number(budget));
      push("Paused campaign created. It will not spend until you enable it.");
      setName("");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "The campaign wasn't created.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function toggle(platform: string, id: string, status: string) {
    const active = status === "ACTIVE" || status === "ENABLE" || status === "ENABLED";
    const next = active ? "PAUSED" : "ACTIVE";
    if (next === "ACTIVE" && !window.confirm("Enable this campaign? It can start spending immediately.")) return;
    setBusy(platform + ":" + id);
    try {
      await api.setAdCampaignStatus(platform, id, next);
      push(next === "ACTIVE" ? "Campaign enabled." : "Campaign paused.");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't change the status.", "error");
    } finally {
      setBusy(null);
    }
  }

  async function changeBudget(platform: string, id: string) {
    const value = Number(budgets[platform + id] ?? "");
    if (!value || value <= 0) return;
    setBusy(platform + ":" + id + ":budget");
    try {
      await api.setAdCampaignBudget(platform, id, value);
      push("Daily budget updated.");
      load();
    } catch (err) {
      push(err instanceof ApiError ? err.message : "We couldn't change the budget.", "error");
    } finally {
      setBusy(null);
    }
  }

  const connected = (accounts ?? []).filter((account) => account.configured);

  return (
    <section className="card mb-16">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <h2 className="section-title" style={{ marginBottom: 4 }}>Live ad accounts</h2>
          <p className="small muted" style={{ margin: 0 }}>
            Read campaigns, pause or enable them, change daily budgets, or create a campaign paused.
          </p>
        </div>
        <Button size="sm" onClick={() => setOpen((value) => !value)}>
          {open ? "Hide" : "Manage"}
        </Button>
      </div>

      {accounts === null ? <Skeleton lines={2} /> : null}
      {accounts && connected.length === 0 ? (
        <div className="setup-callout" style={{ marginTop: 10, marginBottom: 0 }}>
          <div>
            <strong>Connect an ad account to publish and control spend</strong>
            <p className="small muted" style={{ margin: "4px 0 0" }}>
              Add the API key and account ID in Settings → Integrations. Meta, TikTok and Google links below.
            </p>
          </div>
          <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
            <a className="button button-small" href="https://developers.facebook.com/apps/" target="_blank" rel="noreferrer noopener">Meta key ↗</a>
            <a className="button button-small" href="https://developers.tiktok.com/" target="_blank" rel="noreferrer noopener">TikTok key ↗</a>
            <a className="button button-small" href="https://developers.google.com/google-ads/api/docs/get-started/dev-token" target="_blank" rel="noreferrer noopener">Google key ↗</a>
            <a className="button button-small button-primary" href="/settings">Add key</a>
          </div>
        </div>
      ) : null}

      {open && connected.length > 0 ? (
        <div className="stack mt-16">
          {connected.map((account) => (
            <div className="ad-account" key={account.platform}>
              <div className="row-between">
                <strong style={{ textTransform: "capitalize" }}>{account.platform}</strong>
                <span className="row tiny muted" style={{ gap: 8 }}>
                  {account.campaigns?.length ?? 0} campaigns
                  <a
                    href={account.platform === "meta" ? "https://adsmanager.facebook.com/" : account.platform === "tiktok" ? "https://ads.tiktok.com/" : "https://ads.google.com/"}
                    target="_blank"
                    rel="noreferrer noopener"
                  >
                    Ads Manager ↗
                  </a>
                </span>
              </div>
              {account.error ? <p className="form-error">{account.error}</p> : null}
              {(account.campaigns ?? []).map((campaign) => (
                <div className="ad-account-row" key={campaign.id}>
                  <span style={{ minWidth: 0 }}>
                    <strong className="advise-file-name">{campaign.name}</strong>
                    <span className="tiny muted">{campaign.status} · {campaign.id}</span>
                  </span>
                  <span className="tiny">{campaign.budget ? `${Math.round(campaign.budget)} / day` : "budget n/a"}</span>
                  <Button
                    size="sm"
                    variant="quiet"
                    loading={busy === account.platform + ":" + campaign.id}
                    onClick={() => toggle(account.platform, campaign.id, campaign.status)}
                  >
                    {campaign.status === "ACTIVE" || campaign.status === "ENABLE" || campaign.status === "ENABLED" ? "Pause" : "Enable"}
                  </Button>
                  {account.platform === "google" ? (
                    <span className="tiny muted">budget via Google Ads</span>
                  ) : (
                    <div className="row" style={{ gap: 4, flexWrap: "wrap" }}>
                      <input
                        className="input"
                        style={{ width: 76, minHeight: 30, padding: "4px 7px" }}
                        placeholder="Budget"
                        value={budgets[account.platform + campaign.id] ?? ""}
                        onChange={(event) =>
                          setBudgets((current) => ({ ...current, [account.platform + campaign.id]: event.target.value }))
                        }
                      />
                      <Button
                        size="sm"
                        loading={busy === account.platform + ":" + campaign.id + ":budget"}
                        onClick={() => changeBudget(account.platform, campaign.id)}
                      >
                        Set
                      </Button>
                      {account.platform === "meta" ? (
                        <Button size="sm" variant="quiet" onClick={() => setDelivery({ platform: account.platform, campaign })}>
                          Delivery
                        </Button>
                      ) : null}
                    </div>
                  )}
                </div>
              ))}

              <div className="row mt-8" style={{ flexWrap: "wrap" }}>
                <Input
                  style={{ flex: 1, minWidth: 160 }}
                  placeholder="New campaign name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
                <Input
                  style={{ width: 90 }}
                  type="number"
                  min="1"
                  placeholder="Daily"
                  value={budget}
                  onChange={(event) => setBudget(event.target.value)}
                />
                <Button
                  size="sm"
                  loading={busy === account.platform + ":create"}
                  onClick={() => create(account.platform)}
                >
                  Create paused
                </Button>
              </div>
            </div>
          ))}
        </div>
      ) : null}

      {delivery ? (
        <MetaDeliveryModal
          campaign={delivery.campaign}
          onClose={() => setDelivery(null)}
          onDone={() => {
            setDelivery(null);
            load();
          }}
        />
      ) : null}
    </section>
  );
}

function MetaDeliveryModal({
  campaign,
  onClose,
  onDone,
}: {
  campaign: { id: string; name: string };
  onClose: () => void;
  onDone: () => void;
}) {
  const { push } = useToast();
  const [budget, setBudget] = useState("20");
  const [headline, setHeadline] = useState("");
  const [primary, setPrimary] = useState("");
  const [link, setLink] = useState("");
  const [image, setImage] = useState("");
  const [country, setCountry] = useState("NG");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function publish() {
    setBusy(true);
    setError(null);
    try {
      const result = await api.createMetaDelivery(campaign.id, {
        budget: Number(budget),
        headline,
        primaryText: primary,
        link,
        imageUrl: image,
        country,
      });
      push(`Ad set, creative and ad created paused (${result.ad_id}).`);
      onDone();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We couldn't build the ad.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      title={"Build delivery for " + campaign.name}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>Cancel</Button>
          <Button variant="primary" onClick={publish} loading={busy}>Create paused ad</Button>
        </>
      }
    >
      <p className="small muted mb-16">
        Creates an ad set, creative and ad in Meta, all paused. Add the Page ID, landing URL and image URL
        in Settings → Integrations → Meta Ads if you want defaults.
      </p>
      <div className="row tiny" style={{ gap: 10, marginBottom: 12 }}>
        <a href="https://developers.facebook.com/docs/pages/access-tokens" target="_blank" rel="noreferrer noopener">Get a Page ID ↗</a>
        <a href="https://developers.facebook.com/apps/" target="_blank" rel="noreferrer noopener">Get a Meta key ↗</a>
      </div>
      <div className="form-grid">
        <Field label="Daily budget" htmlFor="delivery-budget">
          <Input id="delivery-budget" type="number" min="1" value={budget} onChange={(event) => setBudget(event.target.value)} />
        </Field>
        <Field label="Country code" htmlFor="delivery-country">
          <Input id="delivery-country" maxLength={2} value={country} onChange={(event) => setCountry(event.target.value.toUpperCase())} />
        </Field>
        <Field label="Headline" htmlFor="delivery-headline">
          <Input id="delivery-headline" value={headline} onChange={(event) => setHeadline(event.target.value)} />
        </Field>
      </div>
      <Field label="Primary text" htmlFor="delivery-primary">
        <Textarea id="delivery-primary" rows={3} value={primary} onChange={(event) => setPrimary(event.target.value)} />
      </Field>
      <Field label="Landing link" htmlFor="delivery-link">
        <Input id="delivery-link" value={link} onChange={(event) => setLink(event.target.value)} placeholder="https://…" />
      </Field>
      <Field label="Image URL" htmlFor="delivery-image">
        <Input id="delivery-image" value={image} onChange={(event) => setImage(event.target.value)} placeholder="https://…/ad.jpg" />
      </Field>
      {error ? <p className="form-error">{error}</p> : null}
    </Modal>
  );
}

function CampaignView({
  campaign,
  platformTab,
  onPlatform,
  onCreateTask,
  onExport,
}: {
  campaign: AdCampaign;
  platformTab: string;
  onPlatform: (value: string) => void;
  onCreateTask: () => void;
  onExport: () => void;
}) {
  const { push } = useToast();
  const result = campaign.result;
  const requested = campaign.brief?.platforms ?? ["meta", "tiktok", "youtube"];
  const platformBlock = (result.platforms ?? {})[platformTab as "meta" | "tiktok" | "youtube"];

  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      push("Copied.");
    } catch {
      push("Copy failed. Select the text instead.", "error");
    }
  }

  return (
    <article className="card">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div style={{ minWidth: 0 }}>
          <h2 className="section-title" style={{ marginBottom: 4 }}>
            {result.campaign_name || campaign.title}
          </h2>
          <p className="tiny muted">
            {campaign.model} · {money(campaign.cost_usd)} ·{" "}
            {new Date(campaign.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric" })}
          </p>
        </div>
        <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
          <Button size="sm" onClick={onExport}>
            Export CSV
          </Button>
          <Button size="sm" variant="primary" onClick={onCreateTask}>
            Send to helper
          </Button>
        </div>
      </div>

      <p className="mt-16" style={{ fontSize: 15, lineHeight: 1.6 }}>
        {result.big_idea}
      </p>

      <div className="ad-strategy mt-16">
        <div>
          <span className="ad-label">Audience</span>
          <p>{result.strategy?.audience_summary}</p>
        </div>
        <div>
          <span className="ad-label">Offer</span>
          <p>{result.strategy?.offer}</p>
        </div>
        <div>
          <span className="ad-label">Positioning</span>
          <p>{result.strategy?.positioning}</p>
        </div>
      </div>

      {result.scores ? <ScoreBars scores={result.scores} /> : null}

      <div className="tabs mt-24" role="group" aria-label="Ad platform">
        {requested.map((key) => (
          <button
            key={key}
            className="tab"
            aria-pressed={platformTab === key}
            onClick={() => onPlatform(key)}
          >
            {platforms.find((item) => item.key === key)?.label ?? key}
          </button>
        ))}
      </div>

      {(platformBlock?.concepts ?? []).map((concept, index) => (
        <ConceptCard
          key={`${platformTab}-${index}`}
          platform={platformTab}
          concept={concept}
          onCopy={copy}
        />
      ))}

      {(result.hooks ?? []).length > 0 ? (
        <section className="ad-section">
          <h3 className="section-title mb-8">Hooks to test</h3>
          <div className="ad-hooks">
            {result.hooks.map((hook, index) => (
              <button key={index} className="ad-hook" onClick={() => copy(hook.text)}>
                <span className="ad-hook-text">{hook.text}</span>
                <span className="tiny muted">
                  {hook.category} · {hook.platform}
                </span>
              </button>
            ))}
          </div>
        </section>
      ) : null}

      {(result.testing_plan ?? []).length > 0 ? (
        <section className="ad-section">
          <h3 className="section-title mb-8">Test plan</h3>
          <div className="ad-tests">
            {result.testing_plan.map((test) => (
              <div className="ad-test" key={test.priority}>
                <div className="row-between">
                  <strong>
                    #{test.priority} · {test.element}
                  </strong>
                </div>
                <p><span className="ad-label">A</span> {test.a}</p>
                <p><span className="ad-label">B</span> {test.b}</p>
                <p className="tiny muted">{test.hypothesis}</p>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <section className="ad-section">
        <h3 className="section-title mb-8">Before you spend</h3>
        <div className="ad-compliance">
          <div>
            <span className="ad-label">Claims to verify</span>
            <ul>{result.compliance?.claims_to_verify?.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
          <div>
            <span className="ad-label">Avoid</span>
            <ul>{result.compliance?.avoid?.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
          <div>
            <span className="ad-label">Spec checks</span>
            <ul>{result.compliance?.spec_checks?.map((item, i) => <li key={i}>{item}</li>)}</ul>
          </div>
        </div>
      </section>

      {(result.budget_split ?? []).length > 0 ? (
        <section className="ad-section">
          <h3 className="section-title mb-8">Suggested budget split</h3>
          <div className="ad-budget">
            {result.budget_split.map((item) => (
              <div key={item.platform}>
                <div className="row-between">
                  <strong style={{ textTransform: "capitalize" }}>{item.platform}</strong>
                  <span>{item.percent}%</span>
                </div>
                <div className="ad-bar">
                  <span style={{ width: `${Math.min(100, item.percent)}%` }} />
                </div>
                <p className="tiny muted">{item.why}</p>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <p className="tiny muted mt-24">
        Built from the MIT-licensed Claude Ads, AI Ads Strategist and Creatify ad skills. Specs change by placement —
        verify in Ads Manager before production. No performance is guaranteed.
      </p>
    </article>
  );
}

function ConceptCard({
  platform,
  concept,
  onCopy,
}: {
  platform: string;
  concept: AdConcept;
  onCopy: (text: string) => void;
}) {
  return (
    <div className="ad-concept">
      <div className="row-between" style={{ alignItems: "flex-start" }}>
        <div>
          <span className="ad-label">{platform}</span>
          <h3 style={{ fontSize: 16, fontWeight: 650 }}>{concept.name}</h3>
          <p className="small muted">{concept.angle}</p>
        </div>
        <Button size="sm" onClick={() => onCopy(conceptText(platform, concept))}>
          Copy concept
        </Button>
      </div>

      <div className="ad-hook-box">
        <span className="ad-label">Hook</span>
        <p>{concept.hook}</p>
      </div>

      {platform === "meta" ? (
        <>
          <CopyField label="Primary text" value={concept.primary_text ?? ""} onCopy={onCopy} />
          <div className="form-grid">
            <CopyField label="Headline" value={concept.headline ?? ""} onCopy={onCopy} />
            <CopyField label="Description" value={concept.description ?? ""} onCopy={onCopy} />
          </div>
          <div className="ad-meta-row">
            <span><strong>CTA:</strong> {concept.cta}</span>
            <span><strong>Format:</strong> {concept.format}</span>
            <span><strong>Placement:</strong> {concept.placement}</span>
          </div>
          <CopyField label="Visual direction" value={concept.visual ?? ""} onCopy={onCopy} />
          {concept.script_beats?.length ? (
            <div>
              <span className="ad-label">Script beats</span>
              <ol className="ad-beats">
                {concept.script_beats.map((beat, index) => (
                  <li key={index}>{beat}</li>
                ))}
              </ol>
            </div>
          ) : null}
        </>
      ) : null}

      {platform === "tiktok" ? (
        <>
          {concept.script?.length ? <ScriptTable beats={concept.script} /> : null}
          <CopyField label="Caption" value={concept.caption ?? ""} onCopy={onCopy} />
          <div className="ad-meta-row">
            <span><strong>CTA:</strong> {concept.cta}</span>
            <span><strong>Sound:</strong> {concept.sound}</span>
          </div>
          <div>
            <span className="ad-label">Text overlays</span>
            <div className="chip-row">
              {(concept.text_overlays ?? []).map((item, index) => (
                <span className="chip" key={index}>{item}</span>
              ))}
            </div>
          </div>
          <div>
            <span className="ad-label">Hashtags</span>
            <p className="small muted">{(concept.hashtags ?? []).join(" ")}</p>
          </div>
          <CopyField label="Visual direction" value={concept.visual ?? ""} onCopy={onCopy} />
        </>
      ) : null}

      {platform === "youtube" ? (
        <>
          {concept.script?.length ? <ScriptTable beats={concept.script} /> : null}
          <CopyField label="Voiceover" value={concept.voiceover ?? ""} onCopy={onCopy} />
          <div>
            <span className="ad-label">On-screen text</span>
            <div className="chip-row">
              {(concept.on_screen_text ?? []).map((item, index) => (
                <span className="chip" key={index}>{item}</span>
              ))}
            </div>
          </div>
          <div className="ad-meta-row">
            <span><strong>CTA:</strong> {concept.cta}</span>
            <span><strong>End card:</strong> {concept.end_card}</span>
          </div>
          <CopyField label="Thumbnail direction" value={concept.thumbnail ?? ""} onCopy={onCopy} />
        </>
      ) : null}
    </div>
  );
}

function ScriptTable({ beats }: { beats: NonNullable<AdConcept["script"]> }) {
  return (
    <div className="ad-script">
      {beats.map((beat, index) => (
        <div className="ad-script-row" key={index}>
          <span className="ad-time">{beat.time}</span>
          <span>{beat.line ?? beat.beat}</span>
          {beat.visual ? <span className="tiny muted">{beat.visual}</span> : null}
        </div>
      ))}
    </div>
  );
}

function CopyField({ label, value, onCopy }: { label: string; value: string; onCopy: (text: string) => void }) {
  if (!value) return null;
  return (
    <div className="copy-field">
      <div className="row-between">
        <span className="ad-label">{label}</span>
        <button className="link-button tiny" onClick={() => onCopy(value)}>
          Copy
        </button>
      </div>
      <p>{value}</p>
    </div>
  );
}

function ScoreBars({ scores }: { scores: Record<string, number> }) {
  const labels: Record<string, string> = {
    hook: "Hook",
    clarity: "Clarity",
    platform_fit: "Platform fit",
    distinctiveness: "Distinctiveness",
    proof: "Proof",
    cta: "CTA",
    compliance: "Compliance",
    overall: "Overall",
  };

  return (
    <div className="ad-scores mt-16">
      {Object.entries(labels).map(([key, label]) =>
        scores[key] === undefined ? null : (
          <div key={key} className="ad-score">
            <div className="row-between">
              <span className="tiny muted">{label}</span>
              <span className="tiny">{scores[key]}/10</span>
            </div>
            <div className="ad-bar">
              <span style={{ width: `${Math.min(100, scores[key] * 10)}%` }} />
            </div>
          </div>
        ),
      )}
    </div>
  );
}

function conceptText(platform: string, concept: AdConcept): string {
  const lines = [concept.name, "", "Hook: " + concept.hook, ""];

  if (platform === "meta") {
    lines.push("Primary text:", concept.primary_text ?? "", "", "Headline: " + (concept.headline ?? ""));
    lines.push("Description: " + (concept.description ?? ""), "CTA: " + (concept.cta ?? ""));
    if (concept.script_beats?.length) lines.push("", "Beats:", ...concept.script_beats.map((beat) => "- " + beat));
  }

  if (platform === "tiktok") {
    lines.push("Script:");
    for (const beat of concept.script ?? []) lines.push(`${beat.time} ${beat.line ?? beat.beat ?? ""}`);
    lines.push("", "Caption: " + (concept.caption ?? ""), "Hashtags: " + (concept.hashtags ?? []).join(" "));
    lines.push("CTA: " + (concept.cta ?? ""), "Sound: " + (concept.sound ?? ""));
  }

  if (platform === "youtube") {
    lines.push("Script:");
    for (const beat of concept.script ?? []) lines.push(`${beat.time} ${beat.line ?? beat.beat ?? ""}`);
    lines.push("", "Voiceover: " + (concept.voiceover ?? ""), "CTA: " + (concept.cta ?? ""));
    lines.push("End card: " + (concept.end_card ?? ""));
  }

  return lines.join("\n");
}

function toMarkdown(result: AdResult): string {
  const lines: string[] = [
    "# " + result.campaign_name,
    "",
    result.big_idea,
    "",
    "## Strategy",
    "- Objective: " + result.strategy?.objective,
    "- Audience: " + result.strategy?.audience_summary,
    "- Offer: " + result.strategy?.offer,
    "- Positioning: " + result.strategy?.positioning,
    "",
  ];

  const platformMap = result.platforms ?? {};
  for (const key of ["meta", "tiktok", "youtube"] as const) {
    const block = platformMap[key];
    if (!block) continue;
    lines.push("## " + key, "");
    for (const concept of block.concepts) {
      lines.push("### " + concept.name, concept.hook, "", conceptText(key, concept), "");
    }
  }

  lines.push("## Test plan", "");
  for (const test of result.testing_plan ?? []) {
    lines.push(`- ${test.element}: A) ${test.a} / B) ${test.b} — ${test.hypothesis}`);
  }

  lines.push("", "## Compliance", "");
  for (const item of result.compliance?.claims_to_verify ?? []) lines.push("- Verify: " + item);
  for (const item of result.compliance?.avoid ?? []) lines.push("- Avoid: " + item);
  for (const item of result.compliance?.spec_checks ?? []) lines.push("- Spec: " + item);

  return lines.join("\n");
}
