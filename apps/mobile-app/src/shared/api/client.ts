/**
 * HTTP-клиент к backend. Логи запросов (метод, путь, ok/fail, мс, статус/текст ошибки):
 * в dev включены по умолчанию (`console`), без тел запросов/секретов.
 * Отключить в dev: EXPO_PUBLIC_DEBUG_API_LOGS=0. Включить в production-сборке: EXPO_PUBLIC_DEBUG_API_LOGS=1.
 */
import { Platform } from "react-native";

import type {
  BitrixOAuthSessionView,
  BitrixOAuthStartResult,
  BitrixDealDetail,
  BitrixDealsBundle,
  BitrixDealSummary,
  BitrixTaskDetail,
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
import { getBitrixSessionId } from "../../features/bitrix/sessionStorage";

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

const DEFAULT_REQUEST_TIMEOUT_MS = 30_000;
const VOICE_REQUEST_TIMEOUT_MS = 180_000;

async function request<T>(
  path: string,
  init?: RequestInit,
  timeoutMs = DEFAULT_REQUEST_TIMEOUT_MS,
  bitrixSessionId?: string | null
): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  const started = Date.now();
  const headers = new Headers(init?.headers ?? undefined);
  if (!headers.has("X-TSK-Request-Source")) {
    headers.set("X-TSK-Request-Source", "mobile-app");
  }
  const sessionId =
    bitrixSessionId === undefined ? await getBitrixSessionId() : bitrixSessionId;
  if (sessionId) {
    headers.set("X-TSK-Bitrix-Session", sessionId);
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);
  const mergedInit: RequestInit = { ...init, headers, signal: controller.signal };

  let response: Response;
  try {
    response = await fetch(`${MOBILE_API_BASE_URL}${path}`, mergedInit);
  } catch (err) {
    clearTimeout(timeoutId);
    const durationMs = Date.now() - started;
    const message =
      err instanceof Error && err.name === "AbortError"
        ? `Превышено время ожидания (${Math.round(timeoutMs / 1000)} сек). Whisper может долго обрабатывать первую запись — попробуйте ещё раз.`
        : err instanceof Error
          ? err.message
          : String(err);
    logApiOutcome(method, path, false, durationMs, `error=${message}`);
    void appendRequestLogFailure(path, method, durationMs, message);
    throw new Error(message);
  }
  clearTimeout(timeoutId);

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

  if (response.status === 204 || response.headers.get("content-length") === "0") {
    logApiOutcome(method, path, true, durationMs, `status=${response.status}`);
    return undefined as T;
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

export async function listBitrixTasks(limit = 60, responsibleId?: number, refresh = false) {
  let qs = `limit=${encodeURIComponent(String(limit))}`;
  if (responsibleId != null && responsibleId > 0) {
    qs += `&responsibleId=${encodeURIComponent(String(responsibleId))}`;
  }
  if (refresh) {
    qs += "&refresh=1";
  }
  return request<BitrixTasksBundle>(`/api/v1/mobile/bitrix-tasks?${qs}`);
}

export async function getBitrixTask(taskId: string) {
  const response = await request<{ item: BitrixTaskDetail }>(
    `/api/v1/mobile/bitrix-tasks/${encodeURIComponent(taskId)}`
  );
  return response.item;
}

export async function updateBitrixTaskStatus(taskId: string, status: number) {
  const response = await request<{ item: BitrixTaskDetail }>(
    `/api/v1/mobile/bitrix-tasks/${encodeURIComponent(taskId)}/status`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status })
    }
  );
  return response.item;
}

export async function listBitrixDeals(limit = 50, search = "", refresh = false) {
  let qs = `limit=${encodeURIComponent(String(limit))}`;
  if (search.trim()) {
    qs += `&search=${encodeURIComponent(search.trim())}`;
  }
  if (refresh) {
    qs += "&refresh=1";
  }
  const bundle = await request<BitrixDealsBundle & { docs?: unknown }>(
    `/api/v1/mobile/bitrix-deals?${qs}`
  );
  if (!Array.isArray(bundle.items)) {
    if (bundle && typeof bundle === "object" && "docs" in bundle) {
      throw new Error(
        "Сервер не поддерживает список сделок. Пересоберите backend: docker compose build backend-api && docker compose up -d backend-api"
      );
    }
    throw new Error("Неверный ответ сервера при загрузке сделок Bitrix24");
  }
  return bundle;
}

export async function getBitrixDeal(dealId: string) {
  const response = await request<{ item: BitrixDealDetail }>(
    `/api/v1/mobile/bitrix-deals/${encodeURIComponent(dealId)}`
  );
  return response.item;
}

export async function updateBitrixDealStage(dealId: string, stageId: string) {
  const response = await request<{ item: BitrixDealDetail }>(
    `/api/v1/mobile/bitrix-deals/${encodeURIComponent(dealId)}/stage`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ stageId })
    }
  );
  return response.item;
}

export async function startBitrixOAuth() {
  const response = await request<{ item: BitrixOAuthStartResult }>(
    "/api/v1/mobile/bitrix/oauth/start"
  );
  return response.item;
}

export async function fetchBitrixOAuthSession(sessionId: string) {
  const response = await request<{ item: BitrixOAuthSessionView }>(
    `/api/v1/mobile/bitrix/oauth/session?sessionId=${encodeURIComponent(sessionId)}`,
    undefined,
    DEFAULT_REQUEST_TIMEOUT_MS,
    sessionId
  );
  return response.item;
}

export async function disconnectBitrixOAuth(sessionId: string) {
  await request<void>(
    "/api/v1/mobile/bitrix/oauth/session",
    { method: "DELETE" },
    DEFAULT_REQUEST_TIMEOUT_MS,
    sessionId
  );
}

export async function createMobileBitrixIntentText(input: {
  text: string;
  dealId?: number;
  dealTitle?: string;
  dealHint?: string;
  stageHint?: string;
}) {
  const response = await request<{ item: MobileBitrixIntentResult }>(
    "/api/v1/mobile/bitrix-intent",
    {
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
    },
    VOICE_REQUEST_TIMEOUT_MS
  );
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

  const response = await request<{ item: MobileBitrixIntentResult }>(
    "/api/v1/mobile/bitrix-intent",
    {
      method: "POST",
      body: formData
    },
    VOICE_REQUEST_TIMEOUT_MS
  );
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
    },
    VOICE_REQUEST_TIMEOUT_MS
  );

  return response.item;
}
