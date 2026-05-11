import { useMemo, useState, type FormEvent } from "react";

import { useDashboardData } from "../features/dashboard/useDashboardData";
import type { AdminVoiceBitrixResult } from "../entities/document-job/types";
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
  "mobile.bitrix_intent.error": "Мобильное приложение: Bitrix (ошибка)"
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

  const apiBaseUrl = getApiBaseUrl();

  const readyToSubmitTemplate = templateForm.name.trim() !== "" && templateFile !== null;
  const readyToSubmitJob =
    jobForm.templateId.trim() !== "" && jobForm.sourceName.trim() !== "";

  const templateOptions = useMemo(() => templates, [templates]);

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
      category: "Операции",
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
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Админ-панель TSK</p>
          <h1>Управление шаблонами, заявками, документами и журналом событий</h1>
          <p className="hero-copy">
            Панель работает с постоянным хранилищем шаблонов, заявками в Postgres,
            сформированными документами и операционным журналом обработки.
          </p>
        </div>

        <div className="hero-actions">
          <button onClick={() => void refresh()} disabled={loading}>
            Обновить
          </button>
        </div>
      </section>

      {error ? <p className="banner error">{error}</p> : null}

      <section className="card voice-pipeline-card">
        <h2>Голос → Whisper → Bitrix (тест из админки)</h2>
        <p className="muted">
          Загрузите аудио (m4a, ogg, wav и т.д.). Backend отправит файл в Python-сервис{" "}
          <code>/transcribe</code>, сохранит транскрипт в заявке и попытается выполнить действие в
          Bitrix24, если в корне проекта в файле <code>.env</code> задан{" "}
          <code>BITRIX_WEBHOOK_URL</code> (входящий вебхук REST). Примеры: «сделка 123 следующий
          этап», «сделку «ТЕСТОВЫЙ ПРОЕКТ» следующий этап», «создай задачу: позвонить клиенту».
          Шаблон и название источника можно не указывать (подставится первый шаблон и подпись по
          умолчанию). Поля «какую сделку» и «куда» — текстовые подсказки к парсеру, необязательны.
        </p>
        {voiceError ? <p className="banner error">{voiceError}</p> : null}
        <form className="stacked-form" onSubmit={(e) => void handleVoicePipelineSubmit(e)}>
          <label>
            Шаблон (для учётной заявки, необязательно — первый из списка)
            <select
              value={voiceForm.templateId}
              onChange={(e) => setVoiceForm((c) => ({ ...c, templateId: e.target.value }))}
            >
              <option value="">По умолчанию (первый шаблон)</option>
              {templateOptions.map((t) => (
                <option key={t.id} value={t.id}>
                  {translateTemplateName(t.name)}
                </option>
              ))}
            </select>
          </label>
          <label>
            Название источника (необязательно)
            <input
              value={voiceForm.sourceName}
              onChange={(e) => setVoiceForm((c) => ({ ...c, sourceName: e.target.value }))}
              placeholder="По умолчанию: «Тест голоса из админки»"
            />
          </label>
          <label>
            Какую сделку (текстом, необязательно) — номер или название, если не из голоса
            <input
              value={voiceForm.dealHint}
              onChange={(e) => setVoiceForm((c) => ({ ...c, dealHint: e.target.value }))}
              placeholder="Например: 123 или ООО Ромашка"
            />
          </label>
          <label>
            Куда / целевая стадия (текстом, необязательно) — для команды «на стадию …»
            <input
              value={voiceForm.stageHint}
              onChange={(e) => setVoiceForm((c) => ({ ...c, stageHint: e.target.value }))}
              placeholder="Например: Квалификация"
            />
          </label>
          <label>
            ID сделки Bitrix (необязательно, если произнесёте номер в аудио)
            <input
              value={voiceForm.dealId}
              onChange={(e) => setVoiceForm((c) => ({ ...c, dealId: e.target.value }))}
              placeholder="123"
              inputMode="numeric"
            />
          </label>
          <label>
            Название сделки (если в голосе не распозналось — точное или часть названия в CRM)
            <input
              value={voiceForm.dealTitle}
              onChange={(e) => setVoiceForm((c) => ({ ...c, dealTitle: e.target.value }))}
              placeholder="Например: ТЕСТОВЫЙ ПРОЕКТ 2024"
            />
          </label>
          <label>
            Аудиофайл
            <input
              type="file"
              accept="audio/*,.m4a,.ogg,.oga,.wav,.mp3,.aac,.flac"
              onChange={(e) => setVoiceFile(e.target.files?.[0] ?? null)}
              required
            />
          </label>
          <button type="submit" disabled={voiceBusy}>
            {voiceBusy ? "Обработка…" : "Запустить цепочку"}
          </button>
        </form>
        {voiceResult ? (
          <div className="voice-result">
            <h3>Результат</h3>
            <p>
              <strong>Транскрипт:</strong> {voiceResult.transcript}
            </p>
            <p>
              <strong>Распознанное действие:</strong>{" "}
              {translateFromMap(voiceResult.parsedAction, VOICE_ACTION_LABELS)}
            </p>
            <p>
              <strong>ID сделки (из текста/формы):</strong>{" "}
              {voiceResult.parsedDealId > 0 ? voiceResult.parsedDealId : "—"}
            </p>
            <p>
              <strong>Название сделки (распознано/подобрано):</strong>{" "}
              {voiceResult.parsedDealTitle ? voiceResult.parsedDealTitle : "—"}
            </p>
            <p>
              <strong>Bitrix:</strong> {voiceResult.bitrixConfigured ? "вебхук задан" : "вебхук не задан"}
            </p>
            <ul>
              {voiceResult.bitrixSteps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ul>
            <p className="muted">
              Заявка <code>{voiceResult.job.id}</code>, исходный файл <code>{voiceResult.sourceDocument.id}</code>
              . Смотрите журнал событий ниже.
            </p>
          </div>
        ) : null}
      </section>

      <section className="grid">
        <article className="card">
          <h2>Статус backend</h2>
          <p className="metric">
            {translateFromMap(health?.status ?? "loading", HEALTH_STATUS_LABELS)}
          </p>
          <p>{health?.service ?? "Ожидание ответа от backend..."}</p>
          <p>Среда: {health?.environment ?? "-"}</p>
          <p>Продуктовых API-запросов: {health?.productRequestsTotal ?? 0}</p>
          <p className="subtle">
            Сырой HTTP total с health, `/metrics` и admin polling:{" "}
            {health?.httpRequestsTotalRaw ?? 0}
          </p>
        </article>

        <article className="card">
          <h2>Шаблоны</h2>
          <p className="metric">{summary.templateCount}</p>
          <p>Загружено в постоянное хранилище</p>
        </article>

        <article className="card">
          <h2>Активные заявки</h2>
          <p className="metric">{summary.activeCount}</p>
          <p>{summary.jobCount} всего</p>
        </article>

        <article className="card">
          <h2>Исходные файлы</h2>
          <p className="metric">{summary.sourceDocumentCount}</p>
          <p>{summary.taskCommandCount} команд задач</p>
        </article>
      </section>

      <section className="content-grid two-columns">
        <article className="panel">
          <div className="panel-header">
            <h2>Загрузка шаблона</h2>
            <span>Постоянное хранилище шаблонов</span>
          </div>

          <form className="form-grid" onSubmit={(event) => void handleTemplateSubmit(event)}>
            <label>
              Название
              <input
                value={templateForm.name}
                onChange={(event) =>
                  setTemplateForm((current) => ({
                    ...current,
                    name: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Категория
              <input
                value={templateForm.category}
                onChange={(event) =>
                  setTemplateForm((current) => ({
                    ...current,
                    category: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Версия
              <input
                value={templateForm.version}
                onChange={(event) =>
                  setTemplateForm((current) => ({
                    ...current,
                    version: event.target.value
                  }))
                }
              />
            </label>
            <label className="full-width">
              Описание
              <textarea
                value={templateForm.description}
                onChange={(event) =>
                  setTemplateForm((current) => ({
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
                onChange={(event) => setTemplateFile(event.target.files?.[0] ?? null)}
              />
            </label>
            <div className="form-actions full-width">
              <button
                className="primary"
                type="submit"
                disabled={!readyToSubmitTemplate || creatingTemplate}
              >
                {creatingTemplate ? "Загрузка..." : "Загрузить шаблон"}
              </button>
            </div>
          </form>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>Создание заявки</h2>
            <span>Постановка заявки на генерацию в очередь</span>
          </div>

          <form className="form-grid" onSubmit={(event) => void handleJobSubmit(event)}>
            <label className="full-width">
              Шаблон
              <select
                value={jobForm.templateId}
                onChange={(event) =>
                  setJobForm((current) => ({
                    ...current,
                    templateId: event.target.value
                  }))
                }
              >
                <option value="">Выберите шаблон</option>
                {templateOptions.map((template) => (
                  <option key={template.id} value={template.id}>
                    {translateTemplateName(template.name)} ({template.version})
                  </option>
                ))}
              </select>
            </label>
            <label>
              Название источника
              <input
                value={jobForm.sourceName}
                onChange={(event) =>
                  setJobForm((current) => ({
                    ...current,
                    sourceName: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Инициатор
              <input
                value={jobForm.requestedBy}
                onChange={(event) =>
                  setJobForm((current) => ({
                    ...current,
                    requestedBy: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Канал доставки
              <select
                value={jobForm.deliveryChannel}
                onChange={(event) =>
                  setJobForm((current) => ({
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
                value={jobForm.deliveryAddress}
                onChange={(event) =>
                  setJobForm((current) => ({
                    ...current,
                    deliveryAddress: event.target.value
                  }))
                }
              />
            </label>
            <label className="full-width">
              Параметры / комментарий к заявке
              <textarea
                value={jobForm.payload}
                onChange={(event) =>
                  setJobForm((current) => ({
                    ...current,
                    payload: event.target.value
                  }))
                }
              />
            </label>
            <div className="form-actions full-width">
              <button
                className="primary"
                type="submit"
                disabled={!readyToSubmitJob || creatingJob}
              >
                {creatingJob ? "Создаём..." : "Создать заявку"}
              </button>
            </div>
          </form>
        </article>
      </section>

      <section className="content-grid two-columns">
        <article className="panel">
          <div className="panel-header">
            <h2>Шаблоны документов</h2>
            <span>{templates.length} доступно</span>
          </div>

          <div className="list">
            {templates.map((template) => (
              <div className="list-item" key={template.id}>
                <div>
                  <strong>{translateTemplateName(template.name)}</strong>
                  <p>{translateTemplateDescription(template.description)}</p>
                  <p className="subtle">
                    {template.fileName} · {formatBytes(template.sizeBytes)} KB
                  </p>
                </div>
                <div className="meta">
                  <span>{translateFromMap(template.category, CATEGORY_LABELS)}</span>
                  <span>{template.version}</span>
                  <a
                    href={`${apiBaseUrl}/api/v1/document-templates/${template.id}/download`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Скачать
                  </a>
                </div>
              </div>
            ))}
            {!loading && templates.length === 0 ? (
              <p className="empty">Загрузите первый шаблон, чтобы создавать заявки.</p>
            ) : null}
          </div>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>Заявки на документы</h2>
            <span>{jobs.length} всего</span>
          </div>

          <div className="list">
            {jobs.map((job) => (
              <div className="list-item stacked" key={job.id}>
                <div className="list-topline">
                  <div>
                    <strong>{translateTemplateName(job.templateName)}</strong>
                    <p>{job.sourceName}</p>
                  </div>
                  <div className="meta">
                    <span className={statusClass(job.status)}>
                      {translateFromMap(job.status, STATUS_LABELS)}
                    </span>
                    <span>{formatDate(job.createdAt)}</span>
                  </div>
                </div>

                <div className="job-details">
                  <span>
                    Инициатор: {translateFromMap(job.requestedBy, REQUESTED_BY_LABELS)}
                  </span>
                  <span>
                    Доставка: {translateFromMap(job.deliveryChannel, DELIVERY_CHANNEL_LABELS)}
                  </span>
                  <span>
                    Статус отправки:{" "}
                    {translateFromMap(job.dispatchStatus, DISPATCH_STATUS_LABELS)}
                  </span>
                  <span>Завершено: {formatDate(job.completedAt)}</span>
                  {job.errorMessage ? <span>Ошибка: {translateMessage(job.errorMessage)}</span> : null}
                </div>

                <p className="payload-preview">{job.payload || "Параметры не указаны."}</p>

                <div className="job-actions">
                  <select
                    value={jobStatusDrafts[job.id] ?? job.status}
                    onChange={(event) =>
                      setJobStatusDrafts((current) => ({
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
                      void changeJobStatus(job.id, jobStatusDrafts[job.id] ?? job.status)
                    }
                    disabled={updatingJobId === job.id}
                  >
                    {updatingJobId === job.id ? "Обновляем..." : "Применить статус"}
                  </button>
                </div>
              </div>
            ))}
            {!loading && jobs.length === 0 ? (
              <p className="empty">Создайте первую заявку, чтобы проверить генерацию.</p>
            ) : null}
          </div>
        </article>
      </section>

      <section className="content-grid two-columns">
        <article className="panel">
          <div className="panel-header">
            <h2>Исходные документы</h2>
            <span>{sourceDocuments.length} сохранённых загрузок</span>
          </div>

          <div className="list">
            {sourceDocuments.map((document) => (
              <div className="list-item" key={document.id}>
                <div>
                  <strong>{document.fileName}</strong>
                  <p>{translateFromMap(document.origin, ORIGIN_LABELS)}</p>
                  <p className="subtle">
                    {translateFromMap(document.kind, SOURCE_DOCUMENT_KIND_LABELS)} · заявка{" "}
                    {document.jobId ?? "-"} ·{" "}
                    {formatBytes(document.sizeBytes)} KB
                  </p>
                </div>
                <div className="meta">
                  <span>{formatDate(document.createdAt)}</span>
                  <a
                    href={`${apiBaseUrl}/api/v1/source-documents/${document.id}/download`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Скачать
                  </a>
                </div>
              </div>
            ))}
            {!loading && sourceDocuments.length === 0 ? (
              <p className="empty">
                Здесь появятся загрузки из мобильного приложения и аудиофайлы.
              </p>
            ) : null}
          </div>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>Сгенерированные документы</h2>
            <span>{documents.length} сохранённых файлов</span>
          </div>

          <div className="list">
            {documents.map((document) => (
              <div className="list-item" key={document.id}>
                <div>
                  <strong>{document.fileName}</strong>
                  <p>{translateTemplateName(document.templateName)}</p>
                  <p className="subtle">
                    Заявка: {document.jobId} · {formatBytes(document.sizeBytes)} KB
                  </p>
                </div>
                <div className="meta">
                  <span>{formatDate(document.createdAt)}</span>
                  <a
                    href={`${apiBaseUrl}/api/v1/generated-documents/${document.id}/download`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Скачать
                  </a>
                </div>
              </div>
            ))}
            {!loading && documents.length === 0 ? (
              <p className="empty">
                После обработки здесь появятся сгенерированные документы.
              </p>
            ) : null}
          </div>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>Команды задач</h2>
            <span>{taskCommands.length} зарегистрировано</span>
          </div>

          <div className="list">
            {taskCommands.map((command) => (
              <div className="list-item stacked" key={command.id}>
                <div className="list-topline">
                  <div>
                    <strong>{translateFromMap(command.targetSystem, TASK_TARGET_LABELS)}</strong>
                    <p>{translateCommandText(command.commandText)}</p>
                  </div>
                  <div className="meta">
                    <span className={statusClass(command.status)}>
                      {translateFromMap(command.status, TASK_COMMAND_STATUS_LABELS)}
                    </span>
                    <span>{formatDate(command.createdAt)}</span>
                  </div>
                </div>
                <p className="subtle">
                  {translateFromMap(command.integrationMode, INTEGRATION_MODE_LABELS)} ·{" "}
                  {translateResultMessage(command.resultMessage)}
                </p>
              </div>
            ))}
            {!loading && taskCommands.length === 0 ? (
              <p className="empty">
                Здесь появятся команды для Битрикс и email-согласования.
              </p>
            ) : null}
          </div>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>Журнал событий</h2>
            <span>{summary.eventCount} последних событий</span>
          </div>

          <div className="list">
            {events.map((event) => (
              <div className="list-item stacked" key={event.id}>
                <div className="list-topline">
                  <div>
                    <strong>{translateMessage(event.message)}</strong>
                    <p>{translateFromMap(event.eventType, EVENT_TYPE_LABELS)}</p>
                  </div>
                  <div className="meta">
                    <span className={statusClass(event.level)}>
                      {translateFromMap(event.level, EVENT_LEVEL_LABELS)}
                    </span>
                    <span>{formatDate(event.createdAt)}</span>
                  </div>
                </div>
                <p className="subtle">{formatEventDetails(event.details || "")}</p>
              </div>
            ))}
            {!loading && events.length === 0 ? (
              <p className="empty">Здесь появятся системные события и события заявок.</p>
            ) : null}
          </div>
        </article>
      </section>
    </main>
  );
}
