import { Ionicons } from "@expo/vector-icons";
import { Audio } from "expo-av";
import { StatusBar as ExpoStatusBar } from "expo-status-bar";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Linking,
  Modal,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StatusBar as RNStatusBar,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

import type {
  BitrixDealSummary,
  BitrixNotificationSummary,
  BitrixTaskFilterKey,
  BitrixTaskStats,
  BitrixTaskSummary,
  DocumentJobSummary,
  DocumentTemplateSummary,
  HealthResponse,
  MobileDocumentJobView,
  SourceDocumentSummary
} from "../entities/document-template/types";
import { EstimateDocumentFlowScreen } from "../features/documents/EstimateDocumentFlowScreen";
import {
  createMobileBitrixIntentMultipart,
  createMobileBitrixIntentText,
  generatedDocumentDownloadUrl,
  getHealth,
  getMobileApiBaseUrl,
  getMobileDocumentJob,
  listBitrixDeals,
  listBitrixNotifications,
  listBitrixTasks,
  listJobs,
  listMobileDocumentJobs,
  retryAttachMobileDocumentJob,
  listSourceDocuments,
  listTemplates,
  BitrixIntentError,
  markAllBitrixNotificationsRead,
  markBitrixNotificationRead
} from "../shared/api/client";
import {
  getRequestLog,
  REQUEST_LOG_COLLAPSED_COUNT,
  subscribeRequestLog,
  type RequestLogEntry,
  type RequestLogKind
} from "../shared/api/requestLog";
import { useBitrixAuth } from "../features/bitrix/useBitrixAuth";
import { BitrixDealDetailModal } from "../features/bitrix/BitrixDealDetailModal";
import { BitrixTaskDetailModal } from "../features/bitrix/BitrixTaskDetailModal";
import { bitrixTaskFilterLabel, bitrixTaskMatchesFilter, bitrixTaskStatusRu, formatBitrixDate, formatBitrixMoney, pickLatestAssignedBitrixTask } from "../features/bitrix/bitrixTaskUi";

const HEADER_BLUE = "#2563eb";
const BG = "#e8eef5";

type MainTab = "home" | "tasks" | "deals" | "docs" | "more";

type LogFilterTab = "all" | RequestLogKind | "errors";

type RecordingPhase = "idle" | "recording" | "stopping" | "ready";

type BitrixDealConfirmState = {
  parsedTitle: string;
  transcript: string;
  source: "text" | "voice";
  step: "confirm" | "hint";
};

type SubmitProgressState = {
  title: string;
  message: string;
  elapsedSec: number;
};

const ESTIMATE_SUBMIT_STEPS = [
  "Отправка аудио на сервер…",
  "Распознавание речи (Whisper) — обычно 30–90 сек…",
  "Разбор полей сметы и сохранение…"
];

function bitrixVoiceProgressMessage(elapsedSec: number) {
  if (elapsedSec < 3) {
    return "Отправка аудио на сервер…";
  }
  if (elapsedSec < 45) {
    return "Распознавание речи (Whisper) — обычно 15–60 сек…";
  }
  return "Выполнение действия в Bitrix24…";
}

function bitrixTextProgressMessage(elapsedSec: number) {
  if (elapsedSec < 2) {
    return "Отправка текста на сервер…";
  }
  return "Выполнение действия в Bitrix24…";
}

function estimateProgressMessage(elapsedSec: number) {
  if (elapsedSec < 3) {
    return ESTIMATE_SUBMIT_STEPS[0];
  }
  if (elapsedSec < 45) {
    return ESTIMATE_SUBMIT_STEPS[1];
  }
  return ESTIMATE_SUBMIT_STEPS[2];
}

function actionLabelRu(code: string | undefined) {
  if (!code) {
    return null;
  }
  const map: Record<string, string> = {
    move_next: "Следующий этап",
    move_prev: "Назад по воронке",
    move_stage: "На конкретную стадию",
    create_task: "Новая задача в Bitrix",
    none: "Не распознано"
  };
  return map[code] ?? code;
}

function formatDate(value: string | null) {
  if (!value) {
    return "—";
  }
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "short",
    timeStyle: "short"
  }).format(new Date(value));
}

function jobStatusRu(status: string) {
  switch (status) {
    case "awaiting_review":
      return "На проверке";
    case "awaiting_bitrix_attach":
      return "Ждёт Bitrix";
    case "queued":
      return "В очереди";
    case "running":
      return "Обработка";
    case "completed":
      return "Готово";
    case "failed":
      return "Ошибка";
    default:
      return status;
  }
}

function statusColor(status: string) {
  switch (status) {
    case "completed":
    case "sent":
      return "#15803d";
    case "awaiting_review":
    case "awaiting_bitrix_attach":
      return "#b45309";
    case "running":
    case "pending":
      return "#0369a1";
    case "failed":
      return "#b91c1c";
    default:
      return "#4338ca";
  }
}

