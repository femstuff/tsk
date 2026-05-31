import { useMemo, useState, type FormEvent, type ReactNode } from "react";

import { DonutChart } from "../components/charts/DonutChart";
import { DashboardOverview } from "../components/dashboard/DashboardOverview";
import { AdminShell } from "../components/layout/AdminShell";
import { SectionTabs } from "../components/ui/SectionTabs";
import type { AdminBitrixUser } from "../entities/admin/types";
import type {
  DocumentJob,
  DocumentTemplate,
  GeneratedDocument,
  HealthResponse,
  ProcessingEvent,
  SourceDocument,
  TaskCommand
} from "../entities/document-job/types";
import {
  filterBitrixTasksByKey,
  type DonutSlice,
  type NavSection
} from "../lib/dashboardAnalytics";

type IntegrationItem = {
  name: string;
  status: string;
  tone: "ok" | "warn" | "muted";
};

type JobsTab = "queue" | "templates" | "files" | "create";
type EventsTab = "events" | "commands";

type DashboardPageViewProps = {
  activeSection: NavSection;
  onNavigate: (section: NavSection) => void;
  onRefresh: () => void;
  loading: boolean;
  error: string | null;
  integrations: IntegrationItem[];
  overviewProps: React.ComponentProps<typeof DashboardOverview>;
  bitrixTaskDonut: DonutSlice[];
  bitrixTaskItems: import("../entities/admin/types").AdminBitrixTaskItem[];
  templates: DocumentTemplate[];
  jobs: DocumentJob[];
  documents: GeneratedDocument[];
  sourceDocuments: SourceDocument[];
  taskCommands: TaskCommand[];
  events: ProcessingEvent[];
  visibleEvents: ProcessingEvent[];
  safeEventsPage: number;
  eventsPageCount: number;
  setEventsPage: (value: number | ((page: number) => number)) => void;
  health: HealthResponse | null;
  dashboardMetrics: {
    rps: string;
    errorRate: string;
    latency: string;
    availability: string;
  };
  systemLoad: {
    cpu: number;
    memory: number;
    memoryLabel: string;
  } | null;
  uptimeLabel: string;
  metricsSource: string;
  observabilityError: string | null;
  prometheusUrl: string;
  grafanaUrl: string;
  authorizedUsers: AdminBitrixUser[];
  apiBaseUrl: string;
  templateForm: {
    name: string;
    category: string;
    version: string;
    description: string;
  };
  setTemplateForm: React.Dispatch<
    React.SetStateAction<{
      name: string;
      category: string;
      version: string;
      description: string;
    }>
  >;
  templateFile: File | null;
  setTemplateFile: (file: File | null) => void;
  readyToSubmitTemplate: boolean;
  creatingTemplate: boolean;
  onTemplateSubmit: (event: FormEvent<HTMLFormElement>) => void;
  jobForm: {
    templateId: string;
    sourceName: string;
    requestedBy: string;
    payload: string;
    deliveryChannel: "internal" | "email" | "bitrix";
    deliveryAddress: string;
  };
  setJobForm: React.Dispatch<
    React.SetStateAction<{
      templateId: string;
      sourceName: string;
      requestedBy: string;
      payload: string;
      deliveryChannel: "internal" | "email" | "bitrix";
      deliveryAddress: string;
    }>
  >;
  readyToSubmitJob: boolean;
  creatingJob: boolean;
  onJobSubmit: (event: FormEvent<HTMLFormElement>) => void;
  templateOptions: DocumentTemplate[];
  jobStatusDrafts: Record<string, string>;
  setJobStatusDrafts: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  updatingJobId: string | null;
  onChangeJobStatus: (jobId: string, status: string) => void;
  translateTemplateName: (value: string) => string;
  translateTemplateDescription: (value: string) => string;
  translateFromMap: (value: string, labels: Record<string, string>) => string;
  translateMessage: (value: string) => string;
  translateCommandText: (value: string) => string;
  translateResultMessage: (value: string) => string;
  formatDate: (value: string | null) => string;
  formatBytes: (value: number) => string;
  formatEventDetails: (details: string) => string;
  statusClass: (status: string) => string;
  labels: {
    status: Record<string, string>;
    category: Record<string, string>;
    dispatchStatus: Record<string, string>;
    deliveryChannel: Record<string, string>;
    requestedBy: Record<string, string>;
    origin: Record<string, string>;
    sourceDocumentKind: Record<string, string>;
    taskTarget: Record<string, string>;
    taskCommandStatus: Record<string, string>;
    integrationMode: Record<string, string>;
    eventLevel: Record<string, string>;
    eventType: Record<string, string>;
    healthStatus: Record<string, string>;
  };
  eventsPageSize: number;
};

