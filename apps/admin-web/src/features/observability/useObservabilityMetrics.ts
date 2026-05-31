import { useCallback, useEffect, useState } from "react";

import type { ObservabilityDashboard } from "../../entities/observability/types";
import { getObservabilityDashboard } from "../../shared/api/client";

const pollIntervalMs = 15_000;

export function useObservabilityMetrics() {
  const [snapshot, setSnapshot] = useState<ObservabilityDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    setError(null);
    try {
      const next = await getObservabilityDashboard();
      setSnapshot(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить метрики Prometheus");
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void refresh();

    const intervalId = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void refresh(true);
      }
    }, pollIntervalMs);

    return () => window.clearInterval(intervalId);
  }, [refresh]);

  return { snapshot, loading, error, refresh };
}
