import { useMemo, useState } from "react";

import { DonutChart } from "../charts/DonutChart";
import { SimpleLineChart } from "../charts/SimpleLineChart";
import type { AdminAuthActivity, AdminBitrixTaskItem } from "../../entities/admin/types";
import type { DayCount, DonutSlice } from "../../lib/dashboardAnalytics";
import { filterBitrixTasksByKey } from "../../lib/dashboardAnalytics";
type DashboardOverviewProps = {
  documentsTotal: number;
  documentsWeekDelta: number;
  tasksTotal: number;
  tasksWeekDelta: number;
  tasksLabel?: string;
  usersTotal: number;
  usersWeekDelta: number;
  usersLabel?: string;
  activityToday: number;
  activityLabel?: string;
  documentsTrend: DayCount[];
  jobStatusDonut: DonutSlice[];
  bitrixTaskItems?: AdminBitrixTaskItem[];
  formatTaskDeadline?: (value: string | undefined) => string;
  httpRateSeries: DayCount[];
  metrics: {
    rps: string;
    errorRate: string;
    latency: string;
    availability: string;
  };
  systemLoad: {
    cpu: number;
    memory: number;
    memoryLabel: string;
    uptimeLabel: string;
  };
  metricsSource: string;
  observabilityError: string | null;
  recentAuth: AdminAuthActivity[];
  formatAuthTime: (value: string) => string;
};

function MetricBar({
  label,
  value,
  suffix = "%",
  detail
}: {
  label: string;
  value: number | string;
  suffix?: string;
  detail?: string;
}) {
  const numeric = typeof value === "number" ? value : 0;
  return (
    <div className="metric-bar">
      <div className="metric-bar-head">
        <span>{label}</span>
        <strong>
          {value}
          {typeof value === "number" ? suffix : ""}
        </strong>
      </div>
      {typeof value === "number" ? (
        <div className="metric-bar-track">
          <div className="metric-bar-fill" style={{ width: `${Math.min(100, numeric)}%` }} />
        </div>
      ) : null}
      {detail ? <p className="metric-bar-detail">{detail}</p> : null}
    </div>
  );
}

