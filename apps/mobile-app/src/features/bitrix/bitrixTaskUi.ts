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
  if (name && position) {
    return `${name} (${position})`;
  }
  if (name) {
    return name;
  }
  if (id) {
    return `id ${id}`;
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
