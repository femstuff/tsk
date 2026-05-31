import { SimpleLineChart } from "../charts/SimpleLineChart";
import type { AdminAuthActivity } from "../../entities/admin/types";
import type { DayCount } from "../../lib/dashboardAnalytics";

type DashboardOverviewProps = {
  documentsTotal: number;
  documentsWeekDelta: number;
  tasksTotal: number;
  tasksLabel?: string;
  usersTotal: number;
  usersLabel?: string;
  activityToday: number;
  activityLabel?: string;
  documentsTrend: DayCount[];
  httpRateSeries: DayCount[];
  recentAuth: AdminAuthActivity[];
  formatAuthTime: (value: string) => string;
  onNavigateBitrix?: () => void;
  onNavigateJobs?: () => void;
};

export function DashboardOverview({
  documentsTotal,
  documentsWeekDelta,
  tasksTotal,
  tasksLabel = "в Bitrix24",
  usersTotal,
  usersLabel = "авторизовано",
  activityToday,
  activityLabel = "действий сегодня",
  documentsTrend,
  httpRateSeries,
  recentAuth,
  formatAuthTime,
  onNavigateBitrix,
  onNavigateJobs
}: DashboardOverviewProps) {
  const apiSeries = httpRateSeries.length > 0 ? httpRateSeries : documentsTrend;
  const chartTitle = httpRateSeries.length > 0 ? "Нагрузка API" : "Заявки на документы";
  const chartSubtitle =
    httpRateSeries.length > 0 ? "Prometheus · RPS за последний час" : "Новые заявки за 7 дней";

  return (
    <>
      <section className="kpi-grid">
        <button type="button" className="kpi-card kpi-card-action" onClick={onNavigateJobs}>
          <span className="kpi-label">Документы</span>
          <strong className="kpi-value">{documentsTotal}</strong>
          <span className="kpi-delta">+{documentsWeekDelta} за неделю</span>
        </button>
        <button type="button" className="kpi-card kpi-card-action" onClick={onNavigateBitrix}>
          <span className="kpi-label">Задачи Bitrix24</span>
          <strong className="kpi-value">{tasksTotal}</strong>
          <span className="kpi-delta">{tasksLabel}</span>
        </button>
        <button type="button" className="kpi-card kpi-card-action" onClick={onNavigateBitrix}>
          <span className="kpi-label">Пользователи OAuth</span>
          <strong className="kpi-value">{usersTotal}</strong>
          <span className="kpi-delta">{usersLabel}</span>
        </button>
        <article className="kpi-card">
          <span className="kpi-label">Голос / Bitrix</span>
          <strong className="kpi-value">{activityToday}</strong>
          <span className="kpi-delta">{activityLabel}</span>
        </article>
      </section>

      <section className="dashboard-grid two-up">
        <article className="panel-card">
          <div className="panel-card-head">
            <h2>{chartTitle}</h2>
            <span>{chartSubtitle}</span>
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
            <h2>Авторизации Bitrix24</h2>
            <span>Последние подключения</span>
          </div>
          <ul className="activity-feed">
            {recentAuth.length === 0 ? (
              <li className="activity-empty">Авторизаций пока нет.</li>
            ) : (
              recentAuth.slice(0, 6).map((item, index) => (
                <li key={`${item.bitrixUserId}-${item.occurredAt}-${index}`}>
                  <span className="activity-time">{formatAuthTime(item.occurredAt)}</span>
                  <div>
                    <strong>{item.userName || `id ${item.bitrixUserId}`}</strong>
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
