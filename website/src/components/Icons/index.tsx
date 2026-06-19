import type {ReactNode, SVGProps} from 'react';

/**
 * Tabler-style line icons (24x24, stroke = currentColor, no emoji).
 * Project standard: use line icons instead of emoji for UI decoration.
 * https://tabler.io/icons — geometry kept faithful to the Tabler aesthetic.
 */

export type IconProps = SVGProps<SVGSVGElement> & {size?: number};

function Icon({
  size = 24,
  children,
  ...props
}: IconProps & {children: ReactNode}): ReactNode {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...props}>
      {children}
    </svg>
  );
}

/* ---------------- Feature icons ---------------- */

export const ShieldLockIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M12 3l7 3v5c0 4 -3 7 -7 8c-4 -1 -7 -4 -7 -8v-5l7 -3z" />
    <circle cx="12" cy="11" r="1.4" />
    <path d="M12 12.4v2.6" />
  </Icon>
);

export const CurrencyDollarIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <circle cx="12" cy="12" r="9" />
    <path d="M14.8 9.3a3.2 2.8 0 0 0 -2.8 -1.3h-1a2.4 2.4 0 0 0 0 4.8h1a2.4 2.4 0 0 1 0 4.8h-1a3.2 2.8 0 0 1 -2.8 -1.3" />
    <path d="M12 6v12" />
  </Icon>
);

export const DashboardIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <rect x="4" y="4" width="6" height="8" rx="1.2" />
    <rect x="4" y="15" width="6" height="5" rx="1.2" />
    <rect x="14" y="12" width="6" height="8" rx="1.2" />
    <rect x="14" y="4" width="6" height="5" rx="1.2" />
  </Icon>
);

export const RocketIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M4 13a8 8 0 0 1 7 7a6 6 0 0 0 3 -5a9 9 0 0 0 6 -8a3 3 0 0 0 -3 -3a9 9 0 0 0 -8 6a6 6 0 0 0 -5 3z" />
    <path d="M7 14a6 6 0 0 0 -3 6a6 6 0 0 0 6 -3" />
    <circle cx="15" cy="9" r="1.2" />
  </Icon>
);

export const BoltIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M13 3l0 7l6 0l-8 11l0 -7l-6 0l8 -11z" />
  </Icon>
);

export const ShieldCheckIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M12 3l7 3v5c0 4 -3 7 -7 8c-4 -1 -7 -4 -7 -8v-5l7 -3z" />
    <path d="M9 12l2 2l4 -4" />
  </Icon>
);

/* ---------------- Use-case icons ---------------- */

export const CpuIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <rect x="5" y="5" width="14" height="14" rx="2" />
    <rect x="9" y="9" width="6" height="6" rx="1" />
    <path d="M9 3v2M15 3v2M9 19v2M15 19v2M3 9h2M3 15h2M19 9h2M19 15h2" />
  </Icon>
);

export const BuildingIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M3 21h18" />
    <path d="M5 21v-15a1 1 0 0 1 1 -1h8a1 1 0 0 1 1 1v15" />
    <path d="M15 21v-9a1 1 0 0 1 1 -1h2a1 1 0 0 1 1 1v9" />
    <path d="M8 8h2M8 12h2M8 16h2" />
  </Icon>
);

export const ReportMoneyIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M14 3v4a1 1 0 0 0 1 1h4" />
    <path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z" />
    <path d="M14 14a2 1.6 0 0 0 -2 -1h-.5a1.4 1.4 0 0 0 0 2.6h1a1.4 1.4 0 0 1 0 2.6h-.5a2 1.6 0 0 1 -2 -1" />
    <path d="M12 11.4v.6M12 18v.6" />
  </Icon>
);

/* ---------------- Status / utility icons ---------------- */

export const CheckCircleIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <circle cx="12" cy="12" r="9" />
    <path d="M9 12l2 2l4 -4" />
  </Icon>
);

export const XCircleIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <circle cx="12" cy="12" r="9" />
    <path d="M10 10l4 4M14 10l-4 4" />
  </Icon>
);

export const ArrowRightIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M5 12h14M13 18l6 -6l-6 -6" />
  </Icon>
);

export const GithubIcon = (p: IconProps): ReactNode => (
  <Icon {...p}>
    <path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" />
  </Icon>
);
