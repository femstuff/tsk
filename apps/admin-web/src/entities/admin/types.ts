export type AdminBitrixUser = {
  sessionId: string;
  bitrixUserId: number;
  userName: string;
  portalDomain: string;
  oauthScopes?: string;
  taskCount: number;
  status: string;
  connectedAt: string;
  lastActiveAt: string;
  expiresAt?: string | null;
};

export type AdminAuthActivity = {
  bitrixUserId: number;
  userName: string;
  portalDomain: string;
  status: string;
  occurredAt: string;
  message: string;
};

export type AdminBitrixTaskStats = {
  total: number;
  totalOpen: number;
  inProgress: number;
  overdue: number;
};

export type AdminBitrixTaskItem = {
  id: string;
  title: string;
  status: string;
  statusLabel: string;
  deadline?: string;
  closedDate?: string;
};

export type BitrixTaskFilterKey = "open" | "in_progress" | "overdue" | "completed";

export type AdminDashboard = {
  bitrixTasks: AdminBitrixTaskStats;
  bitrixTaskItems: AdminBitrixTaskItem[];
  authorizedUsers: number;
  voiceActivityToday: number;
  voiceActivityWeek: number;
  users: AdminBitrixUser[];
  recentAuth: AdminAuthActivity[];
};
