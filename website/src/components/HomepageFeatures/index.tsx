import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg?: React.ComponentType<React.ComponentProps<'svg'>>;
  icon?: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: '🔐 Multi-Tenant Isolation',
    icon: '🔐',
    description: (
      <>
        Built on Capsule for true Kubernetes-native multi-tenancy.
        Each team gets isolated resources with shared or exclusive node pools,
        eliminating manual quota configuration.
      </>
    ),
  },
  {
    title: '💰 Real-Time Billing',
    icon: '💰',
    description: (
      <>
        Integrated with OpenCost for automatic cost tracking.
        Per-pod, per-namespace, per-team visibility with customizable pricing
        for CPU, Memory, and GPU resources.
      </>
    ),
  },
  {
    title: '📊 Unified Dashboard',
    icon: '📊',
    description: (
      <>
        Single pane of glass for admins, team leaders, and finance teams.
        Real-time balance monitoring, alerts, auto-suspension,
        and comprehensive usage reports.
      </>
    ),
  },
  {
    title: '🚀 Deploy in Minutes',
    icon: '🚀',
    description: (
      <>
        Zero external dependencies - all data stored in Kubernetes ConfigMaps.
        Install with a single Helm command and get complete GPU resource
        management in under 30 minutes.
      </>
    ),
  },
  {
    title: '⚡ Auto-Deduction & Alerts',
    icon: '⚡',
    description: (
      <>
        Automated billing with prepaid balances and real-time deduction.
        Multi-channel alerts (Webhook, DingTalk, WeChat) with configurable
        thresholds and auto-suspension.
      </>
    ),
  },
  {
    title: '🎯 Production Ready',
    icon: '🎯',
    description: (
      <>
        Cloud-native architecture with horizontal scaling, RBAC integration,
        and comprehensive audit logging. Support for multi-platform
        Docker images and enterprise SSO.
      </>
    ),
  },
];

function Feature({title, Svg, icon, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')} style={{marginBottom: '2rem'}}>
      <div className={clsx('text--center', styles.featureCard)}>
        {icon && <div className={styles.featureIcon}>{icon}</div>}
        <Heading as="h3" className={styles.featureTitle}>{title}</Heading>
        <p className={styles.featureDescription}>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
