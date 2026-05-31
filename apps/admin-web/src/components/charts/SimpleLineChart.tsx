import { useMemo, useState } from "react";

import type { DayCount } from "../../lib/dashboardAnalytics";

type SimpleLineChartProps = {
  data: DayCount[];
  valueFormatter?: (value: number) => string;
  unit?: string;
  maxXTicks?: number;
};

export function SimpleLineChart({
  data,
  valueFormatter,
  unit = "",
  maxXTicks = 6
}: SimpleLineChartProps) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const width = 560;
  const height = 220;
  const padding = { top: 16, right: 16, bottom: 36, left: 36 };
  const innerWidth = width - padding.left - padding.right;
  const innerHeight = height - padding.top - padding.bottom;
  const maxValue = Math.max(0.001, ...data.map((point) => point.value));
  const labelEvery = Math.max(1, Math.ceil(data.length / Math.max(1, maxXTicks)));

  const points = data.map((point, index) => {
    const x =
      padding.left + (data.length <= 1 ? innerWidth / 2 : (index / (data.length - 1)) * innerWidth);
    const y = padding.top + innerHeight - (point.value / maxValue) * innerHeight;
    return { ...point, x, y, index };
  });

  const polyline = points.map((point) => `${point.x},${point.y}`).join(" ");
  const activePoint = activeIndex == null ? null : points[activeIndex];

  const formatValue = useMemo(() => {
    if (valueFormatter) {
      return valueFormatter;
    }
    return (value: number) => `${value.toFixed(2)}${unit}`;
  }, [unit, valueFormatter]);

  if (data.length === 0) {
    return <p className="activity-empty">Нет данных для графика.</p>;
  }

  return (
    <div className="chart-shell">
      {activePoint ? (
        <div className="chart-tooltip">
          <strong>{activePoint.label}</strong>
          <span>{formatValue(activePoint.value)}</span>
        </div>
      ) : null}

      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="chart-svg"
        role="img"
        aria-label="График метрик"
        onMouseLeave={() => setActiveIndex(null)}
      >
        {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
          const y = padding.top + innerHeight * (1 - ratio);
          const value = maxValue * ratio;
          return (
            <g key={ratio}>
              <line
                x1={padding.left}
                y1={y}
                x2={width - padding.right}
                y2={y}
                className="chart-grid-line"
              />
              <text x={8} y={y + 4} className="chart-axis-label">
                {formatValue(value)}
              </text>
            </g>
          );
        })}

        <polyline points={polyline} className="chart-line" fill="none" />
        {points.map((point) => {
          const showLabel =
            point.index % labelEvery === 0 || point.index === data.length - 1 || data.length <= maxXTicks;
          return (
            <g key={`${point.label}-${point.index}`}>
              <circle
                cx={point.x}
                cy={point.y}
                r={activeIndex === point.index ? 6 : 4.5}
                className={`chart-point${activeIndex === point.index ? " active" : ""}`}
                onMouseEnter={() => setActiveIndex(point.index)}
                onFocus={() => setActiveIndex(point.index)}
                tabIndex={0}
              />
              {showLabel ? (
                <text x={point.x} y={height - 10} textAnchor="middle" className="chart-axis-label chart-x-label">
                  {point.label}
                </text>
              ) : null}
            </g>
          );
        })}
      </svg>
    </div>
  );
}
