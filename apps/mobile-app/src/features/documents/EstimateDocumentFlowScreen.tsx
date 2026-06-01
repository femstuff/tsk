import { Ionicons } from "@expo/vector-icons";
import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Linking,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View
} from "react-native";

import type {
  BitrixDealSummary,
  DocumentTemplateSummary,
  EstimatePreview,
  MobileDocumentJobView,
  MobileVoiceRequestResult
} from "../../entities/document-template/types";
import {
  confirmMobileDocumentJob,
  retryAttachMobileDocumentJob,
  createMobileVoiceRequest,
  generatedDocumentDownloadUrl
} from "../../shared/api/client";

const HEADER_BLUE = "#2563eb";

type RecordingPhase = "idle" | "recording" | "stopping" | "ready";

type Step = "deal" | "record" | "processing" | "review" | "done";

type Props = {
  topInset: number;
  template: DocumentTemplateSummary | null;
  deals: BitrixDealSummary[];
  dealsLoading: boolean;
  bitrixConnected: boolean;
  recordingPhase: RecordingPhase;
  audioUri: string | null;
  sourceName: string;
  payloadExtra: string;
  onClose: () => void;
  onSuccess: () => void;
  onSourceNameChange: (value: string) => void;
  onPayloadExtraChange: (value: string) => void;
  onStartRecording: () => void;
  onStopRecording: () => void;
  onResetRecording: () => void;
  onPreview: () => void;
  previewPlayingUri: string | null;
  renderRecordingControls: (
    phase: RecordingPhase,
    uri: string | null,
    onStart: () => void,
    onStop: () => void,
    onReset: () => void,
    onPreview: () => void
  ) => React.ReactNode;
  recordingStatusText: (phase: RecordingPhase) => string;
};

function fieldRow(label: string, value: string) {
  const empty = !value?.trim();
  return (
    <View style={{ marginBottom: 10 }}>
      <Text style={{ fontSize: 12, color: "#64748b", marginBottom: 2 }}>{label}</Text>
      <Text style={{ fontSize: 15, color: empty ? "#94a3b8" : "#0f172a" }}>
        {empty ? "— не распознано —" : value}
      </Text>
    </View>
  );
}

function EstimateReviewBlock({ estimate }: { estimate: EstimatePreview | null | undefined }) {
  if (!estimate) {
    return (
      <Text style={{ color: "#64748b", marginTop: 8 }}>
        Нет данных для проверки. Повторите распознавание.
      </Text>
    );
  }
  const validationWarnings = estimate.validationWarnings ?? [];
  const lineItems = estimate.lineItems ?? [];

  return (
    <View style={{ marginTop: 8 }}>
      {validationWarnings.length > 0 ? (
        <View
          style={{
            backgroundColor: "#fff7ed",
            borderColor: "#fdba74",
            borderWidth: 1,
            borderRadius: 12,
            padding: 12,
            marginBottom: 12
          }}
        >
          <Text style={{ fontWeight: "700", color: "#9a3412", marginBottom: 6 }}>
            Проверьте перед подтверждением
          </Text>
          {validationWarnings.map((w) => (
            <Text key={w} style={{ color: "#c2410c", fontSize: 13, marginBottom: 4 }}>
              • {w}
            </Text>
          ))}
        </View>
      ) : (
        <View
          style={{
            backgroundColor: "#ecfdf5",
            borderColor: "#6ee7b7",
            borderWidth: 1,
            borderRadius: 12,
            padding: 12,
            marginBottom: 12
          }}
        >
          <Text style={{ color: "#047857", fontWeight: "600" }}>
            Основные поля распознаны. Всё равно сверьте значения ниже.
          </Text>
        </View>
      )}
      {fieldRow("Номер сметы", estimate.estimateNumber)}
      {fieldRow("Наименование стройки", estimate.projectName)}
      {fieldRow("Работы / объект", estimate.objectDescription)}
      {fieldRow("Основание (чертежи)", estimate.basis)}
      {fieldRow("Сметная стоимость", estimate.estimatedCost)}
      {fieldRow("Оплата труда", estimate.laborCosts)}
      {fieldRow("Цены на дату", estimate.priceDate)}
      {fieldRow("Итого прямые затраты", estimate.totalDirectCosts)}
      {fieldRow("Всего по смете", estimate.grandTotal)}
      {lineItems.length > 0 ? (
        <View style={{ marginTop: 8 }}>
          <Text style={{ fontSize: 12, color: "#64748b", marginBottom: 6 }}>Строки сметы</Text>
          {lineItems.map((line) => (
            <Text key={`${line.seq}-${line.code}`} style={{ fontSize: 13, color: "#0f172a", marginBottom: 6 }}>
              {line.seq}. {line.description || line.code || "—"} · {line.quantity} {line.unit}
            </Text>
          ))}
        </View>
      ) : null}
      {estimate.rawTranscript ? (
        <View style={{ marginTop: 12 }}>
          <Text style={{ fontSize: 12, color: "#64748b", marginBottom: 4 }}>Транскрипт</Text>
          <Text style={{ fontSize: 13, color: "#334155" }}>{estimate.rawTranscript}</Text>
        </View>
      ) : null}
    </View>
  );
}

