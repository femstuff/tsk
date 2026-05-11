import { Audio } from "expo-av";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

import type {
  DocumentJobSummary,
  DocumentTemplateSummary,
  HealthResponse,
  SourceDocumentSummary,
  TaskCommandSummary
} from "../entities/document-template/types";
import {
  createMobileVoiceRequest,
  createTaskCommand,
  getHealth,
  getMobileApiBaseUrl,
  listJobs,
  listSourceDocuments,
  listTaskCommands,
  listTemplates
} from "../shared/api/client";

function formatDate(value: string | null) {
  if (!value) {
    return "not started";
  }

  return new Intl.DateTimeFormat("en", {
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
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [templates, setTemplates] = useState<DocumentTemplateSummary[]>([]);
  const [jobs, setJobs] = useState<DocumentJobSummary[]>([]);
  const [sourceDocuments, setSourceDocuments] = useState<SourceDocumentSummary[]>([]);
  const [taskCommands, setTaskCommands] = useState<TaskCommandSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [submittingRequest, setSubmittingRequest] = useState(false);
  const [submittingCommand, setSubmittingCommand] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [recording, setRecording] = useState<Audio.Recording | null>(null);
  const [audioUri, setAudioUri] = useState<string | null>(null);
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
  const [quickCommand, setQuickCommand] = useState({
    targetSystem: "bitrix24" as "bitrix24" | "email_approval",
    commandText: ""
  });

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [nextHealth, nextTemplates, nextJobs, nextSourceDocuments, nextTaskCommands] =
        await Promise.all([
          getHealth(),
          listTemplates(),
          listJobs(),
          listSourceDocuments(),
          listTaskCommands()
        ]);

      setHealth(nextHealth);
      setTemplates(nextTemplates);
      const mobileJobs = nextJobs.filter((job) => job.requestedBy === "mobile-app");
      const mobileJobIds = new Set(mobileJobs.map((job) => job.id));
      setJobs(mobileJobs);
      setSourceDocuments(
        nextSourceDocuments.filter((document) => document.origin === "mobile-app")
      );
      setTaskCommands(
        nextTaskCommands.filter(
          (command) => command.jobId === null || mobileJobIds.has(command.jobId)
        )
      );
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const selectedTemplate = useMemo(
    () => templates.find((template) => template.id === requestForm.templateId) ?? null,
    [requestForm.templateId, templates]
  );

  const startRecording = async () => {
    try {
      const permission = await Audio.requestPermissionsAsync();
      if (!permission.granted) {
        Alert.alert("Permission required", "Allow microphone access to record a request.");
        return;
      }

      await Audio.setAudioModeAsync({
        allowsRecordingIOS: true,
        playsInSilentModeIOS: true
      });

      const created = await Audio.Recording.createAsync(
        Audio.RecordingOptionsPresets.HIGH_QUALITY
      );
      setRecording(created.recording);
      setAudioUri(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Unable to start recording");
    }
  };

  const stopRecording = async () => {
    if (!recording) {
      return;
    }

    try {
      await recording.stopAndUnloadAsync();
      const uri = recording.getURI();
      setAudioUri(uri ?? null);
      setRecording(null);
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Unable to stop recording");
    }
  };

  const submitVoiceRequest = async () => {
    if (!audioUri || !requestForm.templateId || !requestForm.sourceName.trim()) {
      return;
    }

    setSubmittingRequest(true);
    try {
      await createMobileVoiceRequest({
        ...requestForm,
        audioUri,
        audioFileName: `voice-request-${Date.now()}.m4a`,
        audioMimeType: "audio/mp4"
      });
      setAudioUri(null);
      setRequestForm((current) => ({
        ...current,
        sourceName: "",
        payload: "",
        taskCommandText: ""
      }));
      await refresh();
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Unable to create request");
    } finally {
      setSubmittingRequest(false);
    }
  };

  const submitQuickCommand = async () => {
    if (!quickCommand.commandText.trim()) {
      return;
    }

    setSubmittingCommand(true);
    try {
      await createTaskCommand({
        targetSystem: quickCommand.targetSystem,
        commandText: quickCommand.commandText
      });
      setQuickCommand((current) => ({ ...current, commandText: "" }));
      await refresh();
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : "Unable to create command");
    } finally {
      setSubmittingCommand(false);
    }
  };

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="dark" />
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.hero}>
          <Text style={styles.eyebrow}>TSK mobile MVP</Text>
          <Text style={styles.title}>Voice requests and task command capture</Text>
          <Text style={styles.subtitle}>
            Record a voice note, bind it to a template, create a backend job, and
            track request, audio, and task-command status from one screen.
          </Text>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Backend target</Text>
          <Text style={styles.panelValue}>{getMobileApiBaseUrl()}</Text>
          <Text style={styles.smallText}>
            For a real device, set `EXPO_PUBLIC_API_BASE_URL` to your host LAN IP.
          </Text>
          <Text style={styles.smallText}>
            Status: {health?.status ?? "loading"} · jobs: {health?.jobsCreatedTotal ?? 0}
          </Text>
        </View>

        {error ? (
          <View style={[styles.panel, styles.errorPanel]}>
            <Text style={styles.errorText}>{error}</Text>
          </View>
        ) : null}

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>1. Choose template</Text>
          {loading ? <ActivityIndicator color="#4338ca" /> : null}
          {templates.map((template) => {
            const selected = template.id === requestForm.templateId;
            return (
              <Pressable
                key={template.id}
                onPress={() =>
                  setRequestForm((current) => ({
                    ...current,
                    templateId: template.id
                  }))
                }
                style={[styles.templateCard, selected && styles.templateCardSelected]}
              >
                <Text style={styles.templateName}>{template.name}</Text>
                <Text style={styles.templateCategory}>
                  {template.category} · {template.version}
                </Text>
                <Text style={styles.templateDescription}>{template.description}</Text>
              </Pressable>
            );
          })}
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>2. Record voice request</Text>
          <View style={styles.row}>
            <Pressable
              onPress={() => void startRecording()}
              disabled={recording !== null}
              style={[styles.button, styles.primaryButton, recording && styles.buttonDisabled]}
            >
              <Text style={styles.buttonText}>Start recording</Text>
            </Pressable>
            <Pressable
              onPress={() => void stopRecording()}
              disabled={recording === null}
              style={[styles.button, recording === null && styles.buttonDisabled]}
            >
              <Text style={styles.buttonTextSecondary}>Stop recording</Text>
            </Pressable>
          </View>
          <Text style={styles.smallText}>
            {recording
              ? "Recording in progress..."
              : audioUri
                ? "Voice note captured and ready to upload."
                : "No recording captured yet."}
          </Text>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>3. Submit document request</Text>
          <TextInput
            placeholder="Request title / source name"
            placeholderTextColor="#94a3b8"
            style={styles.input}
            value={requestForm.sourceName}
            onChangeText={(value) =>
              setRequestForm((current) => ({ ...current, sourceName: value }))
            }
          />
          <TextInput
            placeholder="Notes or manual transcript"
            placeholderTextColor="#94a3b8"
            style={[styles.input, styles.textArea]}
            multiline
            value={requestForm.payload}
            onChangeText={(value) =>
              setRequestForm((current) => ({ ...current, payload: value }))
            }
          />
          <Text style={styles.smallLabel}>Delivery routing</Text>
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
            placeholder="Delivery address or approval target"
            placeholderTextColor="#94a3b8"
            style={styles.input}
            value={requestForm.deliveryAddress}
            onChangeText={(value) =>
              setRequestForm((current) => ({ ...current, deliveryAddress: value }))
            }
          />
          <TextInput
            placeholder="Optional task command, e.g. Create Bitrix follow-up task"
            placeholderTextColor="#94a3b8"
            style={[styles.input, styles.textArea]}
            multiline
            value={requestForm.taskCommandText}
            onChangeText={(value) =>
              setRequestForm((current) => ({ ...current, taskCommandText: value }))
            }
          />
          <Text style={styles.smallLabel}>Task command target</Text>
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
              !audioUri ||
              !selectedTemplate ||
              requestForm.sourceName.trim() === ""
            }
            style={[
              styles.button,
              styles.primaryButton,
              (submittingRequest || !audioUri || !selectedTemplate) && styles.buttonDisabled
            ]}
          >
            <Text style={styles.buttonText}>
              {submittingRequest ? "Submitting..." : "Create voice document request"}
            </Text>
          </Pressable>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Quick task command</Text>
          <TextInput
            placeholder="Create Bitrix24 task for approval"
            placeholderTextColor="#94a3b8"
            style={[styles.input, styles.textArea]}
            multiline
            value={quickCommand.commandText}
            onChangeText={(value) =>
              setQuickCommand((current) => ({ ...current, commandText: value }))
            }
          />
          <View style={styles.segmentRow}>
            {(["bitrix24", "email_approval"] as const).map((option) => (
              <Pressable
                key={option}
                onPress={() =>
                  setQuickCommand((current) => ({ ...current, targetSystem: option }))
                }
                style={[
                  styles.segment,
                  quickCommand.targetSystem === option && styles.segmentSelected
                ]}
              >
                <Text
                  style={
                    quickCommand.targetSystem === option
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
            onPress={() => void submitQuickCommand()}
            disabled={submittingCommand || quickCommand.commandText.trim() === ""}
            style={[
              styles.button,
              submittingCommand || quickCommand.commandText.trim() === ""
                ? styles.buttonDisabled
                : styles.secondaryButton
            ]}
          >
            <Text style={styles.buttonTextSecondary}>
              {submittingCommand ? "Sending..." : "Create command"}
            </Text>
          </Pressable>
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Recent mobile requests</Text>
          {jobs.length === 0 ? <Text style={styles.smallText}>No mobile requests yet.</Text> : null}
          {jobs.map((job) => (
            <View key={job.id} style={styles.listItem}>
              <View style={styles.listHeader}>
                <Text style={styles.listTitle}>{job.sourceName}</Text>
                <Text style={[styles.badge, { color: statusColor(job.status) }]}>
                  {job.status}
                </Text>
              </View>
              <Text style={styles.smallText}>
                {job.templateName} · dispatch {job.dispatchStatus}
              </Text>
              <Text style={styles.smallText}>Created {formatDate(job.createdAt)}</Text>
            </View>
          ))}
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Stored voice artifacts</Text>
          {sourceDocuments.length === 0 ? (
            <Text style={styles.smallText}>No audio files uploaded yet.</Text>
          ) : null}
          {sourceDocuments.map((document) => (
            <View key={document.id} style={styles.listItem}>
              <Text style={styles.listTitle}>{document.fileName}</Text>
              <Text style={styles.smallText}>
                {document.kind} · {Math.round(document.sizeBytes / 1024)} KB
              </Text>
              <Text style={styles.smallText}>Stored {formatDate(document.createdAt)}</Text>
            </View>
          ))}
        </View>

        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Task command status</Text>
          {taskCommands.length === 0 ? (
            <Text style={styles.smallText}>No task commands recorded yet.</Text>
          ) : null}
          {taskCommands.map((command) => (
            <View key={command.id} style={styles.listItem}>
              <View style={styles.listHeader}>
                <Text style={styles.listTitle}>{command.targetSystem}</Text>
                <Text style={[styles.badge, { color: statusColor(command.status) }]}>
                  {command.status}
                </Text>
              </View>
              <Text style={styles.smallText}>{command.commandText}</Text>
              <Text style={styles.smallText}>{command.resultMessage}</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    backgroundColor: "#f8fafc"
  },
  content: {
    gap: 16,
    padding: 20
  },
  hero: {
    gap: 8,
    paddingTop: 8
  },
  eyebrow: {
    color: "#4f46e5",
    fontSize: 12,
    fontWeight: "700",
    letterSpacing: 1,
    textTransform: "uppercase"
  },
  title: {
    color: "#0f172a",
    fontSize: 28,
    fontWeight: "700"
  },
  subtitle: {
    color: "#475569",
    fontSize: 15,
    lineHeight: 22
  },
  panel: {
    backgroundColor: "#ffffff",
    borderColor: "#dbe4ff",
    borderRadius: 20,
    borderWidth: 1,
    gap: 12,
    padding: 16
  },
  errorPanel: {
    borderColor: "#fecaca",
    backgroundColor: "#fef2f2"
  },
  errorText: {
    color: "#b91c1c",
    fontSize: 14
  },
  panelTitle: {
    color: "#0f172a",
    fontSize: 17,
    fontWeight: "700"
  },
  panelValue: {
    color: "#334155",
    fontSize: 14
  },
  templateCard: {
    backgroundColor: "#eef2ff",
    borderRadius: 16,
    gap: 4,
    padding: 14
  },
  templateCardSelected: {
    borderColor: "#4338ca",
    borderWidth: 2
  },
  templateName: {
    color: "#1e293b",
    fontSize: 16,
    fontWeight: "600"
  },
  templateCategory: {
    color: "#4338ca",
    fontSize: 12,
    fontWeight: "700",
    textTransform: "uppercase"
  },
  templateDescription: {
    color: "#475569",
    fontSize: 14,
    lineHeight: 20
  },
  input: {
    borderColor: "#cbd5e1",
    borderRadius: 14,
    borderWidth: 1,
    color: "#0f172a",
    fontSize: 15,
    paddingHorizontal: 14,
    paddingVertical: 12
  },
  textArea: {
    minHeight: 90,
    textAlignVertical: "top"
  },
  smallText: {
    color: "#64748b",
    fontSize: 13,
    lineHeight: 18
  },
  smallLabel: {
    color: "#334155",
    fontSize: 13,
    fontWeight: "600"
  },
  row: {
    flexDirection: "row",
    gap: 12
  },
  button: {
    alignItems: "center",
    borderRadius: 14,
    justifyContent: "center",
    minHeight: 48,
    paddingHorizontal: 16,
    paddingVertical: 12
  },
  primaryButton: {
    backgroundColor: "#4338ca"
  },
  secondaryButton: {
    backgroundColor: "#e2e8f0"
  },
  buttonDisabled: {
    opacity: 0.5
  },
  buttonText: {
    color: "#ffffff",
    fontSize: 14,
    fontWeight: "600"
  },
  buttonTextSecondary: {
    color: "#0f172a",
    fontSize: 14,
    fontWeight: "600"
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
    paddingHorizontal: 14,
    paddingVertical: 8
  },
  segmentSelected: {
    backgroundColor: "#4338ca",
    borderColor: "#4338ca"
  },
  segmentText: {
    color: "#334155",
    fontSize: 13,
    fontWeight: "600"
  },
  segmentTextSelected: {
    color: "#ffffff",
    fontSize: 13,
    fontWeight: "600"
  },
  listItem: {
    backgroundColor: "#f8fafc",
    borderRadius: 14,
    gap: 6,
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
    fontWeight: "600",
    marginRight: 8
  },
  badge: {
    fontSize: 12,
    fontWeight: "700",
    textTransform: "uppercase"
  }
});
