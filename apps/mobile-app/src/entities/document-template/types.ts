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