export function HomeScreen() {
  const topInset = Platform.OS === "android" ? (RNStatusBar.currentHeight ?? 24) : 52;

  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [templates, setTemplates] = useState<DocumentTemplateSummary[]>([]);
  const [jobs, setJobs] = useState<DocumentJobSummary[]>([]);
  const [mobileDocumentJobs, setMobileDocumentJobs] = useState<MobileDocumentJobView[]>([]);
  const [docDetailJob, setDocDetailJob] = useState<MobileDocumentJobView | null>(null);
  const [docAttachRetrying, setDocAttachRetrying] = useState(false);
  const [docDetailLoading, setDocDetailLoading] = useState(false);
  const [sourceDocuments, setSourceDocuments] = useState<SourceDocumentSummary[]>([]);
  const [bitrixTasks, setBitrixTasks] = useState<BitrixTaskSummary[]>([]);
  const [bitrixTaskStats, setBitrixTaskStats] = useState<BitrixTaskStats | null>(null);
  const [bitrixResponsibleId, setBitrixResponsibleId] = useState<number | null>(null);
  const [bitrixAuthMode, setBitrixAuthMode] = useState<string | null>(null);
  const [bitrixTasksError, setBitrixTasksError] = useState<string | null>(null);
  const [bitrixTaskFilter, setBitrixTaskFilter] = useState<BitrixTaskFilterKey>("all");
  const [bitrixNotifications, setBitrixNotifications] = useState<BitrixNotificationSummary[]>([]);
  const [bitrixNotificationsError, setBitrixNotificationsError] = useState<string | null>(null);
  const [bitrixNotificationsLoading, setBitrixNotificationsLoading] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [expandedNotificationId, setExpandedNotificationId] = useState<string | null>(null);
  const [bitrixDealConfirm, setBitrixDealConfirm] = useState<BitrixDealConfirmState | null>(null);
  const [dealConfirmHint, setDealConfirmHint] = useState("");
  const [bitrixDeals, setBitrixDeals] = useState<BitrixDealSummary[]>([]);
  const [bitrixDealsError, setBitrixDealsError] = useState<string | null>(null);
  const [bitrixDealsSearch, setBitrixDealsSearch] = useState("");
  const [bitrixDealsLoading, setBitrixDealsLoading] = useState(false);
  const [bitrixDealsAuthMode, setBitrixDealsAuthMode] = useState<string | null>(null);
  const [selectedDealId, setSelectedDealId] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submittingRequest, setSubmittingRequest] = useState(false);
  const [submittingBitrix, setSubmittingBitrix] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [docRecording, setDocRecording] = useState<Audio.Recording | null>(null);
  const [docRecordingPhase, setDocRecordingPhase] = useState<RecordingPhase>("idle");
  const [docAudioUri, setDocAudioUri] = useState<string | null>(null);
  const [bitrixRecording, setBitrixRecording] = useState<Audio.Recording | null>(null);
  const [bitrixRecordingPhase, setBitrixRecordingPhase] = useState<RecordingPhase>("idle");
  const [bitrixAudioUri, setBitrixAudioUri] = useState<string | null>(null);
  const docRecordingRef = useRef<Audio.Recording | null>(null);
  const bitrixRecordingRef = useRef<Audio.Recording | null>(null);
  const submitElapsedTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [submitProgress, setSubmitProgress] = useState<SubmitProgressState | null>(null);
  const previewSoundRef = useRef<Audio.Sound | null>(null);
  const [previewPlayingUri, setPreviewPlayingUri] = useState<string | null>(null);
  const [requestLogEntries, setRequestLogEntries] = useState<RequestLogEntry[]>([]);
  const [logFilter, setLogFilter] = useState<LogFilterTab>("all");
  const [logExpanded, setLogExpanded] = useState(false);
  const [activeTab, setActiveTab] = useState<MainTab>("home");
  const [docVoiceOpen, setDocVoiceOpen] = useState(false);
  const [bitrixVoiceOpen, setBitrixVoiceOpen] = useState(false);
  const [bitrixAuthOpen, setBitrixAuthOpen] = useState(false);
  const [bitrixIntentText, setBitrixIntentText] = useState("");
  const [bitrixHints, setBitrixHints] = useState({
    dealId: "",
    dealTitle: "",
    dealHint: "",
    stageHint: ""
  });
  const bitrixAuth = useBitrixAuth();

  const [requestForm, setRequestForm] = useState({
    templateId: "",
    sourceName: "",
    requestedBy: "mobile-app",
    payload: "",
    deliveryChannel: "email" as "internal" | "email" | "bitrix",
    deliveryAddress: "ops@example.local",
    taskCommandText: "",
    taskTarget: "bitrix24" as "bitrix24" | "email_approval"
  });
  const refreshAfterSubmit = useCallback(async () => {
    try {
      const [nextJobs, nextSourceDocuments] = await Promise.all([listJobs(), listSourceDocuments()]);
      setJobs(nextJobs.filter((job) => job.requestedBy === "mobile-app"));
      setSourceDocuments(nextSourceDocuments.filter((document) => document.origin === "mobile-app"));
    } catch {
      // Не блокируем успешную отправку из-за фонового обновления списков.
    }
  }, []);

  const clearSubmitProgressTimers = useCallback(() => {
    if (submitElapsedTimerRef.current) {
      clearInterval(submitElapsedTimerRef.current);
      submitElapsedTimerRef.current = null;
    }
  }, []);

  const beginSubmitProgress = useCallback(
    (title: string, messageForElapsed: (elapsedSec: number) => string) => {
      clearSubmitProgressTimers();
      setSubmitProgress({
        title,
        message: messageForElapsed(0),
        elapsedSec: 0
      });

      submitElapsedTimerRef.current = setInterval(() => {
        setSubmitProgress((current) => {
          if (!current) {
            return current;
          }
          const elapsedSec = current.elapsedSec + 1;
          return {
            ...current,
            elapsedSec,
            message: messageForElapsed(elapsedSec)
          };
        });
      }, 1000);
    },
    [clearSubmitProgressTimers]
  );

  const endSubmitProgress = useCallback(() => {
    clearSubmitProgressTimers();
    setSubmitProgress(null);
  }, [clearSubmitProgressTimers]);

  useEffect(() => {
    return () => {
      clearSubmitProgressTimers();
    };
  }, [clearSubmitProgressTimers]);

  const loadBitrixNotifications = useCallback(async () => {
    setBitrixNotificationsLoading(true);
    setBitrixNotificationsError(null);
    try {
      const bundle = await listBitrixNotifications(100);
      setBitrixNotifications(Array.isArray(bundle.items) ? bundle.items : []);
    } catch (err) {
      setBitrixNotifications([]);
      setBitrixNotificationsError(
        err instanceof Error ? err.message : "Не удалось загрузить уведомления Bitrix24"
      );
    } finally {
      setBitrixNotificationsLoading(false);
    }
  }, []);

  const loadBitrixDeals = useCallback(async (search: string, refresh = false) => {
    setBitrixDealsLoading(true);
    setBitrixDealsError(null);
    try {
      const bundle = await listBitrixDeals(80, search, refresh);
      setBitrixDeals(Array.isArray(bundle.items) ? bundle.items : []);
      setBitrixDealsAuthMode(typeof bundle.authMode === "string" ? bundle.authMode : null);
    } catch (err) {
      setBitrixDeals([]);
      setBitrixDealsAuthMode(null);
      setBitrixDealsError(err instanceof Error ? err.message : "Не удалось загрузить сделки Bitrix24");
    } finally {
      setBitrixDealsLoading(false);
    }
  }, []);

  const refresh = useCallback(async (options?: { pull?: boolean }) => {
    if (options?.pull) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    setError(null);
    try {
      const [nextHealth, nextTemplates, nextJobs, nextSourceDocuments, nextMobileDocs] = await Promise.all([
        getHealth(),
        listTemplates(),
        listJobs(),
        listSourceDocuments(),
        listMobileDocumentJobs().catch(() => [] as MobileDocumentJobView[])
      ]);

      let bitrixBundle: Awaited<ReturnType<typeof listBitrixTasks>> | null = null;
      let nextBitrixError: string | null = null;
      try {
        bitrixBundle = await listBitrixTasks(80, undefined, Boolean(options?.pull));
      } catch (bitrixErr) {
        nextBitrixError =
          bitrixErr instanceof Error ? bitrixErr.message : "Не удалось загрузить задачи Bitrix24";
      }
      setBitrixTasksError(nextBitrixError);

      setHealth(nextHealth);
      setTemplates(nextTemplates);
      const mobileJobs = nextJobs.filter((job) => job.requestedBy === "mobile-app");
      setJobs(mobileJobs);
      setSourceDocuments(
        nextSourceDocuments.filter((document) => document.origin === "mobile-app")
      );
      setMobileDocumentJobs(nextMobileDocs);
      if (bitrixBundle && Array.isArray(bitrixBundle.items)) {
        setBitrixTasks(bitrixBundle.items);
        setBitrixTaskStats(bitrixBundle.stats ?? null);
        setBitrixResponsibleId(
          typeof bitrixBundle.responsibleUserId === "number" ? bitrixBundle.responsibleUserId : null
        );
        setBitrixAuthMode(typeof bitrixBundle.authMode === "string" ? bitrixBundle.authMode : null);
      } else {
        setBitrixTasks([]);
        setBitrixTaskStats(null);
        setBitrixResponsibleId(null);
        setBitrixAuthMode(null);
      }

      await loadBitrixDeals(bitrixDealsSearch, Boolean(options?.pull));
      await loadBitrixNotifications();
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [bitrixDealsSearch, loadBitrixDeals, loadBitrixNotifications]);

  const openTaskDetail = useCallback((taskId: string) => {
    setSelectedTaskId(taskId);
  }, []);

  const closeTaskDetail = useCallback(() => {
    setSelectedTaskId(null);
  }, []);

  const openDealDetail = useCallback((dealId: string) => {
    setSelectedDealId(dealId);
  }, []);

  const closeDealDetail = useCallback(() => {
    setSelectedDealId(null);
  }, []);

  useEffect(() => {
    if (activeTab !== "deals") {
      return;
    }
    const timer = setTimeout(() => {
      void loadBitrixDeals(bitrixDealsSearch, true);
    }, 350);
    return () => clearTimeout(timer);
  }, [activeTab, bitrixDealsSearch, loadBitrixDeals]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    let alive = true;
    void getRequestLog().then((entries) => {
      if (alive) {
        setRequestLogEntries(entries);
      }
    });
    const unsubscribe = subscribeRequestLog(() => {
      void getRequestLog().then((entries) => {
        if (alive) {
          setRequestLogEntries(entries);
        }
      });
    });
    return () => {
      alive = false;
      unsubscribe();
    };
  }, []);

  const estimateTemplate = useMemo(() => {
    const matches = templates.filter((template) => {
      const cat = template.category.toLowerCase();
      if (cat === "estimate" || cat === "smeta" || cat === "смета" || cat === "сметы") {
        return true;
      }
      return template.name.toLowerCase().includes("смет");
    });
    const docx =
      matches.find((t) => t.fileName.toLowerCase().endsWith(".docx")) ??
      matches.find((t) => t.fileName.toLowerCase().endsWith(".doc"));
    return docx ?? matches[0] ?? null;
  }, [templates]);

  useEffect(() => {
    if (estimateTemplate && !requestForm.templateId) {
      setRequestForm((current) => ({ ...current, templateId: estimateTemplate.id }));
    }
  }, [estimateTemplate, requestForm.templateId]);

  const selectedTemplate = useMemo(
    () => templates.find((template) => template.id === requestForm.templateId) ?? estimateTemplate,
    [requestForm.templateId, templates, estimateTemplate]
  );

  const filteredRequestLog = useMemo(() => {
    const newestFirst = [...requestLogEntries].reverse();
    if (logFilter === "all") {
      return newestFirst;
    }
    if (logFilter === "errors") {
      return newestFirst.filter((e) => !e.ok);
    }
    return newestFirst.filter((e) => e.kind === logFilter);
  }, [requestLogEntries, logFilter]);

  const latestAssignedBitrixTask = useMemo(
    () => pickLatestAssignedBitrixTask(bitrixTasks, bitrixResponsibleId),
    [bitrixTasks, bitrixResponsibleId]
  );

  const unreadNotificationCount = useMemo(
    () => bitrixNotifications.filter((item) => !item.read).length,
    [bitrixNotifications]
  );

  const filteredBitrixTasks = useMemo(
    () => bitrixTasks.filter((task) => bitrixTaskMatchesFilter(task, bitrixTaskFilter)),
    [bitrixTasks, bitrixTaskFilter]
  );

  const applyTaskFilter = useCallback((filter: BitrixTaskFilterKey) => {
    setBitrixTaskFilter((current) => (current === filter ? "all" : filter));
    setActiveTab("tasks");
  }, []);

  const taskStatBoxStyle = useCallback(
    (filter: BitrixTaskFilterKey) =>
      bitrixTaskFilter === filter ? [styles.bitrixStatBox, styles.bitrixStatBoxActive] : styles.bitrixStatBox,
    [bitrixTaskFilter]
  );

  // Эмулятор Android Studio: ⋯ → Extended controls → Microphone → Virtual headset (или микрофон ПК);
  // при глюках — Cold Boot AVD; в системных настройках приложения включите RECORD_AUDIO.
  const stopPreviewPlayback = useCallback(async () => {
    const sound = previewSoundRef.current;
    previewSoundRef.current = null;
    setPreviewPlayingUri(null);
    if (sound) {
      try {
        await sound.stopAsync();
      } catch {
        // ignore
      }
      try {
        await sound.unloadAsync();
      } catch {
        // ignore
      }
    }
  }, []);

  const togglePreviewPlayback = useCallback(
    async (uri: string | null | undefined) => {
      if (!uri) {
        return;
      }
      if (previewPlayingUri === uri && previewSoundRef.current) {
        await stopPreviewPlayback();
        return;
      }
      await stopPreviewPlayback();
      try {
        await Audio.setAudioModeAsync({
          allowsRecordingIOS: false,
          playsInSilentModeIOS: true
        });
        const { sound } = await Audio.Sound.createAsync({ uri }, { shouldPlay: true });
        previewSoundRef.current = sound;
        setPreviewPlayingUri(uri);
        sound.setOnPlaybackStatusUpdate((status) => {
          if (status.isLoaded && status.didJustFinish) {
            void stopPreviewPlayback();
          }
        });
      } catch (nextError) {
        const message =
          nextError instanceof Error ? nextError.message : "Не удалось воспроизвести";
        setError(message);
        Alert.alert("Прослушать", message);
      }
    },
    [previewPlayingUri, stopPreviewPlayback]
  );

  const resetDocRecordingDraft = useCallback(async () => {
    await stopPreviewPlayback();
    const recording = docRecordingRef.current ?? docRecording;
    if (recording) {
      try {
        await recording.stopAndUnloadAsync();
      } catch {
        // ignore
      }
    }
    docRecordingRef.current = null;
    setDocRecording(null);
    setDocAudioUri(null);
    setDocRecordingPhase("idle");
  }, [docRecording, stopPreviewPlayback]);

  const resetBitrixRecordingDraft = useCallback(async () => {
    await stopPreviewPlayback();
    const recording = bitrixRecordingRef.current ?? bitrixRecording;
    if (recording) {
      try {
        await recording.stopAndUnloadAsync();
      } catch {
        // ignore
      }
    }
    bitrixRecordingRef.current = null;
    setBitrixRecording(null);
    setBitrixAudioUri(null);
    setBitrixRecordingPhase("idle");
  }, [bitrixRecording, stopPreviewPlayback]);

  const startDocRecording = async () => {
    await resetDocRecordingDraft();
    try {
      const permission = await Audio.requestPermissionsAsync();
      if (!permission.granted) {
        Alert.alert("Микрофон", "Разрешите запись аудио в настройках.");
        return;
      }
      await Audio.setAudioModeAsync({
        allowsRecordingIOS: true,
        playsInSilentModeIOS: true
      });
      const created = await Audio.Recording.createAsync(
        Audio.RecordingOptionsPresets.HIGH_QUALITY
      );
      docRecordingRef.current = created.recording;
      setDocRecording(created.recording);
      setDocAudioUri(null);
      setDocRecordingPhase("recording");
    } catch (nextError) {
      const message =
        nextError instanceof Error ? nextError.message : "Не удалось начать запись";
      setError(message);
      Alert.alert("Запись", message);
    }
  };

  const stopDocRecording = async () => {
    const recording = docRecordingRef.current ?? docRecording;
    if (!recording) {
      return;
    }
    setDocRecordingPhase("stopping");
    try {
      await recording.stopAndUnloadAsync();
      const uri = recording.getURI();
      docRecordingRef.current = null;
      setDocRecording(null);
      await Audio.setAudioModeAsync({
        allowsRecordingIOS: false,
        playsInSilentModeIOS: true
      });
      if (uri) {
        setDocAudioUri(uri);
        setDocRecordingPhase("ready");
      } else {
        setDocAudioUri(null);
        setDocRecordingPhase("idle");
        Alert.alert("Запись", "Не удалось сохранить файл. Попробуйте записать ещё раз.");
      }
    } catch (nextError) {
      docRecordingRef.current = null;
      setDocRecording(null);
      setDocRecordingPhase("idle");
      setError(nextError instanceof Error ? nextError.message : "Не удалось остановить запись");
    }
  };

  const startBitrixRecording = async () => {
    await resetBitrixRecordingDraft();
    try {
      const permission = await Audio.requestPermissionsAsync();
      if (!permission.granted) {
        Alert.alert("Микрофон", "Разрешите запись аудио в настройках.");
        return;
      }
      await Audio.setAudioModeAsync({
        allowsRecordingIOS: true,
        playsInSilentModeIOS: true
      });
      const created = await Audio.Recording.createAsync(
        Audio.RecordingOptionsPresets.HIGH_QUALITY
      );
      bitrixRecordingRef.current = created.recording;
      setBitrixRecording(created.recording);
      setBitrixAudioUri(null);
      setBitrixRecordingPhase("recording");
    } catch (nextError) {
      const message =
        nextError instanceof Error ? nextError.message : "Не удалось начать запись";
      setError(message);
      Alert.alert("Запись", message);
    }
  };

  const stopBitrixRecording = async () => {
    const recording = bitrixRecordingRef.current ?? bitrixRecording;
    if (!recording) {
      return;
    }
    setBitrixRecordingPhase("stopping");
    try {
      await recording.stopAndUnloadAsync();
      const uri = recording.getURI();
      bitrixRecordingRef.current = null;
      setBitrixRecording(null);
      await Audio.setAudioModeAsync({
        allowsRecordingIOS: false,
        playsInSilentModeIOS: true
      });
      if (uri) {
        setBitrixAudioUri(uri);
        setBitrixRecordingPhase("ready");
      } else {
        setBitrixAudioUri(null);
        setBitrixRecordingPhase("idle");
        Alert.alert("Запись", "Не удалось сохранить файл. Попробуйте записать ещё раз.");
      }
    } catch (nextError) {
      bitrixRecordingRef.current = null;
      setBitrixRecording(null);
      setBitrixRecordingPhase("idle");
      setError(nextError instanceof Error ? nextError.message : "Не удалось остановить запись");
    }
  };

  const openDocumentJobDetail = async (jobId: string) => {
    setDocDetailLoading(true);
    try {
      const item = await getMobileDocumentJob(jobId);
      setDocDetailJob(item);
    } catch (err) {
      Alert.alert("Ошибка", err instanceof Error ? err.message : "Не удалось открыть документ");
    } finally {
      setDocDetailLoading(false);
    }
  };

  const parseDealId = () => {
    const n = parseInt(bitrixHints.dealId.trim(), 10);
    return Number.isFinite(n) && n > 0 ? n : undefined;
  };

  const resetBitrixFormFields = useCallback(() => {
    setBitrixIntentText("");
    setBitrixHints({
      dealId: "",
      dealTitle: "",
      dealHint: "",
      stageHint: ""
    });
    setDealConfirmHint("");
  }, []);

  const markNotificationReadLocal = useCallback((notificationId: string) => {
    setBitrixNotifications((items) =>
      items.map((item) => (item.id === notificationId ? { ...item, read: true } : item))
    );
  }, []);

  const handleNotificationPress = useCallback(
    async (item: BitrixNotificationSummary) => {
      if (expandedNotificationId === item.id) {
        setExpandedNotificationId(null);
        return;
      }
      setExpandedNotificationId(item.id);
      if (!item.read) {
        markNotificationReadLocal(item.id);
        try {
          await markBitrixNotificationRead(item.id);
        } catch {
          // локально уже отметили — badge обновится
        }
      }
    },
    [expandedNotificationId, markNotificationReadLocal]
  );

  const markAllNotificationsReadHandler = useCallback(async () => {
    if (unreadNotificationCount === 0) {
      return;
    }
    try {
      await markAllBitrixNotificationsRead();
      setBitrixNotifications((items) => items.map((item) => ({ ...item, read: true })));
    } catch (err) {
      Alert.alert(
        "Уведомления",
        err instanceof Error ? err.message : "Не удалось отметить прочитанными"
      );
    }
  }, [unreadNotificationCount]);

  const beginBitrixDealConfirm = useCallback(
    (parsedTitle: string, transcript: string, source: "text" | "voice") => {
      setDealConfirmHint("");
      setBitrixDealConfirm({
        parsedTitle: parsedTitle.trim(),
        transcript: transcript.trim(),
        source,
        step: "confirm"
      });
    },
    []
  );

  const handleBitrixIntentFailure = useCallback(
    (err: unknown, source: "text" | "voice", skipDealConfirm = false) => {
      if (
        !skipDealConfirm &&
        err instanceof BitrixIntentError &&
        err.code === "deal_not_found" &&
        err.item
      ) {
        const title =
          err.item.parsedDealTitle?.trim() ||
          err.item.transcript?.trim() ||
          bitrixHints.dealTitle.trim() ||
          bitrixHints.dealHint.trim();
        beginBitrixDealConfirm(title, err.item.transcript ?? "", source);
        return;
      }
      const message = err instanceof Error ? err.message : "Ошибка Bitrix";
      setError(message);
      Alert.alert("Bitrix24", message);
    },
    [beginBitrixDealConfirm, bitrixHints.dealHint, bitrixHints.dealTitle]
  );

  const submitBitrixText = async (dealHintOverride?: string, skipDealConfirm = false) => {
    if (!bitrixIntentText.trim()) {
      return;
    }
    setSubmittingBitrix(true);
    beginSubmitProgress("Запрос в Bitrix24", bitrixTextProgressMessage);
    const hint = (dealHintOverride ?? bitrixHints.dealHint).trim();
    try {
      const item = await createMobileBitrixIntentText({
        text: bitrixIntentText.trim(),
        dealId: parseDealId(),
        dealTitle: hint || bitrixHints.dealTitle.trim() || undefined,
        dealHint: hint || undefined,
        stageHint: bitrixHints.stageHint.trim() || undefined
      });
      resetBitrixFormFields();
      setBitrixDealConfirm(null);
      void refresh();
      Alert.alert("Bitrix24", (item?.bitrixSteps ?? []).join("\n") || "Запрос выполнен.");
    } catch (nextError) {
      handleBitrixIntentFailure(nextError, "text", skipDealConfirm);
    } finally {
      endSubmitProgress();
      setSubmittingBitrix(false);
    }
  };

  const submitBitrixFromTranscript = async (
    transcript: string,
    dealTitle: string,
    skipDealConfirm = false
  ) => {
    if (!transcript.trim()) {
      Alert.alert("Bitrix24", "Нет распознанного текста для повторной отправки.");
      return;
    }
    setSubmittingBitrix(true);
    beginSubmitProgress("Bitrix24", bitrixTextProgressMessage);
    const hint = dealTitle.trim();
    try {
      const item = await createMobileBitrixIntentText({
        text: transcript.trim(),
        dealId: parseDealId(),
        dealTitle: hint || undefined,
        dealHint: hint || undefined,
        stageHint: bitrixHints.stageHint.trim() || undefined
      });
      setBitrixDealConfirm(null);
      resetBitrixFormFields();
      await stopPreviewPlayback();
      setBitrixAudioUri(null);
      setBitrixRecordingPhase("idle");
      setBitrixVoiceOpen(false);
      void refresh();
      Alert.alert("Bitrix24", (item?.bitrixSteps ?? []).join("\n") || "Запрос выполнен.");
    } catch (nextError) {
      handleBitrixIntentFailure(nextError, "voice", skipDealConfirm);
    } finally {
      endSubmitProgress();
      setSubmittingBitrix(false);
    }
  };

  const submitBitrixVoice = async (dealHintOverride?: string, skipDealConfirm = false) => {
    if (!bitrixAudioUri) {
      Alert.alert("Запись", "Сначала запишите голос или введите текст на главной.");
      return;
    }
    setSubmittingBitrix(true);
    beginSubmitProgress("Голос в Bitrix24", bitrixVoiceProgressMessage);
    const hint = (dealHintOverride ?? bitrixHints.dealHint).trim();
    try {
      const item = await createMobileBitrixIntentMultipart({
        audioUri: bitrixAudioUri,
        audioFileName: `bitrix-intent-${Date.now()}.m4a`,
        audioMimeType: "audio/mp4",
        dealId: parseDealId(),
        dealTitle: hint || bitrixHints.dealTitle.trim() || undefined,
        dealHint: hint || undefined,
        stageHint: bitrixHints.stageHint.trim() || undefined
      });
      await stopPreviewPlayback();
      setBitrixAudioUri(null);
      setBitrixRecordingPhase("idle");
      setBitrixDealConfirm(null);
      resetBitrixFormFields();
      setBitrixVoiceOpen(false);
      void refresh();
      Alert.alert("Bitrix24", (item?.bitrixSteps ?? []).join("\n") || "Запрос выполнен.");
    } catch (nextError) {
      handleBitrixIntentFailure(nextError, "voice", skipDealConfirm);
    } finally {
      endSubmitProgress();
      setSubmittingBitrix(false);
    }
  };

  const retryBitrixWithDealHint = async () => {
    const hint = dealConfirmHint.trim();
    const confirm = bitrixDealConfirm;
    if (!hint || !confirm) {
      Alert.alert("Подсказка", "Введите номер или название сделки.");
      return;
    }
    setBitrixHints((current) => ({ ...current, dealHint: hint, dealTitle: hint }));
    setBitrixDealConfirm(null);
    if (confirm.source === "voice" && confirm.transcript.trim()) {
      await submitBitrixFromTranscript(confirm.transcript, hint, true);
      return;
    }
    await submitBitrixText(hint, true);
  };

  const openEstimateVoice = () => {
    if (estimateTemplate) {
      setRequestForm((current) => ({ ...current, templateId: estimateTemplate.id }));
    }
    void loadBitrixDeals(bitrixDealsSearch, false);
    setDocVoiceOpen(true);
  };

  const renderRecordingControls = (
    phase: RecordingPhase,
    audioUri: string | null,
    onStart: () => void,
    onStop: () => void,
    onReset: () => void,
    onPreview: () => void
  ) => {
    if (phase === "recording") {
      return (
        <Pressable
          onPress={onStop}
          style={[styles.button, styles.secondaryButton, styles.submitWide]}
        >
          <Text style={styles.buttonTextDark}>Стоп</Text>
        </Pressable>
      );
    }
    if (phase === "stopping") {
      return (
        <View style={[styles.button, styles.secondaryButton, styles.submitWide, styles.buttonDisabled]}>
          <ActivityIndicator color="#0f172a" />
          <Text style={[styles.buttonTextDark, { marginTop: 6 }]}>Сохранение записи…</Text>
        </View>
      );
    }
    if (phase === "ready" && audioUri) {
      return (
        <>
          <Pressable onPress={onPreview} style={[styles.button, styles.secondaryButton]}>
            <Text style={styles.buttonTextDark}>
              {previewPlayingUri === audioUri ? "Стоп" : "Прослушать"}
            </Text>
          </Pressable>
          <Pressable onPress={onReset} style={[styles.button, styles.secondaryButton]}>
            <Text style={styles.buttonTextDark}>Перезаписать</Text>
          </Pressable>
        </>
      );
    }
    return (
      <Pressable
        onPress={onStart}
        style={[styles.button, styles.primaryButton, styles.submitWide]}
      >
        <Text style={styles.buttonText}>Начать</Text>
      </Pressable>
    );
  };

  const recordingStatusText = (phase: RecordingPhase) => {
    switch (phase) {
      case "recording":
        return "Идёт запись…";
      case "stopping":
        return "Сохранение записи…";
      case "ready":
        return "Аудио готово — можно прослушать или перезаписать.";
      default:
        return "Запись не сделана.";
    }
  };

  const renderSubmitOverlay = () => {
    if (!submitProgress) {
      return null;
    }
    return (
      <Modal visible transparent animationType="fade" onRequestClose={() => undefined}>
        <View style={styles.progressOverlay}>
          <View style={styles.progressCard}>
            <ActivityIndicator size="large" color={HEADER_BLUE} />
            <Text style={styles.progressTitle}>{submitProgress.title}</Text>
            <Text style={styles.progressMessage}>{submitProgress.message}</Text>
            <Text style={styles.progressElapsed}>Прошло {submitProgress.elapsedSec} сек</Text>
            <Text style={styles.progressHint}>
              Не закрывайте приложение. Распознавание речи может занять до 2 минут.
            </Text>
          </View>
        </View>
      </Modal>
    );
  };

  const renderBitrixSubmitBanner = () => {
    if (!submittingBitrix || submitProgress) {
      return null;
    }
    return (
      <View style={styles.inlineProgress}>
        <ActivityIndicator color={HEADER_BLUE} />
        <Text style={styles.inlineProgressText}>Отправка в Bitrix24…</Text>
      </View>
    );
  };

  const renderBottomNav = () => (
    <>
      <View style={[styles.bottomNav, { paddingBottom: Platform.OS === "android" ? 10 : 18 }]}>
        <Pressable style={styles.navItem} onPress={() => setActiveTab("home")}>
          <Ionicons
            name="home"
            size={22}
            color={activeTab === "home" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "home" && styles.navLabelActive]}>Главная</Text>
        </Pressable>
        <Pressable style={styles.navItem} onPress={() => setActiveTab("tasks")}>
          <Ionicons
            name="checkbox-outline"
            size={22}
            color={activeTab === "tasks" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "tasks" && styles.navLabelActive]}>Задачи</Text>
        </Pressable>
        <Pressable style={styles.navItem} onPress={() => setActiveTab("docs")}>
          <Ionicons
            name="document-text-outline"
            size={22}
            color={activeTab === "docs" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "docs" && styles.navLabelActive]}>Документы</Text>
        </Pressable>
        <Pressable style={styles.navItem} onPress={() => setActiveTab("deals")}>
          <Ionicons
            name="briefcase-outline"
            size={22}
            color={activeTab === "deals" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "deals" && styles.navLabelActive]}>Сделки</Text>
        </Pressable>
        <Pressable style={styles.navItem} onPress={() => setActiveTab("more")}>
          <Ionicons
            name="grid-outline"
            size={22}
            color={activeTab === "more" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "more" && styles.navLabelActive]}>Ещё</Text>
        </Pressable>
      </View>
    </>
  );

  const renderBitrixDealConfirmModal = () => {
    if (!bitrixDealConfirm) {
      return null;
    }
    const titleLabel =
      bitrixDealConfirm.parsedTitle ||
      (bitrixDealConfirm.transcript.length > 80
        ? `${bitrixDealConfirm.transcript.slice(0, 80)}…`
        : bitrixDealConfirm.transcript) ||
      "эту сделку";

    return (
      <Modal
        visible
        transparent
        animationType="fade"
        onRequestClose={() => setBitrixDealConfirm(null)}
      >
        <Pressable style={styles.authOverlay} onPress={() => setBitrixDealConfirm(null)}>
          <Pressable style={styles.authSheet} onPress={() => undefined}>
            {bitrixDealConfirm.step === "confirm" ? (
              <>
                <Text style={styles.authSheetTitle}>Сделка не найдена</Text>
                <Text style={styles.muted}>
                  Вы имели в виду «{titleLabel}»?
                </Text>
                <Pressable
                  onPress={() => {
                    setBitrixDealConfirm(null);
                    Alert.alert(
                      "Сделка не найдена",
                      `Сделки «${titleLabel}» нет в Bitrix24.`
                    );
                  }}
                  style={[styles.button, styles.primaryButton, styles.submitWide]}
                >
                  <Text style={styles.buttonText}>Да, это она</Text>
                </Pressable>
                <Pressable
                  onPress={() =>
                    setBitrixDealConfirm((current) =>
                      current ? { ...current, step: "hint" } : current
                    )
                  }
                  style={[styles.button, styles.secondaryButton, styles.submitWide]}
                >
                  <Text style={styles.buttonTextDark}>Нет, уточнить</Text>
                </Pressable>
              </>
            ) : (
              <>
                <Text style={styles.authSheetTitle}>Уточните сделку</Text>
                <Text style={styles.muted}>
                  Введите номер или название сделки. Повторим запрос с вашей подсказкой — голос
                  заново не распознаём.
                </Text>
                <TextInput
                  placeholder="Номер или название сделки"
                  placeholderTextColor="#94a3b8"
                  style={styles.input}
                  value={dealConfirmHint}
                  onChangeText={setDealConfirmHint}
                  autoFocus
                />
                <Pressable
                  onPress={() => void retryBitrixWithDealHint()}
                  disabled={submittingBitrix || !dealConfirmHint.trim()}
                  style={[
                    styles.button,
                    styles.primaryButton,
                    styles.submitWide,
                    (submittingBitrix || !dealConfirmHint.trim()) && styles.buttonDisabled
                  ]}
                >
                  <Text style={styles.buttonText}>
                    {submittingBitrix ? "Отправка…" : "Отправить снова"}
                  </Text>
                </Pressable>
              </>
            )}
            <Pressable onPress={() => setBitrixDealConfirm(null)} style={styles.authSheetClose}>
              <Text style={styles.linkInline}>Отмена</Text>
            </Pressable>
          </Pressable>
        </Pressable>
      </Modal>
    );
  };

  const renderBitrixAuthModal = () => (
    <Modal
      visible={bitrixAuthOpen}
      transparent
      animationType="fade"
      onRequestClose={() => setBitrixAuthOpen(false)}
    >
      <Pressable style={styles.authOverlay} onPress={() => setBitrixAuthOpen(false)}>
        <Pressable style={styles.authSheet} onPress={() => undefined}>
          <Text style={styles.authSheetTitle}>Bitrix24</Text>
          {bitrixAuth.loading ? (
            <ActivityIndicator color={HEADER_BLUE} />
          ) : bitrixAuth.connected && bitrixAuth.session ? (
            <>
              <Text style={styles.muted}>
                Вы вошли как {bitrixAuth.session.userName || `id ${bitrixAuth.session.bitrixUserId}`}.
                Задачи и уведомления загружаются от вашего аккаунта.
              </Text>
              {bitrixAuth.session.taskScopeGranted === false ? (
                <Text style={styles.errorText}>
                  Нет права «Задачи» у приложения. В Bitrix24 включите scope task в локальном
                  приложении, затем выйдите и войдите снова.
                </Text>
              ) : null}
              <Pressable
                onPress={() =>
                  void bitrixAuth.disconnect().then(() => {
                    setBitrixAuthOpen(false);
                    return refresh();
                  })
                }
                style={[styles.button, styles.secondaryButton, styles.submitWide]}
              >
                <Text style={styles.buttonTextDark}>Выйти из Bitrix24</Text>
              </Pressable>
            </>
          ) : (
            <>
              <Text style={styles.muted}>
                Войдите в Bitrix24, чтобы видеть свои задачи и уведомления (не только пользователя
                вебхука на сервере).
              </Text>
              <Pressable
                onPress={() =>
                  void bitrixAuth.connect().then(() => {
                    setBitrixAuthOpen(false);
                    return refresh();
                  })
                }
                disabled={bitrixAuth.connecting}
                style={[
                  styles.button,
                  styles.primaryButton,
                  styles.submitWide,
                  bitrixAuth.connecting && styles.buttonDisabled
                ]}
              >
                <Text style={styles.buttonText}>
                  {bitrixAuth.connecting ? "Открываем Bitrix…" : "Войти в Bitrix24"}
                </Text>
              </Pressable>
            </>
          )}
          <Pressable onPress={() => setBitrixAuthOpen(false)} style={styles.authSheetClose}>
            <Text style={styles.linkInline}>Закрыть</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );

  const renderHomeDashboard = () => (
    <>
      <View style={styles.card}>
        <View style={styles.cardHeader}>
          <Text style={styles.cardTitle}>Мои задачи Bitrix24</Text>
          <Pressable onPress={() => setActiveTab("tasks")}>
            <Text style={styles.linkInline}>Все задачи</Text>
          </Pressable>
        </View>
        {bitrixTaskStats ? (
          <View style={styles.bitrixStatsRow}>
            <Pressable style={taskStatBoxStyle("open")} onPress={() => applyTaskFilter("open")}>
              <Text style={styles.bitrixStatNum}>{bitrixTaskStats.totalOpen}</Text>
              <Text style={styles.bitrixStatLabel}>Открыто</Text>
            </Pressable>
            <Pressable style={taskStatBoxStyle("in_progress")} onPress={() => applyTaskFilter("in_progress")}>
              <Text style={[styles.bitrixStatNum, { color: "#0369a1" }]}>
                {bitrixTaskStats.inProgress}
              </Text>
              <Text style={styles.bitrixStatLabel}>В работе</Text>
            </Pressable>
            <Pressable style={taskStatBoxStyle("overdue")} onPress={() => applyTaskFilter("overdue")}>
              <Text style={[styles.bitrixStatNum, { color: "#b91c1c" }]}>
                {bitrixTaskStats.overdue}
              </Text>
              <Text style={styles.bitrixStatLabel}>Просрочено</Text>
            </Pressable>
          </View>
        ) : bitrixTasksError ? (
          <Text style={styles.muted}>{bitrixTasksError}</Text>
        ) : bitrixAuth.connected ? (
          <Text style={styles.muted}>
            Задачи не найдены для вашего аккаунта Bitrix (ответственный / соисполнитель).
          </Text>
        ) : (
          <Text style={styles.muted}>
            Нажмите иконку профиля вверху слева, чтобы войти в Bitrix24.
          </Text>
        )}
        {latestAssignedBitrixTask ? (
          <Pressable
            style={styles.notifRow}
            onPress={() => openTaskDetail(latestAssignedBitrixTask.id)}
          >
            <Text style={styles.notifText} numberOfLines={2}>
              {latestAssignedBitrixTask.title || "Без названия"}
            </Text>
            <Ionicons name="chevron-forward" size={18} color="#94a3b8" />
          </Pressable>
        ) : null}
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Запрос в Bitrix24</Text>
        <Text style={styles.muted}>
          Текстом или голосом (кнопка снизу): перенос сделки, стадия, создание задачи — как в
          голосовом разборе на сервере. При необходимости уточните сделку полями ниже.
        </Text>
        <TextInput
          placeholder="Например: переведи сделку «Название» на следующий этап"
          placeholderTextColor="#94a3b8"
          style={[styles.input, styles.textArea]}
          multiline
          value={bitrixIntentText}
          onChangeText={setBitrixIntentText}
        />
        <Text style={styles.sectionLabel}>Подсказки (необязательно)</Text>
        <TextInput
          placeholder="ID сделки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          keyboardType="number-pad"
          value={bitrixHints.dealId}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealId: v }))}
        />
        <TextInput
          placeholder="Название сделки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.dealTitle}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealTitle: v }))}
        />
        <TextInput
          placeholder="Подсказка по сделке (номер или имя)"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.dealHint}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealHint: v }))}
        />
        <TextInput
          placeholder="Целевая стадия (если в тексте неясно)"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.stageHint}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, stageHint: v }))}
        />
        <Pressable
          onPress={() => void submitBitrixText()}
          disabled={submittingBitrix || !bitrixIntentText.trim()}
          style={[
            styles.button,
            styles.primaryButton,
            (submittingBitrix || !bitrixIntentText.trim()) && styles.buttonDisabled
          ]}
        >
          <Text style={styles.buttonText}>
            {submittingBitrix ? "Отправка…" : "Отправить в Bitrix"}
          </Text>
        </Pressable>
        {renderBitrixSubmitBanner()}
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Голос в Bitrix24</Text>
        <Text style={styles.muted}>
          Запишите команду голосом: смена этапа сделки, поиск по названию и другие действия в CRM.
        </Text>
        <Pressable style={styles.docTileWide} onPress={() => setBitrixVoiceOpen(true)}>
          <View style={styles.docIconWrapWide}>
            <Ionicons name="mic-outline" size={32} color={HEADER_BLUE} />
          </View>
          <Text style={styles.docTileLabelWide}>Записать команду</Text>
        </Pressable>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Смета</Text>
        <Text style={styles.muted}>
          Диктуйте поля по фразам: «наименование стройки ТРЦ Кристалл», «сметная стоимость 1 250 000
          рублей», «позиция 1, наименование монтаж кровли, количество 500». Текст подставится в
          шаблон сметы автоматически.
        </Text>
        <Pressable style={styles.docTileWide} onPress={openEstimateVoice}>
          <View style={styles.docIconWrapWide}>
            <Ionicons name="mic-outline" size={32} color={HEADER_BLUE} />
          </View>
          <Text style={styles.docTileLabelWide}>Записать смету голосом</Text>
        </Pressable>
      </View>

      {error ? (
        <View style={styles.errorBanner}>
          <Text style={styles.errorText}>{error}</Text>
          <Pressable onPress={() => void refresh()} style={styles.errorRetry}>
            <Text style={styles.errorRetryText}>Повторить</Text>
          </Pressable>
        </View>
      ) : null}
    </>
  );

  const renderTasksTab = () => (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Задачи Bitrix24</Text>
        <Text style={styles.muted}>
          Список задач Bitrix24. Потяните экран вниз, чтобы обновить. Нажмите задачу — описание и
          смена статуса.
        </Text>
        {bitrixTaskStats ? (
          <View style={styles.bitrixStatsRow}>
            <Pressable style={taskStatBoxStyle("open")} onPress={() => applyTaskFilter("open")}>
              <Text style={styles.bitrixStatNum}>{bitrixTaskStats.totalOpen}</Text>
              <Text style={styles.bitrixStatLabel}>Открыто</Text>
            </Pressable>
            <Pressable style={taskStatBoxStyle("in_progress")} onPress={() => applyTaskFilter("in_progress")}>
              <Text style={[styles.bitrixStatNum, { color: "#0369a1" }]}>
                {bitrixTaskStats.inProgress}
              </Text>
              <Text style={styles.bitrixStatLabel}>В работе</Text>
            </Pressable>
            <Pressable style={taskStatBoxStyle("overdue")} onPress={() => applyTaskFilter("overdue")}>
              <Text style={[styles.bitrixStatNum, { color: "#b91c1c" }]}>
                {bitrixTaskStats.overdue}
              </Text>
              <Text style={styles.bitrixStatLabel}>Просрочено</Text>
            </Pressable>
          </View>
        ) : null}
        {bitrixTaskFilter !== "all" ? (
          <View style={styles.filterBanner}>
            <Text style={styles.filterBannerText}>
              Фильтр: {bitrixTaskFilterLabel(bitrixTaskFilter)} · {filteredBitrixTasks.length} из{" "}
              {bitrixTasks.length}
            </Text>
            <Pressable onPress={() => setBitrixTaskFilter("all")} hitSlop={8}>
              <Text style={styles.linkInline}>Сбросить</Text>
            </Pressable>
          </View>
        ) : null}
        {loading ? <ActivityIndicator color={HEADER_BLUE} /> : null}
        {!loading && filteredBitrixTasks.length === 0 ? (
          <Text style={styles.muted}>
            {bitrixTasksError ??
              (bitrixTaskFilter !== "all"
                ? `Нет задач в категории «${bitrixTaskFilterLabel(bitrixTaskFilter)}».`
                : bitrixAuth.connected
                  ? "Нет задач, где вы ответственный или соисполнитель."
                  : "Войдите в Bitrix24, чтобы видеть свои задачи.")}
          </Text>
        ) : null}
        {filteredBitrixTasks.map((task) => (
          <Pressable
            key={task.id}
            style={styles.listItem}
            onPress={() => openTaskDetail(task.id)}
          >
            <View style={styles.listHeader}>
              <Text style={styles.listTitle}>{task.title || "Без названия"}</Text>
              <Text style={styles.badge}>{bitrixTaskStatusRu(task.status)}</Text>
            </View>
            {task.deadline ? (
              <Text style={styles.muted}>Срок: {task.deadline}</Text>
            ) : (
              <Text style={styles.muted}>Срок не указан</Text>
            )}
            <Text style={styles.muted}>id: {task.id} · нажмите для подробностей</Text>
          </Pressable>
        ))}
      </View>
    </>
  );

  const renderNotificationsScreen = () => (
    <View style={styles.voiceRoot}>
      <View style={[styles.voiceHeader, { paddingTop: topInset + 8 }]}>
        <Pressable onPress={() => setNotificationsOpen(false)} hitSlop={12}>
          <Text style={styles.voiceClose}>Закрыть</Text>
        </Pressable>
        <Text style={styles.voiceTitle}>Уведомления</Text>
        <Pressable
          onPress={() => void markAllNotificationsReadHandler()}
          hitSlop={12}
          disabled={unreadNotificationCount === 0}
        >
          <Text
            style={[
              styles.voiceClose,
              { textAlign: "right" },
              unreadNotificationCount === 0 && { opacity: 0.45 }
            ]}
          >
            Прочитать все
          </Text>
        </Pressable>
      </View>
      <ScrollView
        style={styles.voiceScroll}
        contentContainerStyle={styles.voiceScrollContent}
        refreshControl={
          <RefreshControl
            refreshing={bitrixNotificationsLoading}
            onRefresh={() => void loadBitrixNotifications()}
            tintColor={HEADER_BLUE}
            colors={[HEADER_BLUE]}
          />
        }
      >
        {bitrixNotificationsError ? <Text style={styles.errorText}>{bitrixNotificationsError}</Text> : null}
        {bitrixNotificationsLoading && bitrixNotifications.length === 0 ? (
          <ActivityIndicator color={HEADER_BLUE} />
        ) : null}
        {!bitrixNotificationsLoading && !bitrixNotificationsError && bitrixNotifications.length === 0 ? (
          <Text style={styles.muted}>
            {bitrixAuth.connected
              ? "Уведомлений нет или нет прав im/notifications."
              : "Войдите в Bitrix24 через иконку профиля, чтобы видеть уведомления."}
          </Text>
        ) : null}
        {bitrixNotifications.map((item) => {
          const expanded = expandedNotificationId === item.id;
          return (
            <Pressable
              key={item.id}
              onPress={() => void handleNotificationPress(item)}
              style={[styles.listItem, !item.read && styles.notificationUnread]}
            >
              <View style={styles.listHeader}>
                <Text style={styles.listTitle}>{item.title || item.module || "Уведомление"}</Text>
                {!item.read ? <Text style={styles.badge}>новое</Text> : null}
              </View>
              <Text style={styles.muted} numberOfLines={expanded ? undefined : 2}>
                {item.text || "—"}
              </Text>
              <Text style={styles.muted}>{formatBitrixDate(item.date)}</Text>
              {expanded ? (
                <Text style={styles.linkInline}>{item.read ? "Свернуть" : "Прочитано"}</Text>
              ) : (
                <Text style={styles.muted}>Нажмите, чтобы раскрыть</Text>
              )}
            </Pressable>
          );
        })}
      </ScrollView>
    </View>
  );

  const renderDealsTab = () => (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Сделки Bitrix24</Text>
        <Text style={styles.muted}>
          Карточки сделок с текущим этапом воронки. Нажмите на сделку, чтобы сменить этап вручную.
          {bitrixDealsAuthMode ? ` · источник: ${bitrixDealsAuthMode}` : ""}
          {bitrixDeals.length > 0 ? ` · ${bitrixDeals.length} шт.` : ""}
        </Text>
        <TextInput
          placeholder="Поиск по названию сделки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixDealsSearch}
          onChangeText={setBitrixDealsSearch}
          returnKeyType="search"
        />
        {bitrixDealsError ? <Text style={styles.errorText}>{bitrixDealsError}</Text> : null}
        {bitrixDealsLoading ? <ActivityIndicator color={HEADER_BLUE} /> : null}
        {!bitrixDealsLoading && !bitrixDealsError && bitrixDeals.length === 0 ? (
          <Text style={styles.muted}>
            {bitrixAuth.connected
              ? "Сделки не найдены. Проверьте права CRM в Bitrix или попробуйте другой запрос."
              : "Войдите в Bitrix24, чтобы видеть сделки."}
          </Text>
        ) : null}
        {bitrixDeals.filter((deal) => deal.id).map((deal) => (
          <Pressable key={deal.id} style={styles.listItem} onPress={() => openDealDetail(deal.id)}>
            <View style={styles.listHeader}>
              <Text style={styles.listTitle}>{deal.title || "Без названия"}</Text>
              <Text style={styles.badge}>{deal.stageLabel || deal.stageId || "—"}</Text>
            </View>
            {deal.opportunity ? (
              <Text style={styles.muted}>
                Сумма: {formatBitrixMoney(deal.opportunity, deal.currencyId)}
              </Text>
            ) : null}
            <Text style={styles.muted}>
              id: {deal.id}
              {deal.dateModify || deal.dateCreate
                ? ` · ${formatBitrixDate(deal.dateModify ?? deal.dateCreate)}`
                : ""}
            </Text>
          </Pressable>
        ))}
      </View>
    </>
  );

  const renderDocsTab = () => (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Мои сметы</Text>
        <Text style={styles.muted}>
          Все сформированные документы с этого телефона. Нажмите, чтобы посмотреть распознанные поля и
          скачать файл.
        </Text>
        <Pressable style={[styles.button, styles.primaryButton, { marginTop: 12 }]} onPress={openEstimateVoice}>
          <Text style={styles.buttonText}>Новая смета голосом</Text>
        </Pressable>
        {mobileDocumentJobs.length === 0 ? (
          <Text style={[styles.muted, { marginTop: 12 }]}>Пока нет документов.</Text>
        ) : null}
        {mobileDocumentJobs.map((entry) => (
          <Pressable
            key={entry.job.id}
            style={styles.listItem}
            onPress={() => void openDocumentJobDetail(entry.job.id)}
          >
            <View style={styles.listHeader}>
              <Text style={styles.listTitle}>{entry.job.sourceName}</Text>
              <Text style={[styles.badge, { color: statusColor(entry.job.status) }]}>
                {jobStatusRu(entry.job.status)}
              </Text>
            </View>
            <Text style={styles.muted}>
              {entry.job.templateName}
              {entry.job.bitrixDealTitle ? ` · ${entry.job.bitrixDealTitle}` : ""}
            </Text>
            <Text style={styles.muted}>{formatDate(entry.job.createdAt)}</Text>
          </Pressable>
        ))}
      </View>
    </>
  );

  const renderMoreTab = () => {
    const total = filteredRequestLog.length;
    const shown = logExpanded ? total : Math.min(REQUEST_LOG_COLLAPSED_COUNT, total);
    const slice = filteredRequestLog.slice(0, shown);
    const filterChips: { id: LogFilterTab; label: string }[] = [
      { id: "all", label: "Все" },
      { id: "bitrix", label: "Bitrix" },
      { id: "document", label: "Документы" },
      { id: "data", label: "Справочно" },
      { id: "errors", label: "Ошибки" }
    ];
    return (
      <>
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Сервер</Text>
          <Text style={styles.muted}>{getMobileApiBaseUrl()}</Text>
          <Text style={styles.muted}>
            Статус: {health?.status ?? "…"} · заявок: {health?.jobsCreatedTotal ?? 0}
          </Text>
          <Pressable onPress={() => void refresh()} style={styles.linkBtn}>
            <Text style={styles.linkBtnText}>Обновить</Text>
          </Pressable>
        </View>
        <View style={styles.card}>
        <Text style={styles.cardTitle}>Журнал запросов</Text>
        <Text style={styles.muted}>
          Журнал хранится на телефоне (до 80 записей). Те же действия Bitrix дублируются в админке в
          «Журнал событий».
        </Text>
        <Text style={styles.sectionLabel}>Журнал</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.logFilterRow}>
          {filterChips.map((chip) => (
            <Pressable
              key={chip.id}
              onPress={() => {
                setLogFilter(chip.id);
                setLogExpanded(false);
              }}
              style={[styles.filterChip, logFilter === chip.id && styles.filterChipOn]}
            >
              <Text style={logFilter === chip.id ? styles.filterChipTextOn : styles.filterChipText}>
                {chip.label}
              </Text>
            </Pressable>
          ))}
        </ScrollView>
        {total === 0 ? (
          <Text style={styles.muted}>
            В этой категории пока пусто — смените фильтр или выполните запрос на главной.
          </Text>
        ) : (
          <>
            <Text style={styles.muted}>
              Показано {shown} из {total}
              {!logExpanded && total > REQUEST_LOG_COLLAPSED_COUNT
                ? ` (свернуто, по ${REQUEST_LOG_COLLAPSED_COUNT} последних)`
                : ""}
            </Text>
            {slice.map((entry) => (
              <View key={entry.id} style={styles.logItem}>
                <View style={styles.logItemHeader}>
                  <Text style={styles.logCardTitle}>{entry.titleRu}</Text>
                  <Text
                    style={[
                      styles.logBadge,
                      { color: entry.ok ? "#15803d" : "#b91c1c", borderColor: entry.ok ? "#bbf7d0" : "#fecaca" }
                    ]}
                  >
                    {entry.ok ? "Успешно" : "Ошибка"}
                  </Text>
                </View>
                <Text style={styles.logLineMuted}>
                  {formatDate(entry.at)} · {entry.durationMs} мс
                </Text>
                <Text style={styles.logSummary}>{entry.summary}</Text>
                {entry.transcript ? (
                  <Text style={styles.logDetail}>Распознано: {entry.transcript}</Text>
                ) : null}
                {entry.parsedAction ? (
                  <Text style={styles.logDetail}>
                    Что сделали: {actionLabelRu(entry.parsedAction) ?? entry.parsedAction}
                  </Text>
                ) : null}
                {entry.parsedDealTitle ? (
                  <Text style={styles.logDetail}>Сделка: «{entry.parsedDealTitle}»</Text>
                ) : null}
                {entry.bitrixStepsLine ? (
                  <Text style={styles.logDetail}>В Bitrix: {entry.bitrixStepsLine}</Text>
                ) : null}
                {!entry.ok && entry.errorMessage ? (
                  <Text style={styles.logError}>{entry.errorMessage}</Text>
                ) : null}
              </View>
            ))}
            {total > REQUEST_LOG_COLLAPSED_COUNT ? (
              <Pressable
                onPress={() => setLogExpanded((e) => !e)}
                style={[styles.button, styles.secondaryButton, styles.submitWide]}
              >
                <Text style={styles.buttonTextDark}>
                  {logExpanded ? "Свернуть" : `Показать все (${total})`}
                </Text>
              </Pressable>
            ) : null}
          </>
        )}
        </View>
      </>
    );
  };

  const renderBitrixVoiceFullScreen = () => (
    <View style={styles.voiceRoot}>
      <View style={[styles.voiceHeader, { paddingTop: topInset + 8 }]}>
        <Pressable
          onPress={() => {
            void stopPreviewPlayback();
            setBitrixVoiceOpen(false);
          }}
          hitSlop={12}
        >
          <Text style={styles.voiceClose}>Закрыть</Text>
        </Pressable>
        <Text style={styles.voiceTitle}>Голос в Bitrix24</Text>
        <View style={{ width: 56 }} />
      </View>
      <ScrollView
        style={styles.voiceScroll}
        contentContainerStyle={styles.voiceScrollContent}
        keyboardShouldPersistTaps="handled"
        nestedScrollEnabled
      >
        <Text style={styles.muted}>
          Запишите фразу (например «переведи сделку … на следующий этап»). Если сделка не
          распознается, укажите её в подсказках ниже.
        </Text>
        <Text style={styles.sectionLabel}>Подсказки</Text>
        <TextInput
          placeholder="ID сделки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          keyboardType="number-pad"
          value={bitrixHints.dealId}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealId: v }))}
        />
        <TextInput
          placeholder="Название сделки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.dealTitle}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealTitle: v }))}
        />
        <TextInput
          placeholder="Подсказка по сделке"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.dealHint}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, dealHint: v }))}
        />
        <TextInput
          placeholder="Стадия"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={bitrixHints.stageHint}
          onChangeText={(v) => setBitrixHints((h) => ({ ...h, stageHint: v }))}
        />
        <Text style={styles.sectionLabel}>Запись</Text>
        <View style={styles.row} key={`bitrix-rec-${bitrixRecordingPhase}`}>
          {renderRecordingControls(
            bitrixRecordingPhase,
            bitrixAudioUri,
            () => void startBitrixRecording(),
            () => void stopBitrixRecording(),
            () => void resetBitrixRecordingDraft(),
            () => void togglePreviewPlayback(bitrixAudioUri)
          )}
        </View>
        <Text style={styles.muted}>{recordingStatusText(bitrixRecordingPhase)}</Text>
        <Pressable
          onPress={() => void submitBitrixVoice()}
          disabled={submittingBitrix || bitrixRecordingPhase !== "ready" || !bitrixAudioUri}
          style={[
            styles.button,
            styles.primaryButton,
            styles.submitWide,
            (submittingBitrix || bitrixRecordingPhase !== "ready" || !bitrixAudioUri) &&
              styles.buttonDisabled
          ]}
        >
          <Text style={styles.buttonText}>
            {submittingBitrix ? "Отправка…" : "Отправить в Bitrix"}
          </Text>
        </Pressable>
      </ScrollView>
    </View>
  );

  const renderVoiceFullScreen = () => (
    <EstimateDocumentFlowScreen
      topInset={topInset}
      template={selectedTemplate}
      deals={bitrixDeals}
      dealsLoading={bitrixDealsLoading}
      bitrixConnected={bitrixAuth.connected}
      recordingPhase={docRecordingPhase}
      audioUri={docAudioUri}
      sourceName={requestForm.sourceName}
      payloadExtra={requestForm.payload}
      onClose={() => {
        void stopPreviewPlayback();
        setDocVoiceOpen(false);
      }}
      onSuccess={() => void refresh()}
      onSourceNameChange={(value) => setRequestForm((c) => ({ ...c, sourceName: value }))}
      onPayloadExtraChange={(value) => setRequestForm((c) => ({ ...c, payload: value }))}
      onStartRecording={() => void startDocRecording()}
      onStopRecording={() => void stopDocRecording()}
      onResetRecording={() => void resetDocRecordingDraft()}
      onPreview={() => void togglePreviewPlayback(docAudioUri)}
      previewPlayingUri={previewPlayingUri}
      renderRecordingControls={renderRecordingControls}
      recordingStatusText={recordingStatusText}
    />
  );

  if (bitrixVoiceOpen) {
    return (
      <View style={styles.root}>
        <ExpoStatusBar style="dark" />
        {renderBitrixVoiceFullScreen()}
        {renderSubmitOverlay()}
        {renderBitrixDealConfirmModal()}
      </View>
    );
  }

  if (docVoiceOpen) {
    return (
      <View style={styles.root}>
        <ExpoStatusBar style="dark" />
        {renderVoiceFullScreen()}
        {renderSubmitOverlay()}
      </View>
    );
  }

  if (notificationsOpen) {
    return (
      <View style={styles.root}>
        <ExpoStatusBar style="dark" />
        {renderNotificationsScreen()}
      </View>
    );
  }

  return (
    <View style={styles.root}>
      <ExpoStatusBar style="light" />
      <View style={[styles.header, { paddingTop: topInset + 10 }]}>
        <Pressable onPress={() => setBitrixAuthOpen(true)} hitSlop={12} style={styles.headerSideBtn}>
          <View>
            <Ionicons
              name={bitrixAuth.connected ? "person" : "person-outline"}
              size={24}
              color="#fff"
            />
            {!bitrixAuth.connected && !bitrixAuth.loading ? (
              <View style={styles.headerAuthDot} />
            ) : null}
          </View>
        </Pressable>
        <Text style={styles.headerTitle}>
          {activeTab === "home"
            ? "Главная"
            : activeTab === "tasks"
              ? "Задачи"
              : activeTab === "deals"
                ? "Сделки"
                : activeTab === "docs"
                  ? "Документы"
                  : "Ещё"}
        </Text>
        <Pressable
          onPress={() => {
            setNotificationsOpen(true);
            void loadBitrixNotifications();
          }}
          hitSlop={12}
          style={styles.headerSideBtn}
        >
          <View>
            <Ionicons name="notifications-outline" size={24} color="#fff" />
            {unreadNotificationCount > 0 ? (
              <View style={styles.headerNotifDot}>
                <Text style={styles.headerNotifDotText}>
                  {unreadNotificationCount > 9 ? "9+" : unreadNotificationCount}
                </Text>
              </View>
            ) : null}
          </View>
        </Pressable>
      </View>

      <ScrollView
        style={styles.mainScroll}
        contentContainerStyle={styles.mainScrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => void refresh({ pull: true })}
            tintColor={HEADER_BLUE}
            colors={[HEADER_BLUE]}
          />
        }
      >
        {activeTab === "home" ? renderHomeDashboard() : null}
        {activeTab === "tasks" ? renderTasksTab() : null}
        {activeTab === "deals" ? renderDealsTab() : null}
        {activeTab === "docs" ? renderDocsTab() : null}
        {activeTab === "more" ? renderMoreTab() : null}
      </ScrollView>

      {renderBottomNav()}
      <View
        style={[
          styles.fabWrap,
          { bottom: Platform.OS === "android" ? 76 : 90 }
        ]}
        pointerEvents="box-none"
      >
        <Pressable
          style={styles.fab}
          onPress={() => setBitrixVoiceOpen(true)}
          accessibilityRole="button"
          accessibilityLabel="Голосовая команда в Bitrix24"
        >
          <Ionicons name="mic" size={26} color="#fff" />
        </Pressable>
      </View>
      {renderSubmitOverlay()}
      {renderBitrixAuthModal()}
      {renderBitrixDealConfirmModal()}
      <BitrixTaskDetailModal
        taskId={selectedTaskId}
        visible={selectedTaskId != null}
        onClose={closeTaskDetail}
        onUpdated={() => void refresh({ pull: true })}
      />
      <Modal
        visible={docDetailJob !== null}
        animationType="slide"
        presentationStyle="pageSheet"
        onRequestClose={() => setDocDetailJob(null)}
      >
        <View style={[styles.voiceRoot, { backgroundColor: "#f8fafc" }]}>
          <View style={[styles.voiceHeader, { paddingTop: topInset + 8 }]}>
            <Pressable onPress={() => setDocDetailJob(null)} hitSlop={12}>
              <Text style={styles.voiceClose}>Закрыть</Text>
            </Pressable>
            <Text style={styles.voiceTitle}>Документ</Text>
            <View style={{ width: 56 }} />
          </View>
          {docDetailLoading ? (
            <ActivityIndicator color={HEADER_BLUE} style={{ marginTop: 24 }} />
          ) : docDetailJob ? (
            <ScrollView contentContainerStyle={styles.voiceScrollContent}>
              <Text style={styles.listTitle}>{docDetailJob.job.sourceName}</Text>
              <Text style={styles.muted}>
                {jobStatusRu(docDetailJob.job.status)} · {formatDate(docDetailJob.job.createdAt)}
              </Text>
              {docDetailJob.job.bitrixDealTitle ? (
                <Text style={styles.muted}>Сделка: {docDetailJob.job.bitrixDealTitle}</Text>
              ) : null}
              {(docDetailJob.estimate.validationWarnings ?? []).length > 0 ? (
                <View style={styles.warnBox}>
                  {(docDetailJob.estimate.validationWarnings ?? []).map((w) => (
                    <Text key={w} style={styles.warnText}>
                      • {w}
                    </Text>
                  ))}
                </View>
              ) : null}
              <Text style={styles.sectionLabel}>Поля сметы</Text>
              <Text style={styles.muted}>Стройка: {docDetailJob.estimate.projectName || "—"}</Text>
              <Text style={styles.muted}>Работы: {docDetailJob.estimate.objectDescription || "—"}</Text>
              <Text style={styles.muted}>Основание: {docDetailJob.estimate.basis || "—"}</Text>
              <Text style={styles.muted}>Стоимость: {docDetailJob.estimate.estimatedCost || "—"}</Text>
              <Text style={styles.muted}>Итого: {docDetailJob.estimate.grandTotal || "—"}</Text>
              {docDetailJob.job.errorMessage ? (
                <View style={[styles.warnBox, { marginTop: 12 }]}>
                  <Text style={styles.warnText}>{docDetailJob.job.errorMessage}</Text>
                </View>
              ) : null}
              {docDetailJob.canRetryBitrixAttach ? (
                <Pressable
                  disabled={docAttachRetrying}
                  style={[styles.button, styles.primaryButton, { marginTop: 12 }]}
                  onPress={async () => {
                    setDocAttachRetrying(true);
                    try {
                      const item = await retryAttachMobileDocumentJob(docDetailJob.job.id);
                      setDocDetailJob(item);
                      if (!item.canRetryBitrixAttach) {
                        Alert.alert("Bitrix24", "Смета прикреплена к сделке");
                        void refresh({ pull: true });
                      }
                    } catch (err) {
                      Alert.alert(
                        "Bitrix24",
                        err instanceof Error ? err.message : "Не удалось прикрепить"
                      );
                    } finally {
                      setDocAttachRetrying(false);
                    }
                  }}
                >
                  <Text style={styles.buttonText}>
                    {docAttachRetrying ? "Прикрепление…" : "Повторить прикрепление к сделке"}
                  </Text>
                </Pressable>
              ) : null}
              {docDetailJob.downloadPath ? (
                <Pressable
                  style={[styles.button, styles.primaryButton, { marginTop: 16 }]}
                  onPress={() => void Linking.openURL(generatedDocumentDownloadUrl(docDetailJob.downloadPath!))}
                >
                  <Text style={styles.buttonText}>Открыть файл</Text>
                </Pressable>
              ) : null}
            </ScrollView>
          ) : null}
        </View>
      </Modal>

      <BitrixDealDetailModal
        dealId={selectedDealId}
        visible={selectedDealId != null}
        onClose={closeDealDetail}
        onUpdated={() => void loadBitrixDeals(bitrixDealsSearch, true)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    backgroundColor: BG,
    flex: 1
  },
  header: {
    alignItems: "center",
    backgroundColor: HEADER_BLUE,
    flexDirection: "row",
    justifyContent: "space-between",
    paddingBottom: 14,
    paddingHorizontal: 18
  },
  headerTitle: {
    color: "#fff",
    flex: 1,
    fontSize: 20,
    fontWeight: "700",
    textAlign: "center"
  },
  headerSideBtn: {
    alignItems: "center",
    justifyContent: "center",
    width: 32
  },
  headerAuthDot: {
    backgroundColor: "#fbbf24",
    borderColor: HEADER_BLUE,
    borderRadius: 5,
    borderWidth: 1.5,
    height: 10,
    position: "absolute",
    right: -2,
    top: -2,
    width: 10
  },
  headerNotifDot: {
    alignItems: "center",
    backgroundColor: "#ef4444",
    borderColor: HEADER_BLUE,
    borderRadius: 9,
    borderWidth: 1.5,
    justifyContent: "center",
    minWidth: 18,
    paddingHorizontal: 4,
    position: "absolute",
    right: -8,
    top: -6
  },
  headerNotifDotText: {
    color: "#fff",
    fontSize: 10,
    fontWeight: "700",
    lineHeight: 14
  },
  notificationUnread: {
    backgroundColor: "#eff6ff",
    borderColor: "#bfdbfe",
    borderWidth: 1
  },
  authOverlay: {
    backgroundColor: "rgba(15, 23, 42, 0.45)",
    flex: 1,
    justifyContent: "center",
    padding: 24
  },
  authSheet: {
    backgroundColor: "#fff",
    borderRadius: 16,
    gap: 12,
    padding: 20
  },
  authSheetTitle: {
    color: "#0f172a",
    fontSize: 18,
    fontWeight: "700"
  },
  authSheetClose: {
    alignSelf: "center",
    marginTop: 4
  },
  mainScroll: {
    flex: 1
  },
  mainScrollContent: {
    gap: 14,
    padding: 16,
    paddingBottom: 110
  },
  card: {
    backgroundColor: "#fff",
    borderRadius: 16,
    gap: 12,
    padding: 16,
    shadowColor: "#0f172a",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2
  },
  cardHeader: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between"
  },
  cardTitle: {
    color: "#0f172a",
    fontSize: 17,
    fontWeight: "700"
  },
  badgeCount: {
    backgroundColor: "#dbeafe",
    borderRadius: 999,
    minWidth: 32,
    paddingHorizontal: 10,
    paddingVertical: 4
  },
  badgeCountText: {
    color: HEADER_BLUE,
    fontSize: 14,
    fontWeight: "700",
    textAlign: "center"
  },
  taskRow: {
    alignItems: "center",
    flexDirection: "row",
    gap: 10,
    paddingVertical: 10
  },
  dot: {
    borderRadius: 6,
    height: 12,
    width: 12
  },
  taskRowLabel: {
    color: "#1e293b",
    flex: 1,
    fontSize: 15
  },
  taskRowNum: {
    color: "#64748b",
    fontSize: 15,
    fontWeight: "600",
    marginRight: 4
  },
  docTileWide: {
    alignItems: "center",
    marginTop: 8
  },
  docIconWrapWide: {
    alignItems: "center",
    backgroundColor: "#eff6ff",
    borderRadius: 16,
    height: 72,
    justifyContent: "center",
    marginBottom: 8,
    width: "100%"
  },
  docTileLabelWide: {
    color: "#334155",
    fontSize: 15,
    fontWeight: "700",
    textAlign: "center"
  },
  notifRow: {
    alignItems: "center",
    flexDirection: "row",
    gap: 8,
    marginTop: 4
  },
  notifText: {
    color: "#334155",
    flex: 1,
    fontSize: 14,
    lineHeight: 20
  },
  muted: {
    color: "#64748b",
    fontSize: 13,
    lineHeight: 18
  },
  errorBanner: {
    backgroundColor: "#fef2f2",
    borderColor: "#fecaca",
    borderRadius: 14,
    borderWidth: 1,
    gap: 10,
    padding: 14
  },
  errorText: {
    color: "#b91c1c",
    fontSize: 14,
    fontWeight: "600"
  },
  errorRetry: {
    alignSelf: "flex-start"
  },
  errorRetryText: {
    color: HEADER_BLUE,
    fontSize: 14,
    fontWeight: "700"
  },
  linkBtn: {
    alignSelf: "flex-start",
    marginTop: 4
  },
  linkBtnText: {
    color: HEADER_BLUE,
    fontSize: 14,
    fontWeight: "700"
  },
  linkInline: {
    color: HEADER_BLUE,
    fontSize: 14,
    fontWeight: "700"
  },
  bitrixStatsRow: {
    flexDirection: "row",
    gap: 10,
    justifyContent: "space-between",
    marginTop: 10
  },
  bitrixStatBox: {
    alignItems: "center",
    backgroundColor: "#f8fafc",
    borderColor: "transparent",
    borderRadius: 12,
    borderWidth: 2,
    flex: 1,
    paddingVertical: 12
  },
  bitrixStatBoxActive: {
    backgroundColor: "#eff6ff",
    borderColor: HEADER_BLUE
  },
  filterBanner: {
    alignItems: "center",
    backgroundColor: "#eff6ff",
    borderRadius: 10,
    flexDirection: "row",
    gap: 10,
    justifyContent: "space-between",
    paddingHorizontal: 12,
    paddingVertical: 10
  },
  filterBannerText: {
    color: "#1e3a8a",
    flex: 1,
    fontSize: 13,
    fontWeight: "600"
  },
  bitrixStatNum: {
    color: "#0f172a",
    fontSize: 22,
    fontWeight: "800"
  },
  bitrixStatLabel: {
    color: "#64748b",
    fontSize: 12,
    fontWeight: "600",
    marginTop: 4
  },
  bottomNav: {
    alignItems: "flex-end",
    backgroundColor: "#fff",
    borderTopColor: "#e2e8f0",
    borderTopWidth: 1,
    bottom: 0,
    flexDirection: "row",
    justifyContent: "space-around",
    left: 0,
    paddingTop: 8,
    position: "absolute",
    right: 0
  },
  navItem: {
    alignItems: "center",
    flex: 1,
    gap: 2,
    minWidth: 0,
    paddingVertical: 4
  },
  navLabel: {
    color: "#64748b",
    fontSize: 9,
    fontWeight: "600",
    textAlign: "center"
  },
  navLabelActive: {
    color: HEADER_BLUE
  },
  fabWrap: {
    pointerEvents: "box-none",
    position: "absolute",
    right: 16,
    zIndex: 30
  },
  fab: {
    alignItems: "center",
    backgroundColor: HEADER_BLUE,
    borderRadius: 28,
    elevation: 8,
    height: 56,
    justifyContent: "center",
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.25,
    shadowRadius: 6,
    width: 56
  },
  listItem: {
    backgroundColor: "#f8fafc",
    borderRadius: 12,
    gap: 4,
    padding: 12
  },
  listHeader: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between"
  },
  listTitle: {
    color: "#0f172a",
    flex: 1,
    fontSize: 15,
    fontWeight: "600"
  },
  badge: {
    fontSize: 11,
    fontWeight: "700",
    textTransform: "uppercase"
  },
  voiceRoot: {
    backgroundColor: "#fff",
    flex: 1
  },
  voiceHeader: {
    alignItems: "center",
    borderBottomColor: "#e2e8f0",
    borderBottomWidth: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    paddingBottom: 12,
    paddingHorizontal: 16
  },
  voiceClose: {
    color: HEADER_BLUE,
    fontSize: 16,
    fontWeight: "600",
    width: 72
  },
  voiceTitle: {
    color: "#0f172a",
    fontSize: 17,
    fontWeight: "700"
  },
  voiceScroll: {
    flex: 1
  },
  voiceScrollContent: {
    gap: 12,
    padding: 16,
    paddingBottom: 40
  },
  sectionLabel: {
    color: "#0f172a",
    fontSize: 14,
    fontWeight: "700",
    marginTop: 8
  },
  templateCard: {
    backgroundColor: "#f1f5f9",
    borderRadius: 12,
    borderWidth: 2,
    borderColor: "transparent",
    padding: 12
  },
  templateCardSelected: {
    borderColor: HEADER_BLUE,
    backgroundColor: "#eff6ff"
  },
  templateName: {
    color: "#0f172a",
    fontSize: 15,
    fontWeight: "600"
  },
  templateMeta: {
    color: "#64748b",
    fontSize: 12
  },
  row: {
    flexDirection: "row",
    gap: 10
  },
  button: {
    alignItems: "center",
    borderRadius: 12,
    flex: 1,
    justifyContent: "center",
    minHeight: 44,
    paddingHorizontal: 12
  },
  primaryButton: {
    backgroundColor: HEADER_BLUE
  },
  secondaryButton: {
    backgroundColor: "#e2e8f0"
  },
  buttonDisabled: {
    opacity: 0.45
  },
  buttonText: {
    color: "#fff",
    fontSize: 14,
    fontWeight: "600"
  },
  buttonTextDark: {
    color: "#0f172a",
    fontSize: 14,
    fontWeight: "600"
  },
  submitWide: {
    flex: 0,
    marginTop: 8,
    width: "100%"
  },
  input: {
    borderColor: "#cbd5e1",
    borderRadius: 12,
    borderWidth: 1,
    color: "#0f172a",
    fontSize: 15,
    paddingHorizontal: 12,
    paddingVertical: 10
  },
  textArea: {
    minHeight: 80,
    textAlignVertical: "top"
  },
  segmentRow: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8
  },
  segment: {
    borderColor: "#cbd5e1",
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 6
  },
  segmentSelected: {
    backgroundColor: HEADER_BLUE,
    borderColor: HEADER_BLUE
  },
  segmentText: {
    color: "#334155",
    fontSize: 12,
    fontWeight: "600"
  },
  segmentTextSelected: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "600"
  },
  warnBox: {
    backgroundColor: "#fffbeb",
    borderRadius: 10,
    padding: 10
  },
  warnText: {
    color: "#92400e",
    fontSize: 13
  },
  cardMuted: {
    backgroundColor: "#f1f5f9"
  },
  collapseHeader: {
    alignItems: "center",
    flexDirection: "row",
    gap: 10
  },
  logFilterRow: {
    marginBottom: 8,
    marginTop: 4,
    maxHeight: 44
  },
  filterChip: {
    borderColor: "#cbd5e1",
    borderRadius: 999,
    borderWidth: 1,
    marginRight: 8,
    paddingHorizontal: 14,
    paddingVertical: 8
  },
  filterChipOn: {
    backgroundColor: HEADER_BLUE,
    borderColor: HEADER_BLUE
  },
  filterChipText: {
    color: "#475569",
    fontSize: 13,
    fontWeight: "600"
  },
  filterChipTextOn: {
    color: "#fff",
    fontSize: 13,
    fontWeight: "600"
  },
  logItemHeader: {
    alignItems: "center",
    flexDirection: "row",
    gap: 8,
    justifyContent: "space-between"
  },
  logCardTitle: {
    color: "#0f172a",
    flex: 1,
    fontSize: 15,
    fontWeight: "700"
  },
  logBadge: {
    borderRadius: 8,
    borderWidth: 1,
    fontSize: 11,
    fontWeight: "700",
    overflow: "hidden",
    paddingHorizontal: 8,
    paddingVertical: 3
  },
  logItem: {
    backgroundColor: "#f8fafc",
    borderRadius: 10,
    gap: 4,
    marginBottom: 8,
    padding: 10
  },
  logLine: {
    color: "#0f172a",
    fontSize: 12,
    fontWeight: "600"
  },
  logLineMuted: {
    color: "#64748b",
    fontSize: 11
  },
  logSummary: {
    color: "#1e293b",
    fontSize: 13,
    marginTop: 2
  },
  logDetail: {
    color: "#475569",
    fontSize: 12
  },
  logError: {
    color: "#b91c1c",
    fontSize: 12,
    fontWeight: "600"
  },
  progressOverlay: {
    alignItems: "center",
    backgroundColor: "rgba(15, 23, 42, 0.55)",
    flex: 1,
    justifyContent: "center",
    padding: 24
  },
  progressCard: {
    alignItems: "center",
    backgroundColor: "#fff",
    borderRadius: 16,
    gap: 10,
    maxWidth: 340,
    padding: 24,
    width: "100%"
  },
  progressTitle: {
    color: "#0f172a",
    fontSize: 18,
    fontWeight: "700",
    textAlign: "center"
  },
  progressStep: {
    color: HEADER_BLUE,
    fontSize: 13,
    fontWeight: "700"
  },
  progressMessage: {
    color: "#334155",
    fontSize: 15,
    lineHeight: 22,
    textAlign: "center"
  },
  progressElapsed: {
    color: "#64748b",
    fontSize: 13
  },
  progressHint: {
    color: "#64748b",
    fontSize: 12,
    lineHeight: 18,
    textAlign: "center"
  },
  inlineProgress: {
    alignItems: "center",
    backgroundColor: "#eff6ff",
    borderRadius: 12,
    flexDirection: "row",
    gap: 10,
    marginTop: 4,
    padding: 12
  },
  inlineProgressText: {
    color: "#1e40af",
    flex: 1,
    fontSize: 14,
    fontWeight: "600"
  }
});
