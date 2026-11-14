export type Company = {
  id: string;
  name: string;
  business_key: string;
  website: string;
  support_email: string;
  description: string;
  audience: string;
  tone: string;
  monthly_task_limit: number;
  monthly_budget_usd: number;
  spend_usd: number;
  country: string;
  ticker: string;
  parent_company_id: string;
  is_product: boolean;
  role: "owner" | "editor" | "viewer";
  created_at: string;
  agent_count: number;
  task_count: number;
};

export type Goal = {
  id: string;
  title: string;
  target: number;
  current: number;
  due_date: string;
  created_at: string;
  completed_at: string | null;
};

export type Routine = {
  id: string;
  title: string;
  details: string;
  agent_id: string;
  agent: Agent | null;
  cadence: "daily" | "weekdays" | "weekly" | "monthly";
  enabled: boolean;
  next_run_at: string;
  last_run_at: string | null;
  created_at: string;
};

export type BusinessPreset = {
  key: string;
  name: string;
  category: string;
  emoji: string;
  blurb: string;
  agents: number;
  tasks: number;
  integrations: string[];
};

export type PresetDetail = {
  key: string;
  name: string;
  category: string;
  emoji: string;
  blurb: string;
  agents: { name: string; role: string; emoji: string }[];
  tasks: { title: string; agent: string }[];
  integrations: string[];
};

export type Skill = {
  key: string;
  name: string;
  description: string;
  category: string;
  chars: number;
  source: string;
  source_url: string;
};

export type Playbook = {
  key: string;
  name: string;
  summary: string;
  file: string;
};

export type MarketingReport = {
  ok: boolean;
  report: string;
  model: string;
  cost?: number;
  evidence: { url: string; title: string; description: string; characters: number; truncated: boolean };
  task?: Task;
};

export type Member = {
  user_id: string;
  name: string;
  email: string;
  country: string;
  ticker: string;
  parent_company_id: string;
  is_product: boolean;
  role: "owner" | "editor" | "viewer";
  is_owner: boolean;
};

export type Invite = { id: string; email: string; role: string; created_at: string };

export type AdviceCheck = { label: string; done: boolean; evidence: string; action: string };

export type AdviceAgent = {
  name: string;
  role: string;
  emoji: string;
  instructions: string;
  skills: string[];
};

export type AdviceSection = {
  key: string;
  title: string;
  summary: string;
  status: "done" | "partial" | "todo";
  progress: number;
  checks: AdviceCheck[];
  actions: string[];
  agents: AdviceAgent[];
  country_note: string;
};

export type CountryProfile = {
  name: string;
  currency: string;
  registration: string[];
  tax: string[];
  payments: string[];
  ads: string[];
  data: string[];
  notes: string[];
};

export type AdviceReport = {
  company_name: string;
  country: CountryProfile;
  stage: string;
  stage_label: string;
  score: number;
  done: number;
  total: number;
  sections: AdviceSection[];
  next: string[];
  has_model: boolean;
};

export type CompanyAdvice = {
  analysis: {
    headline: string;
    constraints: { title: string; why: string; fix: string }[];
    next_7_days: string[];
    opportunities: string[];
    risks: string[];
  };
  model: string;
  created_at: string;
};

export type ChatSource = { title: string; url: string };

export type CompanyFile = {
  id: string;
  name: string;
  mime: string;
  size: number;
  has_text: boolean;
  created_at: string;
};

export type FinancialSnapshot = {
  id: string;
  period: string;
  revenue: number;
  cogs: number;
  opex: number;
  cash: number;
  receivables: number;
  payables: number;
  source: string;
  audited: boolean;
  audit_status: "self_reported" | "evidence_attached" | "discrepancies" | "verified";
  audit_note: string;
  auditor: string;
  audit_file_id: string;
  created_at: string;
};

export type FinancialModel = {
  period: string;
  revenue: number;
  cogs: number;
  gross_profit: number;
  gross_margin: number;
  opex: number;
  net_profit: number;
  cash: number;
  receivables: number;
  payables: number;
  runway_months: number;
  break_even_sales: number;
  annual_revenue: number;
  quality: string;
  source: string;
};

export type PortfolioRow = {
  id: string;
  name: string;
  is_product: boolean;
  country: string;
  currency: string;
  original_currency: string;
  stage: string;
  score: number;
  revenue: number;
  net_profit: number;
  cash: number;
  tasks: number;
  agents: number;
  audit_state: string;
};

