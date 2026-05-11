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

export type DocumentTemplate = {
  id: string;
  name: string;
  category: string;
  version: string;
  description: string;
  fileName: string;
  mimeType: string;
  storageKey: string;
  sizeBytes: number;
  createdAt: string;
};

export type DocumentJob = {
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

export type GeneratedDocument = {
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

export type SourceDocument = {
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

/** Ответ тестового контура «голос → Whisper → Bitrix». */
export type AdminVoiceBitrixResult = {
  transcript: string;
  job: DocumentJob;
  sourceDocument: SourceDocument;
  parsedAction: string;
  parsedDealId: number;
  parsedDealTitle: string;
  bitrixSteps: string[];
  bitrixConfigured: boolean;
};

export type TaskCommand = {
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

export type ProcessingEvent = {
  id: string;
  jobId: string | null;
  level: string;
  eventType: string;
  message: string;
  details: string;
  createdAt: string;
};
