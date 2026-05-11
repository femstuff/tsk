import { Platform } from "react-native";

import type {
  DocumentJobSummary,
  DocumentTemplateSummary,
  HealthResponse,
  MobileVoiceRequestResult,
  SourceDocumentSummary,
  TaskCommandSummary
} from "../../entities/document-template/types";

const MOBILE_API_BASE_URL =
  process.env.EXPO_PUBLIC_API_BASE_URL ??
  Platform.select({
    android: "http://10.0.2.2:8080",
    default: "http://localhost:8080"
  }) ??
  "http://localhost:8080";

type CollectionResponse<T> = {
  items: T[];
};

type ItemResponse<T> = {
  item: T;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${MOBILE_API_BASE_URL}${path}`, init);
  if (!response.ok) {
    const text = await response.text();
    try {
      const parsed = JSON.parse(text) as { error?: string };
      throw new Error(parsed.error ?? `Request failed: ${response.status}`);
    } catch {
      throw new Error(text || `Request failed: ${response.status}`);
    }
  }

  return (await response.json()) as T;
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
  return response.items;
}

export async function listJobs() {
  const response = await request<CollectionResponse<DocumentJobSummary>>(
    "/api/v1/document-jobs"
  );
  return response.items;
}

export async function listSourceDocuments() {
  const response = await request<CollectionResponse<SourceDocumentSummary>>(
    "/api/v1/source-documents"
  );
  return response.items;
}

export async function listTaskCommands() {
  const response = await request<CollectionResponse<TaskCommandSummary>>(
    "/api/v1/task-commands"
  );
  return response.items;
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
