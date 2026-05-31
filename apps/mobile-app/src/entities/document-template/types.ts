export type DocumentTemplateSummary = {
  id: string;
  name: string;
  category: string;
  description: string;
  version: string;
  fileName: string;
  sizeBytes: number;
};

export type DocumentJobSummary = {
  id: string;
  templateId: string;
  templateName: string;
  sourceName: string;
  requestedBy: string;
  payload: string;
  deliveryChannel: "internal" | "email" | "bitrix";
  deliveryAddress: string;
  dispatchStatus: "not_required" | "pending" | "sent" | "failed";
  status: string;
  errorMessage: string;
  resultDocumentId: string | null;
  createdAt: string;
  updatedAt: string;
  startedAt: string | null;
  completedAt: string | null;
};

export type SourceDocumentSummary = {
  id: string;
  jobId: string | null;
  templateId: string;
  kind: string;
  origin: string;
  fileName: string;
  mimeType: string;
  storageKey: string;
  sizeBytes: number;
  createdAt: string;
};

export type TaskCommandSummary = {
  id: string;
  jobId: string | null;
  sourceDocumentId: string | null;
  targetSystem: "bitrix24" | "email_approval";
  commandText: string;
  status: "recorded" | "pending" | "sent" | "failed";
  integrationMode: string;
  externalReference: string;
  resultMessage: string;
  createdAt: string;
  updatedAt: string;
};

export type HealthResponse = {
  status: string;
  service: string;
  environment: string;
  database: string;
  storageRoot: string;
  uptimeSeconds: number;
  productRequestsTotal: number;
  httpRequestsTotalRaw: number;
  jobsCreatedTotal: number;
  errorsTotal: number;
};

export type MobileVoiceRequestResult = {
  job: DocumentJobSummary;
  sourceDocument: SourceDocumentSummary;
  taskCommand?: TaskCommandSummary;
};

export type BitrixTaskPerson = {
  id?: string;
  name?: string;
  workPosition?: string;
};

/** Задача из Bitrix24 (контекст вебхука на сервере). */
export type BitrixTaskSummary = {
  id: string;
  title: string;
  status: string;
  deadline?: string;
  closedDate?: string;
};

export type BitrixTaskDetail = BitrixTaskSummary & {
  description?: string;
  priority?: string;
  mark?: string;
  creator?: BitrixTaskPerson;
  responsible?: BitrixTaskPerson;
  accomplices?: BitrixTaskPerson[];
  auditors?: BitrixTaskPerson[];
  createdDate?: string;
  changedDate?: string;
  dateStart?: string;
  startDatePlan?: string;
  endDatePlan?: string;
  groupId?: string;
  stageId?: string;
  parentId?: string;
  commentsCount?: string;
  timeEstimate?: string;
  durationFact?: string;
  tags?: string[];
  crmLinks?: string[];
  favorite?: boolean;
  availableActions?: Record<string, boolean>;
};

export type BitrixTaskStats = {
  totalOpen: number;
  inProgress: number;
  overdue: number;
};

export type BitrixTasksBundle = {
  responsibleUserId: number;
  stats: BitrixTaskStats;
  items: BitrixTaskSummary[];
  authMode?: string;
};

export type BitrixDealSummary = {
  id: string;
  title: string;
  stageId: string;
  stageLabel: string;
  categoryId: string;
  opportunity?: string;
  currencyId?: string;
  assignedById?: string;
  dateCreate?: string;
  dateModify?: string;
  closed?: string;
};

export type BitrixDealStageOption = {
  id: string;
  label: string;
};

export type BitrixDealDetail = BitrixDealSummary & {
  assignedBy?: BitrixTaskPerson;
  companyId?: string;
  contactId?: string;
  comments?: string;
  stageOptions?: BitrixDealStageOption[];
};

export type BitrixDealsBundle = {
  items: BitrixDealSummary[];
  authMode?: string;
};

export type BitrixOAuthStartResult = {
  authorizeUrl: string;
  sessionId: string;
  state: string;
};

export type BitrixOAuthSessionView = {
  sessionId: string;
  connected: boolean;
  bitrixUserId: number;
  userName: string;
  portalDomain: string;
  oauthScopes?: string;
  taskScopeGranted?: boolean;
  authMode: string;
};

/** Ответ после разбора запроса и вызова Bitrix. */
export type MobileBitrixIntentResult = {
  transcript: string;
  parsedAction: string;
  parsedDealId: number;
  parsedDealTitle: string;
  bitrixSteps: string[];
  bitrixConfigured: boolean;
};
