import { useMemo, useState } from "react";

import type { DonutSlice } from "../../lib/dashboardAnalytics";

type DonutChartProps = {
  slices: DonutSlice[];
  selectedFilterKey?: string | null;
  onSliceSelect?: (slice: DonutSlice) => void;
};

export function DonutChart({ slices, selectedFilterKey = null, onSliceSelect }: DonutChartProps) {
  const [hoverLabel, setHoverLabel] = useState<string | null>(null);
  const total = slices.reduce((sum, slice) => sum + slice.value, 0);
  const radius = 72;
  const stroke = 24;
  const center = 96;

  const activeLabel =
    hoverLabel ??
    (selectedFilterKey ? (slices.find((slice) => slice.filterKey === selectedFilterKey)?.label ?? null) : null);

  const arcs = useMemo(() => {
    let currentOffset = 0;
    return slices.map((slice) => {
      const fraction = total > 0 ? slice.value / total : 0;
      const dash = fraction * 2 * Math.PI * radius;
      const gap = 2 * Math.PI * radius - dash;
      const arc = {
        ...slice,
        dash,
        gap,
        offset: currentOffset,
        percent: Math.round(fraction * 100)
      };
      currentOffset += dash;
      return arc;
    });
  }, [radius, slices, total]);

  const handleSelect = (slice: DonutSlice) => {
    if (!slice.filterKey || !onSliceSelect) {
      return;
    }
    onSliceSelect(slice);
  };

  return (
    <div className="donut-layout">
      <svg viewBox="0 0 192 192" className="donut-svg" role="img" aria-label="Статусы">
        <g transform={`rotate(-90 ${center} ${center})`}>
          {arcs.map((arc) => {
            const selected = selectedFilterKey != null && arc.filterKey === selectedFilterKey;
            const dimmed = selectedFilterKey != null && !selected;
            return (
              <circle
                key={arc.label}
                cx={center}
                cy={center}
                r={radius}
                fill="none"
                stroke={arc.color}
                strokeWidth={activeLabel && activeLabel !== arc.label ? stroke - 6 : stroke}
                strokeDasharray={`${arc.dash} ${arc.gap}`}
                strokeDashoffset={-arc.offset}
                opacity={dimmed ? 0.35 : 1}
                className={`donut-arc${selected || activeLabel === arc.label ? " active" : ""}`}
                onMouseEnter={() => setHoverLabel(arc.label)}
                onMouseLeave={() => setHoverLabel(null)}
                onClick={() => handleSelect(arc)}
              />
            );
          })}
        </g>
        <text x={center} y={center - 4} textAnchor="middle" className="donut-total">
          {activeLabel ? (arcs.find((arc) => arc.label === activeLabel)?.value ?? total) : total}
        </text>
        <text x={center} y={center + 16} textAnchor="middle" className="donut-caption">
          {activeLabel ?? "всего"}
        </text>
      </svg>

      <ul className="donut-legend">
        {arcs.map((arc) => {
          const selected = selectedFilterKey != null && arc.filterKey === selectedFilterKey;
          return (
            <li key={arc.label}>
              <button
                type="button"
                className={`donut-legend-button${selected || activeLabel === arc.label ? " active" : ""}`}
                onMouseEnter={() => setHoverLabel(arc.label)}
                onMouseLeave={() => setHoverLabel(null)}
                onFocus={() => setHoverLabel(arc.label)}
                onBlur={() => setHoverLabel(null)}
                onClick={() => handleSelect(arc)}
              >
                <span className="donut-dot" style={{ background: arc.color }} />
                <span>
                  {arc.label}: {arc.percent}%
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
