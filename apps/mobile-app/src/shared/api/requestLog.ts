import AsyncStorage from "@react-native-async-storage/async-storage";

import type { MobileBitrixIntentResult } from "../../entities/document-template/types";

const STORAGE_KEY = "@tsk/mobile-app/request-log-v1";
export const REQUEST_LOG_MAX = 80;
/** Сколько последних записей показывать в свёрнутом режиме на экране «Ещё». */
export const REQUEST_LOG_COLLAPSED_COUNT = 6;

export type RequestLogKind = "bitrix" | "document" | "data" | "other";

export type RequestLogEntry = {
  id: string;
  at: string;
  method: string;
  endpoint: string;
  ok: boolean;
  durationMs: number;
  /** Короткий заголовок для человека */
  titleRu: string;
  /** Одна строка итога */
  summary: string;
  kind: RequestLogKind;
  errorMessage?: string;
  transcript?: string;
  parsedAction?: string;
  parsedDealTitle?: string;
  bitrixStepsLine?: string;
};

const listeners = new Set<() => void>();

export function subscribeRequestLog(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function notifyListeners(): void {
  for (const listener of listeners) {
    try {
      listener();
    } catch {
      // ignore subscriber errors
    }
  }
}

function truncate(value: string, max: number): string {
  if (value.length <= max) {
    return value;
  }
  return `${value.slice(0, max - 1)}…`;
}

function newId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

function isBitrixIntentPath(path: string): boolean {
  return path.includes("/mobile/bitrix-intent");
}

function logKindAndTitle(path: string, method: string): { kind: RequestLogKind; titleRu: string } {
  if (path.includes("/mobile/bitrix-intent")) {
    return { kind: "bitrix", titleRu: "Запрос в Bitrix24" };
  }
  if (path.includes("/mobile/voice-requests")) {
    return { kind: "document", titleRu: "Голос → документ по шаблону" };
  }
  if (path.includes("/mobile/bitrix-tasks")) {
    return { kind: "data", titleRu: "Список задач Bitrix" };
  }
  if (path.includes("/document-jobs")) {
    return { kind: "data", titleRu: "Заявки на документы" };
  }
  if (path.includes("/document-templates")) {
    return { kind: "data", titleRu: "Шаблоны" };
  }
  if (path.includes("/source-documents")) {
    return { kind: "data", titleRu: "Исходные файлы" };
  }
  if (path.includes("/health")) {
    return { kind: "data", titleRu: "Проверка сервера" };
  }
  return { kind: "other", titleRu: `${method} ${path}` };
}

function extractItem(body: unknown): unknown {
  if (body && typeof body === "object" && "item" in body) {
    return (body as { item: unknown }).item;
  }
  return undefined;
}

function countItems(body: unknown): number | null {
  if (!body || typeof body !== "object" || !("items" in body)) {
    return null;
  }
  const raw = (body as { items?: unknown }).items;
  return Array.isArray(raw) ? raw.length : null;
}

const BITRIX_ACTION_RU: Record<string, string> = {
  move_next: "Следующий этап сделки",
  move_prev: "Предыдущий этап",
  move_stage: "Переход на стадию",
  create_task: "Создание задачи",
  none: "Действие не распознано"
};

function summarizeBitrixItem(item: MobileBitrixIntentResult | null | undefined): {
  summary: string;
  transcript?: string;
  parsedAction?: string;
  parsedDealTitle?: string;
  bitrixStepsLine?: string;
} {
  if (!item) {
    return { summary: "Ответ без данных — обратитесь к разработчику" };
  }
  const stepsJoined = (item.bitrixSteps ?? []).join(" · ");
  const act =
    BITRIX_ACTION_RU[item.parsedAction] ??
    (item.parsedAction ? `Действие: ${item.parsedAction}` : "");
  const deal = item.parsedDealTitle ? `Сделка: «${truncate(item.parsedDealTitle, 80)}»` : "";
  const parts = [act, deal, stepsJoined ? `Результат: ${stepsJoined}` : ""].filter(Boolean);
  const summary = parts.length ? truncate(parts.join(" · "), 240) : "Запрос обработан";
  return {
    summary,
    transcript: item.transcript ? truncate(item.transcript, 480) : undefined,
    parsedAction: item.parsedAction || undefined,
    parsedDealTitle: item.parsedDealTitle || undefined,
    bitrixStepsLine: stepsJoined ? truncate(stepsJoined, 280) : undefined
  };
}

function buildSuccessEntry(
  path: string,
  method: string,
  durationMs: number,
  body: unknown
): Omit<RequestLogEntry, "id" | "at"> {
  const { kind, titleRu } = logKindAndTitle(path, method);
  if (isBitrixIntentPath(path)) {
    const item = extractItem(body) as MobileBitrixIntentResult | undefined;
    const extra = summarizeBitrixItem(item);
    return {
      method,
      endpoint: path,
      ok: true,
      durationMs,
      kind,
      titleRu,
      summary: extra.summary,
      transcript: extra.transcript,
      parsedAction: extra.parsedAction,
      parsedDealTitle: extra.parsedDealTitle,
      bitrixStepsLine: extra.bitrixStepsLine
    };
  }
  if (path.includes("/mobile/bitrix-tasks") && body && typeof body === "object") {
    const b = body as {
      stats?: { totalOpen?: number; inProgress?: number; overdue?: number };
      responsibleUserId?: number;
      items?: unknown[];
    };
    const st = b.stats;
    const n = Array.isArray(b.items) ? b.items.length : 0;
    const summary = st
      ? `Ответственный id ${b.responsibleUserId ?? "?"}: открыто ${st.totalOpen ?? 0}, в работе ${st.inProgress ?? 0}, просрочено ${st.overdue ?? 0} · в ответе ${n} задач`
      : `Задачи: в ответе ${n} записей`;
    return {
      method,
      endpoint: path,
      ok: true,
      durationMs,
      kind,
      titleRu,
      summary
    };
  }
  if (path.includes("/mobile/voice-requests")) {
    const item = extractItem(body) as { job?: { id?: string; sourceName?: string } } | undefined;
    const id = item?.job?.id;
    const name = item?.job?.sourceName;
    return {
      method,
      endpoint: path,
      ok: true,
      durationMs,
      kind,
      titleRu,
      summary: id
        ? `Заявка создана${name ? ` «${truncate(name, 40)}»` : ""}, номер ${id}`
        : "Голосовая заявка принята"
    };
  }
  const n = countItems(body);
  if (n != null) {
    return {
      method,
      endpoint: path,
      ok: true,
      durationMs,
      kind,
      titleRu,
      summary:
        kind === "data"
          ? `Загружено записей: ${n}`
          : `${method} ${path} · элементов: ${n}`
    };
  }
  return {
    method,
    endpoint: path,
    ok: true,
    durationMs,
    kind,
    titleRu,
    summary: "Готово"
  };
}

async function persist(entries: RequestLogEntry[]): Promise<void> {
  const trimmed = entries.slice(-REQUEST_LOG_MAX);
  await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(trimmed));
}

