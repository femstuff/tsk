import { useMemo, useState, type FormEvent } from "react";

import { DashboardPageView } from "./DashboardPageView";
import { useAdminDashboard } from "../features/admin/useAdminDashboard";
import { useDashboardData } from "../features/dashboard/useDashboardData";
import { useObservabilityMetrics } from "../features/observability/useObservabilityMetrics";
import type { AdminVoiceBitrixResult } from "../entities/document-job/types";
import {
  buildDocumentsTrend,
  buildJobStatusDonut,
  bitrixTaskDonut,
  countWithinDays,
  deriveMetrics,
  deriveSystemLoad,
  type NavSection
} from "../lib/dashboardAnalytics";
import {
  formatPrometheusMetrics,
  formatUptime,
  prometheusHttpSeries,
  prometheusJobDonut,
  prometheusSystemLoad
} from "../lib/observabilityFormat";
import { getApiBaseUrl, runAdminVoiceBitrixPipeline } from "../shared/api/client";

const STATUS_LABELS: Record<string, string> = {
  queued: "В очереди",
  running: "В работе",
  completed: "Завершено",
  failed: "Ошибка",
  cancelled: "Отменено"
};

const DISPATCH_STATUS_LABELS: Record<string, string> = {
  not_required: "Не требуется",
  pending: "Ожидает отправки",
  sent: "Отправлено",
  failed: "Ошибка отправки"
};

const DELIVERY_CHANNEL_LABELS: Record<string, string> = {
  internal: "Внутренний",
  email: "Email",
  bitrix: "Битрикс24"
};

const CATEGORY_LABELS: Record<string, string> = {
  operations: "Операции",
  sales: "Продажи",
  "customer-success": "Сопровождение клиентов",
  general: "Общее"
};

const SOURCE_DOCUMENT_KIND_LABELS: Record<string, string> = {
  voice_recording: "Голосовая запись",
  attachment: "Вложение"
};

const TASK_TARGET_LABELS: Record<string, string> = {
  bitrix24: "Битрикс24",
  email_approval: "Email-согласование"
};

const TASK_COMMAND_STATUS_LABELS: Record<string, string> = {
  recorded: "Зарегистрирована",
  pending: "Ожидает выполнения",
  sent: "Отправлена",
  failed: "Ошибка"
};

const INTEGRATION_MODE_LABELS: Record<string, string> = {
  stub: "Заглушка",
  webhook: "Вебхук"
};

const EVENT_LEVEL_LABELS: Record<string, string> = {
  info: "Инфо",
  warning: "Предупреждение",
  error: "Ошибка"
};

const EVENT_TYPE_LABELS: Record<string, string> = {
  "system.seeded": "Загрузка стартовых шаблонов",
  "template.created": "Шаблон загружен",
  "job.queued": "Заявка поставлена в очередь",
  "source_document.uploaded": "Исходный документ загружен",
  "job.status.updated": "Статус заявки обновлён",
  "job.started": "Обработка заявки начата",
  "document.generated": "Документ сформирован",
  "dispatch.email_approval": "Зафиксировано email-согласование",
  "dispatch.bitrix_fallback": "Запущен резервный сценарий Битрикс",
  "dispatch.bitrix_failed": "Ошибка отправки в Битрикс",
  "dispatch.bitrix_sent": "Отправлено в Битрикс",
  "task_command.recorded": "Команда задачи зарегистрирована",
  "job.failed": "Генерация завершилась ошибкой",
  "voice.transcribed": "Транскрипция голоса",
  "voice.bitrix_skipped": "Bitrix не настроен",
  "voice.bitrix_task_failed": "Ошибка создания задачи Bitrix",
  "voice.bitrix_task_created": "Задача в Bitrix создана",
  "voice.bitrix_no_deal": "Нет ID сделки для Bitrix",
  "voice.bitrix_deal_move_failed": "Ошибка перемещения сделки",
  "voice.bitrix_deal_moved": "Сделка в Bitrix обновлена",
  "voice.bitrix_no_action": "Действие Bitrix не распознано",
  "mobile.bitrix_intent.ok": "Мобильное приложение: Bitrix (успех)",
  "mobile.bitrix_intent.error": "Мобильное приложение: Bitrix (ошибка)",
  "bitrix.oauth.connected": "Авторизация Bitrix24"
};

const VOICE_ACTION_LABELS: Record<string, string> = {
  none: "Не определено",
  move_next: "Следующая стадия сделки",
  move_prev: "Предыдущая стадия",
  move_stage: "Переход на стадию",
  create_task: "Создать задачу"
};

