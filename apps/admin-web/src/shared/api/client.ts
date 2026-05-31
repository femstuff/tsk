import type { AdminDashboard } from "../../entities/admin/types";
import type {
  AdminVoiceBitrixResult,
  DocumentJob,
  DocumentTemplate,
  GeneratedDocument,
  HealthResponse,
  ProcessingEvent,
  SourceDocument,
  TaskCommand
} from "../../entities/document-job/types";
import type { ObservabilityDashboard } from "../../entities/observability/types";

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

type CollectionResponse<T> = {
  items: T[];
};

type ItemResponse<T> = {
  item: T;
};

const API_ERROR_LABELS: Record<string, string> = {
  "invalid multipart payload": "Некорректные multipart-данные",
  "template file is required": "Нужно выбрать файл шаблона",
  "unable to read uploaded file": "Не удалось прочитать загруженный файл",
  "invalid JSON payload": "Некорректные JSON-данные",
  "audio file is required": "Нужно выбрать аудиофайл",
  "unable to read uploaded audio": "Не удалось прочитать загруженное аудио",
  "name is required": "Нужно указать название",
  "status is invalid": "Указан некорректный статус",
  "templateId is required": "Нужно выбрать шаблон",
  "sourceName is required": "Нужно указать название источника",
  "deliveryChannel is invalid": "Указан некорректный канал доставки",
  "voice recording is required": "Нужна голосовая запись",
  "whisper is not configured (set WHISPER_BASE_URL)": "Не настроен Whisper (задайте WHISPER_BASE_URL для backend)",
  "transcription:": "Ошибка транскрипции",
  "нет шаблонов: загрузите шаблон или укажите templateId":
    "Нет ни одного шаблона: загрузите шаблон в админке или укажите шаблон в форме"
};

function translateApiErrorMessage(message: string) {
  return API_ERROR_LABELS[message] ?? message;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (!(init?.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers });

  if (!response.ok) {
    const text = await response.text();
    try {
      const parsed = JSON.parse(text) as { error?: string };
      throw new Error(
        translateApiErrorMessage(parsed.error ?? `Ошибка запроса: ${response.status}`)
      );
    } catch {
      throw new Error(
        translateApiErrorMessage(text || `Ошибка запроса: ${response.status}`)
      );
    }
  }

  return (await response.json()) as T;
}

export function getApiBaseUrl() {
  return API_BASE_URL;
}

export function getHealth(init?: RequestInit) {
  return request<HealthResponse>("/api/v1/health", init);
}

export async function listDocumentTemplates(init?: RequestInit) {
  const response = await request<CollectionResponse<DocumentTemplate>>(
    "/api/v1/document-templates",
    init
  );

  return response.items;
}

export async function listDocumentJobs(init?: RequestInit) {
  const response = await request<CollectionResponse<DocumentJob>>(
    "/api/v1/document-jobs",
    init
  );

  return response.items;
}

export async function listGeneratedDocuments(jobId?: string, init?: RequestInit) {
  const query = jobId ? `?jobId=${encodeURIComponent(jobId)}` : "";
  const response = await request<CollectionResponse<GeneratedDocument>>(
    `/api/v1/generated-documents${query}`,
    init
  );

  return response.items;
}

export async function listSourceDocuments(jobId?: string, init?: RequestInit) {
  const query = jobId ? `?jobId=${encodeURIComponent(jobId)}` : "";
  const response = await request<CollectionResponse<SourceDocument>>(
    `/api/v1/source-documents${query}`,
    init
  );

  return response.items;
}

export async function listTaskCommands(jobId?: string, init?: RequestInit) {
  const query = jobId ? `?jobId=${encodeURIComponent(jobId)}` : "";
  const response = await request<CollectionResponse<TaskCommand>>(
    `/api/v1/task-commands${query}`,
    init
  );

  return response.items;
}

export async function listProcessingEvents(limit = 120, init?: RequestInit) {
  const response = await request<CollectionResponse<ProcessingEvent>>(
    `/api/v1/processing-events?limit=${limit}`,
    init
  );

  return response.items;
}

export async function createDocumentTemplate(formData: FormData) {
  const response = await request<ItemResponse<DocumentTemplate>>(
    "/api/v1/document-templates",
    {
      method: "POST",
      body: formData
    }
  );

  return response.item;
}

export async function createDocumentJob(input: {
  templateId: string;
  sourceName: string;
  requestedBy: string;
  payload: string;
  deliveryChannel: "internal" | "email" | "bitrix";
  deliveryAddress: string;
}) {
  const response = await request<ItemResponse<DocumentJob>>("/api/v1/document-jobs", {
    method: "POST",
    body: JSON.stringify(input)
  });

  return response.item;
}

export async function runAdminVoiceBitrixPipeline(formData: FormData) {
  const response = await request<ItemResponse<AdminVoiceBitrixResult>>(
    "/api/v1/admin/voice-bitrix-pipeline",
    {
      method: "POST",
      body: formData
    }
  );

  return response.item;
}

export async function updateDocumentJobStatus(
  jobId: string,
  input: { status: string; note: string }
) {
  const response = await request<ItemResponse<DocumentJob>>(
    `/api/v1/document-jobs/${jobId}/status`,
    {
      method: "PATCH",
      body: JSON.stringify(input)
    }
  );

  return response.item;
}

export async function getObservabilityDashboard(init?: RequestInit) {
  const response = await request<ItemResponse<ObservabilityDashboard>>(
    "/api/v1/admin/observability/dashboard",
    init
  );
  return response.item;
}

export async function getAdminDashboard(init?: RequestInit) {
  const response = await request<ItemResponse<AdminDashboard>>("/api/v1/admin/dashboard", init);
  return response.item;
}