export type PortfolioTotal = {
  currency: string;
  revenue: number;
  net_profit: number;
  cash: number;
};

export type AdAccountCampaign = {
  id: string;
  name: string;
  status: string;
  budget: number;
  currency: string;
};

export type AdAccount = {
  platform: "meta" | "tiktok" | "google";
  configured: boolean;
  campaigns: AdAccountCampaign[];
  error?: string;
};

export type Integration = {
  kind: string;
  configured: boolean;
  details: Record<string, string>;
};

export type AdsBrief = {
  title: string;
  product: string;
  audience: string;
  problem: string;
  proof: string;
  price: string;
  url: string;
  objective: string;
  tone: string;
  extra: string;
  platforms: string[];
};

export type AdScriptBeat = { time: string; beat?: string; line?: string; visual?: string };

export type AdConcept = {
  name: string;
  angle: string;
  hook: string;
  primary_text?: string;
  headline?: string;
  description?: string;
  cta?: string;
  format?: string;
  placement?: string;
  visual?: string;
  script_beats?: string[];
  script?: AdScriptBeat[];
  caption?: string;
  hashtags?: string[];
  sound?: string;
  text_overlays?: string[];
  voiceover?: string;
  on_screen_text?: string[];
  end_card?: string;
  thumbnail?: string;
};

export type AdResult = {
  campaign_name: string;
  big_idea: string;
  strategy: {
    objective: string;
    audience_summary: string;
    offer: string;
    positioning: string;
    proof: string[];
    unknowns: string[];
  };
  personas: { name: string; who: string; pain: string; trigger: string; objection: string; platforms: string[] }[];
  angles: { name: string; insight: string; promise: string; proof: string; emotion: string }[];
  hooks: { text: string; category: string; platform: string; why: string }[];
  platforms: {
    meta?: { concepts: AdConcept[]; notes: string[] };
    tiktok?: { concepts: AdConcept[]; notes: string[] };
    youtube?: { concepts: AdConcept[]; notes: string[] };
  };
  testing_plan: { priority: number; element: string; a: string; b: string; hypothesis: string }[];
  budget_split: { platform: string; percent: number; why: string }[];
  compliance: { claims_to_verify: string[]; avoid: string[]; spec_checks: string[] };
  scores: Record<string, number>;
};

export type AdCampaign = {
  id: string;
  title: string;
  brief: AdsBrief;
  result: AdResult;
  model: string;
  cost_usd: number;
  created_at: string;
};

export type Announcement = {
  id: number;
  title: string;
  body: string;
  level: "info" | "success" | "warning";
  published_at: string;
};

export type Notification = {
  id: string;
  kind: string;
  title: string;
  body: string;
  task_id: string;
  read_at: string | null;
  created_at: string;
};

export type Provider = {
  key: string;
  name: string;
  kind: string;
  base_url: string;
  keys_url: string;
  docs_url: string;
  free: boolean;
  free_note: string;
  default_model: string;
  models: string[];
  configured: boolean;
  platform?: boolean;
  last4: string;
};

export type User = {
  id: string;
  email: string;
  name: string;
  created_at: string;
};

export type AgentStatus = "ready" | "working" | "offline" | "not_connected";

export type Agent = {
  id: string;
  name: string;
  role: string;
  emoji: string;
  instructions: string;
  skills: string[];
  monthly_budget_usd: number;
  can_delegate: boolean;
  model_provider: string;
  model_name: string;
  status: AgentStatus;
  open_tasks: number;
  working_tasks: number;
  last_seen_at: string | null;
  created_at: string;
};

export type Comment = {
  id: string;
  author_type: "you" | "helper" | "system";
  author_name: string;
  body: string;
  created_at: string;
};

export type TaskStatus = "todo" | "working" | "review" | "done";

export type Task = {
  id: string;
  title: string;
  details: string;
  status: TaskStatus;
  agent_id: string | null;
  agent: { id: string; name: string; emoji: string; role: string } | null;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
  comments?: Comment[];
};

export type ActivityItem = {
  id: number;
  kind: string;
  message: string;
  task_id?: string;
  agent_id?: string;
  created_at: string;
};

export type Session = {
  id: string;
  device: string;
  ip_address: string;
  last_used_at: string;
  created_at: string;
};