export function DashboardOverview({
  documentsTotal,
  documentsWeekDelta,
  tasksTotal,
  tasksWeekDelta,
  tasksLabel = "в Bitrix24",
  usersTotal,
  usersWeekDelta,
  usersLabel = "авторизовано",
  activityToday,
  activityLabel = "голосовых действий сегодня",
  documentsTrend,
  jobStatusDonut,
  bitrixTaskItems = [],
  formatTaskDeadline = (value) => value?.trim() || "—",
  httpRateSeries,
  metrics,
  systemLoad,
  metricsSource,
  observabilityError,
  recentAuth,
  formatAuthTime
}: DashboardOverviewProps) {
  const [selectedFilterKey, setSelectedFilterKey] = useState<string | null>(null);
  const apiSeries = httpRateSeries.length > 0 ? httpRateSeries : documentsTrend;

  const selectedSlice = jobStatusDonut.find((slice) => slice.filterKey === selectedFilterKey);
  const filteredTasks = useMemo(
    () => (selectedFilterKey ? filterBitrixTasksByKey(bitrixTaskItems, selectedFilterKey) : []),
    [bitrixTaskItems, selectedFilterKey]
  );

  const handleSliceSelect = (slice: DonutSlice) => {
    if (!slice.filterKey) {
      return;
    }
    setSelectedFilterKey((current) => (current === slice.filterKey ? null : slice.filterKey ?? null));
  };

  return (
    <>
      {observabilityError ? <p className="banner error">{observabilityError}</p> : null}

      <section className="kpi-grid">
        <article className="kpi-card">
          <span className="kpi-label">Документы</span>
          <strong className="kpi-value">{documentsTotal}</strong>
          <span className="kpi-delta">+{documentsWeekDelta} за неделю</span>
        </article>
        <article className="kpi-card">
          <span className="kpi-label">Задачи Bitrix24</span>
          <strong className="kpi-value">{tasksTotal}</strong>
          <span className="kpi-delta">{tasksLabel}</span>
        </article>
        <article className="kpi-card">
          <span className="kpi-label">Пользователи</span>
          <strong className="kpi-value">{usersTotal}</strong>
          <span className="kpi-delta">{usersLabel}</span>
        </article>
        <article className="kpi-card">
          <span className="kpi-label">Голос / Bitrix</span>
          <strong className="kpi-value">{activityToday}</strong>
          <span className="kpi-delta">{activityLabel}</span>
        </article>
      </section>

      <section className="dashboard-grid two-up">
        <article className="panel-card">
          <div className="panel-card-head">
            <h2>{httpRateSeries.length > 0 ? "Нагрузка API" : "Документы"}</h2>
            <span>{httpRateSeries.length > 0 ? "Prometheus · RPS за 1 ч" : "Заявки за 7 дней"}</span>
          </div>
          <SimpleLineChart
            data={apiSeries}
            valueFormatter={
              httpRateSeries.length > 0 ? (value) => `${value.toFixed(2)} req/s` : undefined
            }
          />
        </article>

        <article className="panel-card">
          <div className="panel-card-head">
            <h2>Задачи Bitrix24</h2>
            <span>{tasksTotal} в выборке · клик по сегменту</span>
          </div>
          <DonutChart
            slices={jobStatusDonut}
            selectedFilterKey={selectedFilterKey}
            onSliceSelect={handleSliceSelect}
          />
          {selectedFilterKey && selectedSlice ? (
            <div className="bitrix-task-filter-panel">
              <div className="bitrix-task-filter-head">
                <strong>{selectedSlice.label}</strong>
                <span>{filteredTasks.length} задач</span>
                <button type="button" className="bitrix-task-filter-clear" onClick={() => setSelectedFilterKey(null)}>
                  Сбросить
                </button>
              </div>
              {filteredTasks.length === 0 ? (
                <p className="activity-empty">Нет задач с этим статусом в текущей выборке.</p>
              ) : (
                <ul className="bitrix-task-list">
                  {filteredTasks.map((task) => (
                    <li key={task.id} className="bitrix-task-item">
                      <div>
                        <strong>{task.title || `Задача #${task.id}`}</strong>
                        <p className="subtle">ID {task.id}</p>
                      </div>
                      <div className="bitrix-task-meta">
                        <span className="status-pill">{task.statusLabel}</span>
                        <span>{formatTaskDeadline(task.deadline)}</span>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}
        </article>
      </section>

      <section className="dashboard-grid three-up">
        <article className="panel-card">
          <div className="panel-card-head">
            <h2>Метрики</h2>
            <span>{metricsSource}</span>
          </div>
          <div className="metrics-grid">
            <div>
              <span>RPS</span>
              <strong>{metrics.rps}</strong>
            </div>
            <div>
              <span>Ошибки</span>
              <strong>{metrics.errorRate}</strong>
            </div>
            <div>
              <span>Латентность p95</span>
              <strong>{metrics.latency}</strong>
            </div>
            <div>
              <span>Доступность</span>
              <strong>{metrics.availability}</strong>
            </div>
          </div>
        </article>

        <article className="panel-card">
          <div className="panel-card-head">
            <h2>Системные показатели</h2>
            <span>{metricsSource}</span>
          </div>
          <MetricBar label="CPU" value={systemLoad.cpu} />
          <MetricBar
            label="Память"
            value={systemLoad.memory}
            detail={`${systemLoad.memoryLabel} GB resident`}
          />
          <MetricBar label="Uptime" value={systemLoad.uptimeLabel} suffix="" detail="backend API" />
        </article>

        <article className="panel-card">
          <div className="panel-card-head">
            <h2>Последняя активность</h2>
            <span>Авторизации Bitrix24</span>
          </div>
          <ul className="activity-feed">
            {recentAuth.length === 0 ? (
              <li className="activity-empty">Авторизаций пока нет.</li>
            ) : (
              recentAuth.map((item, index) => (
                <li key={`${item.bitrixUserId}-${item.occurredAt}-${index}`}>
                  <span className="activity-time">{formatAuthTime(item.occurredAt)}</span>
                  <div>
                    <strong>
                      {item.message}: {item.userName || `id ${item.bitrixUserId}`}
                    </strong>
                    <p>
                      {item.portalDomain} · {item.status === "active" ? "в сети" : "выход"}
                    </p>
                  </div>
                </li>
              ))
            )}
          </ul>
        </article>
      </section>
    </>
  );
}
