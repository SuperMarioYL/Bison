import React from 'react';
import { Empty } from 'antd';

export interface DonutDatum {
  name: string;
  value: number;
  color: string;
}

export interface DonutChartProps {
  data: DonutDatum[];
  size?: number;
  /** Label shown under the total in the ring center. */
  centerLabel?: string;
}

/**
 * Lightweight SVG donut chart with a center total and a legend. Pure SVG/CSS
 * (no charting dependency) so it stays out of the heavy echarts bundle while
 * still looking production-grade. Theme-aware via CSS variables.
 */
const DonutChart: React.FC<DonutChartProps> = ({ data, size = 168, centerLabel }) => {
  const total = data.reduce((sum, d) => sum + d.value, 0);
  if (total === 0) {
    return <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  const stroke = 18;
  const radius = (size - stroke) / 2;
  const circ = 2 * Math.PI * radius;

  let runningOffset = 0;
  const segments = data
    .filter(d => d.value > 0)
    .map(d => {
      const frac = d.value / total;
      const seg = { ...d, frac, dash: frac * circ, offset: runningOffset };
      runningOffset += frac * circ;
      return seg;
    });

  return (
    <div className="donut-chart">
      <div className="donut-ring" style={{ width: size, height: size }}>
        <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
          <g transform={`rotate(-90 ${size / 2} ${size / 2})`}>
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke="var(--border-light)"
              strokeWidth={stroke}
            />
            {segments.map((s, i) => (
              <circle
                key={s.name + i}
                cx={size / 2}
                cy={size / 2}
                r={radius}
                fill="none"
                stroke={s.color}
                strokeWidth={stroke}
                strokeDasharray={`${s.dash} ${circ - s.dash}`}
                strokeDashoffset={-s.offset}
              />
            ))}
          </g>
        </svg>
        <div className="donut-center">
          <div className="donut-total">{total}</div>
          {centerLabel && <div className="donut-label">{centerLabel}</div>}
        </div>
      </div>
      <div className="donut-legend">
        {segments.map(s => (
          <div className="donut-legend-item" key={s.name}>
            <span className="donut-dot" style={{ background: s.color }} />
            <span className="donut-name">{s.name}</span>
            <span className="donut-val">{s.value}</span>
            <span className="donut-pct">{(s.frac * 100).toFixed(0)}%</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DonutChart;
