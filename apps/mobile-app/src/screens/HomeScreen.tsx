import { Ionicons } from "@expo/vector-icons";
import { Audio } from "expo-av";
import { StatusBar as ExpoStatusBar } from "expo-status-bar";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StatusBar as RNStatusBar,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

import type {
  BitrixTaskStats,
  BitrixTaskSummary,
  DocumentJobSummary,
  DocumentTemplateSummary,
  HealthResponse,
  SourceDocumentSummary
} from "../entities/document-template/types";
import {
  createMobileBitrixIntentMultipart,
  createMobileBitrixIntentText,
  createMobileVoiceRequest,
  getHealth,
  getMobileApiBaseUrl,
  listBitrixTasks,
  listJobs,
  listSourceDocuments,
  listTemplates
} from "../shared/api/client";
import {
  getRequestLog,
  REQUEST_LOG_COLLAPSED_COUNT,
  subscribeRequestLog,
  type RequestLogEntry,
  type RequestLogKind
} from "../shared/api/requestLog";

const HEADER_BLUE = "#2563eb";
const BG = "#e8eef5";

type MainTab = "home" | "tasks" | "docs" | "more";

type LogFilterTab = "all" | RequestLogKind | "errors";

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

function bitrixTaskStatusRu(status: string) {
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

function formatDate(value: string | null) {
  if (!value) {
    return "—";
  }
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "short",
    timeStyle: "short"
  }).format(new Date(value));
}

