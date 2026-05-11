/**
 * HTTP-клиент к backend. Логи запросов (метод, путь, ok/fail, мс, статус/текст ошибки):
 * в dev включены по умолчанию (`console`), без тел запросов/секретов.
 * Отключить в dev: EXPO_PUBLIC_DEBUG_API_LOGS=0. Включить в production-сборке: EXPO_PUBLIC_DEBUG_API_LOGS=1.
 */
import { Platform } from "react-native";

import type {
  BitrixTasksBundle,
  BitrixTaskSummary,
  DocumentJobSummary,
  DocumentTemplateSummary,
  HealthResponse,
  MobileBitrixIntentResult,
  MobileVoiceRequestResult,
  SourceDocumentSummary,
  TaskCommandSummary
} from "../../entities/document-template/types";

import { appendRequestLogFailure, appendRequestLogSuccess } from "./requestLog";

const MOBILE_API_BASE_URL =
  process.env.EXPO_PUBLIC_API_BASE_URL ??
  Platform.select({
    android: "http://10.0.2.2:8080",
    default: "http://localhost:8080"
  }) ??
  "http://localhost:8080";

type CollectionResponse<T> = {
  items?: T[] | null;
};

type ItemResponse<T> = {
  item: T;
};

function itemsFrom<T>(response: CollectionResponse<T>): T[] {
  const raw = response.items;
  return Array.isArray(raw) ? raw : [];
}

const DEBUG_API_LOGS =
  process.env.EXPO_PUBLIC_DEBUG_API_LOGS === "1" ||
  (typeof __DEV__ !== "undefined" &&
    __DEV__ &&
    process.env.EXPO_PUBLIC_DEBUG_API_LOGS !== "0");

function logApiOutcome(
  method: string,
  path: string,
  ok: boolean,
  durationMs: number,
  detail: string
) {
  if (!DEBUG_API_LOGS) {
    return;
  }
  const outcome = ok ? "ok" : "fail";
  console.log(`[API] ${method} ${path} ${outcome} ${durationMs}ms ${detail}`);
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  const started = Date.now();
  const headers = new Headers(init?.headers ?? undefined);
  if (!headers.has("X-TSK-Request-Source")) {
    headers.set("X-TSK-Request-Source", "mobile-app");
  }
  const mergedInit: RequestInit = { ...init, headers };
  let response: Response;
  try {
    response = await fetch(`${MOBILE_API_BASE_URL}${path}`, mergedInit);
  } catch (err) {
    const durationMs = Date.now() - started;
    const message = err instanceof Error ? err.message : String(err);
    logApiOutcome(method, path, false, durationMs, `error=${message}`);
    void appendRequestLogFailure(path, method, durationMs, message);
    throw err;
  }

  const durationMs = Date.now() - started;

  if (!response.ok) {
    const text = await response.text();
    let thrown: Error;
    try {
      const parsed = JSON.parse(text) as { error?: string };
      thrown = new Error(parsed.error ?? `Request failed: ${response.status}`);
    } catch {
      thrown = new Error(text || `Request failed: ${response.status}`);
    }
    logApiOutcome(method, path, false, durationMs, `status=${response.status} ${thrown.message}`);
    void appendRequestLogFailure(path, method, durationMs, thrown.message);
    throw thrown;
  }

  const json = (await response.json()) as T;
  logApiOutcome(method, path, true, durationMs, `status=${response.status}`);
  void appendRequestLogSuccess(path, method, durationMs, json as unknown);

  return json;
}

export function getMobileApiBaseUrl() {
  return MOBILE_API_BASE_URL;
}

export function getHealth() {
  return request<HealthResponse>("/api/v1/health");
}

export async function listTemplates() {
  const response = await request<CollectionResponse<DocumentTemplateSummary>>(
    "/api/v1/document-templates"
  );
  return itemsFrom(response);
}

export async function listJobs() {
  const response = await request<CollectionResponse<DocumentJobSummary>>(
    "/api/v1/document-jobs"
  );
  return itemsFrom(response);
}

export async function listSourceDocuments() {
  const response = await request<CollectionResponse<SourceDocumentSummary>>(
    "/api/v1/source-documents"
  );
  return itemsFrom(response);
}

