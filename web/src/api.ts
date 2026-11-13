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
  evidence: { url: string; title: string; description: string; characters: number; truncated: boolean };
  task?: Task;
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

  dashboard: () => request<Dashboard>("/api/dashboard"),
  activity: () => request<{ ok: boolean; activity: ActivityItem[] }>("/api/activity"),

  agents: () => request<{ ok: boolean; agents: Agent[] }>("/api/agents"),
  createAgent: (input: { name: string; role: string; emoji: string; instructions: string }) =>
    request<{ ok: boolean; agent: Agent; apiKey: string }>("/api/agents", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  agent: (id: string) =>
    request<{ ok: boolean; agent: Agent; tasks: Task[]; status: AgentStatus }>(`/api/agents/${id}`),
  updateAgent: (id: string, input: Partial<Pick<Agent, "name" | "role" | "emoji" | "instructions" | "skills" | "model_provider" | "model_name">> & { modelProvider?: string; modelName?: string }) =>
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