function normalizeStoredEntry(row: Partial<RequestLogEntry> & Record<string, unknown>): RequestLogEntry {
  const endpoint = String(row.endpoint ?? "");
  const method = String(row.method ?? "GET");
  const fallback = logKindAndTitle(endpoint, method);
  return {
    id: String(row.id ?? newId()),
    at: String(row.at ?? new Date().toISOString()),
    method,
    endpoint,
    ok: Boolean(row.ok),
    durationMs: typeof row.durationMs === "number" ? row.durationMs : 0,
    kind: (row.kind as RequestLogKind) ?? fallback.kind,
    titleRu: typeof row.titleRu === "string" ? row.titleRu : fallback.titleRu,
    summary: String(row.summary ?? ""),
    errorMessage: typeof row.errorMessage === "string" ? row.errorMessage : undefined,
    transcript: typeof row.transcript === "string" ? row.transcript : undefined,
    parsedAction: typeof row.parsedAction === "string" ? row.parsedAction : undefined,
    parsedDealTitle: typeof row.parsedDealTitle === "string" ? row.parsedDealTitle : undefined,
    bitrixStepsLine: typeof row.bitrixStepsLine === "string" ? row.bitrixStepsLine : undefined
  };
}

export async function getRequestLog(): Promise<RequestLogEntry[]> {
  try {
    const raw = await AsyncStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.map((row) => normalizeStoredEntry(row as Partial<RequestLogEntry>));
  } catch {
    return [];
  }
}

export async function appendRequestLogSuccess(
  path: string,
  method: string,
  durationMs: number,
  body: unknown
): Promise<void> {
  try {
    const prev = await getRequestLog();
    const partial = buildSuccessEntry(path, method, durationMs, body);
    const next: RequestLogEntry = {
      id: newId(),
      at: new Date().toISOString(),
      ...partial
    };
    await persist([...prev, next]);
    notifyListeners();
  } catch {
    // persistence must not break API calls
  }
}

export async function appendRequestLogFailure(
  path: string,
  method: string,
  durationMs: number,
  errorMessage: string
): Promise<void> {
  try {
    const prev = await getRequestLog();
    const { kind, titleRu } = logKindAndTitle(path, method);
    const next: RequestLogEntry = {
      id: newId(),
      at: new Date().toISOString(),
      method,
      endpoint: path,
      ok: false,
      durationMs,
      kind,
      titleRu,
      summary: truncate(errorMessage, 200),
      errorMessage: truncate(errorMessage, 400)
    };
    await persist([...prev, next]);
    notifyListeners();
  } catch {
    // ignore
  }
}
