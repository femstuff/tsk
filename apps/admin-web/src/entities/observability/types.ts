export type ObservabilityPoint = {
  label: string;
  value: number;
};

export type ObservabilityDashboard = {
  available: boolean;
  rps?: number;
  errorRatePercent?: number;
  latencyP95Seconds?: number;
  uptimeSeconds?: number;
  memoryBytes?: number;
  cpuCores?: number;
  httpRateSeries: ObservabilityPoint[];
  jobStatusSeries: ObservabilityPoint[];
  queriedAt: string;
};
