import { useCallback, useEffect, useMemo, useState } from "react";

import type {
  DocumentJob,
  DocumentTemplate,
  GeneratedDocument,
  HealthResponse,
  ProcessingEvent,
  SourceDocument,
  TaskCommand
} from "../../entities/document-job/types";
import {
  createDocumentTemplate,
  createDocumentJob,
  getHealth,
  listDocumentJobs,
  listDocumentTemplates,
  listGeneratedDocuments,
  listSourceDocuments,
  listTaskCommands,
  listProcessingEvents,
  updateDocumentJobStatus
} from "../../shared/api/client";

type DashboardState = {
  health: HealthResponse | null;
  templates: DocumentTemplate[];
  jobs: DocumentJob[];
  documents: GeneratedDocument[];
  sourceDocuments: SourceDocument[];
  taskCommands: TaskCommand[];
  events: ProcessingEvent[];
  loading: boolean;
  error: string | null;
};

const initialState: DashboardState = {
  health: null,
  templates: [],
  jobs: [],
  documents: [],
  sourceDocuments: [],
  taskCommands: [],
  events: [],
  loading: true,
  error: null
};

const dashboardPollIntervalMs = 15000;
const dashboardPollRequestInit: RequestInit = {
  headers: {
    "X-TSK-Request-Source": "admin-poll"
  }
};

export function useDashboardData() {
  const [state, setState] = useState<DashboardState>(initialState);
  const [creatingTemplate, setCreatingTemplate] = useState(false);
  const [creatingJob, setCreatingJob] = useState(false);
  const [updatingJobId, setUpdatingJobId] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    setState((current) => ({
      ...current,
      loading: silent ? current.loading : true,
      error: null
    }));

    try {
      const [health, templates, jobs, documents, sourceDocuments, taskCommands, events] =
        await Promise.all([
        getHealth(dashboardPollRequestInit),
        listDocumentTemplates(dashboardPollRequestInit),
        listDocumentJobs(dashboardPollRequestInit),
        listGeneratedDocuments(undefined, dashboardPollRequestInit),
        listSourceDocuments(undefined, dashboardPollRequestInit),
        listTaskCommands(undefined, dashboardPollRequestInit),
        listProcessingEvents(50, dashboardPollRequestInit)
      ]);

      setState({
        health,
        templates,
        jobs,
        documents,
        sourceDocuments,
        taskCommands,
        events,
        loading: false,
        error: null
      });
    } catch (error) {
      setState((current) => ({
        ...current,
        loading: false,
        error: error instanceof Error ? error.message : "Неизвестная ошибка"
      }));
    }
  }, []);

  useEffect(() => {
    void refresh();

    const intervalId = window.setInterval(() => {
      if (document.visibilityState !== "visible") {
        return;
      }

      void refresh(true);
    }, dashboardPollIntervalMs);

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void refresh(true);
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [refresh]);

  const submitTemplate = useCallback(
    async (formData: FormData) => {
      setCreatingTemplate(true);
      try {
        await createDocumentTemplate(formData);
        await refresh();
      } finally {
        setCreatingTemplate(false);
      }
    },
    [refresh]
  );

  const submitJob = useCallback(
    async (input: {
      templateId: string;
      sourceName: string;
      requestedBy: string;
      payload: string;
      deliveryChannel: "internal" | "email" | "bitrix";
      deliveryAddress: string;
    }) => {
      setCreatingJob(true);
      try {
        await createDocumentJob(input);
        await refresh();
      } finally {
        setCreatingJob(false);
      }
    },
    [refresh]
  );

  const changeJobStatus = useCallback(
    async (jobId: string, status: string, note = "") => {
      setUpdatingJobId(jobId);
      try {
        await updateDocumentJobStatus(jobId, { status, note });
        await refresh();
      } finally {
        setUpdatingJobId(null);
      }
    },
    [refresh]
  );

  const summary = useMemo(() => {
    const activeCount = state.jobs.filter(
      (job) => job.status === "queued" || job.status === "running"
    ).length;
    const failedCount = state.jobs.filter((job) => job.status === "failed").length;

    return {
      templateCount: state.templates.length,
      jobCount: state.jobs.length,
      activeCount,
      documentCount: state.documents.length,
      sourceDocumentCount: state.sourceDocuments.length,
      taskCommandCount: state.taskCommands.length,
      failedCount,
      eventCount: state.events.length
    };
  }, [
    state.documents.length,
    state.events.length,
    state.jobs,
    state.sourceDocuments.length,
    state.taskCommands.length,
    state.templates.length
  ]);

  return {
    ...state,
    creatingTemplate,
    creatingJob,
    updatingJobId,
    refresh,
    submitTemplate,
    submitJob,
    changeJobStatus,
    summary
  };
}
