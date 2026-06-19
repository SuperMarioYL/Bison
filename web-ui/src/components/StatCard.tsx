import React from 'react';
import { Spin } from 'antd';

export interface StatCardProps {
  /** Metric label, e.g. "集群节点". */
  title: string;
  /** Primary value — number or short string. */
  value: React.ReactNode;
  /** Icon rendered inside the gradient tile. */
  icon: React.ReactNode;
  /** CSS gradient for the icon tile (defaults to blue). */
  gradient?: string;
  /** Optional unit/suffix shown after the value. */
  suffix?: React.ReactNode;
  /** Optional small helper text under the value. */
  hint?: React.ReactNode;
  /** Custom color for the value text (e.g. status colors). */
  valueColor?: string;
  onClick?: () => void;
  loading?: boolean;
}

/**
 * KPI card: a gradient icon tile beside a label + large value. Used on the
 * dashboard summary row. Renders as a plain element (not Ant Card) so the
 * gradient wash and hover lift are fully controllable.
 */
const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  icon,
  gradient = 'var(--gradient-blue)',
  suffix,
  hint,
  valueColor,
  onClick,
  loading,
}) => (
  <div
    className={`stat-card${onClick ? ' stat-card-clickable' : ''}`}
    onClick={onClick}
    style={{ ['--stat-gradient' as string]: gradient } as React.CSSProperties}
  >
    <div className="stat-card-icon" style={{ background: gradient }}>
      {icon}
    </div>
    <div className="stat-card-body">
      <div className="stat-card-title">{title}</div>
      {loading ? (
        <Spin size="small" />
      ) : (
        <div className="stat-card-value" style={valueColor ? { color: valueColor } : undefined}>
          {value}
          {suffix && <span className="stat-card-suffix">{suffix}</span>}
        </div>
      )}
      {hint && <div className="stat-card-hint">{hint}</div>}
    </div>
  </div>
);

export default StatCard;