function Section({ visible, children }: { visible: boolean; children: ReactNode }) {
  if (!visible) {
    return null;
  }
  return <>{children}</>;
}

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

export function DashboardPageView(props: DashboardPageViewProps) {
  const {
    activeSection,
    onNavigate,
    onRefresh,
    loading,
    error,
    integrations,
    overviewProps,
    bitrixTaskDonut,
    bitrixTaskItems
  } = props;

  const [jobsTab, setJobsTab] = useState<JobsTab>("queue");
  const [eventsTab, setEventsTab] = useState<EventsTab>("events");
  const [bitrixFilterKey, setBitrixFilterKey] = useState<string | null>(null);

  const selectedSlice = bitrixTaskDonut.find((slice) => slice.filterKey === bitrixFilterKey);
  const filteredBitrixTasks = useMemo(
    () => (bitrixFilterKey ? filterBitrixTasksByKey(bitrixTaskItems, bitrixFilterKey) : []),
    [bitrixFilterKey, bitrixTaskItems]
  );

  const recentEvents = props.events.slice(0, 5);

  const handleBitrixSliceSelect = (slice: DonutSlice) => {
    if (!slice.filterKey) {
      return;
    }
    setBitrixFilterKey((current) => (current === slice.filterKey ? null : slice.filterKey ?? null));
  };

  return (
    <AdminShell
      activeSection={activeSection}
      onNavigate={onNavigate}
      onRefresh={onRefresh}
      loading={loading}
      integrations={integrations}
    >
      {error ? <p className="banner error">{error}</p> : null}
      {props.observabilityError ? <p className="banner warn">{props.observabilityError}</p> : null}

      <Section visible={activeSection === "overview"}>
        <DashboardOverview
          {...overviewProps}
          onNavigateBitrix={() => onNavigate("bitrix")}
          onNavigateJobs={() => onNavigate("jobs")}
        />

        <article className="panel overview-recent-events">
          <div className="panel-header">
            <h2>Последние события</h2>
            <button type="button" className="link-button" onClick={() => onNavigate("events")}>
              Весь журнал →
            </button>
          </div>
          <div className="list compact-list">
            {recentEvents.length === 0 ? (
              <p className="empty">Событий пока нет.</p>
            ) : (
              recentEvents.map((event) => (
                <div className="list-item stacked" key={event.id}>
                  <div className="list-topline">
                    <div>
                      <strong>{props.translateMessage(event.message)}</strong>
                      <p className="subtle">
                        {props.translateFromMap(event.eventType, props.labels.eventType)}
                      </p>
                    </div>
                    <div className="meta">
                      <span className={props.statusClass(event.level)}>
                        {props.translateFromMap(event.level, props.labels.eventLevel)}
                      </span>
                      <span>{props.formatDate(event.createdAt)}</span>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </article>
      </Section>

      <Section visible={activeSection === "bitrix"}>
        <section className="dashboard-grid two-up">
          <article className="panel-card">
            <div className="panel-card-head">
              <h2>Задачи Bitrix24</h2>
              <span>{bitrixTaskItems.length} в выборке · клик по сегменту</span>
            </div>
            {bitrixTaskDonut.length > 0 ? (
              <DonutChart
                slices={bitrixTaskDonut}
                selectedFilterKey={bitrixFilterKey}
                onSliceSelect={handleBitrixSliceSelect}
              />
            ) : (
              <p className="empty">Нет данных о задачах. Подключите Bitrix24 через мобильное приложение.</p>
            )}
            {bitrixFilterKey && selectedSlice ? (
              <div className="bitrix-task-filter-panel">
                <div className="bitrix-task-filter-head">
                  <strong>{selectedSlice.label}</strong>
                  <span>{filteredBitrixTasks.length} задач</span>
                  <button
                    type="button"
                    className="bitrix-task-filter-clear"
                    onClick={() => setBitrixFilterKey(null)}
                  >
                    Сбросить
                  </button>
                </div>
                {filteredBitrixTasks.length === 0 ? (
                  <p className="activity-empty">Нет задач с этим статусом.</p>
                ) : (
                  <ul className="bitrix-task-list">
                    {filteredBitrixTasks.map((task) => (
                      <li key={task.id} className="bitrix-task-item">
                        <div>
                          <strong>{task.title || `Задача #${task.id}`}</strong>
                          <p className="subtle">ID {task.id}</p>
                        </div>
                        <div className="bitrix-task-meta">
                          <span className="status-pill">{task.statusLabel}</span>
                          <span>{task.deadline?.trim() ? props.formatDate(task.deadline) : "—"}</span>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ) : null}
          </article>

          <article className="panel-card">
            <div className="panel-card-head">
              <h2>Подключённые пользователи</h2>
              <span>{props.authorizedUsers.length} OAuth-сессий</span>
            </div>
            <div className="user-grid compact-user-grid">
              {props.authorizedUsers.map((user) => (
                <details className="user-card" key={user.sessionId}>
                  <summary className="user-card-summary">
                    <div>
                      <strong>{user.userName || `id ${user.bitrixUserId}`}</strong>
                      <p className="subtle">{user.portalDomain}</p>
                    </div>
                    <span className="user-card-meta">{user.taskCount} задач</span>
                  </summary>
                  <dl className="user-details">
                    <div>
                      <dt>Bitrix ID</dt>
                      <dd>{user.bitrixUserId}</dd>
                    </div>
                    <div>
                      <dt>Scopes</dt>
                      <dd>{user.oauthScopes || "—"}</dd>
                    </div>
                    <div>
                      <dt>Подключён</dt>
                      <dd>{props.formatDate(user.connectedAt)}</dd>
                    </div>
                    <div>
                      <dt>Активность</dt>
                      <dd>{props.formatDate(user.lastActiveAt)}</dd>
                    </div>
                    <div>
                      <dt>Токен до</dt>
                      <dd>{props.formatDate(user.expiresAt ?? null)}</dd>
                    </div>
                  </dl>
                </details>
              ))}
              {!loading && props.authorizedUsers.length === 0 ? (
                <p className="empty">Пользователи появятся после входа в мобильном приложении.</p>
              ) : null}
            </div>
          </article>
        </section>
      </Section>

      <Section visible={activeSection === "jobs"}>
        <SectionTabs
          active={jobsTab}
          onChange={setJobsTab}
          items={[
            { id: "queue", label: "Заявки", count: props.jobs.length },
            { id: "templates", label: "Шаблоны", count: props.templates.length },
            { id: "files", label: "Файлы", count: props.documents.length + props.sourceDocuments.length },
            { id: "create", label: "Новая заявка" }
          ]}
        />

        {jobsTab === "queue" ? (
          <article className="panel">
            <div className="panel-header">
              <h2>Очередь заявок</h2>
              <span>{props.jobs.length} всего</span>
            </div>
            <div className="list">
              {props.jobs.map((job) => (
                <div className="list-item stacked" key={job.id}>
                  <div className="list-topline">
                    <div>
                      <strong>{props.translateTemplateName(job.templateName)}</strong>
                      <p>{job.sourceName}</p>
                    </div>
                    <div className="meta">
                      <span className={props.statusClass(job.status)}>
                        {props.translateFromMap(job.status, props.labels.status)}
                      </span>
                      <span>{props.formatDate(job.createdAt)}</span>
                    </div>
                  </div>
                  <div className="job-details">
                    <span>
                      Инициатор: {props.translateFromMap(job.requestedBy, props.labels.requestedBy)}
                    </span>
                    <span>
                      Доставка: {props.translateFromMap(job.deliveryChannel, props.labels.deliveryChannel)}
                    </span>
                    <span>
                      Отправка: {props.translateFromMap(job.dispatchStatus, props.labels.dispatchStatus)}
                    </span>
                    <span>Завершено: {props.formatDate(job.completedAt)}</span>
                    {job.errorMessage ? (
                      <span>Ошибка: {props.translateMessage(job.errorMessage)}</span>
                    ) : null}
                  </div>
                  <p className="payload-preview">{job.payload || "Параметры не указаны."}</p>
                  <div className="job-actions">
                    <select
                      value={props.jobStatusDrafts[job.id] ?? job.status}
                      onChange={(event) =>
                        props.setJobStatusDrafts((current) => ({
                          ...current,
                          [job.id]: event.target.value
                        }))
                      }
                    >
                      <option value="queued">В очереди</option>
                      <option value="running">В работе</option>
                      <option value="completed">Завершено</option>
                      <option value="failed">Ошибка</option>
                      <option value="cancelled">Отменено</option>
                    </select>
                    <button
                      onClick={() =>
                        void props.onChangeJobStatus(
                          job.id,
                          props.jobStatusDrafts[job.id] ?? job.status
                        )
                      }
                      disabled={props.updatingJobId === job.id}
                    >
                      {props.updatingJobId === job.id ? "Обновляем…" : "Применить"}
                    </button>
                  </div>
                </div>
              ))}
              {!loading && props.jobs.length === 0 ? (
                <p className="empty">Создайте заявку во вкладке «Новая заявка».</p>
              ) : null}
            </div>
          </article>
        ) : null}

        {jobsTab === "templates" ? (
          <section className="content-grid two-columns">
            <article className="panel">
              <div className="panel-header">
                <h2>Загрузить шаблон</h2>
              </div>
              <form className="form-grid" onSubmit={(event) => void props.onTemplateSubmit(event)}>
                <label>
                  Название
                  <input
                    value={props.templateForm.name}
                    onChange={(event) =>
                      props.setTemplateForm((current) => ({ ...current, name: event.target.value }))
                    }
                  />
                </label>
                <label>
                  Категория
                  <input
                    value={props.templateForm.category}
                    onChange={(event) =>
                      props.setTemplateForm((current) => ({ ...current, category: event.target.value }))
                    }
                  />
                </label>
                <label>
                  Версия
                  <input
                    value={props.templateForm.version}
                    onChange={(event) =>
                      props.setTemplateForm((current) => ({ ...current, version: event.target.value }))
                    }
                  />
                </label>
                <label className="full-width">
                  Описание
                  <textarea
                    value={props.templateForm.description}
                    onChange={(event) =>
                      props.setTemplateForm((current) => ({
                        ...current,
                        description: event.target.value
                      }))
                    }
                  />
                </label>
                <label className="full-width">
                  Файл
                  <input
                    type="file"
                    onChange={(event) => props.setTemplateFile(event.target.files?.[0] ?? null)}
                  />
                </label>
                <div className="form-actions full-width">
                  <button
                    className="primary"
                    type="submit"
                    disabled={!props.readyToSubmitTemplate || props.creatingTemplate}
                  >
                    {props.creatingTemplate ? "Загрузка…" : "Загрузить"}
                  </button>
                </div>
              </form>
            </article>

            <article className="panel">
              <div className="panel-header">
                <h2>Библиотека шаблонов</h2>
                <span>{props.templates.length} доступно</span>
              </div>
              <div className="list">
                {props.templates.map((template) => (
                  <div className="list-item" key={template.id}>
                    <div>
                      <strong>{props.translateTemplateName(template.name)}</strong>
                      <p>{props.translateTemplateDescription(template.description)}</p>
                      <p className="subtle">
                        {template.fileName} · {props.formatBytes(template.sizeBytes)} KB
                      </p>
                    </div>
                    <div className="meta">
                      <span>{props.translateFromMap(template.category, props.labels.category)}</span>
                      <span>{template.version}</span>
                      <a
                        href={`${props.apiBaseUrl}/api/v1/document-templates/${template.id}/download`}
                        target="_blank"
                        rel="noreferrer"
                      >
                        Скачать
                      </a>
                    </div>
                  </div>
                ))}
                {!loading && props.templates.length === 0 ? (
                  <p className="empty">Загрузите первый шаблон.</p>
                ) : null}
              </div>
            </article>
          </section>
        ) : null}

        {jobsTab === "files" ? (
          <section className="content-grid two-columns">
            <article className="panel">
              <div className="panel-header">
                <h2>Исходные файлы</h2>
                <span>{props.sourceDocuments.length}</span>
              </div>
              <div className="list">
                {props.sourceDocuments.map((document) => (
                  <div className="list-item" key={document.id}>
                    <div>
                      <strong>{document.fileName}</strong>
                      <p>{props.translateFromMap(document.origin, props.labels.origin)}</p>
                      <p className="subtle">
                        {props.translateFromMap(document.kind, props.labels.sourceDocumentKind)} · заявка{" "}
                        {document.jobId ?? "—"}
                      </p>
                    </div>
                    <div className="meta">
                      <span>{props.formatDate(document.createdAt)}</span>
                      <a
                        href={`${props.apiBaseUrl}/api/v1/source-documents/${document.id}/download`}
                        target="_blank"
                        rel="noreferrer"
                      >
                        Скачать
                      </a>
                    </div>
                  </div>
                ))}
                {!loading && props.sourceDocuments.length === 0 ? (
                  <p className="empty">Загрузки из мобильного приложения появятся здесь.</p>
                ) : null}
              </div>
            </article>

            <article className="panel">
              <div className="panel-header">
                <h2>Сгенерированные документы</h2>
                <span>{props.documents.length}</span>
              </div>
              <div className="list">
                {props.documents.map((document) => (
                  <div className="list-item" key={document.id}>
                    <div>
                      <strong>{document.fileName}</strong>
                      <p>{props.translateTemplateName(document.templateName)}</p>
                      <p className="subtle">Заявка {document.jobId}</p>
                    </div>
                    <div className="meta">
                      <span>{props.formatDate(document.createdAt)}</span>
                      <a
                        href={`${props.apiBaseUrl}/api/v1/generated-documents/${document.id}/download`}
                        target="_blank"
                        rel="noreferrer"
                      >
                        Скачать
                      </a>
                    </div>
                  </div>
                ))}
                {!loading && props.documents.length === 0 ? (
                  <p className="empty">После обработки заявок файлы появятся здесь.</p>
                ) : null}
              </div>
            </article>
          </section>
        ) : null}

        {jobsTab === "create" ? (
          <article className="panel panel-narrow">
            <div className="panel-header">
              <h2>Новая заявка</h2>
              <span>Постановка в очередь на генерацию</span>
            </div>
            <form className="form-grid" onSubmit={(event) => void props.onJobSubmit(event)}>
              <label className="full-width">
                Шаблон
                <select
                  value={props.jobForm.templateId}
                  onChange={(event) =>
                    props.setJobForm((current) => ({ ...current, templateId: event.target.value }))
                  }
                >
                  <option value="">Выберите шаблон</option>
                  {props.templateOptions.map((template) => (
                    <option key={template.id} value={template.id}>
                      {props.translateTemplateName(template.name)} ({template.version})
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Название источника
                <input
                  value={props.jobForm.sourceName}
                  onChange={(event) =>
                    props.setJobForm((current) => ({ ...current, sourceName: event.target.value }))
                  }
                />
              </label>
              <label>
                Инициатор
                <input
                  value={props.jobForm.requestedBy}
                  onChange={(event) =>
                    props.setJobForm((current) => ({ ...current, requestedBy: event.target.value }))
                  }
                />
              </label>
              <label>
                Канал доставки
                <select
                  value={props.jobForm.deliveryChannel}
                  onChange={(event) =>
                    props.setJobForm((current) => ({
                      ...current,
                      deliveryChannel: event.target.value as "internal" | "email" | "bitrix"
                    }))
                  }
                >
                  <option value="internal">Внутренний</option>
                  <option value="email">Email</option>
                  <option value="bitrix">Битрикс24</option>
                </select>
              </label>
              <label>
                Адрес доставки
                <input
                  value={props.jobForm.deliveryAddress}
                  onChange={(event) =>
                    props.setJobForm((current) => ({
                      ...current,
                      deliveryAddress: event.target.value
                    }))
                  }
                />
              </label>
              <label className="full-width">
                Параметры / комментарий
                <textarea
                  value={props.jobForm.payload}
                  onChange={(event) =>
                    props.setJobForm((current) => ({ ...current, payload: event.target.value }))
                  }
                />
              </label>
              <div className="form-actions full-width">
                <button
                  className="primary"
                  type="submit"
                  disabled={!props.readyToSubmitJob || props.creatingJob}
                >
                  {props.creatingJob ? "Создаём…" : "Создать заявку"}
                </button>
              </div>
            </form>
          </article>
        ) : null}
      </Section>

      <Section visible={activeSection === "events"}>
        <SectionTabs
          active={eventsTab}
          onChange={setEventsTab}
          items={[
            { id: "events", label: "События", count: props.events.length },
            { id: "commands", label: "Команды API", count: props.taskCommands.length }
          ]}
        />

        {eventsTab === "events" ? (
          <article className="panel">
            <div className="panel-header">
              <h2>Журнал событий</h2>
              <span>
                {props.events.length} всего · стр. {props.safeEventsPage + 1}/{props.eventsPageCount}
              </span>
            </div>
            <div className="list">
              {props.visibleEvents.map((event) => (
                <div className="list-item stacked" key={event.id}>
                  <div className="list-topline">
                    <div>
                      <strong>{props.translateMessage(event.message)}</strong>
                      <p>{props.translateFromMap(event.eventType, props.labels.eventType)}</p>
                    </div>
                    <div className="meta">
                      <span className={props.statusClass(event.level)}>
                        {props.translateFromMap(event.level, props.labels.eventLevel)}
                      </span>
                      <span>{props.formatDate(event.createdAt)}</span>
                    </div>
                  </div>
                  <p className="subtle">{props.formatEventDetails(event.details || "")}</p>
                </div>
              ))}
              {!loading && props.events.length === 0 ? (
                <p className="empty">Системные события появятся после работы платформы.</p>
              ) : null}
            </div>
            {props.events.length > props.eventsPageSize ? (
              <div className="list-pagination">
                <button
                  type="button"
                  disabled={props.safeEventsPage <= 0}
                  onClick={() => props.setEventsPage((page) => Math.max(0, page - 1))}
                >
                  Назад
                </button>
                <span className="subtle">
                  {props.safeEventsPage * props.eventsPageSize + 1}–
                  {Math.min((props.safeEventsPage + 1) * props.eventsPageSize, props.events.length)} из{" "}
                  {props.events.length}
                </span>
                <button
                  type="button"
                  disabled={props.safeEventsPage >= props.eventsPageCount - 1}
                  onClick={() =>
                    props.setEventsPage((page) => Math.min(props.eventsPageCount - 1, page + 1))
                  }
                >
                  Вперёд
                </button>
              </div>
            ) : null}
          </article>
        ) : null}

        {eventsTab === "commands" ? (
          <article className="panel">
            <div className="panel-header">
              <h2>Команды backend</h2>
              <span>{props.taskCommands.length} зарегистрировано</span>
            </div>
            <p className="subtle panel-intro">
              Действия API: отправка в Bitrix, email-согласование и другие интеграционные команды.
            </p>
            <div className="list">
              {props.taskCommands.map((command) => (
                <div className="list-item stacked" key={command.id}>
                  <div className="list-topline">
                    <div>
                      <strong>
                        {props.translateFromMap(command.targetSystem, props.labels.taskTarget)}
                      </strong>
                      <p>{props.translateCommandText(command.commandText)}</p>
                    </div>
                    <div className="meta">
                      <span className={props.statusClass(command.status)}>
                        {props.translateFromMap(command.status, props.labels.taskCommandStatus)}
                      </span>
                      <span>{props.formatDate(command.createdAt)}</span>
                    </div>
                  </div>
                  <p className="subtle">
                    {props.translateFromMap(command.integrationMode, props.labels.integrationMode)} ·{" "}
                    {props.translateResultMessage(command.resultMessage)}
                  </p>
                </div>
              ))}
              {!loading && props.taskCommands.length === 0 ? (
                <p className="empty">Команды появятся при работе интеграций.</p>
              ) : null}
            </div>
          </article>
        ) : null}
      </Section>

      <Section visible={activeSection === "health"}>
        <section className="content-grid two-columns">
          <article className="panel">
            <div className="panel-header">
              <h2>Backend API</h2>
              <span>{props.health?.service ?? "—"}</span>
            </div>
            <p className="metric">
              {props.translateFromMap(props.health?.status ?? "loading", props.labels.healthStatus)}
            </p>
            <dl className="health-dl">
              <div>
                <dt>Среда</dt>
                <dd>{props.health?.environment ?? "—"}</dd>
              </div>
              <div>
                <dt>База данных</dt>
                <dd>{props.health?.database ?? "—"}</dd>
              </div>
              <div>
                <dt>Uptime</dt>
                <dd>{props.uptimeLabel}</dd>
              </div>
              <div>
                <dt>API-запросов</dt>
                <dd>{props.health?.productRequestsTotal ?? 0}</dd>
              </div>
              <div>
                <dt>Ошибок</dt>
                <dd>{props.health?.errorsTotal ?? 0}</dd>
              </div>
              <div>
                <dt>Заявок создано</dt>
                <dd>{props.health?.jobsCreatedTotal ?? 0}</dd>
              </div>
            </dl>
            <a href={`${props.apiBaseUrl}/api/v1/health`} target="_blank" rel="noreferrer">
              Открыть health endpoint →
            </a>
          </article>

          <article className="panel">
            <div className="panel-header">
              <h2>Метрики</h2>
              <span>{props.metricsSource}</span>
            </div>
            <div className="metrics-grid">
              <div>
                <span>RPS</span>
                <strong>{props.dashboardMetrics.rps}</strong>
              </div>
              <div>
                <span>Ошибки</span>
                <strong>{props.dashboardMetrics.errorRate}</strong>
              </div>
              <div>
                <span>Латентность p95</span>
                <strong>{props.dashboardMetrics.latency}</strong>
              </div>
              <div>
                <span>Доступность</span>
                <strong>{props.dashboardMetrics.availability}</strong>
              </div>
            </div>
            {props.systemLoad ? (
              <div className="health-load-bars">
                <MetricBar label="CPU" value={props.systemLoad.cpu} />
                <MetricBar
                  label="Память"
                  value={props.systemLoad.memory}
                  detail={
                    props.systemLoad.memoryLabel !== "—"
                      ? `${props.systemLoad.memoryLabel} GB`
                      : undefined
                  }
                />
              </div>
            ) : null}
          </article>
        </section>

        <section className="integration-grid">
          {integrations.map((item) => (
            <article className={`integration-card tone-${item.tone}`} key={item.name}>
              <strong>{item.name}</strong>
              <p>{item.status}</p>
              {item.name === "Prometheus" && props.prometheusUrl ? (
                <a href={props.prometheusUrl} target="_blank" rel="noreferrer">
                  Открыть Prometheus
                </a>
              ) : null}
              {item.name === "Grafana" && props.grafanaUrl ? (
                <a href={props.grafanaUrl} target="_blank" rel="noreferrer">
                  Открыть Grafana
                </a>
              ) : null}
            </article>
          ))}
        </section>
      </Section>
    </AdminShell>
  );
}
