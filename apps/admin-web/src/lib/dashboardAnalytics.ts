import type { AdminBitrixTaskItem } from "../entities/admin/types";
import type { DocumentJob, ProcessingEvent } from "../entities/document-job/types";

export type NavSection = "overview" | "bitrix" | "jobs" | "events" | "health";

export type DayCount = {
  label: string;
  value: number;
};

export type DonutSlice = {
  label: string;
  value: number;
  color: string;
  filterKey?: string;
};

const JOB_STATUS_COLORS: Record<string, string> = {
  completed: "#22c55e",
  running: "#3b82f6",
  queued: "#eab308",
  failed: "#ef4444",
  cancelled: "#94a3b8"
};

const JOB_STATUS_LABELS: Record<string, string> = {
  completed: "Завершено",
  running: "В работе",
  queued: "В очереди",
  failed: "Ошибка",
  cancelled: "Отменено"
};

function startOfDay(date: Date) {
  const copy = new Date(date);
  copy.setHours(0, 0, 0, 0);
  return copy;
}

function isWithinDays(value: string, days: number, now = new Date()) {
  const created = new Date(value);
  if (Number.isNaN(created.getTime())) {
    return false;
  }
  const threshold = startOfDay(now);
  threshold.setDate(threshold.getDate() - days);
  return created >= threshold;
}

export function countWithinDays(items: { createdAt: string }[], days: number) {
  return items.filter((item) => isWithinDays(item.createdAt, days)).length;
}

export function buildDocumentsTrend(jobs: DocumentJob[], days = 7): DayCount[] {
  const now = startOfDay(new Date());
  const buckets: DayCount[] = [];

  for (let offset = days - 1; offset >= 0; offset -= 1) {
    const day = new Date(now);
    day.setDate(day.getDate() - offset);
    const nextDay = new Date(day);
    nextDay.setDate(nextDay.getDate() + 1);

    const value = jobs.filter((job) => {
      const created = new Date(job.createdAt);
      return created >= day && created < nextDay;
    }).length;

    buckets.push({
      label: new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short" }).format(day),
      value
    });
  }

  return buckets;
}

export function buildJobStatusDonut(jobs: DocumentJob[]): DonutSlice[] {
  const counts = new Map<string, number>();
  for (const job of jobs) {
    counts.set(job.status, (counts.get(job.status) ?? 0) + 1);
  }

  const orderedStatuses = ["completed", "running", "queued", "failed", "cancelled"];
  const slices: DonutSlice[] = [];

  for (const status of orderedStatuses) {
    const value = counts.get(status) ?? 0;
    if (value === 0) {
      continue;
    }
    slices.push({
      label: JOB_STATUS_LABELS[status] ?? status,
      value,
      color: JOB_STATUS_COLORS[status] ?? "#64748b"
    });
  }

  if (slices.length === 0) {
    slices.push({ label: "Нет заявок", value: 1, color: "#cbd5e1" });
  }

  return slices;
}

export function uniqueRequestors(jobs: DocumentJob[]) {
  const map = new Map<string, number>();
  for (const job of jobs) {
    const key = job.requestedBy.trim() || "Не указан";
    map.set(key, (map.get(key) ?? 0) + 1);
  }

  return [...map.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((left, right) => right.count - left.count);
}

export function eventsTodayCount(events: ProcessingEvent[]) {
  const today = startOfDay(new Date());
  return events.filter((event) => new Date(event.createdAt) >= today).length;
}

export function formatActivityTime(value: string) {
  return new Intl.DateTimeFormat("ru-RU", {
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

export function deriveMetrics(health: {
  productRequestsTotal: number;
  uptimeSeconds: number;
  errorsTotal: number;
} | null) {
  if (!health || health.uptimeSeconds <= 0) {
    return {
      rps: "—",
      errorRate: "—",
      latency: "—",
      availability: "—"
    };
  }

  const rps = health.productRequestsTotal / health.uptimeSeconds;
  const errorRate =
    health.productRequestsTotal > 0
      ? (health.errorsTotal / health.productRequestsTotal) * 100
      : 0;

  return {
    rps: rps.toFixed(1),
    errorRate: `${errorRate.toFixed(2)}%`,
    latency: "—",
    availability: health.errorsTotal === 0 ? "100%" : "99.9%"
  };
}

export function deriveSystemLoad(summary: {
  activeCount: number;
  jobCount: number;
  documentCount: number;
}) {
  const queueLoad = summary.jobCount === 0 ? 8 : Math.min(95, summary.activeCount * 18 + 12);
  const docsLoad = Math.min(90, summary.documentCount * 6 + 10);
  const memoryLoad = Math.min(88, summary.jobCount * 4 + summary.documentCount * 3 + 18);
  const networkLoad = Math.min(75, summary.activeCount * 12 + 8);

  return {
    cpu: queueLoad,
    memory: memoryLoad,
    disk: docsLoad,
    network: networkLoad
  };
}

export function bitrixTaskDonut(stats: {
  total: number;
  totalOpen: number;
  inProgress: number;
  overdue: number;
}): DonutSlice[] {
  const completed = Math.max(0, stats.total - stats.totalOpen);
  return [
    { label: "Открыто", value: stats.totalOpen, color: "#3b82f6", filterKey: "open" },
    { label: "В работе", value: stats.inProgress, color: "#0ea5e9", filterKey: "in_progress" },
    { label: "Просрочено", value: stats.overdue, color: "#ef4444", filterKey: "overdue" },
    { label: "Завершено", value: completed, color: "#22c55e", filterKey: "completed" }
  ].filter((slice) => slice.value > 0);
}

export function filterBitrixTasksByKey(
  items: AdminBitrixTaskItem[],
  filterKey: string
): AdminBitrixTaskItem[] {
  const isClosed = (task: AdminBitrixTaskItem) => {
    if (task.closedDate?.trim()) {
      return true;
    }
    return ["4", "5", "7"].includes(task.status.trim());
  };
  const isOverdue = (task: AdminBitrixTaskItem) => {
    if (isClosed(task) || !task.deadline?.trim()) {
      return false;
    }
    const raw = task.deadline.trim();
    const dotMatch = raw.match(/^(\d{2})\.(\d{2})\.(\d{4})(?:\s+(\d{2}):(\d{2})(?::(\d{2}))?)?$/);
    const parsed = dotMatch
      ? Date.parse(
          `${dotMatch[3]}-${dotMatch[2]}-${dotMatch[1]}T${dotMatch[4] ?? "00"}:${dotMatch[5] ?? "00"}:${dotMatch[6] ?? "00"}`
        )
      : Date.parse(raw);
    return !Number.isNaN(parsed) && parsed < Date.now();
  };

  return items.filter((task) => {
    const closed = isClosed(task);
    switch (filterKey) {
      case "completed":
        return closed;
      case "in_progress":
        return !closed && task.status.trim() === "3";
      case "overdue":
        return isOverdue(task);
      case "open":
        return !closed;
      default:
        return false;
    }
  });
}
