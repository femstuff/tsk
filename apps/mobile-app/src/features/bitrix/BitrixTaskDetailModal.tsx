import { Ionicons } from "@expo/vector-icons";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Linking,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View
} from "react-native";

import type { BitrixTaskDetail } from "../../entities/document-template/types";
import { getBitrixTask, updateBitrixTaskStatus } from "../../shared/api/client";
import { useBitrixAuth } from "./useBitrixAuth";
import {
  bitrixMarkRu,
  bitrixPriorityRu,
  bitrixTaskStatusActions,
  bitrixTaskStatusRu,
  formatBitrixAuthor,
  formatBitrixDate,
  formatBitrixDuration,
  formatBitrixDurationType,
  formatBitrixFileSize,
  formatBitrixList,
  formatBitrixPeople,
  formatBitrixPerson,
  formatBitrixReference,
  resolveBitrixFileUrl,
  stripBitrixDescription
} from "./bitrixTaskUi";

const HEADER_BLUE = "#2563eb";

type Props = {
  taskId: string | null;
  visible: boolean;
  onClose: () => void;
  onUpdated: () => void;
};

function InfoRow({
  label,
  value,
  alwaysShow = false
}: {
  label: string;
  value: string;
  alwaysShow?: boolean;
}) {
  const text = value?.trim() || "—";
  if (!alwaysShow && text === "—") {
    return null;
  }
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{label}</Text>
      <Text style={styles.infoValue}>{text}</Text>
    </View>
  );
}

