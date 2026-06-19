import React from 'react';

export interface PageHeaderProps {
  /** Main title — usually a short noun phrase. */
  title: React.ReactNode;
  /** Optional one-line description shown beneath the title. */
  subtitle?: React.ReactNode;
  /** Optional leading icon rendered inside a gradient tile. */
  icon?: React.ReactNode;
  /** CSS gradient for the icon tile (defaults to the brand gradient). */
  gradient?: string;
  /** Right-aligned content such as action buttons or filters. */
  extra?: React.ReactNode;
}

/**
 * Consistent page header used across list/detail pages. Provides a gradient
 * icon tile, a title + subtitle stack, and a slot for right-aligned actions.
 */
const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  subtitle,
  icon,
  gradient = 'var(--gradient-brand)',
  extra,
}) => (
  <div className="page-header">
    <div className="page-header-left">
      {icon && (
        <div className="page-header-icon" style={{ background: gradient }}>
          {icon}
        </div>
      )}
      <div className="page-header-text">
        <h1 className="page-header-title">{title}</h1>
        {subtitle && <div className="page-header-subtitle">{subtitle}</div>}
      </div>
    </div>
    {extra && <div className="page-header-extra">{extra}</div>}
  </div>
);

export default PageHeader;
