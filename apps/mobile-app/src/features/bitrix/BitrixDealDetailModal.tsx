import { Ionicons } from "@expo/vector-icons";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

import type { BitrixDealField, BitrixDealDetail } from "../../entities/document-template/types";
import { getBitrixDeal, updateBitrixDealFields, updateBitrixDealStage } from "../../shared/api/client";
import { formatBitrixDate, formatBitrixPerson } from "./bitrixTaskUi";

const HEADER_BLUE = "#2563eb";
const VISIBLE_FIELDS_COLLAPSED = 5;

const HIDDEN_DEAL_FIELD_KEYS = new Set([
  "ASSIGNED_BY_ID",
  "CREATED_BY_ID",
  "MODIFY_BY_ID",
  "MOVED_BY_ID",
  "LAST_ACTIVITY_BY",
  "STAGE_ID",
  "STAGE_SEMANTIC_ID",
  "IS_MANUAL_OPPORTUNITY",
  "IS_NEW",
  "IS_RECURRING",
  "IS_REPEATED_APPROACH",
  "IS_RETURN_CUSTOMER",
  "MOVED_TIME",
  "LAST_ACTIVITY_TIME"
]);

function dealStageIdsEqual(current: string, optionId: string) {
  const left = current.trim().toUpperCase();
  const right = optionId.trim().toUpperCase();
  if (left === right) {
    return true;
  }
  const leftShort = left.includes(":") ? (left.split(":").pop() ?? left) : left;
  const rightShort = right.includes(":") ? (right.split(":").pop() ?? right) : right;
  return leftShort === rightShort;
}

type Props = {
  dealId: string | null;
  visible: boolean;
  onClose: () => void;
  onUpdated: () => void;
};

function formatDealFieldValue(key: string, value: string) {
  const upper = key.toUpperCase();
  if (
    upper.includes("DATE") ||
    upper.endsWith("_TIME") ||
    upper === "BEGINDATE" ||
    upper === "CLOSEDATE"
  ) {
    const formatted = formatBitrixDate(value);
    return formatted !== "—" ? formatted : value;
  }
  return value;
}

function formatDealFieldDisplay(field: BitrixDealField) {
  const display = formatDealFieldValue(field.key, field.value);
  const raw = field.rawValue?.trim() ?? "";
  if (!display || display === "—") {
    return display;
  }
  if (raw && raw !== "0" && raw !== display && !display.includes("(#") && !display.includes(`#${raw}`)) {
    return `${display} (#${raw})`;
  }
  return display;
}

