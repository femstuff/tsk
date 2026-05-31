import type { BitrixTaskFilterKey, BitrixTaskSummary } from "../../entities/document-template/types";

export function bitrixTaskStatusRu(status: string) {
  const map: Record<string, string> = {
    "1": "Новая",
    "2": "Ждёт выполнения",
    "3": "В работе",
    "4": "Ждёт контроля",
    "5": "Завершена",
    "6": "Отложена",
    "7": "Отклонена"
  };
  return map[String(status).trim()] ?? `Статус ${status}`;
}

export function stripBitrixDescription(text: string) {
  return text
    .replace(/\[\/[^\]]+\]/gi, "\n")
    .replace(/\[[^\]]+\]/g, "")
    .replace(/&nbsp;/gi, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

export type BitrixStatusAction = {
  label: string;
  status: number;
};

export function bitrixTaskStatusActions(currentStatus: string): BitrixStatusAction[] {
  const st = String(currentStatus).trim();
  const actions: BitrixStatusAction[] = [];
  if (["1", "2", "6"].includes(st)) {
    actions.push({ label: "В работе", status: 3 });
  }
  if (["2", "3", "4"].includes(st)) {
    actions.push({ label: "Завершить", status: 5 });
  }
  if (["2", "3"].includes(st)) {
    actions.push({ label: "Отложить", status: 6 });
  }
  if (["3", "6"].includes(st)) {
    actions.push({ label: "Ждёт выполнения", status: 2 });
  }
  return actions;
}

export function formatBitrixDate(value: string | undefined) {
  const raw = String(value ?? "").trim();
  if (!raw) {
    return "—";
  }
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw;
  }
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

export function formatBitrixDuration(secondsRaw: string | undefined) {
  const sec = Number(String(secondsRaw ?? "").trim());
  if (!Number.isFinite(sec) || sec <= 0) {
    return "—";
  }
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (h > 0) {
    return `${h} ч ${m} мин`;
  }
  return `${m} мин`;
}

export function bitrixPriorityRu(priority: string | undefined) {
  switch (String(priority ?? "").trim()) {
    case "2":
      return "Высокий";
    case "1":
      return "Средний";
    case "0":
      return "Низкий";
    default:
      return priority ? `Приоритет ${priority}` : "—";
  }
}

export function bitrixMarkRu(mark: string | undefined) {
  switch (String(mark ?? "").trim().toUpperCase()) {
    case "P":
      return "Положительная";
    case "N":
      return "Отрицательная";
    default:
      return mark ? mark : "—";
  }
}

export function formatBitrixPerson(person?: { id?: string; name?: string; workPosition?: string }) {
  if (!person) {
    return "—";
  }
  const name = String(person.name ?? "").trim();
  const id = String(person.id ?? "").trim();
  const position = String(person.workPosition ?? "").trim();
  const nameWithId = name && id ? `${name} (#${id})` : name;
  if (nameWithId && position) {
    return `${nameWithId} · ${position}`;
  }
  if (nameWithId) {
    return nameWithId;
  }
  if (id) {
    return `#${id}`;
  }
  return "—";
}

/** Название сущности с опциональным id: «Текст (#123)». */
export function formatBitrixReference(title?: string, id?: string) {
  const label = String(title ?? "").trim();
  const num = String(id ?? "").trim();
  if (label && num) {
    return `${label} (#${num})`;
  }
  if (label) {
    return label;
  }
  if (num) {
    return `#${num}`;
  }
  return "—";
}

export function formatBitrixAuthor(name?: string, id?: string) {
  const author = String(name ?? "").trim();
  const num = String(id ?? "").trim();
  if (author && num) {
    return `${author} (#${num})`;
  }
  if (author) {
    return author;
  }
  if (num) {
    return `#${num}`;
  }
  return "—";
}

export function formatBitrixPeople(
  people?: { id?: string; name?: string; workPosition?: string }[]
) {
  if (!people?.length) {
    return "—";
  }
  return people.map((p) => formatBitrixPerson(p)).join(", ");
}

export function formatBitrixList(items?: string[]) {
  if (!items?.length) {
    return "—";
  }
  return items.join(", ");
}

export function bitrixTaskFilterLabel(filter: BitrixTaskFilterKey) {
  switch (filter) {
    case "open":
      return "Открытые";
    case "in_progress":
      return "В работе";
    case "overdue":
      return "Просроченные";
    default:
      return "Все";
  }
}

function parseBitrixDeadlineMs(raw: string | undefined) {
  const value = String(raw ?? "").trim();
  if (!value) {
    return null;
  }
  const iso = Date.parse(value);
  if (!Number.isNaN(iso)) {
    return iso;
  }
  const match = value.match(/^(\d{2})\.(\d{2})\.(\d{4})(?:\s+(\d{2}):(\d{2})(?::(\d{2}))?)?$/);
  if (!match) {
    return null;
  }
  const [, dd, mm, yyyy, hh = "0", min = "0", sec = "0"] = match;
  return new Date(
    Number(yyyy),
    Number(mm) - 1,
    Number(dd),
    Number(hh),
    Number(min),
    Number(sec)
  ).getTime();
}

export function isBitrixTaskClosed(task: BitrixTaskSummary) {
  if (String(task.closedDate ?? "").trim()) {
    return true;
  }
  return ["4", "5", "7"].includes(String(task.status ?? "").trim());
}

export function bitrixTaskMatchesFilter(
  task: BitrixTaskSummary,
  filter: BitrixTaskFilterKey,
  now = new Date()
) {
  if (filter === "all") {
    return true;
  }
  const closed = isBitrixTaskClosed(task);
  switch (filter) {
    case "open":
      return !closed;
    case "in_progress":
      return !closed && String(task.status ?? "").trim() === "3";
    case "overdue": {
      if (closed) {
        return false;
      }
      const deadlineMs = parseBitrixDeadlineMs(task.deadline);
      return deadlineMs != null && now.getTime() > deadlineMs;
    }
    default:
      return true;
  }
}

export function formatBitrixFileSize(sizeRaw: string | undefined) {
  const size = Number(String(sizeRaw ?? "").trim());
  if (!Number.isFinite(size) || size <= 0) {
    return "";
  }
  if (size >= 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} МБ`;
  }
  if (size >= 1024) {
    return `${Math.round(size / 1024)} КБ`;
  }
  return `${size} Б`;
}

export function formatBitrixBool(value: boolean | undefined) {
  if (value == null) {
    return "—";
  }
  return value ? "Да" : "Нет";
}

export function formatBitrixDurationType(value: string | undefined) {
  switch (String(value ?? "").trim()) {
    case "days":
      return "Дни";
    case "hours":
      return "Часы";
    case "mins":
      return "Минуты";
    default:
      return value?.trim() ? value : "—";
  }
}

export function formatBitrixAvailableActions(actions?: Record<string, boolean>) {
  if (!actions) {
    return "—";
  }
  const labels = Object.entries(actions)
    .filter(([, enabled]) => enabled)
    .map(([key]) => key);
  return labels.length > 0 ? labels.join(", ") : "—";
}

export function resolveBitrixFileUrl(url: string | undefined, portalDomain?: string) {
  const raw = String(url ?? "").trim();
  if (!raw) {
    return "";
  }
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return raw;
  }
  const host = String(portalDomain ?? "")
    .trim()
    .replace(/^https?:\/\//, "")
    .replace(/\/+$/, "");
  if (raw.startsWith("/") && host) {
    return `https://${host}${raw}`;
  }
  return raw;
}

function bitrixTaskRecencyMs(task: BitrixTaskSummary) {
  for (const raw of [task.changedDate, task.createdDate]) {
    const parsed = Date.parse(String(raw ?? "").trim());
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  const id = Number(task.id);
  return Number.isFinite(id) ? id : 0;
}

export function compareBitrixTaskRecency(a: BitrixTaskSummary, b: BitrixTaskSummary) {
  return bitrixTaskRecencyMs(b) - bitrixTaskRecencyMs(a);
}

/** Последняя назначенная на пользователя задача (по CHANGED_DATE / CREATED_DATE). */
export function pickLatestAssignedBitrixTask(
  tasks: BitrixTaskSummary[],
  responsibleUserId: number | null
): BitrixTaskSummary | null {
  if (tasks.length === 0) {
    return null;
  }
  const rid =
    responsibleUserId != null && responsibleUserId > 0 ? String(responsibleUserId) : null;
  const assigned = rid ? tasks.filter((task) => (task.responsibleId ?? "") === rid) : tasks;
  const pool = assigned.length > 0 ? assigned : tasks;
  return [...pool].sort(compareBitrixTaskRecency)[0] ?? null;
}
