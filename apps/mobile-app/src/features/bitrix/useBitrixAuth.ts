import * as WebBrowser from "expo-web-browser";
import { useCallback, useEffect, useState } from "react";
import { Alert } from "react-native";

import type { BitrixOAuthSessionView } from "../../entities/document-template/types";
import {
  disconnectBitrixOAuth,
  fetchBitrixOAuthSession,
  startBitrixOAuth
} from "../../shared/api/client";
import { clearBitrixSessionId, getBitrixSessionId, setBitrixSessionId } from "./sessionStorage";

WebBrowser.maybeCompleteAuthSession();

export function useBitrixAuth() {
  const [session, setSession] = useState<BitrixOAuthSessionView | null>(null);
  const [loading, setLoading] = useState(true);
  const [connecting, setConnecting] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const sessionId = await getBitrixSessionId();
      if (!sessionId) {
        setSession(null);
        return;
      }
      const view = await fetchBitrixOAuthSession(sessionId);
      if (view.connected) {
        setSession(view);
      } else {
        await clearBitrixSessionId();
        setSession(null);
      }
    } catch {
      setSession(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const connect = useCallback(async () => {
    setConnecting(true);
    try {
      const start = await startBitrixOAuth();
      await setBitrixSessionId(start.sessionId);

      const result = await WebBrowser.openAuthSessionAsync(
        start.authorizeUrl,
        "tsk://bitrix-oauth"
      );

      if (result.type !== "success" || !result.url) {
        await clearBitrixSessionId();
        setSession(null);
        if (result.type === "cancel") {
          Alert.alert("Bitrix24", "Вход отменён.");
        }
        return;
      }

      const parsedSessionId = (() => {
        try {
          return new URL(result.url).searchParams.get("sessionId");
        } catch {
          const match = result.url.match(/[?&]sessionId=([^&]+)/);
          return match ? decodeURIComponent(match[1]) : null;
        }
      })();
      const sessionId = parsedSessionId?.trim() || start.sessionId;
      await setBitrixSessionId(sessionId);

      const view = await fetchBitrixOAuthSession(sessionId);
      if (!view.connected) {
        throw new Error("Сессия Bitrix не активирована");
      }
      setSession(view);
      Alert.alert("Bitrix24", view.userName ? `Вы вошли как ${view.userName}` : "Вход выполнен.");
    } catch (err) {
      await clearBitrixSessionId();
      setSession(null);
      Alert.alert(
        "Bitrix24",
        err instanceof Error ? err.message : "Не удалось войти в Bitrix24"
      );
    } finally {
      setConnecting(false);
    }
  }, []);

  const disconnect = useCallback(async () => {
    const sessionId = await getBitrixSessionId();
    if (sessionId) {
      try {
        await disconnectBitrixOAuth(sessionId);
      } catch {
        // Локально всё равно разлогиниваем.
      }
    }
    await clearBitrixSessionId();
    setSession(null);
  }, []);

  return {
    session,
    loading,
    connecting,
    connected: Boolean(session?.connected),
    refresh,
    connect,
    disconnect
  };
}
