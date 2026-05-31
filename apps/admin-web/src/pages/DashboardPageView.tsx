import type { FormEvent, ReactNode } from "react";

import { DashboardOverview } from "../components/dashboard/DashboardOverview";
import { AdminShell } from "../components/layout/AdminShell";
import type { AdminBitrixUser } from "../entities/admin/types";
import type { AdminVoiceBitrixResult, DocumentJob, DocumentTemplate, GeneratedDocument, HealthResponse, ProcessingEvent, SourceDocument, TaskCommand } from "../entities/document-job/types";
import type { NavSection } from "../lib/dashboardAnalytics";

type IntegrationItem = {
  name: string;
  status: string;
  tone: "ok" | "warn" | "muted";
};

type DashboardPageViewProps = {
  activeSection: NavSection;
  onNavigate: (section: NavSection) => void;
  onRefresh: () => void;
  loading: boolean;
  error: string | null;
  integrations: IntegrationItem[];
  overviewProps: React.ComponentProps<typeof DashboardOverview>;
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
  voiceForm: {
    templateId: string;
    sourceName: string;
    dealId: string;
    dealTitle: string;
    dealHint: string;
    stageHint: string;
  };
  setVoiceForm: React.Dispatch<
    React.SetStateAction<{
      templateId: string;
      sourceName: string;
      dealId: string;
      dealTitle: string;
      dealHint: string;
      stageHint: string;
    }>
  >;
  voiceFile: File | null;
  setVoiceFile: (file: File | null) => void;
  voiceBusy: boolean;
  voiceResult: AdminVoiceBitrixResult | null;
  voiceError: string | null;
  onVoicePipelineSubmit: (event: FormEvent<HTMLFormElement>) => void;
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
    voiceAction: Record<string, string>;
  };
  eventsPageSize: number;
};

function Section({ visible, children }: { visible: boolean; children: ReactNode }) {
  if (!visible) {
    return null;
  }
  return <>{children}</>;
}