function DealFieldRow({
  field,
  editing,
  draftValue,
  saving,
  onStartEdit,
  onDraftChange,
  onSave,
  onCancel
}: {
  field: BitrixDealField;
  editing: boolean;
  draftValue: string;
  saving: boolean;
  onStartEdit: () => void;
  onDraftChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
}) {
  const displayValue = formatDealFieldDisplay(field);
  const hasOptions = (field.options?.length ?? 0) > 0;

  return (
    <View style={styles.infoRow}>
      <View style={styles.fieldHeader}>
        <Text style={styles.infoLabel}>{field.label}</Text>
        {field.editable && !editing ? (
          <Pressable onPress={onStartEdit} hitSlop={8}>
            <Ionicons name="pencil" size={16} color={HEADER_BLUE} />
          </Pressable>
        ) : null}
      </View>

      {editing ? (
        <View style={styles.editBlock}>
          {hasOptions ? (
            <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.optionRow}>
              {(field.options ?? []).map((option) => {
                const active = draftValue === option.id;
                return (
                  <Pressable
                    key={option.id}
                    onPress={() => onDraftChange(option.id)}
                    style={[styles.optionChip, active && styles.optionChipActive]}
                  >
                    <Text style={[styles.optionChipText, active && styles.optionChipTextActive]}>
                      {option.label}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>
          ) : (
            <TextInput
              value={draftValue}
              onChangeText={onDraftChange}
              style={styles.fieldInput}
              multiline={field.type === "text" || field.key === "COMMENTS"}
              editable={!saving}
            />
          )}
          <View style={styles.editActions}>
            <Pressable disabled={saving} onPress={onCancel} style={styles.editBtnSecondary}>
              <Text style={styles.editBtnSecondaryText}>Отмена</Text>
            </Pressable>
            <Pressable disabled={saving} onPress={onSave} style={styles.editBtnPrimary}>
              {saving ? (
                <ActivityIndicator color="#fff" size="small" />
              ) : (
                <Text style={styles.editBtnPrimaryText}>Сохранить</Text>
              )}
            </Pressable>
          </View>
        </View>
      ) : (
        <Text style={styles.infoValue}>{displayValue || "—"}</Text>
      )}
    </View>
  );
}

export function BitrixDealDetailModal({ dealId, visible, onClose, onUpdated }: Props) {
  const [detail, setDetail] = useState<BitrixDealDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [savingField, setSavingField] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldsExpanded, setFieldsExpanded] = useState(false);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [draftValue, setDraftValue] = useState("");

  const load = useCallback(async () => {
    if (!dealId) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const item = await getBitrixDeal(dealId);
      setDetail(item);
    } catch (err) {
      setDetail(null);
      setError(err instanceof Error ? err.message : "Не удалось загрузить сделку");
    } finally {
      setLoading(false);
    }
  }, [dealId]);

  useEffect(() => {
    if (visible && dealId) {
      void load();
    }
    if (!visible) {
      setDetail(null);
      setError(null);
      setFieldsExpanded(false);
      setEditingKey(null);
      setDraftValue("");
    }
  }, [dealId, load, visible]);

  const changeStage = async (stageId: string) => {
    if (!dealId || updating) {
      return;
    }
    setUpdating(true);
    setError(null);
    try {
      const item = await updateBitrixDealStage(dealId, stageId);
      setDetail(item);
      onUpdated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сменить этап");
    } finally {
      setUpdating(false);
    }
  };

  const startEditField = (field: BitrixDealField) => {
    setEditingKey(field.key);
    setDraftValue(field.rawValue?.trim() || field.value?.trim() || "");
  };

  const saveField = async (fieldKey: string) => {
    if (!dealId || savingField) {
      return;
    }
    setSavingField(true);
    setError(null);
    try {
      const item = await updateBitrixDealFields(dealId, { [fieldKey]: draftValue });
      setDetail(item);
      setEditingKey(null);
      setDraftValue("");
      onUpdated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось сохранить поле");
    } finally {
      setSavingField(false);
    }
  };

  const dealFields = (detail?.fields ?? []).filter(
    (field) => !HIDDEN_DEAL_FIELD_KEYS.has(field.key.toUpperCase())
  );
  const visibleFields =
    fieldsExpanded || dealFields.length <= VISIBLE_FIELDS_COLLAPSED
      ? dealFields
      : dealFields.slice(0, VISIBLE_FIELDS_COLLAPSED);
  const hiddenCount = Math.max(0, dealFields.length - VISIBLE_FIELDS_COLLAPSED);
  const stageOptions = detail?.stageOptions ?? [];

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose}>
      <View style={styles.root}>
        <View style={styles.header}>
          <Pressable onPress={onClose} hitSlop={12}>
            <Ionicons name="close" size={28} color="#fff" />
          </Pressable>
          <Text style={styles.headerTitle}>Сделка Bitrix24</Text>
          <Pressable onPress={() => void load()} hitSlop={12}>
            <Ionicons name="refresh" size={24} color="#fff" />
          </Pressable>
        </View>

        {loading ? (
          <View style={styles.center}>
            <ActivityIndicator color={HEADER_BLUE} size="large" />
          </View>
        ) : (
          <ScrollView contentContainerStyle={styles.body}>
            {error ? <Text style={styles.error}>{error}</Text> : null}
            {detail ? (
              <>
                <Text style={styles.title}>{detail.title || `Сделка #${detail.id}`}</Text>
                <View style={styles.stageBadge}>
                  <Text style={styles.stageBadgeText}>{detail.stageLabel || detail.stageId || "—"}</Text>
                </View>

                <View style={styles.infoCard}>
                  <Text style={styles.sectionTitle}>Сменить этап</Text>
                  {updating ? <ActivityIndicator color={HEADER_BLUE} style={{ marginBottom: 12 }} /> : null}
                  {stageOptions.length > 0 ? (
                    stageOptions.map((stage) => {
                      const active = dealStageIdsEqual(detail.stageId ?? "", stage.id);
                      return (
                        <Pressable
                          key={stage.id}
                          disabled={updating || active}
                          onPress={() => void changeStage(stage.id)}
                          style={[styles.stageOption, active && styles.stageOptionActive]}
                        >
                          <Text style={[styles.stageOptionText, active && styles.stageOptionTextActive]}>
                            {stage.label}
                          </Text>
                          {active ? <Text style={styles.stageCurrentMark}>текущий</Text> : null}
                        </Pressable>
                      );
                    })
                  ) : (
                    <Text style={styles.sectionHint}>
                      Этапы недоступны. Нажмите ↻ для обновления или проверьте права CRM в Bitrix24.
                    </Text>
                  )}
                </View>

                {(detail.assignedBy?.name ||
                  detail.createdBy?.name ||
                  detail.modifiedBy?.name) && (
                  <View style={styles.infoCard}>
                    <Text style={styles.sectionTitle}>Участники</Text>
                    {detail.assignedBy?.name ? (
                      <View style={styles.infoRow}>
                        <Text style={styles.infoLabel}>Ответственный</Text>
                        <Text style={styles.infoValue}>{formatBitrixPerson(detail.assignedBy)}</Text>
                      </View>
                    ) : null}
                    {detail.createdBy?.name ? (
                      <View style={styles.infoRow}>
                        <Text style={styles.infoLabel}>Кем создана</Text>
                        <Text style={styles.infoValue}>{formatBitrixPerson(detail.createdBy)}</Text>
                      </View>
                    ) : null}
                    {detail.modifiedBy?.name ? (
                      <View style={styles.infoRow}>
                        <Text style={styles.infoLabel}>Кем изменена</Text>
                        <Text style={styles.infoValue}>{formatBitrixPerson(detail.modifiedBy)}</Text>
                      </View>
                    ) : null}
                  </View>
                )}

                {dealFields.length > 0 ? (
                  <View style={styles.infoCard}>
                    <Text style={styles.sectionTitle}>Поля сделки</Text>
                    <Text style={styles.sectionHint}>
                      Первые {VISIBLE_FIELDS_COLLAPSED} полей; нажмите карандаш для редактирования.
                    </Text>
                    {visibleFields.map((field) => (
                      <DealFieldRow
                        key={field.key}
                        field={field}
                        editing={editingKey === field.key}
                        draftValue={draftValue}
                        saving={savingField}
                        onStartEdit={() => startEditField(field)}
                        onDraftChange={setDraftValue}
                        onSave={() => void saveField(field.key)}
                        onCancel={() => {
                          setEditingKey(null);
                          setDraftValue("");
                        }}
                      />
                    ))}
                    {hiddenCount > 0 ? (
                      <Pressable
                        onPress={() => setFieldsExpanded((prev) => !prev)}
                        style={styles.expandBtn}
                      >
                        <Text style={styles.expandBtnText}>
                          {fieldsExpanded ? "Свернуть" : `Показать ещё ${hiddenCount}`}
                        </Text>
                        <Ionicons
                          name={fieldsExpanded ? "chevron-up" : "chevron-down"}
                          size={18}
                          color={HEADER_BLUE}
                        />
                      </Pressable>
                    ) : null}
                  </View>
                ) : (
                  <View style={styles.infoCard}>
                    <Text style={styles.sectionTitle}>Поля сделки</Text>
                    <View style={styles.infoRow}>
                      <Text style={styles.infoLabel}>ID</Text>
                      <Text style={styles.infoValue}>{detail.id}</Text>
                    </View>
                    <View style={styles.infoRow}>
                      <Text style={styles.infoLabel}>Сумма</Text>
                      <Text style={styles.infoValue}>
                        {detail.opportunity
                          ? `${detail.opportunity} ${detail.currencyId ?? ""}`.trim()
                          : "—"}
                      </Text>
                    </View>
                    {detail.comments ? (
                      <View style={styles.infoRow}>
                        <Text style={styles.infoLabel}>Комментарий</Text>
                        <Text style={styles.infoValue}>{detail.comments}</Text>
                      </View>
                    ) : null}
                  </View>
                )}
              </>
            ) : null}
          </ScrollView>
        )}
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
    backgroundColor: HEADER_BLUE,
    flexDirection: "row",
    justifyContent: "space-between",
    paddingBottom: 14,
    paddingHorizontal: 16,
    paddingTop: 52
  },
  headerTitle: {
    color: "#fff",
    fontSize: 17,
    fontWeight: "700"
  },
  center: {
    alignItems: "center",
    flex: 1,
    justifyContent: "center"
  },
  body: {
    gap: 12,
    padding: 16,
    paddingBottom: 32
  },
  title: {
    color: "#0f172a",
    fontSize: 20,
    fontWeight: "700"
  },
  stageBadge: {
    alignSelf: "flex-start",
    backgroundColor: "#dbeafe",
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 6
  },
  stageBadgeText: {
    color: "#1d4ed8",
    fontSize: 13,
    fontWeight: "600"
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
    gap: 4
  },
  fieldHeader: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between"
  },
  infoLabel: {
    color: "#64748b",
    flex: 1,
    fontSize: 12,
    fontWeight: "600",
    textTransform: "uppercase"
  },
  infoValue: {
    color: "#0f172a",
    fontSize: 15,
    lineHeight: 21
  },
  sectionTitle: {
    color: "#0f172a",
    fontSize: 16,
    fontWeight: "700"
  },
  sectionHint: {
    color: "#64748b",
    fontSize: 13,
    lineHeight: 18,
    marginBottom: 4
  },
  stageOption: {
    alignItems: "center",
    backgroundColor: "#f8fafc",
    borderColor: "#cbd5e1",
    borderRadius: 12,
    borderWidth: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    paddingHorizontal: 14,
    paddingVertical: 12
  },
  stageOptionActive: {
    backgroundColor: "#eff6ff",
    borderColor: HEADER_BLUE
  },
  stageOptionText: {
    color: "#0f172a",
    flex: 1,
    fontSize: 15
  },
  stageOptionTextActive: {
    color: HEADER_BLUE,
    fontWeight: "700"
  },
  stageCurrentMark: {
    color: "#64748b",
    fontSize: 12
  },
  expandBtn: {
    alignItems: "center",
    flexDirection: "row",
    gap: 4,
    justifyContent: "center",
    marginTop: 4,
    paddingVertical: 8
  },
  expandBtnText: {
    color: HEADER_BLUE,
    fontSize: 14,
    fontWeight: "600"
  },
  editBlock: {
    gap: 8
  },
  fieldInput: {
    backgroundColor: "#f8fafc",
    borderColor: "#cbd5e1",
    borderRadius: 8,
    borderWidth: 1,
    color: "#0f172a",
    fontSize: 15,
    minHeight: 40,
    paddingHorizontal: 10,
    paddingVertical: 8
  },
  optionRow: {
    gap: 8,
    paddingVertical: 4
  },
  optionChip: {
    backgroundColor: "#f1f5f9",
    borderColor: "#cbd5e1",
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 8
  },
  optionChipActive: {
    backgroundColor: "#eff6ff",
    borderColor: HEADER_BLUE
  },
  optionChipText: {
    color: "#334155",
    fontSize: 14
  },
  optionChipTextActive: {
    color: HEADER_BLUE,
    fontWeight: "600"
  },
  editActions: {
    flexDirection: "row",
    gap: 8,
    justifyContent: "flex-end"
  },
  editBtnSecondary: {
    borderColor: "#cbd5e1",
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 8
  },
  editBtnSecondaryText: {
    color: "#475569",
    fontSize: 14,
    fontWeight: "600"
  },
  editBtnPrimary: {
    backgroundColor: HEADER_BLUE,
    borderRadius: 8,
    minWidth: 96,
    paddingHorizontal: 14,
    paddingVertical: 8
  },
  editBtnPrimaryText: {
    color: "#fff",
    fontSize: 14,
    fontWeight: "600",
    textAlign: "center"
  },
  error: {
    color: "#b91c1c",
    marginBottom: 12
  }
});
