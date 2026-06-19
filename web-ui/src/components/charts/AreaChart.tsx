import React, { useId, useState } from 'react';
import { Empty } from 'antd';

export interface AreaPoint {
  date: string;
  value: number;
}

export interface AreaChartProps {
  data: AreaPoint[];
  height?: number;
  /** Line/fill color (defaults to the theme accent). */
  color?: string;
  /** Prefix for the tooltip value, e.g. "$". */
  valuePrefix?: string;
  /** Decimal places for the tooltip value. */
  precision?: number;
}

/** Build a smooth (Catmull-Rom → cubic bezier) path through the points. */
function smoothPath(pts: { x: number; y: number }[]): string {
  if (pts.length === 0) return '';
  if (pts.length === 1) return `M ${pts[0].x} ${pts[0].y}`;
  let d = `M ${pts[0].x} ${pts[0].y}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] || pts[i];
    const p1 = pts[i];
    const p2 = pts[i + 1];
    const p3 = pts[i + 2] || p2;
    const cp1x = p1.x + (p2.x - p0.x) / 6;
    const cp1y = p1.y + (p2.y - p0.y) / 6;
    const cp2x = p2.x - (p3.x - p1.x) / 6;
    const cp2y = p2.y - (p3.y - p1.y) / 6;
    d += ` C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`;
  }
  return d;
}

/**
 * Responsive SVG area/line chart with a gradient fill, grid lines and a
 * hover tooltip. Pure SVG/CSS — no charting dependency. The viewBox uses a
 * fixed coordinate space and `non-scaling-stroke` so the line stays crisp at
 * any width; the hover cursor/dot/tooltip are HTML overlays.
 */
const AreaChart: React.FC<AreaChartProps> = ({
  data,
  height = 200,
  color = 'var(--accent)',
  valuePrefix = '',
  precision = 2,
}) => {
  const gradientId = useId().replace(/:/g, '');
  const [hover, setHover] = useState<number | null>(null);

  if (!data || data.length === 0) {
    return <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  const W = 600;
  const padX = 10;
  const padTop = 14;
  const padBottom = 26;
  const innerH = height - padTop - padBottom;
  const n = data.length;
  const maxV = Math.max(...data.map(d => d.value), 1);

  const xFor = (i: number) =>
    n === 1 ? W / 2 : padX + (i / (n - 1)) * (W - 2 * padX);
  const yFor = (v: number) => padTop + innerH - (v / maxV) * innerH;

  const pts = data.map((d, i) => ({ x: xFor(i), y: yFor(d.value) }));
  const line = smoothPath(pts);
  const baseY = padTop + innerH;
  const area = `${line} L ${pts[n - 1].x} ${baseY} L ${pts[0].x} ${baseY} Z`;
  const gridLines = [0, 0.25, 0.5, 0.75, 1].map(f => padTop + innerH - f * innerH);

  const handleMove = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const frac = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    setHover(Math.round(frac * (n - 1)));
  };

  return (
    <div
      className="area-chart"
      style={{ height }}
      onMouseMove={handleMove}
      onMouseLeave={() => setHover(null)}
    >
      <svg
        className="area-svg"
        width="100%"
        height={height}
        viewBox={`0 0 ${W} ${height}`}
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.35" />
            <stop offset="100%" stopColor={color} stopOpacity="0.02" />
          </linearGradient>
        </defs>
        {gridLines.map((y, i) => (
          <line
            key={i}
            x1={padX}
            y1={y}
            x2={W - padX}
            y2={y}
            stroke="var(--border-light)"
            strokeWidth="1"
            vectorEffect="non-scaling-stroke"
          />
        ))}
        <path d={area} fill={`url(#${gradientId})`} />
        <path
          d={line}
          fill="none"
          stroke={color}
          strokeWidth="2.5"
          strokeLinejoin="round"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
      </svg>

      <div className="area-xlabels">
        <span>{data[0].date}</span>
        {n > 1 && <span>{data[n - 1].date}</span>}
      </div>

      {hover !== null && (
        <>
          <div
            className="area-cursor"
            style={{ left: `${(pts[hover].x / W) * 100}%`, height: baseY }}
          />
          <div
            className="area-dot"
            style={{
              left: `${(pts[hover].x / W) * 100}%`,
              top: pts[hover].y,
              background: color,
            }}
          />
          <div
            className="area-tooltip"
            style={{ left: `${(pts[hover].x / W) * 100}%`, top: pts[hover].y }}
          >
            <div className="area-tt-date">{data[hover].date}</div>
            <div className="area-tt-val">
              {valuePrefix}
              {data[hover].value.toFixed(precision)}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default AreaChart;