export function EstimateDocumentFlowScreen({
  topInset,
  template,
  deals,
  dealsLoading,
  bitrixConnected,
  recordingPhase,
  audioUri,
  sourceName,
  payloadExtra,
  onClose,
  onSuccess,
  onSourceNameChange,
  onPayloadExtraChange,
  onStartRecording,
  onStopRecording,
  onResetRecording,
  onPreview,
  previewPlayingUri,
  renderRecordingControls,
  recordingStatusText
}: Props) {
  const [step, setStep] = useState<Step>("deal");
  const [selectedDeal, setSelectedDeal] = useState<{ id: number; title: string } | null>(null);
  const [voiceResult, setVoiceResult] = useState<MobileVoiceRequestResult | null>(null);
  const [confirmResult, setConfirmResult] = useState<MobileDocumentJobView | null>(null);
  const [busy, setBusy] = useState(false);

  const resetFlow = useCallback(() => {
    setStep("deal");
    setSelectedDeal(null);
    setVoiceResult(null);
    setConfirmResult(null);
    onResetRecording();
  }, [onResetRecording]);

  const handleClose = () => {
    resetFlow();
    onClose();
  };

  const submitVoice = async () => {
    if (!template || !selectedDeal || !audioUri) {
      return;
    }
    setBusy(true);
    setStep("processing");
    try {
      const item = await createMobileVoiceRequest({
        templateId: template.id,
        sourceName: sourceName.trim() || selectedDeal.title,
        requestedBy: "mobile-app",
        payload: payloadExtra,
        deliveryChannel: "internal",
        deliveryAddress: "",
        taskCommandText: "",
        taskTarget: "bitrix24",
        dealId: selectedDeal.id,
        dealTitle: selectedDeal.title,
        audioUri,
        audioFileName: `voice-estimate-${Date.now()}.m4a`,
        audioMimeType: "audio/mp4"
      });
      setVoiceResult(item);
      setStep("review");
    } catch (err) {
      setStep("record");
      Alert.alert("Ошибка", err instanceof Error ? err.message : "Не удалось обработать запись");
    } finally {
      setBusy(false);
    }
  };

  const submitConfirm = async () => {
    if (!voiceResult?.job.id) {
      return;
    }
    setBusy(true);
    try {
      const item = await confirmMobileDocumentJob(voiceResult.job.id);
      setConfirmResult(item);
      setStep("done");
      if (!item.canRetryBitrixAttach) {
        onSuccess();
      }
    } catch (err) {
      Alert.alert("Ошибка", err instanceof Error ? err.message : "Не удалось подтвердить смету");
    } finally {
      setBusy(false);
    }
  };

  const openDownload = async (path?: string) => {
    if (!path) {
      return;
    }
    const url = generatedDocumentDownloadUrl(path);
    try {
      await Linking.openURL(url);
    } catch {
      Alert.alert("Файл", `Откройте в браузере:\n${url}`);
    }
  };

  let title = "Новый документ";
  if (step === "deal") {
    title = "Смета · сделка";
  } else if (step === "record") {
    title = "Запись голоса";
  } else if (step === "review") {
    title = "Проверка полей";
  } else if (step === "done") {
    title = "Готово";
  }

  return (
    <View style={{ flex: 1, backgroundColor: "#f8fafc" }}>
      <View
        style={{
          flexDirection: "row",
          alignItems: "center",
          paddingTop: topInset + 8,
          paddingHorizontal: 16,
          paddingBottom: 12,
          borderBottomWidth: 1,
          borderBottomColor: "#e2e8f0",
          backgroundColor: "#fff"
        }}
      >
        <Pressable onPress={handleClose} hitSlop={12}>
          <Text style={{ color: HEADER_BLUE, fontSize: 16 }}>Закрыть</Text>
        </Pressable>
        <Text style={{ flex: 1, textAlign: "center", fontSize: 17, fontWeight: "700", color: "#0f172a" }}>
          {title}
        </Text>
        <View style={{ width: 56 }} />
      </View>

      <ScrollView
        style={{ flex: 1 }}
        contentContainerStyle={{ padding: 16, paddingBottom: 32 }}
        keyboardShouldPersistTaps="handled"
      >
        <View
          style={{
            alignSelf: "flex-start",
            backgroundColor: "#eff6ff",
            paddingHorizontal: 12,
            paddingVertical: 6,
            borderRadius: 999,
            marginBottom: 12
          }}
        >
          <Text style={{ color: HEADER_BLUE, fontWeight: "600" }}>Тип: Смета</Text>
        </View>

        {step === "deal" ? (
          <>
            <Text style={{ color: "#64748b", marginBottom: 12 }}>
              Выберите сделку Bitrix24 — к ней будет прикреплён итоговый файл после подтверждения.
            </Text>
            {!bitrixConnected ? (
              <Text style={{ color: "#b45309", marginBottom: 12 }}>
                Войдите в Bitrix24 (иконка профиля), чтобы выбрать сделку.
              </Text>
            ) : null}
            {dealsLoading ? <ActivityIndicator color={HEADER_BLUE} /> : null}
            {deals
              .filter((d) => d.id)
              .map((deal) => {
                const id = Number(deal.id);
                const selected = selectedDeal?.id === id;
                return (
                  <Pressable
                    key={deal.id}
                    onPress={() => setSelectedDeal({ id, title: deal.title || `Сделка #${deal.id}` })}
                    style={{
                      padding: 14,
                      borderRadius: 12,
                      borderWidth: 1,
                      borderColor: selected ? HEADER_BLUE : "#e2e8f0",
                      backgroundColor: selected ? "#eff6ff" : "#fff",
                      marginBottom: 8
                    }}
                  >
                    <Text style={{ fontWeight: "600", color: "#0f172a" }}>{deal.title || "Без названия"}</Text>
                    <Text style={{ color: "#64748b", fontSize: 12, marginTop: 4 }}>id: {deal.id}</Text>
                  </Pressable>
                );
              })}
            <Pressable
              disabled={!selectedDeal}
              onPress={() => setStep("record")}
              style={{
                marginTop: 16,
                backgroundColor: selectedDeal ? HEADER_BLUE : "#94a3b8",
                padding: 14,
                borderRadius: 12,
                alignItems: "center"
              }}
            >
              <Text style={{ color: "#fff", fontWeight: "700" }}>Далее · запись голоса</Text>
            </Pressable>
          </>
        ) : null}

        {step === "record" ? (
          <>
            {selectedDeal ? (
              <Text style={{ color: "#64748b", marginBottom: 12 }}>
                Сделка: {selectedDeal.title} (id {selectedDeal.id})
              </Text>
            ) : null}
            {!template ? (
              <Text style={{ color: "#b45309" }}>
                Шаблон сметы не найден. Перезапустите backend (docker compose up -d --build
                backend-api) — при старте загрузится Word-шаблон docx.
              </Text>
            ) : (
              <Text style={{ color: "#64748b", marginBottom: 12 }}>
                Диктуйте поля: стройка, основание, стоимость, позиции работ. После записи вы проверите
                распознанные поля перед формированием файла.
              </Text>
            )}
            <View style={{ flexDirection: "row", marginBottom: 8 }}>
              {renderRecordingControls(
                recordingPhase,
                audioUri,
                onStartRecording,
                onStopRecording,
                onResetRecording,
                onPreview
              )}
            </View>
            <Text style={{ color: "#64748b", marginBottom: 12 }}>{recordingStatusText(recordingPhase)}</Text>
            <TextInput
              placeholder="Название заявки (необязательно)"
              placeholderTextColor="#94a3b8"
              style={{
                borderWidth: 1,
                borderColor: "#cbd5e1",
                borderRadius: 12,
                padding: 12,
                marginBottom: 8,
                backgroundColor: "#fff"
              }}
              value={sourceName}
              onChangeText={onSourceNameChange}
            />
            <TextInput
              placeholder="Дополнение текстом"
              placeholderTextColor="#94a3b8"
              multiline
              style={{
                borderWidth: 1,
                borderColor: "#cbd5e1",
                borderRadius: 12,
                padding: 12,
                minHeight: 72,
                marginBottom: 12,
                backgroundColor: "#fff"
              }}
              value={payloadExtra}
              onChangeText={onPayloadExtraChange}
            />
            <Pressable
              disabled={busy || recordingPhase !== "ready" || !audioUri || !template || !selectedDeal}
              onPress={() => void submitVoice()}
              style={{
                backgroundColor:
                  recordingPhase === "ready" && audioUri && template ? HEADER_BLUE : "#94a3b8",
                padding: 14,
                borderRadius: 12,
                alignItems: "center"
              }}
            >
              <Text style={{ color: "#fff", fontWeight: "700" }}>Распознать и показать поля</Text>
            </Pressable>
            <Pressable onPress={() => setStep("deal")} style={{ marginTop: 12, alignItems: "center" }}>
              <Text style={{ color: HEADER_BLUE }}>Назад к выбору сделки</Text>
            </Pressable>
          </>
        ) : null}

        {step === "processing" ? (
          <View style={{ alignItems: "center", paddingVertical: 40 }}>
            <ActivityIndicator size="large" color={HEADER_BLUE} />
            <Text style={{ marginTop: 16, color: "#64748b" }}>Whisper и разбор полей…</Text>
          </View>
        ) : null}

        {step === "review" && voiceResult ? (
          <>
            <EstimateReviewBlock estimate={voiceResult.estimate} />
            <Pressable
              disabled={busy}
              onPress={() => void submitConfirm()}
              style={{
                marginTop: 16,
                backgroundColor: HEADER_BLUE,
                padding: 14,
                borderRadius: 12,
                alignItems: "center"
              }}
            >
              <Text style={{ color: "#fff", fontWeight: "700" }}>
                {busy ? "Формирование…" : "Подтвердить и прикрепить к сделке"}
              </Text>
            </Pressable>
            <Pressable
              onPress={() => {
                setVoiceResult(null);
                setStep("record");
                onResetRecording();
              }}
              style={{ marginTop: 12, alignItems: "center" }}
            >
              <Text style={{ color: "#64748b" }}>Перезаписать голос</Text>
            </Pressable>
          </>
        ) : null}

        {step === "done" && confirmResult ? (
          <>
            <View
              style={{
                backgroundColor: confirmResult.canRetryBitrixAttach ? "#fff7ed" : "#ecfdf5",
                borderRadius: 12,
                padding: 16,
                marginBottom: 12,
                borderWidth: confirmResult.canRetryBitrixAttach ? 1 : 0,
                borderColor: "#fdba74"
              }}
            >
              <Ionicons
                name={confirmResult.canRetryBitrixAttach ? "warning" : "checkmark-circle"}
                size={32}
                color={confirmResult.canRetryBitrixAttach ? "#c2410c" : "#059669"}
              />
              <Text style={{ fontWeight: "700", fontSize: 16, marginTop: 8, color: "#0f172a" }}>
                {confirmResult.canRetryBitrixAttach
                  ? "Документ готов, Bitrix не прикрепил"
                  : "Смета сформирована (Word)"}
              </Text>
              <Text style={{ color: "#64748b", marginTop: 6 }}>
                {confirmResult.canRetryBitrixAttach
                  ? (confirmResult.job.errorMessage ||
                    "Укажите BITRIX_DEAL_ESTIMATE_FIELD в .env или проверьте поле «Смета» в CRM.")
                  : `Файл .docx сохранён${confirmResult.job.bitrixDealTitle ? ` и прикреплён к сделке «${confirmResult.job.bitrixDealTitle}»` : ""}. Откройте во вкладке «Документы».`}
              </Text>
            </View>
            {confirmResult.downloadPath || confirmResult.job.resultDocumentId ? (
              <Pressable
                onPress={() =>
                  void openDownload(
                    confirmResult.downloadPath ??
                      (confirmResult.job.resultDocumentId
                        ? `/api/v1/generated-documents/${confirmResult.job.resultDocumentId}/download`
                        : undefined)
                  )
                }
                style={{
                  borderWidth: 1,
                  borderColor: HEADER_BLUE,
                  borderRadius: 12,
                  padding: 14,
                  alignItems: "center",
                  marginBottom: 8
                }}
              >
                <Text style={{ color: HEADER_BLUE, fontWeight: "600" }}>Открыть файл сметы</Text>
              </Pressable>
            ) : null}
            {confirmResult.canRetryBitrixAttach ? (
              <Pressable
                disabled={busy}
                onPress={async () => {
                  if (!confirmResult.job.id) {
                    return;
                  }
                  setBusy(true);
                  try {
                    const item = await retryAttachMobileDocumentJob(confirmResult.job.id);
                    setConfirmResult(item);
                    if (!item.canRetryBitrixAttach) {
                      onSuccess();
                    }
                  } catch (err) {
                    Alert.alert(
                      "Bitrix",
                      err instanceof Error ? err.message : "Не удалось прикрепить к сделке"
                    );
                  } finally {
                    setBusy(false);
                  }
                }}
                style={{
                  backgroundColor: HEADER_BLUE,
                  padding: 14,
                  borderRadius: 12,
                  alignItems: "center",
                  marginBottom: 8
                }}
              >
                <Text style={{ color: "#fff", fontWeight: "700" }}>
                  {busy ? "Прикрепление…" : "Повторить прикрепление к сделке"}
                </Text>
              </Pressable>
            ) : null}
            <Pressable
              onPress={handleClose}
              style={{
                backgroundColor: confirmResult.canRetryBitrixAttach ? "#fff" : HEADER_BLUE,
                borderWidth: confirmResult.canRetryBitrixAttach ? 1 : 0,
                borderColor: "#cbd5e1",
                padding: 14,
                borderRadius: 12,
                alignItems: "center"
              }}
            >
              <Text
                style={{
                  color: confirmResult.canRetryBitrixAttach ? HEADER_BLUE : "#fff",
                  fontWeight: "700"
                }}
              >
                {confirmResult.canRetryBitrixAttach ? "Закрыть" : "Готово"}
              </Text>
            </Pressable>
          </>
        ) : null}
      </ScrollView>
    </View>
  );
}
