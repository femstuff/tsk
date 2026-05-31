import { Ionicons } from "@expo/vector-icons";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Keyboard,
  Linking,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

import type { BitrixTaskComment, BitrixTaskDetail, BitrixTaskFile } from "../../entities/document-template/types";
import { addBitrixTaskComment, getBitrixTask } from "../../shared/api/client";
import {
  formatBitrixAuthor,
  formatBitrixDate,
  resolveBitrixFileUrl,
  stripBitrixDescription
} from "./bitrixTaskUi";

const HEADER_BLUE = "#2563eb";

type ChatPanelProps = {
  taskId: string | null;
  taskTitle?: string;
  portalDomain?: string;
  onBack: () => void;
  onUpdated: () => void;
};

function CommentItem({
  comment,
  portalDomain,
  onOpenFile
}: {
  comment: BitrixTaskComment;
  portalDomain?: string;
  onOpenFile: (file: BitrixTaskFile) => void;
}) {
  return (
    <View style={styles.commentBubble}>
      <View style={styles.commentHeader}>
        <Text style={styles.commentAuthor}>{formatBitrixAuthor(comment.authorName, comment.authorId)}</Text>
        <Text style={styles.commentDate}>{formatBitrixDate(comment.postDate)}</Text>
      </View>
      <Text style={styles.commentMessage}>{stripBitrixDescription(comment.message) || "—"}</Text>
      {(comment.files?.length ?? 0) > 0 ? (
        <View style={styles.commentFiles}>
          {comment.files?.map((file, index) => (
            <Pressable
              key={file.id ?? `${file.name}-${index}`}
              style={styles.commentFileChip}
              onPress={() => onOpenFile(file)}
            >
              <Ionicons name="attach" size={14} color={HEADER_BLUE} />
              <Text style={styles.commentFileText}>{file.name || "Файл"}</Text>
            </Pressable>
          ))}
        </View>
      ) : null}
    </View>
  );
}

/** Экран чата без Modal — для вложения в карточку задачи. */
export function BitrixTaskChatPanel({
  taskId,
  taskTitle,
  portalDomain,
  onBack,
  onUpdated
}: ChatPanelProps) {
  const listRef = useRef<FlatList<BitrixTaskComment>>(null);
  const [detail, setDetail] = useState<BitrixTaskDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [sendingComment, setSendingComment] = useState(false);
  const [commentText, setCommentText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [composerPadding, setComposerPadding] = useState(Platform.OS === "ios" ? 28 : 16);

  const scrollToEnd = useCallback(() => {
    requestAnimationFrame(() => {
      listRef.current?.scrollToEnd({ animated: true });
    });
  }, []);

  const load = useCallback(async () => {
    if (!taskId) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const item = await getBitrixTask(taskId);
      setDetail(item);
      scrollToEnd();
    } catch (err) {
      setDetail(null);
      setError(err instanceof Error ? err.message : "Не удалось загрузить чат");
    } finally {
      setLoading(false);
    }
  }, [scrollToEnd, taskId]);

  useEffect(() => {
    if (taskId) {
      void load();
    }
    return () => {
      setDetail(null);
      setError(null);
      setCommentText("");
    };
  }, [load, taskId]);

  useEffect(() => {
    const showEvent = Platform.OS === "ios" ? "keyboardWillShow" : "keyboardDidShow";
    const hideEvent = Platform.OS === "ios" ? "keyboardWillHide" : "keyboardDidHide";
    const showSub = Keyboard.addListener(showEvent, (event) => {
      setComposerPadding(Math.max(12, event.endCoordinates.height));
      scrollToEnd();
    });
    const hideSub = Keyboard.addListener(hideEvent, () => {
      setComposerPadding(Platform.OS === "ios" ? 28 : 16);
    });
    return () => {
      showSub.remove();
      hideSub.remove();
    };
  }, [scrollToEnd]);

  const openFile = async (file: BitrixTaskFile) => {
    const resolved = resolveBitrixFileUrl(file.downloadUrl || file.viewUrl, portalDomain, file.id);
    if (!resolved) {
      Alert.alert("Файл", "Не удалось получить ссылку на файл.");
      return;
    }
    try {
      const canOpen = await Linking.canOpenURL(resolved);
      if (!canOpen) {
        Alert.alert("Файл", "Не удалось открыть ссылку на этом устройстве.");
        return;
      }
      await Linking.openURL(resolved);
    } catch {
      Alert.alert("Файл", "Не удалось открыть файл. Возможно, нужен вход в Bitrix24 в браузере.");
    }
  };

  const handleSendComment = async () => {
    const message = commentText.trim();
    if (!taskId || !message || sendingComment) {
      return;
    }
    setSendingComment(true);
    setError(null);
    try {
      const item = await addBitrixTaskComment(taskId, message);
      setDetail(item);
      setCommentText("");
      onUpdated();
      scrollToEnd();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить сообщение");
    } finally {
      setSendingComment(false);
    }
  };

  const comments = detail?.comments ?? [];
  const title = detail?.title?.trim() || taskTitle?.trim() || "Чат задачи";

  return (
    <View style={styles.root}>
      <View style={styles.header}>
        <Pressable onPress={onBack} hitSlop={12} style={styles.headerBtn}>
          <Ionicons name="chevron-back" size={26} color="#0f172a" />
        </Pressable>
        <View style={styles.headerCenter}>
          <Text style={styles.headerTitle} numberOfLines={1}>
            {title}
          </Text>
          <Text style={styles.headerSubtitle}>{comments.length} сообщений</Text>
        </View>
        <Pressable onPress={() => void load()} hitSlop={12} style={styles.headerBtn}>
          <Ionicons name="refresh" size={22} color={HEADER_BLUE} />
        </Pressable>
      </View>

      {loading ? (
        <View style={styles.center}>
          <ActivityIndicator color={HEADER_BLUE} size="large" />
        </View>
      ) : (
        <FlatList
          ref={listRef}
          data={comments}
          keyExtractor={(item) => item.id}
          style={styles.list}
          contentContainerStyle={[styles.listContent, comments.length === 0 ? styles.listContentEmpty : null]}
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode={Platform.OS === "ios" ? "interactive" : "on-drag"}
          onContentSizeChange={scrollToEnd}
          ListEmptyComponent={<Text style={styles.emptyText}>Сообщений пока нет — напишите первым.</Text>}
          renderItem={({ item }) => (
            <CommentItem comment={item} portalDomain={portalDomain} onOpenFile={(file) => void openFile(file)} />
          )}
        />
      )}

      {error ? <Text style={styles.error}>{error}</Text> : null}

      <View style={[styles.composer, { paddingBottom: composerPadding }]}>
        <TextInput
          placeholder="Сообщение…"
          placeholderTextColor="#94a3b8"
          style={styles.composerInput}
          value={commentText}
          onChangeText={setCommentText}
          multiline
          maxLength={4000}
          onFocus={scrollToEnd}
        />
        <Pressable
          style={[styles.composerSend, (!commentText.trim() || sendingComment) && styles.composerSendDisabled]}
          disabled={!commentText.trim() || sendingComment}
          onPress={() => void handleSendComment()}
        >
          {sendingComment ? (
            <ActivityIndicator color="#fff" size="small" />
          ) : (
            <Ionicons name="send" size={20} color="#fff" />
          )}
        </Pressable>
      </View>
    </View>
  );
}

