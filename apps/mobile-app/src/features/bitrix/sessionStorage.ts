import * as SecureStore from "expo-secure-store";

const BITRIX_SESSION_KEY = "tsk.bitrix.sessionId";

export async function getBitrixSessionId(): Promise<string | null> {
  try {
    const value = await SecureStore.getItemAsync(BITRIX_SESSION_KEY);
    return value?.trim() || null;
  } catch {
    return null;
  }
}

export async function setBitrixSessionId(sessionId: string | null): Promise<void> {
  if (!sessionId?.trim()) {
    await SecureStore.deleteItemAsync(BITRIX_SESSION_KEY);
    return;
  }
  await SecureStore.setItemAsync(BITRIX_SESSION_KEY, sessionId.trim());
}

export async function clearBitrixSessionId(): Promise<void> {
  await SecureStore.deleteItemAsync(BITRIX_SESSION_KEY);
}