const TEMPLATE_NAME_LABELS: Record<string, string> = {
  "Service Brief": "Служебное резюме",
  "Sales Follow-up": "Повторное коммерческое предложение",
  "Onboarding Pack": "Пакет онбординга клиента"
};

const TEMPLATE_DESCRIPTION_LABELS: Record<string, string> = {
  "Operational summary template for incoming document requests.":
    "Шаблон операционного резюме для входящих запросов на документы.",
  "Follow-up proposal template for sales or Bitrix-driven handoff.":
    "Шаблон последующего предложения для продаж или передачи через Битрикс.",
  "Generate a short operational summary for a client request.":
    "Краткое операционное резюме по клиентскому запросу.",
  "Prepare a follow-up document for a newly qualified lead.":
    "Последующий документ для нового квалифицированного лида.",
  "Assemble the first-touch onboarding template bundle.":
    "Стартовый комплект шаблонов для онбординга клиента."
};

const MESSAGE_LABELS: Record<string, string> = {
  "Seeded default document templates": "Загружены стартовые шаблоны документов",
  "Document job queued for processing": "Заявка на документ поставлена в очередь на обработку",
  "Stored mobile voice recording": "Сохранена голосовая запись из мобильного приложения",
  "Picked queued job for generation": "Заявка из очереди взята в обработку",
  "Generated document artifact": "Сформирован итоговый документ",
  "Email approval flow recorded": "Сценарий email-согласования зарегистрирован",
  "Bitrix webhook not configured, fallback approval recorded":
    "Вебхук Битрикс не настроен, зарегистрирован резервный сценарий согласования",
  "Bitrix card handoff completed": "Передача в Битрикс завершена",
  "Bitrix delivery failed": "Ошибка отправки в Битрикс",
  "Task command stored": "Команда задачи сохранена",
  "Мобильное приложение: Bitrix — выполнено": "Мобильное приложение: запрос в Bitrix выполнен",
  "Мобильное приложение: Bitrix — ошибка": "Мобильное приложение: ошибка запроса в Bitrix",
  "Document generation failed": "Ошибка генерации документа",
  "Marked as failed from admin": "Статус изменён на «Ошибка» из админ-панели",
  "Cancelled from admin": "Заявка отменена из админ-панели"
};

const REQUESTED_BY_LABELS: Record<string, string> = {
  "admin-web": "Админ-панель",
  "mobile-app": "Мобильное приложение"
};

const ORIGIN_LABELS: Record<string, string> = {
  "mobile-app": "Мобильное приложение",
  "admin-web": "Админ-панель"
};

const HEALTH_STATUS_LABELS: Record<string, string> = {
  ok: "Доступен",
  loading: "Загрузка"
};

const EVENTS_PAGE_SIZE = 5;

function translateFromMap(value: string, labels: Record<string, string>) {
  return labels[value] ?? value;
}

function translateTemplateName(value: string) {
  return translateFromMap(value, TEMPLATE_NAME_LABELS);
}

function translateTemplateDescription(value: string) {
  return translateFromMap(value, TEMPLATE_DESCRIPTION_LABELS);
}

function formatEventDetails(details: string) {
  const trimmed = details.trim();
  if (!trimmed) {
    return "Без дополнительных деталей.";
  }
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return trimmed;
    }
  }
  return trimmed;
}

function translateMessage(value: string) {
  const jobStatusMatch = value.match(/^Job status changed to (.+)$/);
  if (jobStatusMatch) {
    return `Статус заявки изменён: ${translateFromMap(jobStatusMatch[1], STATUS_LABELS)}`;
  }

  const uploadedTemplateMatch = value.match(/^Uploaded document template (.+)$/);
  if (uploadedTemplateMatch) {
    return `Загружен шаблон документа ${translateTemplateName(uploadedTemplateMatch[1])}`;
  }

  return translateFromMap(value, MESSAGE_LABELS);
}

function translateCommandText(value: string) {
  const emailApprovalMatch = value.match(/^Approve generated document (.+) for (.+)$/);
  if (emailApprovalMatch) {
    return `Согласовать сформированный документ ${emailApprovalMatch[1]} для ${emailApprovalMatch[2]}`;
  }

  const fallbackApprovalMatch = value.match(
    /^Fallback approval for Bitrix delivery of (.+) to (.+)$/
  );
  if (fallbackApprovalMatch) {
    return `Резервное согласование для отправки в Битрикс: ${fallbackApprovalMatch[1]} -> ${fallbackApprovalMatch[2]}`;
  }

  return value;
}