type ModalProps = ChatPanelProps & {
  visible: boolean;
  onClose: () => void;
};

/** Отдельный Modal-обёртка (если нужен снаружи карточки задачи). */
export function BitrixTaskChatModal({ visible, onClose, ...panelProps }: ModalProps) {
  return (
    <Modal visible={visible} animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <BitrixTaskChatPanel {...panelProps} onBack={onClose} />
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
    gap: 8,
    paddingHorizontal: 8,
    paddingVertical: 12
  },
  headerBtn: {
    alignItems: "center",
    justifyContent: "center",
    width: 40
  },
  headerCenter: {
    flex: 1,
    minWidth: 0
  },
  headerTitle: {
    color: "#0f172a",
    fontSize: 16,
    fontWeight: "700"
  },
  headerSubtitle: {
    color: "#64748b",
    fontSize: 12,
    marginTop: 2
  },
  center: {
    alignItems: "center",
    flex: 1,
    justifyContent: "center"
  },
  list: {
    flex: 1
  },
  listContent: {
    gap: 10,
    padding: 16,
    paddingBottom: 8
  },
  listContentEmpty: {
    flexGrow: 1,
    justifyContent: "center"
  },
  emptyText: {
    color: "#64748b",
    fontSize: 15,
    textAlign: "center"
  },
  commentBubble: {
    backgroundColor: "#fff",
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
  composer: {
    alignItems: "flex-end",
    backgroundColor: "#fff",
    borderTopColor: "#e2e8f0",
    borderTopWidth: 1,
    flexDirection: "row",
    gap: 10,
    paddingHorizontal: 12,
    paddingTop: 10
  },
  composerInput: {
    backgroundColor: "#f8fafc",
    borderColor: "#cbd5e1",
    borderRadius: 14,
    borderWidth: 1,
    color: "#0f172a",
    flex: 1,
    fontSize: 15,
    maxHeight: 120,
    minHeight: 44,
    paddingHorizontal: 12,
    paddingVertical: 10
  },
  composerSend: {
    alignItems: "center",
    backgroundColor: HEADER_BLUE,
    borderRadius: 22,
    height: 44,
    justifyContent: "center",
    width: 44
  },
  composerSendDisabled: {
    opacity: 0.45
  },
  error: {
    color: "#b91c1c",
    fontSize: 13,
    paddingHorizontal: 16,
    paddingVertical: 6
  }
});