export function DashboardPageView(props: DashboardPageViewProps) {
  const {
    activeSection,
    onNavigate,
    onRefresh,
    loading,
    error,
    integrations,
    overviewProps
  } = props;

  return (
    <AdminShell
      activeSection={activeSection}
      onNavigate={onNavigate}
      onRefresh={onRefresh}
      loading={loading}
      integrations={integrations}
    >
      {error ? <p className="banner error">{error}</p> : null}

      <Section visible={activeSection === "dashboard"}>
        <DashboardOverview {...overviewProps} />
      </Section>

      <Section visible={activeSection === "templates"}>
        <section className="content-grid two-columns">
          <article className="panel">
            <div className="panel-header">
              <h2>Загрузка шаблона</h2>
              <span>Постоянное хранилище шаблонов</span>
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
                    props.setTemplateForm((current) => ({
                      ...current,
                      category: event.target.value
                    }))
                  }
                />
              </label>
              <label>
                Версия
                <input
                  value={props.templateForm.version}
                  onChange={(event) =>
                    props.setTemplateForm((current) => ({
                      ...current,
                      version: event.target.value
                    }))
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
                Файл шаблона
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
                  {props.creatingTemplate ? "Загрузка..." : "Загрузить шаблон"}
                </button>
              </div>
            </form>
          </article>

          <article className="panel">
            <div className="panel-header">
              <h2>Шаблоны документов</h2>
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
                <p className="empty">Загрузите первый шаблон, чтобы создавать заявки.</p>
              ) : null}
            </div>
          </article>
        </section>
      </Section>

      <Section visible={activeSection === "documents"}>
        <section className="content-grid two-columns">
          <article className="panel">
            <div className="panel-header">
              <h2>Заявки на документы</h2>
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
                      Инициатор:{" "}
                      {props.translateFromMap(job.requestedBy, props.labels.requestedBy)}
                    </span>
                    <span>
                      Доставка:{" "}
                      {props.translateFromMap(job.deliveryChannel, props.labels.deliveryChannel)}
                    </span>
                    <span>
                      Статус отправки:{" "}
                      {props.translateFromMap(job.dispatchStatus, props.labels.dispatchStatus)}
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
                      {props.updatingJobId === job.id ? "Обновляем..." : "Применить статус"}
                    </button>
                  </div>
                </div>
              ))}
              {!loading && props.jobs.length === 0 ? (
                <p className="empty">Создайте первую заявку, чтобы проверить генерацию.</p>
              ) : null}
            </div>
          </article>

          <article className="panel">
            <div className="panel-header">
              <h2>Исходные документы</h2>
              <span>{props.sourceDocuments.length} сохранённых загрузок</span>
            </div>
            <div className="list">
              {props.sourceDocuments.map((document) => (
                <div className="list-item" key={document.id}>
                  <div>
                    <strong>{document.fileName}</strong>
                    <p>{props.translateFromMap(document.origin, props.labels.origin)}</p>
                    <p className="subtle">
                      {props.translateFromMap(document.kind, props.labels.sourceDocumentKind)} ·
                      заявка {document.jobId ?? "-"} · {props.formatBytes(document.sizeBytes)} KB
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
                <p className="empty">
                  Здесь появятся загрузки из мобильного приложения и аудиофайлы.
                </p>
              ) : null}
            </div>
          </article>

          <article className="panel">
            <div className="panel-header">
              <h2>Сгенерированные документы</h2>
              <span>{props.documents.length} сохранённых файлов</span>
            </div>
            <div className="list">
              {props.documents.map((document) => (
                <div className="list-item" key={document.id}>
                  <div>
                    <strong>{document.fileName}</strong>
                    <p>{props.translateTemplateName(document.templateName)}</p>
                    <p className="subtle">
                      Заявка: {document.jobId} · {props.formatBytes(document.sizeBytes)} KB
                    </p>
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
                <p className="empty">После обработки здесь появятся сгенерированные документы.</p>
              ) : null}
            </div>
          </article>
        </section>
      </Section>

      <Section visible={activeSection === "tasks"}>
        <article className="panel">
          <div className="panel-header">
            <h2>Команды backend</h2>
            <span>{props.taskCommands.length} зарегистрировано</span>
          </div>
          <p className="subtle panel-intro">
            Запросы и действия, которые приходят на API и выполняются системой (Bitrix, email и др.) — не задачи Bitrix24.
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
              <p className="empty">Здесь появятся команды для Битрикс и email-согласования.</p>
            ) : null}
          </div>
        </article>
      </Section>

      <Section visible={activeSection === "users"}>
        <section className="user-grid">
          {props.authorizedUsers.map((user) => (
            <details className="user-card" key={user.sessionId}>
              <summary className="user-card-summary">
                <div>
                  <strong>{user.userName || `id ${user.bitrixUserId}`}</strong>
                  <p className="subtle">Bitrix ID {user.bitrixUserId}</p>
                </div>
                <span className="user-card-meta">{user.taskCount} задач</span>
              </summary>
              <dl className="user-details">
                <div>
                  <dt>ID сессии</dt>
                  <dd>{user.sessionId}</dd>
                </div>
                <div>
                  <dt>Bitrix user ID</dt>
                  <dd>{user.bitrixUserId}</dd>
                </div>
                <div>
                  <dt>Имя</dt>
                  <dd>{user.userName || "—"}</dd>
                </div>
                <div>
                  <dt>Портал</dt>
                  <dd>{user.portalDomain}</dd>
                </div>
                <div>
                  <dt>OAuth scopes</dt>
                  <dd>{user.oauthScopes || "—"}</dd>
                </div>
                <div>
                  <dt>Задач в Bitrix</dt>
                  <dd>{user.taskCount}</dd>
                </div>
                <div>
                  <dt>Статус</dt>
                  <dd>{user.status === "active" ? "Активна" : user.status}</dd>
                </div>
                <div>
                  <dt>Подключён</dt>
                  <dd>{props.formatDate(user.connectedAt)}</dd>
                </div>
                <div>
                  <dt>Последняя активность</dt>
                  <dd>{props.formatDate(user.lastActiveAt)}</dd>
                </div>
                <div>
                  <dt>Токен истекает</dt>
                  <dd>{props.formatDate(user.expiresAt ?? null)}</dd>
                </div>
              </dl>
            </details>
          ))}
          {!loading && props.authorizedUsers.length === 0 ? (
            <p className="empty">Авторизованные пользователи Bitrix24 появятся после входа в мобильном приложении.</p>
          ) : null}
        </section>
      </Section>

      <Section visible={activeSection === "activity"}>
        <article className="panel">
          <div className="panel-header">
            <h2>Журнал событий</h2>
            <span>
              {props.events.length} всего · страница {props.safeEventsPage + 1} из{" "}
              {props.eventsPageCount}
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
              <p className="empty">Здесь появятся системные события и события заявок.</p>
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
                {Math.min((props.safeEventsPage + 1) * props.eventsPageSize, props.events.length)}{" "}
                из {props.events.length}
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
      </Section>

      <Section visible={activeSection === "metrics"}>
        <section className="content-grid two-columns">
          <article className="panel">
            <div className="panel-header">
              <h2>Статус backend</h2>
              <span>{props.health?.service ?? "—"}</span>
            </div>
            <p className="metric">
              {props.translateFromMap(props.health?.status ?? "loading", props.labels.healthStatus)}
            </p>
            <p>Среда: {props.health?.environment ?? "-"}</p>
            <p>База данных: {props.health?.database ?? "-"}</p>
            <p>Продуктовых API-запросов: {props.health?.productRequestsTotal ?? 0}</p>
            <p>Ошибок: {props.health?.errorsTotal ?? 0}</p>
            <p>Заявок создано: {props.health?.jobsCreatedTotal ?? 0}</p>
            <p className="subtle">
              Сырой HTTP total: {props.health?.httpRequestsTotalRaw ?? 0} · uptime:{" "}
              {props.health?.uptimeSeconds ?? 0} сек
            </p>
          </article>
          <article className="panel">
            <div className="panel-header">
              <h2>Операционные метрики</h2>
              <span>Расчёт из health</span>
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
          </article>
        </section>
      </Section>

      <Section visible={activeSection === "integrations"}>
        <section className="integration-grid">
          {integrations.map((item) => (
            <article className="integration-card" key={item.name}>
              <strong>{item.name}</strong>
              <p>{item.status}</p>
              {item.name === "Prometheus" ? (
                <a href="http://localhost:9090" target="_blank" rel="noreferrer">
                  Открыть Prometheus
                </a>
              ) : null}
              {item.name === "Grafana" ? (
                <a href="http://localhost:3000" target="_blank" rel="noreferrer">
                  Открыть Grafana
                </a>
              ) : null}
              {item.name === "Go API" ? (
                <a href={`${props.apiBaseUrl}/api/v1/health`} target="_blank" rel="noreferrer">
                  Health endpoint
                </a>
              ) : null}
            </article>
          ))}
        </section>
      </Section>

      <Section visible={activeSection === "settings"}>
        <section className="content-grid two-columns">
          <article className="panel">
            <div className="panel-header">
              <h2>Создание заявки</h2>
              <span>Постановка заявки на генерацию в очередь</span>
            </div>
            <form className="form-grid" onSubmit={(event) => void props.onJobSubmit(event)}>
              <label className="full-width">
                Шаблон
                <select
                  value={props.jobForm.templateId}
                  onChange={(event) =>
                    props.setJobForm((current) => ({
                      ...current,
                      templateId: event.target.value
                    }))
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
                    props.setJobForm((current) => ({
                      ...current,
                      requestedBy: event.target.value
                    }))
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
                Параметры / комментарий к заявке
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
                  {props.creatingJob ? "Создаём..." : "Создать заявку"}
                </button>
              </div>
            </form>
          </article>

          <details className="panel voice-pipeline-collapsible">
            <summary>Тест: голос → Whisper → Bitrix (для разработки)</summary>
            <div className="voice-pipeline-body">
              <p className="muted">
                Служебный контур для проверки транскрипции и команд Bitrix. Загрузите аудио —
                backend вызовет <code>/transcribe</code> и попытается выполнить действие в Bitrix24
                при заданном <code>BITRIX_WEBHOOK_URL</code>.
              </p>
              {props.voiceError ? <p className="banner error">{props.voiceError}</p> : null}
              <form
                className="stacked-form"
                onSubmit={(event) => void props.onVoicePipelineSubmit(event)}
              >
                <label>
                  Шаблон (необязательно)
                  <select
                    value={props.voiceForm.templateId}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({
                        ...current,
                        templateId: event.target.value
                      }))
                    }
                  >
                    <option value="">По умолчанию (первый шаблон)</option>
                    {props.templateOptions.map((template) => (
                      <option key={template.id} value={template.id}>
                        {props.translateTemplateName(template.name)}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Название источника (необязательно)
                  <input
                    value={props.voiceForm.sourceName}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({
                        ...current,
                        sourceName: event.target.value
                      }))
                    }
                    placeholder="Тест голоса из админки"
                  />
                </label>
                <label>
                  Сделка (номер или название, необязательно)
                  <input
                    value={props.voiceForm.dealHint}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({ ...current, dealHint: event.target.value }))
                    }
                    placeholder="123 или ООО Ромашка"
                  />
                </label>
                <label>
                  Целевая стадия (необязательно)
                  <input
                    value={props.voiceForm.stageHint}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({
                        ...current,
                        stageHint: event.target.value
                      }))
                    }
                    placeholder="Квалификация"
                  />
                </label>
                <label>
                  ID сделки Bitrix (необязательно)
                  <input
                    value={props.voiceForm.dealId}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({ ...current, dealId: event.target.value }))
                    }
                    placeholder="123"
                    inputMode="numeric"
                  />
                </label>
                <label>
                  Название сделки (необязательно)
                  <input
                    value={props.voiceForm.dealTitle}
                    onChange={(event) =>
                      props.setVoiceForm((current) => ({
                        ...current,
                        dealTitle: event.target.value
                      }))
                    }
                    placeholder="ТЕСТОВЫЙ ПРОЕКТ 2024"
                  />
                </label>
                <label>
                  Аудиофайл
                  <input
                    type="file"
                    accept="audio/*,.m4a,.ogg,.oga,.wav,.mp3,.aac,.flac"
                    onChange={(event) => props.setVoiceFile(event.target.files?.[0] ?? null)}
                    required
                  />
                </label>
                <button type="submit" disabled={props.voiceBusy}>
                  {props.voiceBusy ? "Обработка…" : "Запустить цепочку"}
                </button>
              </form>
              {props.voiceResult ? (
                <div className="voice-result">
                  <h3>Результат</h3>
                  <p>
                    <strong>Транскрипт:</strong> {props.voiceResult.transcript}
                  </p>
                  <p>
                    <strong>Распознанное действие:</strong>{" "}
                    {props.translateFromMap(
                      props.voiceResult.parsedAction,
                      props.labels.voiceAction
                    )}
                  </p>
                  <p>
                    <strong>ID сделки:</strong>{" "}
                    {props.voiceResult.parsedDealId > 0 ? props.voiceResult.parsedDealId : "—"}
                  </p>
                  <p>
                    <strong>Название сделки:</strong>{" "}
                    {props.voiceResult.parsedDealTitle ? props.voiceResult.parsedDealTitle : "—"}
                  </p>
                  <p>
                    <strong>Bitrix:</strong>{" "}
                    {props.voiceResult.bitrixConfigured ? "вебхук задан" : "вебхук не задан"}
                  </p>
                  <ul>
                    {props.voiceResult.bitrixSteps.map((step) => (
                      <li key={step}>{step}</li>
                    ))}
                  </ul>
                  <p className="muted">
                    Заявка <code>{props.voiceResult.job.id}</code>, файл{" "}
                    <code>{props.voiceResult.sourceDocument.id}</code>
                  </p>
                </div>
              ) : null}
            </div>
          </details>
        </section>
      </Section>
    </AdminShell>
  );
}