export type Dashboard = {
  ok: boolean;
  greeting: string;
  name: string;
  company: { id: string; name: string; business_key: string; website: string };
  counts: Record<TaskStatus, number>;
  needsYou: Task[];
  working: Task[];
  upNext: Task[];
  agents: Agent[];
  activity: ActivityItem[];
  goals: Goal[];
};

export class ApiError extends Error {
  status: number;
  retryAfter: number;

  constructor(message: string, status: number, retryAfter = 0) {
    super(message);
    this.status = status;
    this.retryAfter = retryAfter;
  }
}

let activeCompany = "";

export function setCompany(id: string) {
  activeCompany = id;
  if (id) {
    window.localStorage.setItem("crew:company", id);
  } else {
    window.localStorage.removeItem("crew:company");
  }
}

export function rememberedCompany(): string {
  return activeCompany || window.localStorage.getItem("crew:company") || "";
}

function readCookie(name: string): string {
  const prefix = `${name}=`;
  const parts = document.cookie.split("; ");
  for (const part of parts) {
    if (part.startsWith(prefix)) return decodeURIComponent(part.slice(prefix.length));
  }
  return "";
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  options: { silent?: boolean } = {},
): Promise<T> {
  const method = (init.method ?? "GET").toUpperCase();
  const headers = new Headers(init.headers);

  headers.set("Accept", "application/json");

  if (init.body) {
    headers.set("Content-Type", "application/json");
  }

  if (activeCompany) {
    headers.set("X-Crew-Company", activeCompany);
  }

  if (method !== "GET" && method !== "HEAD") {
    const csrf = readCookie("crew_csrf");
    if (!csrf) {
      await fetch("/api/csrf", { credentials: "same-origin" });
    }
    headers.set("X-CSRF-Token", readCookie("crew_csrf"));
  }

  const response = await fetch(path, {
    ...init,
    method,
    headers,
    credentials: "same-origin",
  });

  const text = await response.text();
  let payload: Record<string, unknown> = {};

  if (text) {
    try {
      payload = JSON.parse(text) as Record<string, unknown>;
    } catch {
      throw new ApiError("We couldn't read the server's reply. Please refresh and try again.", response.status);
    }
  }

  if (response.status === 401 && !options.silent) {
    // Let the app drop to the sign-in screen instead of rendering an error on
    // whichever page happened to be open when the session ended.
    window.dispatchEvent(new Event("crew:unauthorized"));
  }

  if (!response.ok) {
    throw new ApiError(
      typeof payload.error === "string" ? payload.error : "Something went wrong. Please try again.",
      response.status,
      typeof payload.retry_after === "number" ? payload.retry_after : 0,
    );
  }

  return payload as T;
}