function translateResultMessage(value: string) {
  const emailApprovalMatch = value.match(/^Email approval flow recorded for (.+)$/);
  if (emailApprovalMatch) {
    return `Сценарий email-согласования зарегистрирован для ${emailApprovalMatch[1]}`;
  }

  return translateFromMap(value, {
    "Command recorded": "Команда зарегистрирована",
    "Command recorded only": "Команда только зарегистрирована",
    "Bitrix task command sent": "Команда отправлена в Битрикс",
    "Bitrix webhook not configured; command recorded":
      "Вебхук Битрикс не настроен; команда только зарегистрирована"
  });
}

function formatDate(value: string | null) {
  if (!value) {
    return "Не запускалось";
  }

  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(new Date(value));
}

function formatBytes(value: number) {
  return new Intl.NumberFormat("ru-RU", {
    maximumFractionDigits: 1
  }).format(value / 1024);
}

function statusClass(status: string) {
  return `status-pill status-${status}`;
}

export function DashboardPage() {
  const {
    health,
    templates,
    jobs,
    documents,
    sourceDocuments,
    taskCommands,
    events,
    loading,
    error,
    creatingTemplate,
    creatingJob,
    updatingJobId,
    refresh,
    submitTemplate,
    submitJob,
    changeJobStatus,
    summary
  } = useDashboardData();
  const {
    snapshot: observability,
    error: observabilityError
  } = useObservabilityMetrics();
  const {
    dashboard: adminDashboard,
    loading: adminDashboardLoading,
    error: adminDashboardError
  } = useAdminDashboard();

  const [templateForm, setTemplateForm] = useState({
    name: "",
    category: "Операции",
    version: "v1",
    description: ""
  });
  const [templateFile, setTemplateFile] = useState<File | null>(null);
  const [jobForm, setJobForm] = useState({
    templateId: "",
    sourceName: "",
    requestedBy: "Админ-панель",
    payload: "",
    deliveryChannel: "internal" as "internal" | "email" | "bitrix",
    deliveryAddress: ""
  });
  const [jobStatusDrafts, setJobStatusDrafts] = useState<Record<string, string>>({});
  const [voiceForm, setVoiceForm] = useState({
    templateId: "",
    sourceName: "",
    dealId: "",
    dealTitle: "",
    dealHint: "",
    stageHint: ""
  });
  const [voiceFile, setVoiceFile] = useState<File | null>(null);
  const [voiceBusy, setVoiceBusy] = useState(false);
  const [voiceResult, setVoiceResult] = useState<AdminVoiceBitrixResult | null>(null);
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const [eventsPage, setEventsPage] = useState(0);
  const [activeSection, setActiveSection] = useState<NavSection>("dashboard");

  const apiBaseUrl = getApiBaseUrl();

  const documentsTrend = useMemo(() => buildDocumentsTrend(jobs), [jobs]);
  const fallbackJobDonut = useMemo(() => buildJobStatusDonut(jobs), [jobs]);
  const prometheusDonut = useMemo(() => prometheusJobDonut(observability), [observability]);
  const bitrixDonut = useMemo(
    () => (adminDashboard ? bitrixTaskDonut(adminDashboard.bitrixTasks) : []),
    [adminDashboard]
  );
  const jobStatusDonut =
    bitrixDonut.length > 0 ? bitrixDonut : prometheusDonut.length > 0 ? prometheusDonut : fallbackJobDonut;
  const dashboardMetrics = useMemo(
    () =>
      observability?.available
        ? formatPrometheusMetrics(observability)
        : deriveMetrics(health),
    [health, observability]
  );
  const httpRateSeries = useMemo(() => prometheusHttpSeries(observability), [observability]);
  const prometheusLoad = useMemo(() => prometheusSystemLoad(observability), [observability]);
  const systemLoad = useMemo(() => {
    const healthUptime = health?.uptimeSeconds ?? 0;
    const promUptime = observability?.uptimeSeconds ?? 0;
    const uptimeSeconds = Math.max(healthUptime, promUptime);

    if (prometheusLoad) {
      return {
        cpu: prometheusLoad.cpu,
        memory: prometheusLoad.memory,
        memoryLabel: prometheusLoad.memoryGb,
        uptimeLabel: formatUptime(uptimeSeconds)
      };
    }
    return {
      cpu: deriveSystemLoad(summary).cpu,
      memory: deriveSystemLoad(summary).memory,
      memoryLabel: "—",
      uptimeLabel: formatUptime(uptimeSeconds)
    };
  }, [health?.uptimeSeconds, observability?.uptimeSeconds, prometheusLoad, summary]);
  const metricsSource = observability?.available ? "Prometheus · live" : "Health endpoint";
  const formatAuthTime = (value: string) => formatDate(value);

  const integrations = useMemo(() => {
    const bitrixActive =
      taskCommands.some((command) => command.integrationMode === "webhook") ||
      events.some((event) => event.eventType.includes("bitrix"));

    return [
      {
        name: "Bitrix24",
        status: bitrixActive ? "Подключено" : "Не настроено",
        tone: bitrixActive ? ("ok" as const) : ("warn" as const)
      },
      {
        name: "Prometheus",
        status: observability?.available ? "Подключено" : "Нет данных",
        tone: observability?.available ? ("ok" as const) : ("warn" as const)
      },
      {
        name: "Grafana",
        status: observability?.available ? "Подключено" : "Нет данных Prometheus",
        tone: observability?.available ? ("ok" as const) : ("warn" as const)
      },
      {
        name: "Go API",
        status: health?.status === "ok" ? "Работает" : "Проверка",
        tone: health?.status === "ok" ? ("ok" as const) : ("warn" as const)
      }
    ];
  }, [events, health?.status, observability?.available, taskCommands]);

  const readyToSubmitTemplate = templateForm.name.trim() !== "" && templateFile !== null;
  const readyToSubmitJob =
    jobForm.templateId.trim() !== "" && jobForm.sourceName.trim() !== "";

  const templateOptions = useMemo(() => templates, [templates]);

  const eventsPageCount = Math.max(1, Math.ceil(events.length / EVENTS_PAGE_SIZE));
  const safeEventsPage = Math.min(eventsPage, eventsPageCount - 1);
  const visibleEvents = useMemo(() => {
    const start = safeEventsPage * EVENTS_PAGE_SIZE;
    return events.slice(start, start + EVENTS_PAGE_SIZE);
  }, [events, safeEventsPage]);

  const handleTemplateSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!templateFile) {
      return;
    }

    const formData = new FormData();
    formData.set("name", templateForm.name);
    formData.set("category", templateForm.category);
    formData.set("version", templateForm.version);
    formData.set("description", templateForm.description);
    formData.set("file", templateFile);

    await submitTemplate(formData);
    setTemplateForm({
      name: "",
      category: "estimate",
      version: "v1",
      description: ""
    });
    setTemplateFile(null);
  };

  const handleJobSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    await submitJob(jobForm);
    setJobForm((current) => ({
      ...current,
      sourceName: "",
      payload: "",
      deliveryAddress: ""
    }));
  };

  const handleVoicePipelineSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setVoiceError(null);
    if (!voiceFile) {
      setVoiceError("Выберите аудиофайл.");
      return;
    }

    setVoiceBusy(true);
    try {
      const formData = new FormData();
      formData.set("audio", voiceFile);
      if (voiceForm.templateId.trim() !== "") {
        formData.set("templateId", voiceForm.templateId.trim());
      }
      if (voiceForm.sourceName.trim() !== "") {
        formData.set("sourceName", voiceForm.sourceName.trim());
      }
      if (voiceForm.dealId.trim() !== "") {
        formData.set("dealId", voiceForm.dealId.trim());
      }
      if (voiceForm.dealTitle.trim() !== "") {
        formData.set("dealTitle", voiceForm.dealTitle.trim());
      }
      if (voiceForm.dealHint.trim() !== "") {
        formData.set("dealHint", voiceForm.dealHint.trim());
      }
      if (voiceForm.stageHint.trim() !== "") {
        formData.set("stageHint", voiceForm.stageHint.trim());
      }

      const item = await runAdminVoiceBitrixPipeline(formData);
      setVoiceResult(item);
      await refresh();
    } catch (err) {
      setVoiceResult(null);
      setVoiceError(err instanceof Error ? err.message : "Не удалось выполнить цепочку");
    } finally {
      setVoiceBusy(false);
    }
  };

  return (
    <DashboardPageView
      activeSection={activeSection}
      onNavigate={setActiveSection}
      onRefresh={() => void refresh()}
      loading={loading || adminDashboardLoading}
      error={error ?? adminDashboardError}
      integrations={integrations}
      overviewProps={{
        documentsTotal: summary.documentCount,
        documentsWeekDelta: countWithinDays(documents, 7),
        tasksTotal: adminDashboard?.bitrixTasks.total ?? 0,
        tasksWeekDelta: 0,
        tasksLabel: adminDashboard
          ? `${adminDashboard.bitrixTasks.totalOpen} открыто · ${adminDashboard.bitrixTasks.inProgress} в работе`
          : "загрузка…",
        usersTotal: adminDashboard?.authorizedUsers ?? 0,
        usersWeekDelta: 0,
        usersLabel: "в Bitrix24 (OAuth)",
        activityToday: adminDashboard?.voiceActivityToday ?? 0,
        activityLabel: adminDashboard
          ? `${adminDashboard.voiceActivityWeek} за 7 дней (голос + Bitrix)`
          : "голосовых действий",
        documentsTrend,
        jobStatusDonut,
        bitrixTaskItems: adminDashboard?.bitrixTaskItems ?? [],
        formatTaskDeadline: (value) => (value?.trim() ? formatDate(value) : "—"),
        httpRateSeries,
        metrics: dashboardMetrics,
        systemLoad,
        metricsSource,
        observabilityError: observabilityError ?? adminDashboardError,
        recentAuth: adminDashboard?.recentAuth ?? [],
        formatAuthTime
      }}
      templates={templates}
      jobs={jobs}
      documents={documents}
      sourceDocuments={sourceDocuments}
      taskCommands={taskCommands}
      events={events}
      visibleEvents={visibleEvents}
      safeEventsPage={safeEventsPage}
      eventsPageCount={eventsPageCount}
      setEventsPage={setEventsPage}
      health={health}
      dashboardMetrics={dashboardMetrics}
      authorizedUsers={adminDashboard?.users ?? []}
      apiBaseUrl={apiBaseUrl}
      templateForm={templateForm}
      setTemplateForm={setTemplateForm}
      templateFile={templateFile}
      setTemplateFile={setTemplateFile}
      readyToSubmitTemplate={readyToSubmitTemplate}
      creatingTemplate={creatingTemplate}
      onTemplateSubmit={handleTemplateSubmit}
      jobForm={jobForm}
      setJobForm={setJobForm}
      readyToSubmitJob={readyToSubmitJob}
      creatingJob={creatingJob}
      onJobSubmit={handleJobSubmit}
      templateOptions={templateOptions}
      jobStatusDrafts={jobStatusDrafts}
      setJobStatusDrafts={setJobStatusDrafts}
      updatingJobId={updatingJobId}
      onChangeJobStatus={(jobId, status) => void changeJobStatus(jobId, status)}
      voiceForm={voiceForm}
      setVoiceForm={setVoiceForm}
      voiceFile={voiceFile}
      setVoiceFile={setVoiceFile}
      voiceBusy={voiceBusy}
      voiceResult={voiceResult}
      voiceError={voiceError}
      onVoicePipelineSubmit={handleVoicePipelineSubmit}
      translateTemplateName={translateTemplateName}
      translateTemplateDescription={translateTemplateDescription}
      translateFromMap={translateFromMap}
      translateMessage={translateMessage}
      translateCommandText={translateCommandText}
      translateResultMessage={translateResultMessage}
      formatDate={formatDate}
      formatBytes={formatBytes}
      formatEventDetails={formatEventDetails}
      statusClass={statusClass}
      labels={{
        status: STATUS_LABELS,
        category: CATEGORY_LABELS,
        dispatchStatus: DISPATCH_STATUS_LABELS,
        deliveryChannel: DELIVERY_CHANNEL_LABELS,
        requestedBy: REQUESTED_BY_LABELS,
        origin: ORIGIN_LABELS,
        sourceDocumentKind: SOURCE_DOCUMENT_KIND_LABELS,
        taskTarget: TASK_TARGET_LABELS,
        taskCommandStatus: TASK_COMMAND_STATUS_LABELS,
        integrationMode: INTEGRATION_MODE_LABELS,
        eventLevel: EVENT_LEVEL_LABELS,
        eventType: EVENT_TYPE_LABELS,
        healthStatus: HEALTH_STATUS_LABELS,
        voiceAction: VOICE_ACTION_LABELS
      }}
      eventsPageSize={EVENTS_PAGE_SIZE}
    />
  );
}
