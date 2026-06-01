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
  bitrixDealId?: number | null;
  bitrixDealTitle?: string;
  createdAt: string;
  updatedAt: string;
  startedAt: string | null;
  completedAt: string | null;
};

export type EstimateLineItem = {
  seq: number;
  code: string;
  description: string;
  unit: string;
  quantity: string;
  basePricePerUnit: string;
  basePriceTotal: string;
  currentPricePerUnit: string;
  currentPriceTotal: string;
};

export type EstimatePreview = {
  estimateNumber: string;
  projectName: string;
  objectDescription: string;
  basis: string;
  estimatedCost: string;
  laborCosts: string;
  priceDate: string;
  approver: string;
  totalDirectCosts: string;
  grandTotal: string;
  lineItems: EstimateLineItem[];
  rawTranscript: string;
  validationWarnings: string[];
};

export type GeneratedDocumentSummary = {
  id: string;
  jobId: string;
  templateId: string;
  templateName: string;
  fileName: string;
  mimeType: string;
  storageKey: string;
  sizeBytes: number;
  createdAt: string;
};

export type MobileDocumentJobView = {
  job: DocumentJobSummary;
  estimate: EstimatePreview;
  generatedDocument?: GeneratedDocumentSummary;
  downloadPath?: string;
  canRetryBitrixAttach?: boolean;
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
  estimate: EstimatePreview;
  transcript: string;
  generatedDocument?: GeneratedDocumentSummary;
};

export type BitrixTaskPerson = {
  id?: string;
  name?: string;
  workPosition?: string;
};

export type BitrixTaskFile = {
  id?: string;
  name?: string;
  size?: string;
  downloadUrl?: string;
  viewUrl?: string;
};

export type BitrixTaskComment = {
  id: string;
  authorId?: string;
  authorName?: string;
  postDate?: string;
  message: string;
  files?: BitrixTaskFile[];
};

export type BitrixTaskChecklistItem = {
  id?: string;
  title: string;
  isComplete: boolean;
  sortIndex?: number;
};

export type BitrixTaskFilterKey = "all" | "open" | "in_progress" | "overdue";

/** Задача из Bitrix24 (контекст вебхука на сервере). */
export type BitrixTaskSummary = {
  id: string;
  title: string;
  status: string;
  deadline?: string;
  closedDate?: string;
  createdDate?: string;
  changedDate?: string;
  responsibleId?: string;
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
  groupTitle?: string;
  stageId?: string;
  stageLabel?: string;
  parentId?: string;
  parentTitle?: string;
  commentsCount?: string;
  timeEstimate?: string;
  durationFact?: string;
  durationPlan?: string;
  durationType?: string;
  tags?: string[];
  crmLinks?: string[];
  favorite?: boolean;
  allowTimeTracking?: boolean;
  taskControl?: boolean;
  multitask?: boolean;
  forumTopicId?: string;
  availableActions?: Record<string, boolean>;
  checklist?: BitrixTaskChecklistItem[];
  files?: BitrixTaskFile[];
  comments?: BitrixTaskComment[];
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

export type BitrixDealFieldOption = {
  id: string;
  label: string;
};

export type BitrixDealField = {
  key: string;
  label: string;
  value: string;
  rawValue?: string;
  editable?: boolean;
  type?: string;
  options?: BitrixDealFieldOption[];
};

export type BitrixDealDetail = BitrixDealSummary & {
  assignedBy?: BitrixTaskPerson;
  createdBy?: BitrixTaskPerson;
  modifiedBy?: BitrixTaskPerson;
  companyId?: string;
  companyTitle?: string;
  contactId?: string;
  contactTitle?: string;
  leadId?: string;
  leadTitle?: string;
  quoteId?: string;
  comments?: string;
  additionalInfo?: string;
  typeId?: string;
  probability?: string;
  taxValue?: string;
  beginDate?: string;
  closeDate?: string;
  sourceId?: string;
  sourceDescription?: string;
  utmSource?: string;
  utmMedium?: string;
  utmCampaign?: string;
  utmContent?: string;
  utmTerm?: string;
  opened?: boolean;
  isNew?: boolean;
  isRecurring?: boolean;
  isReturnCustomer?: boolean;
  fields?: BitrixDealField[];
  stageOptions?: BitrixDealStageOption[];
};

export type BitrixDealsBundle = {
  items: BitrixDealSummary[];
  authMode?: string;
};

export type BitrixNotificationSummary = {
  id: string;
  title?: string;
  text: string;
  date: string;
  read: boolean;
  module?: string;
  tag?: string;
};

export type BitrixNotificationsBundle = {
  items: BitrixNotificationSummary[];
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
