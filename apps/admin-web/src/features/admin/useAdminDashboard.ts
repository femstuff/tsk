import { useCallback, useEffect, useState } from "react";

import type { AdminDashboard } from "../../entities/admin/types";
import { getAdminDashboard } from "../../shared/api/client";

const pollIntervalMs = 15000;
const pollInit: RequestInit = {
  headers: { "X-TSK-Request-Source": "admin-poll" }
};

export function useAdminDashboard() {
  const [dashboard, setDashboard] = useState<AdminDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    setError(null);
    try {
      const item = await getAdminDashboard(pollInit);
      setDashboard(item);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить дашборд");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void refresh(true);
      }
    }, pollIntervalMs);
    return () => window.clearInterval(id);
  }, [refresh]);

  return { dashboard, loading, error, refresh };
}