export function BitrixTaskDetailModal({ taskId, visible, onClose, onUpdated }: Props) {
  const bitrixAuth = useBitrixAuth();
  const portalDomain = bitrixAuth.session?.portalDomain;
  const [detail, setDetail] = useState<BitrixTaskDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!taskId) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const item = await getBitrixTask(taskId);
      setDetail(item);
    } catch (err) {
      setDetail(null);
      setError(err instanceof Error ? err.message : "Не удалось загрузить задачу");
    } finally {
      setLoading(false);
    }
  }, [taskId]);

  useEffect(() => {
    if (visible && taskId) {
      void load();
    }
    if (!visible) {
      setDetail(null);
      setError(null);
    }
  }, [visible, taskId, load]);

  const handleStatusChange = async (status: number) => {
    if (!taskId || updating) {
      return;
    }
    setUpdating(true);
    setError(null);
    try {
      const item = await updateBitrixTaskStatus(taskId, status);
      setDetail(item);
      onUpdated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось обновить статус");
    } finally {
      setUpdating(false);
    }
  };

  const openFile = async (url?: string) => {
    const resolved = resolveBitrixFileUrl(url, portalDomain);
    if (!resolved) {
      return;
    }
    try {
      await Linking.openURL(resolved);
    } catch {
      setError("Не удалось открыть файл");
    }
  };

  const actions = detail ? bitrixTaskStatusActions(detail.status) : [];
  const description = detail?.description ? stripBitrixDescription(detail.description) : "";

  return (
    <Modal visible={visible} animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <View style={styles.root}>
        <View style={styles.header}>
          <Pressable onPress={onClose} hitSlop={12} style={styles.closeBtn}>
            <Ionicons name="close" size={26} color="#0f172a" />
          </Pressable>
          <Text style={styles.headerTitle}>Задача Bitrix24</Text>
          <Pressable onPress={() => void load()} hitSlop={12} style={styles.closeBtn}>
            <Ionicons name="refresh" size={22} color={HEADER_BLUE} />
          </Pressable>
        </View>

        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          {loading ? <ActivityIndicator color={HEADER_BLUE} style={styles.loader} /> : null}

          {!loading && detail ? (
            <>
              <Text style={styles.title}>{detail.title || "Без названия"}</Text>
              <View style={styles.metaRow}>
                <Text style={styles.badge}>{bitrixTaskStatusRu(detail.status)}</Text>
                <Text style={styles.taskIdMeta}>#{detail.id}</Text>
              </View>

              <View style={styles.infoCard}>
                <Text style={styles.sectionTitle}>Участники</Text>
                <InfoRow label="Постановщик" value={formatBitrixPerson(detail.creator)} alwaysShow />
                <InfoRow label="Исполнитель" value={formatBitrixPerson(detail.responsible)} alwaysShow />
                <InfoRow label="Соисполнители" value={formatBitrixPeople(detail.accomplices)} />
                <InfoRow label="Наблюдатели" value={formatBitrixPeople(detail.auditors)} />
              </View>

              <View style={styles.infoCard}>
                <Text style={styles.sectionTitle}>Сроки</Text>
                <InfoRow label="Поставлена" value={formatBitrixDate(detail.createdDate)} alwaysShow />
                <InfoRow label="Изменена" value={formatBitrixDate(detail.changedDate)} />
                <InfoRow label="Крайний срок" value={formatBitrixDate(detail.deadline)} alwaysShow />
                <InfoRow label="Начало" value={formatBitrixDate(detail.dateStart)} />
                <InfoRow label="План начала" value={formatBitrixDate(detail.startDatePlan)} />
                <InfoRow label="План окончания" value={formatBitrixDate(detail.endDatePlan)} />
                <InfoRow label="Закрыта" value={formatBitrixDate(detail.closedDate)} />
              </View>

              <View style={styles.infoCard}>
                <Text style={styles.sectionTitle}>Связи и контекст</Text>
                <InfoRow
                  label="Родительская задача"
                  value={formatBitrixReference(detail.parentTitle, detail.parentId)}
                />
                <InfoRow
                  label="Группа / проект"
                  value={formatBitrixReference(detail.groupTitle, detail.groupId)}
                />
                <InfoRow
                  label="Стадия"
                  value={formatBitrixReference(detail.stageLabel, detail.stageId)}
                />
                <InfoRow label="CRM" value={formatBitrixList(detail.crmLinks)} />
                <InfoRow label="Теги" value={formatBitrixList(detail.tags)} />
              </View>

              <View style={styles.infoCard}>
                <Text style={styles.sectionTitle}>Дополнительно</Text>
                <InfoRow label="Приоритет" value={bitrixPriorityRu(detail.priority)} />
                <InfoRow label="Оценка" value={bitrixMarkRu(detail.mark)} />
                <InfoRow label="Плановое время" value={formatBitrixDuration(detail.timeEstimate)} />
                <InfoRow label="План длительности" value={formatBitrixDuration(detail.durationPlan)} />
                <InfoRow label="Тип длительности" value={formatBitrixDurationType(detail.durationType)} />
                <InfoRow label="Затрачено" value={formatBitrixDuration(detail.durationFact)} />
              </View>

              {(detail.checklist?.length ?? 0) > 0 ? (
                <View style={styles.infoCard}>
                  <Text style={styles.sectionTitle}>Чек-лист</Text>
                  {detail.checklist?.map((item, index) => (
                    <View key={item.id ?? `${item.title}-${index}`} style={styles.checklistRow}>
                      <Ionicons
                        name={item.isComplete ? "checkbox" : "square-outline"}
                        size={18}
                        color={item.isComplete ? "#16a34a" : "#64748b"}
                      />
                      <Text style={[styles.checklistText, item.isComplete && styles.checklistDone]}>
                        {item.title}
                      </Text>
                    </View>
                  ))}
                </View>
              ) : null}

              {(detail.files?.length ?? 0) > 0 ? (
                <View style={styles.infoCard}>
                  <Text style={styles.sectionTitle}>Файлы</Text>
                  {detail.files?.map((file, index) => {
                    const sizeLabel = formatBitrixFileSize(file.size);
                    const openUrl = file.downloadUrl || file.viewUrl;
                    return (
                      <Pressable
                        key={file.id ?? `${file.name}-${index}`}
                        style={styles.fileRow}
                        onPress={() => void openFile(openUrl)}
                        disabled={!openUrl}
                      >
                        <Ionicons name="document-attach-outline" size={20} color={HEADER_BLUE} />
                        <View style={styles.fileMeta}>
                          <Text style={styles.fileName}>{file.name || file.id || "Файл"}</Text>
                          {sizeLabel ? <Text style={styles.fileSize}>{sizeLabel}</Text> : null}
                        </View>
                        {openUrl ? <Ionicons name="open-outline" size={18} color="#64748b" /> : null}
                      </Pressable>
                    );
                  })}
                </View>
              ) : null}

              <Text style={styles.sectionTitle}>Описание</Text>
              <Text style={styles.description}>{description || "Описание не указано."}</Text>

              <View style={styles.infoCard}>
                <Text style={styles.sectionTitle}>Чат задачи</Text>
                {(detail.comments?.length ?? 0) === 0 ? (
                  <Text style={styles.meta}>Комментариев пока нет.</Text>
                ) : (
                  detail.comments?.map((comment) => (
                    <View key={comment.id} style={styles.commentBubble}>
                      <View style={styles.commentHeader}>
                        <Text style={styles.commentAuthor}>
                          {formatBitrixAuthor(comment.authorName, comment.authorId)}
                        </Text>
                        <Text style={styles.commentDate}>{formatBitrixDate(comment.postDate)}</Text>
                      </View>
                      <Text style={styles.commentMessage}>
                        {stripBitrixDescription(comment.message) || "—"}
                      </Text>
                      {(comment.files?.length ?? 0) > 0 ? (
                        <View style={styles.commentFiles}>
                          {comment.files?.map((file, index) => {
                            const openUrl = file.downloadUrl || file.viewUrl;
                            return (
                              <Pressable
                                key={file.id ?? `${file.name}-${index}`}
                                style={styles.commentFileChip}
                                onPress={() => void openFile(openUrl)}
                                disabled={!openUrl}
                              >
                                <Ionicons name="attach" size={14} color={HEADER_BLUE} />
                                <Text style={styles.commentFileText}>{file.name || "Файл"}</Text>
                              </Pressable>
                            );
                          })}
                        </View>
                      ) : null}
                    </View>
                  ))
                )}
              </View>

              {actions.length > 0 ? (
                <>
                  <Text style={styles.sectionTitle}>Действия</Text>
                  <View style={styles.actionsRow}>
                    {actions.map((action) => (
                      <Pressable
                        key={action.status}
                        disabled={updating}
                        onPress={() => void handleStatusChange(action.status)}
                        style={[styles.actionBtn, updating && styles.actionBtnDisabled]}
                      >
                        <Text style={styles.actionBtnText}>{action.label}</Text>
                      </Pressable>
                    ))}
                  </View>
                </>
              ) : (
                <Text style={styles.meta}>Для этого статуса нет доступных действий.</Text>
              )}
            </>
          ) : null}

          {!loading && !detail && error ? <Text style={styles.error}>{error}</Text> : null}
          {error && detail ? <Text style={styles.error}>{error}</Text> : null}
          {updating ? <ActivityIndicator color={HEADER_BLUE} style={styles.loader} /> : null}
        </ScrollView>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: {
    backgroundColor: "#f8fafc",
    flex: 1
  },
  header: {
    alignItems: "center",
    borderBottomColor: "#e2e8f0",
    borderBottomWidth: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    paddingHorizontal: 12,
    paddingVertical: 14
  },
  headerTitle: {
    color: "#0f172a",
    fontSize: 17,
    fontWeight: "700"
  },
  closeBtn: {
    alignItems: "center",
    width: 36
  },
  content: {
    gap: 12,
    padding: 16,
    paddingBottom: 40
  },
  loader: {
    marginVertical: 24
  },
  title: {
    color: "#0f172a",
    fontSize: 22,
    fontWeight: "700"
  },
  metaRow: {
    alignItems: "center",
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10
  },
  badge: {
    backgroundColor: "#dbeafe",
    borderRadius: 999,
    color: "#1d4ed8",
    fontSize: 13,
    fontWeight: "600",
    overflow: "hidden",
    paddingHorizontal: 10,
    paddingVertical: 4
  },
  taskIdMeta: {
    color: "#64748b",
    fontSize: 13
  },
  infoCard: {
    backgroundColor: "#fff",
    borderColor: "#e2e8f0",
    borderRadius: 12,
    borderWidth: 1,
    gap: 8,
    padding: 14
  },
  infoRow: {
    gap: 2
  },
  infoLabel: {
    color: "#64748b",
    fontSize: 12,
    fontWeight: "600",
    textTransform: "uppercase"
  },
  infoValue: {
    color: "#0f172a",
    fontSize: 15,
    lineHeight: 21
  },
  meta: {
    color: "#64748b",
    fontSize: 14
  },
  sectionTitle: {
    color: "#0f172a",
    fontSize: 16,
    fontWeight: "700",
    marginTop: 4
  },
  description: {
    backgroundColor: "#fff",
    borderColor: "#e2e8f0",
    borderRadius: 12,
    borderWidth: 1,
    color: "#334155",
    fontSize: 15,
    lineHeight: 22,
    padding: 14
  },
  checklistRow: {
    alignItems: "center",
    flexDirection: "row",
    gap: 8
  },
  checklistText: {
    color: "#0f172a",
    flex: 1,
    fontSize: 15
  },
  checklistDone: {
    color: "#64748b",
    textDecorationLine: "line-through"
  },
  fileRow: {
    alignItems: "center",
    backgroundColor: "#f8fafc",
    borderRadius: 10,
    flexDirection: "row",
    gap: 10,
    paddingHorizontal: 10,
    paddingVertical: 8
  },
  fileMeta: {
    flex: 1,
    gap: 2
  },
  fileName: {
    color: "#0f172a",
    fontSize: 14,
    fontWeight: "600"
  },
  fileSize: {
    color: "#64748b",
    fontSize: 12
  },
  commentBubble: {
    backgroundColor: "#f8fafc",
    borderColor: "#e2e8f0",
    borderRadius: 12,
    borderWidth: 1,
    gap: 6,
    padding: 12
  },
  commentHeader: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between"
  },
  commentAuthor: {
    color: "#0f172a",
    flex: 1,
    fontSize: 14,
    fontWeight: "700"
  },
  commentDate: {
    color: "#64748b",
    fontSize: 12
  },
  commentMessage: {
    color: "#334155",
    fontSize: 14,
    lineHeight: 20
  },
  commentFiles: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 6
  },
  commentFileChip: {
    alignItems: "center",
    backgroundColor: "#eff6ff",
    borderRadius: 999,
    flexDirection: "row",
    gap: 4,
    paddingHorizontal: 10,
    paddingVertical: 6
  },
  commentFileText: {
    color: HEADER_BLUE,
    fontSize: 12,
    fontWeight: "600"
  },
  actionsRow: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8
  },
  actionBtn: {
    backgroundColor: HEADER_BLUE,
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 10
  },
  actionBtnDisabled: {
    opacity: 0.6
  },
  actionBtnText: {
    color: "#fff",
    fontSize: 14,
    fontWeight: "600"
  },
  error: {
    color: "#b91c1c",
    fontSize: 14
  }
});