export async function listTaskCommands() {
  const response = await request<CollectionResponse<TaskCommandSummary>>(
    "/api/v1/task-commands"
  );
  return itemsFrom(response);
}

export async function createTaskCommand(input: {
  jobId?: string | null;
  sourceDocumentId?: string | null;
  targetSystem: "bitrix24" | "email_approval";
  commandText: string;
}) {
  const response = await request<ItemResponse<TaskCommandSummary>>("/api/v1/task-commands", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      jobId: input.jobId ?? null,
      sourceDocumentId: input.sourceDocumentId ?? null,
      targetSystem: input.targetSystem,
      commandText: input.commandText
    })
  });
  return response.item;
}

export async function listBitrixTasks(limit = 60, responsibleId?: number) {
  let qs = `limit=${encodeURIComponent(String(limit))}`;
  if (responsibleId != null && responsibleId > 0) {
    qs += `&responsibleId=${encodeURIComponent(String(responsibleId))}`;
  }
  return request<BitrixTasksBundle>(`/api/v1/mobile/bitrix-tasks?${qs}`);
}

export async function createMobileBitrixIntentText(input: {
  text: string;
  dealId?: number;
  dealTitle?: string;
  dealHint?: string;
  stageHint?: string;
}) {
  const response = await request<{ item: MobileBitrixIntentResult }>("/api/v1/mobile/bitrix-intent", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      text: input.text,
      dealId: input.dealId ?? 0,
      dealTitle: input.dealTitle ?? "",
      dealHint: input.dealHint ?? "",
      stageHint: input.stageHint ?? ""
    })
  });
  return response.item;
}

export async function createMobileBitrixIntentMultipart(input: {
  text?: string;
  audioUri?: string;
  audioFileName?: string;
  audioMimeType?: string;
  dealId?: number;
  dealTitle?: string;
  dealHint?: string;
  stageHint?: string;
}) {
  const formData = new FormData();
  if (input.text?.trim()) {
    formData.append("text", input.text.trim());
  }
  if (input.audioUri) {
    formData.append("audio", {
      uri: input.audioUri,
      name: input.audioFileName ?? `bitrix-intent-${Date.now()}.m4a`,
      type: input.audioMimeType ?? "audio/mp4"
    } as any);
  }
  if (input.dealId != null && input.dealId > 0) {
    formData.append("dealId", String(input.dealId));
  }
  if (input.dealTitle?.trim()) {
    formData.append("dealTitle", input.dealTitle.trim());
  }
  if (input.dealHint?.trim()) {
    formData.append("dealHint", input.dealHint.trim());
  }
  if (input.stageHint?.trim()) {
    formData.append("stageHint", input.stageHint.trim());
  }

  const response = await request<{ item: MobileBitrixIntentResult }>("/api/v1/mobile/bitrix-intent", {
    method: "POST",
    body: formData
  });
  return response.item;
}

export async function createMobileVoiceRequest(input: {
  templateId: string;
  sourceName: string;
  requestedBy: string;
  payload: string;
  deliveryChannel: "internal" | "email" | "bitrix";
  deliveryAddress: string;
  taskCommandText: string;
  taskTarget: "bitrix24" | "email_approval";
  audioUri: string;
  audioFileName: string;
  audioMimeType: string;
}) {
  const formData = new FormData();
  formData.append("templateId", input.templateId);
  formData.append("sourceName", input.sourceName);
  formData.append("requestedBy", input.requestedBy);
  formData.append("payload", input.payload);
  formData.append("deliveryChannel", input.deliveryChannel);
  formData.append("deliveryAddress", input.deliveryAddress);
  formData.append("taskCommandText", input.taskCommandText);
  formData.append("taskTarget", input.taskTarget);
  formData.append("audio", {
    uri: input.audioUri,
    name: input.audioFileName,
    type: input.audioMimeType
  } as any);

  const response = await request<ItemResponse<MobileVoiceRequestResult>>(
    "/api/v1/mobile/voice-requests",
    {
      method: "POST",
      body: formData
    }
  );

  return response.item;
}