function statusColor(status: string) {
  switch (status) {
    case "completed":
    case "sent":
      return "#15803d";
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
  const [sourceDocuments, setSourceDocuments] = useState<SourceDocumentSummary[]>([]);
  const [bitrixTasks, setBitrixTasks] = useState<BitrixTaskSummary[]>([]);
  const [bitrixTaskStats, setBitrixTaskStats] = useState<BitrixTaskStats | null>(null);
  const [bitrixResponsibleId, setBitrixResponsibleId] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [submittingRequest, setSubmittingRequest] = useState(false);
  const [submittingBitrix, setSubmittingBitrix] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [docRecording, setDocRecording] = useState<Audio.Recording | null>(null);
  const [docAudioUri, setDocAudioUri] = useState<string | null>(null);
  const [bitrixRecording, setBitrixRecording] = useState<Audio.Recording | null>(null);
  const [bitrixAudioUri, setBitrixAudioUri] = useState<string | null>(null);
  const previewSoundRef = useRef<Audio.Sound | null>(null);
  const [previewPlayingUri, setPreviewPlayingUri] = useState<string | null>(null);
  const [requestLogEntries, setRequestLogEntries] = useState<RequestLogEntry[]>([]);
  const [logFilter, setLogFilter] = useState<LogFilterTab>("all");
  const [logExpanded, setLogExpanded] = useState(false);
  const [docVoiceSectionOpen, setDocVoiceSectionOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<MainTab>("home");
  const [docVoiceOpen, setDocVoiceOpen] = useState(false);
  const [bitrixVoiceOpen, setBitrixVoiceOpen] = useState(false);
  const [bitrixIntentText, setBitrixIntentText] = useState("");
  const [bitrixHints, setBitrixHints] = useState({
    dealId: "",
    dealTitle: "",
    dealHint: "",
    stageHint: ""
  });

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
  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [nextHealth, nextTemplates, nextJobs, nextSourceDocuments, bitrixBundle] =
        await Promise.all([
          getHealth(),
          listTemplates(),
          listJobs(),
          listSourceDocuments(),
          listBitrixTasks(80).catch(() => null)
        ]);
      setHealth(nextHealth);
      setTemplates(nextTemplates);
      const mobileJobs = nextJobs.filter((job) => job.requestedBy === "mobile-app");
      setJobs(mobileJobs);
      setSourceDocuments(
        nextSourceDocuments.filter((document) => document.origin === "mobile-app")
      );
      if (bitrixBundle && Array.isArray(bitrixBundle.items)) {
        setBitrixTasks(bitrixBundle.items);
        setBitrixTaskStats(bitrixBundle.stats ?? null);
        setBitrixResponsibleId(
          typeof bitrixBundle.responsibleUserId === "number" ? bitrixBundle.responsibleUserId : null
        );
      } else {
        setBitrixTasks([]);
        setBitrixTaskStats(null);
        setBitrixResponsibleId(null);
      }
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
    }
  }, []);

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

  useEffect(() => {
    return () => {
      const sound = previewSoundRef.current;
      previewSoundRef.current = null;
      void sound?.unloadAsync();
    };
  }, []);

  const selectedTemplate = useMemo(
    () => templates.find((template) => template.id === requestForm.templateId) ?? null,
    [requestForm.templateId, templates]
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

  const notificationPreview = useMemo(() => {
    const t = bitrixTasks[0];
    if (t?.title) {
      return t.title.length > 72 ? `${t.title.slice(0, 72)}…` : t.title;
    }
    if (jobs[0]) {
      return `${jobs[0].templateName}: ${jobs[0].status}`;
    }
    return "Нет свежих задач из Bitrix";
  }, [bitrixTasks, jobs]);

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
    if (docRecording) {
      try {
        await docRecording.stopAndUnloadAsync();
      } catch {
        // ignore
      }
      setDocRecording(null);
    }
    setDocAudioUri(null);
  }, [docRecording, stopPreviewPlayback]);

  const resetBitrixRecordingDraft = useCallback(async () => {
    await stopPreviewPlayback();
    if (bitrixRecording) {
      try {
        await bitrixRecording.stopAndUnloadAsync();
      } catch {
        // ignore
      }
      setBitrixRecording(null);
    }
    setBitrixAudioUri(null);
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
      setDocRecording(created.recording);
    } catch (nextError) {
      const message =
        nextError instanceof Error ? nextError.message : "Не удалось начать запись";
      setError(message);
      Alert.alert("Запись", message);
    }
  };

  const stopDocRecording = async () => {
    if (!docRecording) {
      return;
    }
    try {
      await docRecording.stopAndUnloadAsync();
      const uri = docRecording.getURI();
      setDocAudioUri(uri ?? null);
      setDocRecording(null);
    } catch (nextError) {
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
      setBitrixRecording(created.recording);
    } catch (nextError) {
      const message =
        nextError instanceof Error ? nextError.message : "Не удалось начать запись";
      setError(message);
      Alert.alert("Запись", message);
    }
  };

  const stopBitrixRecording = async () => {
    if (!bitrixRecording) {
      return;
    }
    try {
      await bitrixRecording.stopAndUnloadAsync();
      const uri = bitrixRecording.getURI();
      setBitrixAudioUri(uri ?? null);
      setBitrixRecording(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Не удалось остановить запись");
    }
  };

  const submitVoiceRequest = async () => {
    if (!docAudioUri || !requestForm.templateId || !requestForm.sourceName.trim()) {
      return;
    }
    setSubmittingRequest(true);
    try {
      await createMobileVoiceRequest({
        ...requestForm,
        audioUri: docAudioUri,
        audioFileName: `voice-request-${Date.now()}.m4a`,
        audioMimeType: "audio/mp4"
      });
      await stopPreviewPlayback();
      setDocAudioUri(null);
      setRequestForm((current) => ({
        ...current,
        sourceName: "",
        payload: "",
        taskCommandText: ""
      }));
      await refresh();
      setDocVoiceOpen(false);
      Alert.alert("Готово", "Голосовая заявка отправлена на сервер.");
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Ошибка отправки");
    } finally {
      setSubmittingRequest(false);
    }
  };

  const parseDealId = () => {
    const n = parseInt(bitrixHints.dealId.trim(), 10);
    return Number.isFinite(n) && n > 0 ? n : undefined;
  };

  const submitBitrixText = async () => {
    if (!bitrixIntentText.trim()) {
      return;
    }
    setSubmittingBitrix(true);
    try {
      const item = await createMobileBitrixIntentText({
        text: bitrixIntentText.trim(),
        dealId: parseDealId(),
        dealTitle: bitrixHints.dealTitle.trim() || undefined,
        dealHint: bitrixHints.dealHint.trim() || undefined,
        stageHint: bitrixHints.stageHint.trim() || undefined
      });
      setBitrixIntentText("");
      await refresh();
      Alert.alert("Bitrix24", (item?.bitrixSteps ?? []).join("\n"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Ошибка Bitrix");
    } finally {
      setSubmittingBitrix(false);
    }
  };

  const submitBitrixVoice = async () => {
    if (!bitrixAudioUri) {
      Alert.alert("Запись", "Сначала запишите голос или введите текст на главной.");
      return;
    }
    setSubmittingBitrix(true);
    try {
      const item = await createMobileBitrixIntentMultipart({
        audioUri: bitrixAudioUri,
        audioFileName: `bitrix-intent-${Date.now()}.m4a`,
        audioMimeType: "audio/mp4",
        dealId: parseDealId(),
        dealTitle: bitrixHints.dealTitle.trim() || undefined,
        dealHint: bitrixHints.dealHint.trim() || undefined,
        stageHint: bitrixHints.stageHint.trim() || undefined
      });
      await stopPreviewPlayback();
      setBitrixAudioUri(null);
      setBitrixVoiceOpen(false);
      await refresh();
      Alert.alert("Bitrix24", (item?.bitrixSteps ?? []).join("\n"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Ошибка Bitrix");
    } finally {
      setSubmittingBitrix(false);
    }
  };

  const openDocTile = (title: string) => {
    Alert.alert(title, "Раздел в разработке. Пока используй список файлов во вкладке «Документы».");
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
        <View style={styles.navSpacer} />
        <Pressable style={styles.navItem} onPress={() => setActiveTab("docs")}>
          <Ionicons
            name="document-text-outline"
            size={22}
            color={activeTab === "docs" ? HEADER_BLUE : "#64748b"}
          />
          <Text style={[styles.navLabel, activeTab === "docs" && styles.navLabelActive]}>Документы</Text>
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
      <View style={styles.fabWrap} pointerEvents="box-none">
        <Pressable style={styles.fab} onPress={() => setBitrixVoiceOpen(true)} accessibilityRole="button">
          <Ionicons name="mic" size={28} color="#fff" />
        </Pressable>
      </View>
    </>
  );

  const renderHomeDashboard = () => (
    <>
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
          <Text style={styles.buttonText}>{submittingBitrix ? "Отправка…" : "Отправить в Bitrix"}</Text>
        </Pressable>
      </View>

      <View style={styles.card}>
        <View style={styles.cardHeader}>
          <Text style={styles.cardTitle}>Мои задачи Bitrix24</Text>
          <Pressable onPress={() => setActiveTab("tasks")}>
            <Text style={styles.linkInline}>Все задачи</Text>
          </Pressable>
        </View>
        <Text style={styles.muted}>
          Счётчики по задачам, где вы — ответственный (пользователь из URL вебхука на сервере). До{" "}
          {80} шт. в выборке; точные итоги по порталу — после OAuth под вашим аккаунтом.
        </Text>
        {bitrixTaskStats ? (
          <View style={styles.bitrixStatsRow}>
            <Pressable style={styles.bitrixStatBox} onPress={() => setActiveTab("tasks")}>
              <Text style={styles.bitrixStatNum}>{bitrixTaskStats.totalOpen}</Text>
              <Text style={styles.bitrixStatLabel}>Открыто</Text>
            </Pressable>
            <Pressable style={styles.bitrixStatBox} onPress={() => setActiveTab("tasks")}>
              <Text style={[styles.bitrixStatNum, { color: "#0369a1" }]}>
                {bitrixTaskStats.inProgress}
              </Text>
              <Text style={styles.bitrixStatLabel}>В работе</Text>
            </Pressable>
            <Pressable style={styles.bitrixStatBox} onPress={() => setActiveTab("tasks")}>
              <Text style={[styles.bitrixStatNum, { color: "#b91c1c" }]}>
                {bitrixTaskStats.overdue}
              </Text>
              <Text style={styles.bitrixStatLabel}>Просрочено</Text>
            </Pressable>
          </View>
        ) : (
          <Text style={styles.muted}>
            Не удалось загрузить (проверьте BITRIX_WEBHOOK_URL: в пути должно быть …/rest/ID/токен/).
          </Text>
        )}
        {bitrixResponsibleId != null ? (
          <Text style={styles.muted}>Ответственный в Bitrix, id: {bitrixResponsibleId}</Text>
        ) : null}
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Документы</Text>
        <View style={styles.docGrid}>
          <Pressable style={styles.docTile} onPress={() => openDocTile("Договоры")}>
            <View style={styles.docIconWrap}>
              <Ionicons name="document-text-outline" size={26} color={HEADER_BLUE} />
            </View>
            <Text style={styles.docTileLabel}>Договоры</Text>
          </Pressable>
          <Pressable style={styles.docTile} onPress={() => openDocTile("Сметы")}>
            <View style={styles.docIconWrap}>
              <Ionicons name="list-outline" size={26} color={HEADER_BLUE} />
            </View>
            <Text style={styles.docTileLabel}>Сметы</Text>
          </Pressable>
          <Pressable style={styles.docTile} onPress={() => openDocTile("Акты")}>
            <View style={styles.docIconWrap}>
              <Ionicons name="reader-outline" size={26} color={HEADER_BLUE} />
            </View>
            <Text style={styles.docTileLabel}>Акты</Text>
          </Pressable>
          <Pressable style={styles.docTile} onPress={() => openDocTile("Чертежи")}>
            <View style={styles.docIconWrap}>
              <Ionicons name="analytics-outline" size={26} color={HEADER_BLUE} />
            </View>
            <Text style={styles.docTileLabel}>Чертежи</Text>
          </Pressable>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Свежая задача из Bitrix</Text>
        <Pressable style={styles.notifRow} onPress={() => setActiveTab("tasks")}>
          <Text style={styles.notifText}>{notificationPreview}</Text>
          <Ionicons name="chevron-forward" size={18} color="#94a3b8" />
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

      <View style={[styles.card, styles.cardMuted]}>
        <Pressable
          onPress={() => setDocVoiceSectionOpen((v) => !v)}
          style={styles.collapseHeader}
        >
          <View style={{ flex: 1 }}>
            <Text style={styles.cardTitle}>Голос → документ по шаблону</Text>
            <Text style={styles.muted}>
              Редкий сценарий: заявка на генерацию документа, не путать с запросом в Bitrix выше.
            </Text>
          </View>
          <Ionicons
            name={docVoiceSectionOpen ? "chevron-up" : "chevron-down"}
            size={22}
            color="#64748b"
          />
        </Pressable>
        {docVoiceSectionOpen ? (
          <Pressable
            onPress={() => setDocVoiceOpen(true)}
            style={[styles.button, styles.secondaryButton, { marginTop: 12 }]}
          >
            <Text style={styles.buttonTextDark}>Открыть форму записи</Text>
          </Pressable>
        ) : null}
      </View>

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
    </>
  );

  const renderTasksTab = () => (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Задачи Bitrix24</Text>
        <Text style={styles.muted}>
          Только задачи, где вы указаны ответственным (по пользователю из входящего вебхука). Счётчики
          на главной считаются по этому же списку (ограничение выборки — см. лимит в API).
        </Text>
        {bitrixTaskStats ? (
          <View style={styles.bitrixStatsRow}>
            <View style={styles.bitrixStatBox}>
              <Text style={styles.bitrixStatNum}>{bitrixTaskStats.totalOpen}</Text>
              <Text style={styles.bitrixStatLabel}>Открыто</Text>
            </View>
            <View style={styles.bitrixStatBox}>
              <Text style={[styles.bitrixStatNum, { color: "#0369a1" }]}>
                {bitrixTaskStats.inProgress}
              </Text>
              <Text style={styles.bitrixStatLabel}>В работе</Text>
            </View>
            <View style={styles.bitrixStatBox}>
              <Text style={[styles.bitrixStatNum, { color: "#b91c1c" }]}>
                {bitrixTaskStats.overdue}
              </Text>
              <Text style={styles.bitrixStatLabel}>Просрочено</Text>
            </View>
          </View>
        ) : null}
        {loading ? <ActivityIndicator color={HEADER_BLUE} /> : null}
        {!loading && bitrixTasks.length === 0 ? (
          <Text style={styles.muted}>Нет задач или не настроен / не разобран вебхук Bitrix.</Text>
        ) : null}
        {bitrixTasks.map((task) => (
          <View key={task.id} style={styles.listItem}>
            <View style={styles.listHeader}>
              <Text style={styles.listTitle}>{task.title || "Без названия"}</Text>
              <Text style={styles.badge}>{bitrixTaskStatusRu(task.status)}</Text>
            </View>
            {task.deadline ? (
              <Text style={styles.muted}>Срок: {task.deadline}</Text>
            ) : (
              <Text style={styles.muted}>Срок не указан</Text>
            )}
            <Text style={styles.muted}>id: {task.id}</Text>
          </View>
        ))}
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Заявки на документы (TSK)</Text>
        {jobs.length === 0 ? (
          <Text style={styles.muted}>Пока нет заявок с этого телефона.</Text>
        ) : null}
        {jobs.map((job) => (
          <View key={job.id} style={styles.listItem}>
            <View style={styles.listHeader}>
              <Text style={styles.listTitle}>{job.sourceName}</Text>
              <Text style={[styles.badge, { color: statusColor(job.status) }]}>{job.status}</Text>
            </View>
            <Text style={styles.muted}>
              {job.templateName} · {job.dispatchStatus}
            </Text>
            <Text style={styles.muted}>{formatDate(job.createdAt)}</Text>
          </View>
        ))}
      </View>
    </>
  );

  const renderDocsTab = () => (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>Файлы</Text>
      {sourceDocuments.length === 0 ? (
        <Text style={styles.muted}>Нет загруженных голосовых файлов.</Text>
      ) : null}
      {sourceDocuments.map((document) => (
        <View key={document.id} style={styles.listItem}>
          <Text style={styles.listTitle}>{document.fileName}</Text>
          <Text style={styles.muted}>
            {document.kind} · {Math.round(document.sizeBytes / 1024)} KB
          </Text>
          <Text style={styles.muted}>{formatDate(document.createdAt)}</Text>
        </View>
      ))}
    </View>
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
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Ещё</Text>
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
          Запишите фразу (например «переведи сделку … на следующий этап»). Нужен WHISPER_BASE_URL на
          сервере.
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
        <View style={styles.row}>
          {bitrixRecording ? (
            <Pressable
              onPress={() => void stopBitrixRecording()}
              style={[styles.button, styles.secondaryButton, styles.submitWide]}
            >
              <Text style={styles.buttonTextDark}>Стоп</Text>
            </Pressable>
          ) : !bitrixAudioUri ? (
            <Pressable
              onPress={() => void startBitrixRecording()}
              style={[styles.button, styles.primaryButton, styles.submitWide]}
            >
              <Text style={styles.buttonText}>Начать</Text>
            </Pressable>
          ) : (
            <>
              <Pressable
                onPress={() => void togglePreviewPlayback(bitrixAudioUri)}
                style={[styles.button, styles.secondaryButton]}
              >
                <Text style={styles.buttonTextDark}>
                  {previewPlayingUri === bitrixAudioUri ? "Стоп" : "Прослушать"}
                </Text>
              </Pressable>
              <Pressable
                onPress={() => void resetBitrixRecordingDraft()}
                style={[styles.button, styles.secondaryButton]}
              >
                <Text style={styles.buttonTextDark}>Перезаписать</Text>
              </Pressable>
            </>
          )}
        </View>
        <Text style={styles.muted}>
          {bitrixRecording
            ? "Идёт запись…"
            : bitrixAudioUri
              ? "Аудио готово."
              : "Запись не сделана."}
        </Text>
        <Pressable
          onPress={() => void submitBitrixVoice()}
          disabled={submittingBitrix || !bitrixAudioUri}
          style={[
            styles.button,
            styles.primaryButton,
            styles.submitWide,
            (submittingBitrix || !bitrixAudioUri) && styles.buttonDisabled
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
    <View style={styles.voiceRoot}>
      <View style={[styles.voiceHeader, { paddingTop: topInset + 8 }]}>
        <Pressable
          onPress={() => {
            void stopPreviewPlayback();
            setDocVoiceOpen(false);
          }}
          hitSlop={12}
        >
          <Text style={styles.voiceClose}>Закрыть</Text>
        </Pressable>
        <Text style={styles.voiceTitle}>Голосовая заявка</Text>
        <View style={{ width: 56 }} />
      </View>
      <ScrollView
        style={styles.voiceScroll}
        contentContainerStyle={styles.voiceScrollContent}
        keyboardShouldPersistTaps="handled"
        nestedScrollEnabled
      >
        <Text style={styles.muted}>
          Выбери шаблон, запиши голос, заполни название и при необходимости текст — отправка на
          backend.
        </Text>

        <Text style={styles.sectionLabel}>Шаблон</Text>
        {loading ? <ActivityIndicator color={HEADER_BLUE} /> : null}
        {!loading && templates.length === 0 ? (
          <View style={styles.warnBox}>
            <Text style={styles.warnText}>
              Нет шаблонов. Загрузите шаблон в админке (:5173), затем «Обновить» на главной.
            </Text>
          </View>
        ) : null}
        {templates.map((template) => {
          const selected = template.id === requestForm.templateId;
          return (
            <Pressable
              key={template.id}
              onPress={() =>
                setRequestForm((current) => ({ ...current, templateId: template.id }))
              }
              style={[styles.templateCard, selected && styles.templateCardSelected]}
            >
              <Text style={styles.templateName}>{template.name}</Text>
              <Text style={styles.templateMeta}>
                {template.category} · {template.version}
              </Text>
            </Pressable>
          );
        })}

        <Text style={styles.sectionLabel}>Запись</Text>
        <View style={styles.row}>
          {docRecording ? (
            <Pressable
              onPress={() => void stopDocRecording()}
              style={[styles.button, styles.secondaryButton, styles.submitWide]}
            >
              <Text style={styles.buttonTextDark}>Стоп</Text>
            </Pressable>
          ) : !docAudioUri ? (
            <Pressable
              onPress={() => void startDocRecording()}
              style={[styles.button, styles.primaryButton, styles.submitWide]}
            >
              <Text style={styles.buttonText}>Начать</Text>
            </Pressable>
          ) : (
            <>
              <Pressable
                onPress={() => void togglePreviewPlayback(docAudioUri)}
                style={[styles.button, styles.secondaryButton]}
              >
                <Text style={styles.buttonTextDark}>
                  {previewPlayingUri === docAudioUri ? "Стоп" : "Прослушать"}
                </Text>
              </Pressable>
              <Pressable
                onPress={() => void resetDocRecordingDraft()}
                style={[styles.button, styles.secondaryButton]}
              >
                <Text style={styles.buttonTextDark}>Перезаписать</Text>
              </Pressable>
            </>
          )}
        </View>
        <Text style={styles.muted}>
          {docRecording
            ? "Идёт запись…"
            : docAudioUri
              ? "Аудио готово."
              : "Запись не сделана."}
        </Text>

        <Text style={styles.sectionLabel}>Данные заявки</Text>
        <TextInput
          placeholder="Название / источник *"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={requestForm.sourceName}
          onChangeText={(value) =>
            setRequestForm((current) => ({ ...current, sourceName: value }))
          }
        />
        <TextInput
          placeholder="Заметки (текст вручную)"
          placeholderTextColor="#94a3b8"
          style={[styles.input, styles.textArea]}
          multiline
          value={requestForm.payload}
          onChangeText={(value) =>
            setRequestForm((current) => ({ ...current, payload: value }))
          }
        />
        <Text style={styles.sectionLabel}>Канал доставки</Text>
        <View style={styles.segmentRow}>
          {(["internal", "email", "bitrix"] as const).map((option) => (
            <Pressable
              key={option}
              onPress={() =>
                setRequestForm((current) => ({ ...current, deliveryChannel: option }))
              }
              style={[
                styles.segment,
                requestForm.deliveryChannel === option && styles.segmentSelected
              ]}
            >
              <Text
                style={
                  requestForm.deliveryChannel === option
                    ? styles.segmentTextSelected
                    : styles.segmentText
                }
              >
                {option}
              </Text>
            </Pressable>
          ))}
        </View>
        <TextInput
          placeholder="Адрес доставки"
          placeholderTextColor="#94a3b8"
          style={styles.input}
          value={requestForm.deliveryAddress}
          onChangeText={(value) =>
            setRequestForm((current) => ({ ...current, deliveryAddress: value }))
          }
        />
        <TextInput
          placeholder="Команда задачи (необязательно)"
          placeholderTextColor="#94a3b8"
          style={[styles.input, styles.textArea]}
          multiline
          value={requestForm.taskCommandText}
          onChangeText={(value) =>
            setRequestForm((current) => ({ ...current, taskCommandText: value }))
          }
        />
        <Text style={styles.sectionLabel}>Цель команды</Text>
        <View style={styles.segmentRow}>
          {(["bitrix24", "email_approval"] as const).map((option) => (
            <Pressable
              key={option}
              onPress={() =>
                setRequestForm((current) => ({ ...current, taskTarget: option }))
              }
              style={[
                styles.segment,
                requestForm.taskTarget === option && styles.segmentSelected
              ]}
            >
              <Text
                style={
                  requestForm.taskTarget === option
                    ? styles.segmentTextSelected
                    : styles.segmentText
                }
              >
                {option}
              </Text>
            </Pressable>
          ))}
        </View>

        <Pressable
          onPress={() => void submitVoiceRequest()}
          disabled={
            submittingRequest ||
            !docAudioUri ||
            !selectedTemplate ||
            requestForm.sourceName.trim() === ""
          }
          style={[
            styles.button,
            styles.primaryButton,
            styles.submitWide,
            (submittingRequest || !docAudioUri || !selectedTemplate) && styles.buttonDisabled
          ]}
        >
          <Text style={styles.buttonText}>
            {submittingRequest ? "Отправка…" : "Отправить заявку"}
          </Text>
        </Pressable>
      </ScrollView>
    </View>
  );

  if (bitrixVoiceOpen) {
    return (
      <View style={styles.root}>
        <ExpoStatusBar style="dark" />
        {renderBitrixVoiceFullScreen()}
      </View>
    );
  }

  if (docVoiceOpen) {
    return (
      <View style={styles.root}>
        <ExpoStatusBar style="dark" />
        {renderVoiceFullScreen()}
      </View>
    );
  }

  return (
    <View style={styles.root}>
      <ExpoStatusBar style="light" />
      <View style={[styles.header, { paddingTop: topInset + 10 }]}>
        <Text style={styles.headerTitle}>Главная</Text>
        <Pressable onPress={() => void refresh()} hitSlop={12}>
          <Ionicons name="notifications-outline" size={24} color="#fff" />
        </Pressable>
      </View>

      <ScrollView
        style={styles.mainScroll}
        contentContainerStyle={styles.mainScrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator
      >
        {activeTab === "home" ? renderHomeDashboard() : null}
        {activeTab === "tasks" ? renderTasksTab() : null}
        {activeTab === "docs" ? renderDocsTab() : null}
        {activeTab === "more" ? renderMoreTab() : null}
      </ScrollView>

      {renderBottomNav()}
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
  mainScroll: {
    flex: 1
  },
  mainScrollContent: {
    gap: 14,
    padding: 16,
    paddingBottom: 100
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
  docGrid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10,
    justifyContent: "space-between",
    marginTop: 4
  },
  docTile: {
    alignItems: "center",
    flexBasis: "23%",
    maxWidth: "24%"
  },
  docIconWrap: {
    alignItems: "center",
    backgroundColor: "#eff6ff",
    borderRadius: 14,
    height: 56,
    justifyContent: "center",
    marginBottom: 6,
    width: "100%"
  },
  docTileLabel: {
    color: "#334155",
    fontSize: 11,
    fontWeight: "600",
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
    borderRadius: 12,
    flex: 1,
    paddingVertical: 12
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
    paddingVertical: 4
  },
  navSpacer: {
    width: 56
  },
  navLabel: {
    color: "#64748b",
    fontSize: 10,
    fontWeight: "600"
  },
  navLabelActive: {
    color: HEADER_BLUE
  },
  fabWrap: {
    alignItems: "center",
    bottom: 28,
    left: 0,
    pointerEvents: "box-none",
    position: "absolute",
    right: 0,
    zIndex: 20
  },
  fab: {
    alignItems: "center",
    backgroundColor: HEADER_BLUE,
    borderRadius: 32,
    elevation: 6,
    height: 56,
    justifyContent: "center",
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
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
  }
});
