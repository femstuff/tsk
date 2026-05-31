import type { ObservabilityDashboard } from "../entities/observability/types";
import type { DayCount, DonutSlice } from "./dashboardAnalytics";

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

export function formatUptime(seconds: number | null | undefined): string {
  if (seconds == null || seconds <= 0) {
    return "—";
  }
  if (seconds < 60) {
    return `${Math.round(seconds)} сек`;
  }
  if (seconds < 3600) {
    return `${(seconds / 60).toFixed(1)} мин`;
  }
  if (seconds < 86400) {
    return `${(seconds / 3600).toFixed(1)} ч`;
  }
  return `${(seconds / 86400).toFixed(1)} д`;
}

export function formatPrometheusMetrics(snapshot: ObservabilityDashboard | null) {
  if (!snapshot?.available) {
    return {
      rps: "—",
      errorRate: "—",
      latency: "—",
      availability: "—"
    };
  }

  const errorRate =
    snapshot.errorRatePercent != null ? `${snapshot.errorRatePercent.toFixed(2)}%` : "—";
  const latency =
    snapshot.latencyP95Seconds != null
      ? `${Math.round(snapshot.latencyP95Seconds * 1000)} ms`
      : "—";
  const availability =
    snapshot.errorRatePercent != null
      ? `${Math.max(0, 100 - snapshot.errorRatePercent).toFixed(2)}%`
      : snapshot.uptimeSeconds != null
        ? "100%"
        : "—";

  return {
    rps: snapshot.rps != null ? snapshot.rps.toFixed(2) : "—",
    errorRate,
    latency,
    availability
  };
}

export function prometheusHttpSeries(snapshot: ObservabilityDashboard | null): DayCount[] {
  if (!snapshot?.httpRateSeries?.length) {
    return [];
  }
  return snapshot.httpRateSeries.map((point) => ({
    label: point.label,
    value: point.value
  }));
}

export function prometheusJobDonut(snapshot: ObservabilityDashboard | null): DonutSlice[] {
  if (!snapshot?.jobStatusSeries?.length) {
    return [];
  }

  return snapshot.jobStatusSeries
    .filter((point) => point.value > 0)
    .map((point) => ({
      label: JOB_STATUS_LABELS[point.label] ?? point.label,
      value: point.value,
      color: JOB_STATUS_COLORS[point.label] ?? "#64748b"
    }));
}

export function prometheusSystemLoad(snapshot: ObservabilityDashboard | null) {
  if (!snapshot?.available) {
    return null;
  }

  const cpuPercent =
    snapshot.cpuCores != null ? Math.min(100, Math.round(snapshot.cpuCores * 100)) : null;
  const memoryGb =
    snapshot.memoryBytes != null ? snapshot.memoryBytes / (1024 * 1024 * 1024) : null;
  const memoryPercent =
    memoryGb != null ? Math.min(100, Math.round((memoryGb / 2) * 100)) : null;

  return {
    cpu: cpuPercent ?? 0,
    memory: memoryPercent ?? 0,
    memoryGb: memoryGb != null ? memoryGb.toFixed(2) : "—",
    uptimeSeconds: snapshot.uptimeSeconds ?? null,
    uptimeLabel: formatUptime(snapshot.uptimeSeconds)
  };
}
