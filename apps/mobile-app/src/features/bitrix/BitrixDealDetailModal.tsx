import { Ionicons } from "@expo/vector-icons";
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View
} from "react-native";

import type { BitrixDealDetail } from "../../entities/document-template/types";
import { getBitrixDeal, updateBitrixDealStage } from "../../shared/api/client";
import { formatBitrixDate, formatBitrixPerson } from "./bitrixTaskUi";

const HEADER_BLUE = "#2563eb";

type Props = {
  dealId: string | null;
  visible: boolean;
  onClose: () => void;
  onUpdated: () => void;
};

function InfoRow({ label, value }: { label: string; value: string }) {
  const text = value?.trim() || "—";
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{label}</Text>
      <Text style={styles.infoValue}>{text}</Text>
    </View>
  );
}

export function BitrixDealDetailModal({ dealId, visible, onClose, onUpdated }: Props) {
  const [detail, setDetail] = useState<BitrixDealDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose}>
      <View style={styles.root}>
        <View style={styles.header}>
          <Pressable onPress={onClose} hitSlop={12}>
            <Ionicons name="close" size={28} color="#fff" />
          </Pressable>
          <Text style={styles.headerTitle}>Сделка Bitrix24</Text>
          <View style={{ width: 28 }} />
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

                <InfoRow label="ID" value={detail.id} />
                <InfoRow
                  label="Сумма"
                  value={detail.opportunity ? `${detail.opportunity} ${detail.currencyId ?? ""}`.trim() : "—"}
                />
                <InfoRow
                  label="Ответственный"
                  value={formatBitrixPerson(
                    detail.assignedBy ??
                      (detail.assignedById ? { id: detail.assignedById } : undefined)
                  )}
                />
                <InfoRow label="Создана" value={formatBitrixDate(detail.dateCreate)} />
                <InfoRow label="Изменена" value={formatBitrixDate(detail.dateModify)} />
                <InfoRow label="Комментарий" value={detail.comments ?? "—"} />

                <Text style={styles.sectionTitle}>Сменить этап</Text>
                {updating ? <ActivityIndicator color={HEADER_BLUE} style={{ marginBottom: 12 }} /> : null}
                {(detail.stageOptions ?? []).map((stage) => {
                  const active = stage.id === detail.stageId;
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
                })}
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
    padding: 16,
    paddingBottom: 32
  },
  title: {
    color: "#0f172a",
    fontSize: 20,
    fontWeight: "700",
    marginBottom: 8
  },
  stageBadge: {
    alignSelf: "flex-start",
    backgroundColor: "#dbeafe",
    borderRadius: 999,
    marginBottom: 16,
    paddingHorizontal: 12,
    paddingVertical: 6
  },
  stageBadgeText: {
    color: "#1d4ed8",
    fontSize: 13,
    fontWeight: "600"
  },
  infoRow: {
    borderBottomColor: "#e2e8f0",
    borderBottomWidth: 1,
    marginBottom: 10,
    paddingBottom: 8
  },
  infoLabel: {
    color: "#64748b",
    fontSize: 12,
    marginBottom: 2
  },
  infoValue: {
    color: "#0f172a",
    fontSize: 15
  },
  sectionTitle: {
    color: "#0f172a",
    fontSize: 16,
    fontWeight: "700",
    marginBottom: 10,
    marginTop: 12
  },
  stageOption: {
    alignItems: "center",
    backgroundColor: "#fff",
    borderColor: "#cbd5e1",
    borderRadius: 12,
    borderWidth: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    marginBottom: 8,
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
  error: {
    color: "#b91c1c",
    marginBottom: 12
  }
});