export const api = {
  csrf: () => fetch("/api/csrf", { credentials: "same-origin" }),

  // The initial session probe must not look like an expiry, or a first-time
  // visitor would be told they were signed out.
  me: () => request<{ ok: boolean; user: User }>("/api/me", {}, { silent: true }),
  updateMe: (name: string) =>
    request<{ ok: boolean; user: User }>("/api/me", {
      method: "PATCH",
      body: JSON.stringify({ name }),
    }),
  deleteMe: () => request<{ ok: boolean }>("/api/me", { method: "DELETE" }),

  requestCode: (email: string) =>
    request<{ ok: boolean; requestId: string; resendAfter: number; message: string }>(
      "/api/auth/request",
      { method: "POST", body: JSON.stringify({ email }) },
    ),
  resendCode: (requestId: string) =>
    request<{ ok: boolean; resendAfter: number }>("/api/auth/resend", {
      method: "POST",
      body: JSON.stringify({ requestId }),
    }),
  verifyCode: (requestId: string, code: string) =>
    request<{ ok: boolean; user: User; isNew: boolean }>("/api/auth/verify", {
      method: "POST",
      body: JSON.stringify({ requestId, code }),
    }),
  logout: (reason: string) =>
    request<{ ok: boolean }>("/api/auth/logout", {
      method: "POST",
      body: JSON.stringify({ reason }),
    }),

  sessions: () => request<{ ok: boolean; sessions: Session[] }>("/api/me/sessions"),
  revokeSession: (id: string) =>
    request<{ ok: boolean }>(`/api/me/sessions/${id}`, { method: "DELETE" }),

  companies: () => request<{ ok: boolean; companies: Company[] }>("/api/companies"),
  createCompany: (input: Record<string, unknown>) =>
    request<{ ok: boolean; company: Company; agentsCreated: number; tasksCreated: number }>("/api/companies", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  company: (id: string) => request<{ ok: boolean; company: Company }>(`/api/companies/${id}`),
  updateCompany: (id: string, input: Record<string, unknown>) =>
    request<{ ok: boolean; company: Company }>(`/api/companies/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteCompany: (id: string) =>
    request<{ ok: boolean }>(`/api/companies/${id}`, { method: "DELETE" }),

  files: () => request<{ ok: boolean; files: CompanyFile[] }>("/api/files"),
  uploadFile: async (file: File) => {
    const headers = new Headers();
    headers.set("Accept", "application/json");
    if (activeCompany) headers.set("X-Crew-Company", activeCompany);
    const csrf = readCookie("crew_csrf");
    if (!csrf) await fetch("/api/csrf", { credentials: "same-origin" });
    headers.set("X-CSRF-Token", readCookie("crew_csrf"));
    const form = new FormData();
    form.append("file", file);
    const response = await fetch("/api/files", {
      method: "POST",
      body: form,
      headers,
      credentials: "same-origin",
    });
    const payload = (await response.json().catch(() => ({}))) as { file?: CompanyFile; error?: string };
    if (!response.ok) throw new ApiError(payload.error || "We couldn't upload that file.", response.status);
    return payload as { file: CompanyFile };
  },
  deleteFile: (id: string) => request<{ ok: boolean }>(`/api/files/${id}`, { method: "DELETE" }),
  downloadFile: async (id: string) => {
    const headers = new Headers();
    if (activeCompany) headers.set("X-Crew-Company", activeCompany);
    const response = await fetch(`/api/files/${id}/download`, { headers, credentials: "same-origin" });
    if (!response.ok) throw new ApiError("We couldn't download that file.", response.status);
    return response.blob();
  },
  financials: () =>
    request<{ ok: boolean; snapshots: FinancialSnapshot[]; model: FinancialModel | null }>("/api/financials"),
  saveFinancials: (input: {
    period: string;
    revenue: number;
    cogs: number;
    opex: number;
    cash: number;
    receivables: number;
    payables: number;
    source: string;
    audited: boolean;
  }) =>
    request<{ ok: boolean; snapshot: FinancialSnapshot; model: FinancialModel }>("/api/financials", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  deleteFinancials: (id: string) =>
    request<{ ok: boolean }>(`/api/financials/${id}`, { method: "DELETE" }),
  auditFinancials: (id: string, fileId: string, auditor: string) =>
    request<{ ok: boolean; snapshot: FinancialSnapshot; status: string; note: string }>(
      `/api/financials/${id}/audit`,
      { method: "POST", body: JSON.stringify({ fileId, auditor }) },
    ),
  integrations: () =>
    request<{ ok: boolean; integrations: Integration[]; email: { available: boolean; from: string } }>(
      "/api/integrations",
    ),
  saveIntegration: (kind: string, config: Record<string, string>) =>
    request<{ ok: boolean; integration: Integration }>(`/api/integrations/${kind}`, {
      method: "PUT",
      body: JSON.stringify(config),
    }),
  deleteIntegration: (kind: string) =>
    request<{ ok: boolean }>(`/api/integrations/${kind}`, { method: "DELETE" }),
  testIntegration: (kind: string) =>
    request<{ ok: boolean; detail: Record<string, unknown> }>(`/api/integrations/${kind}/test`, { method: "POST" }),
  integrationOverview: (kind: string) =>
    request<{ ok: boolean; overview: Record<string, unknown> }>(`/api/integrations/${kind}/overview`),
  sendEmail: (input: { to: string; subject: string; body: string }) =>
    request<{ ok: boolean }>("/api/send/email", { method: "POST", body: JSON.stringify(input) }),
  sendWhatsApp: (input: { to: string; message: string; template?: boolean }) =>
    request<{ ok: boolean }>("/api/send/whatsapp", { method: "POST", body: JSON.stringify(input) }),

  researchStatus: () =>
    request<{ ok: boolean; searxng: boolean; brave: boolean; search_url: string }>("/api/research/status"),

  adAccounts: () => request<{ ok: boolean; accounts: AdAccount[] }>("/api/adaccounts"),
  createAdCampaign: (platform: string, name: string, budget: number) =>
    request<{ ok: boolean; id: string }>(`/api/adaccounts/${platform}/campaigns`, {
      method: "POST",
      body: JSON.stringify({ name, budget }),
    }),
  setAdCampaignStatus: (platform: string, id: string, status: "PAUSED" | "ACTIVE") =>
    request<{ ok: boolean }>(`/api/adaccounts/${platform}/campaigns/${id}/status`, {
      method: "POST",
      body: JSON.stringify({ status }),
    }),
  createMetaDelivery: (
    campaignId: string,
    input: { budget: number; headline: string; primaryText: string; link: string; imageUrl: string; country: string },
  ) =>
    request<{ ok: boolean; ad_set_id: string; creative_id: string; ad_id: string }>(
      `/api/adaccounts/meta/campaigns/${campaignId}/delivery`,
      { method: "POST", body: JSON.stringify(input) },
    ),
  setAdCampaignBudget: (platform: string, id: string, budget: number) =>
    request<{ ok: boolean }>(`/api/adaccounts/${platform}/campaigns/${id}/budget`, {
      method: "POST",
      body: JSON.stringify({ budget }),
    }),

  portfolio: (base = "USD") =>
    request<{
      ok: boolean;
      parent: Company;
      products: Company[];
      rows: PortfolioRow[];
      totals: PortfolioTotal[];
      base: string;
      rates_available: boolean;
      fx_key: boolean;
    }>(`/api/portfolio?base=${encodeURIComponent(base)}`),
  products: () => request<{ ok: boolean; products: Company[] }>("/api/products"),
  createProduct: (input: { name: string; description: string; audience: string; website: string }) =>
    request<{ ok: boolean; product: Company }>("/api/products", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  deleteProduct: (id: string) =>
    request<{ ok: boolean }>(`/api/products/${id}`, { method: "DELETE" }),

  members: () =>
    request<{ ok: boolean; members: Member[]; invites: Invite[]; role: string }>("/api/members"),
  addMember: (email: string, role: string) =>
    request<{ ok: boolean }>("/api/members", { method: "POST", body: JSON.stringify({ email, role }) }),
  removeMember: (id: string) => request<{ ok: boolean }>(`/api/members/${id}`, { method: "DELETE" }),
  deleteInvite: (id: string) => request<{ ok: boolean }>(`/api/invites/${id}`, { method: "DELETE" }),

  skills: () =>
    request<{ ok: boolean; skills: Skill[]; categories: string[]; count: number; playbooks: Playbook[]; sources: { name: string; url: string; license: string }[] }>(
      "/api/skills",
    ),
  skill: (key: string) => request<{ ok: boolean; skill: Skill; body: string }>(`/api/skills/${key}`),
  goals: () => request<{ ok: boolean; goals: Goal[] }>("/api/goals"),
  createGoal: (input: { title: string; target: number; dueDate: string }) =>
    request<{ ok: boolean; goal: Goal }>("/api/goals", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateGoal: (id: string, input: Record<string, unknown>) =>
    request<{ ok: boolean; goal: Goal }>(`/api/goals/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteGoal: (id: string) => request<{ ok: boolean }>(`/api/goals/${id}`, { method: "DELETE" }),

  routines: () => request<{ ok: boolean; routines: Routine[] }>("/api/routines"),
  createRoutine: (input: { title: string; details: string; agentId: string; cadence: string }) =>
    request<{ ok: boolean; routine: Routine }>("/api/routines", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateRoutine: (id: string, input: Record<string, unknown>) =>
    request<{ ok: boolean; routine: Routine }>(`/api/routines/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteRoutine: (id: string) => request<{ ok: boolean }>(`/api/routines/${id}`, { method: "DELETE" }),

  analyzeMarketing: (input: { url: string; focus: string; playbook: string; agentId?: string }) =>
    request<MarketingReport>("/api/marketing/analyze", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  presets: () =>
    request<{ ok: boolean; businesses: BusinessPreset[]; categories: string[] }>("/api/presets"),
  preset: (key: string) => request<{ ok: boolean } & PresetDetail>(`/api/presets/${key}`),

  providers: () => request<{ ok: boolean; providers: Provider[] }>("/api/providers"),
  setProviderKey: (key: string, apiKey: string) =>
    request<{ ok: boolean; provider: string; configured: boolean; last4: string }>(`/api/providers/${key}`, {
      method: "PUT",
      body: JSON.stringify({ apiKey }),
    }),
  deleteProviderKey: (key: string) =>
    request<{ ok: boolean }>(`/api/providers/${key}`, { method: "DELETE" }),
  testProvider: (key: string) =>
    request<{ ok: boolean; message: string }>(`/api/providers/${key}/test`, { method: "POST" }),

  advice: () =>
    request<{ ok: boolean; report: AdviceReport; advice?: CompanyAdvice; company: Company }>("/api/advice"),
  studyCompany: () =>
    request<{ ok: boolean; advice: CompanyAdvice; cost: number }>("/api/advice/study", { method: "POST" }),
  adviceChat: (message: string, history: { role: string; content: string }[]) =>
    request<{ ok: boolean; answer: string; sources: ChatSource[]; cost: number }>("/api/advice/chat", {
      method: "POST",
      body: JSON.stringify({ message, history }),
    }),

  generateAds: (brief: AdsBrief) =>
    request<{ ok: boolean; campaign: AdCampaign }>("/api/ads/generate", {
      method: "POST",
      body: JSON.stringify(brief),
    }),
  adCampaigns: () => request<{ ok: boolean; campaigns: AdCampaign[] }>("/api/ads/campaigns"),
  adCampaign: (id: string) => request<{ ok: boolean; campaign: AdCampaign }>(`/api/ads/campaigns/${id}`),
  deleteAdCampaign: (id: string) =>
    request<{ ok: boolean }>(`/api/ads/campaigns/${id}`, { method: "DELETE" }),

  announcements: () => request<{ ok: boolean; announcements: Announcement[] }>("/api/announcements"),
  notifications: () =>
    request<{ ok: boolean; notifications: Notification[]; unread: number }>("/api/notifications"),
  readNotification: (id: string) =>
    request<{ ok: boolean }>(`/api/notifications/${id}/read`, { method: "POST" }),
  readAllNotifications: () =>
    request<{ ok: boolean }>("/api/notifications/read-all", { method: "POST" }),

  dashboard: () => request<Dashboard>("/api/dashboard"),
  activity: () => request<{ ok: boolean; activity: ActivityItem[] }>("/api/activity"),

  agents: () => request<{ ok: boolean; agents: Agent[] }>("/api/agents"),
  createAgent: (input: { name: string; role: string; emoji: string; instructions: string; skills?: string[] }) =>
    request<{ ok: boolean; agent: Agent; apiKey: string }>("/api/agents", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  agent: (id: string) =>
    request<{ ok: boolean; agent: Agent; tasks: Task[]; status: AgentStatus; spend_usd: number }>(`/api/agents/${id}`),
  updateAgent: (id: string, input: Partial<Pick<Agent, "name" | "role" | "emoji" | "instructions" | "skills" | "model_provider" | "model_name" | "monthly_budget_usd" | "can_delegate">> & { modelProvider?: string; modelName?: string; monthlyBudgetUsd?: number; canDelegate?: boolean }) =>
    request<{ ok: boolean; agent: Agent }>(`/api/agents/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteAgent: (id: string) =>
    request<{ ok: boolean }>(`/api/agents/${id}`, { method: "DELETE" }),
  rotateKey: (id: string) =>
    request<{ ok: boolean; apiKey: string }>(`/api/agents/${id}/rotate-key`, { method: "POST" }),

  tasks: (status?: string) =>
    request<{ ok: boolean; tasks: Task[] }>(`/api/tasks${status ? `?status=${status}` : ""}`),
  createTask: (input: { title: string; details: string; agentId: string }) =>
    request<{ ok: boolean; task: Task }>("/api/tasks", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  task: (id: string) => request<{ ok: boolean; task: Task }>(`/api/tasks/${id}`),
  updateTask: (id: string, input: Record<string, unknown>) =>
    request<{ ok: boolean; task: Task }>(`/api/tasks/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteTask: (id: string) => request<{ ok: boolean }>(`/api/tasks/${id}`, { method: "DELETE" }),
  comment: (id: string, body: string) =>
    request<{ ok: boolean; comment: Comment }>(`/api/tasks/${id}/comments`, {
      method: "POST",
      body: JSON.stringify({ body }),
    }),
  approve: (id: string) =>
    request<{ ok: boolean; task: Task }>(`/api/tasks/${id}/approve`, { method: "POST" }),
  sendBack: (id: string, note: string) =>
    request<{ ok: boolean; task: Task }>(`/api/tasks/${id}/send-back`, {
      method: "POST",
      body: JSON.stringify({ note }),
    }),
};
