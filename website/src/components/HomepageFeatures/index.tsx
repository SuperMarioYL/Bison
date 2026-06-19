import type {ComponentType, ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import Translate from '@docusaurus/Translate';
import {
  ShieldLockIcon,
  CurrencyDollarIcon,
  DashboardIcon,
  RocketIcon,
  BoltIcon,
  ShieldCheckIcon,
  type IconProps,
} from '../Icons';
import styles from './styles.module.css';

type FeatureItem = {
  Icon: ComponentType<IconProps>;
  color: string;
  title: ReactNode;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    Icon: ShieldLockIcon,
    color: '#5E5CE6',
    title: (
      <Translate id="homepage.feature.multitenant.title">
        Multi-Tenant Isolation
      </Translate>
    ),
    description: (
      <Translate id="homepage.feature.multitenant.desc">
        Built on Capsule for true Kubernetes-native multi-tenancy. Each team gets
        isolated resources with shared or exclusive node pools, eliminating manual
        quota configuration.
      </Translate>
    ),
  },
  {
    Icon: CurrencyDollarIcon,
    color: '#34C759',
    title: (
      <Translate id="homepage.feature.billing.title">Real-Time Billing</Translate>
    ),
    description: (
      <Translate id="homepage.feature.billing.desc">
        Integrated with OpenCost for automatic cost tracking. Per-pod, per-namespace,
        per-team visibility with customizable pricing for CPU, Memory, and GPU
        resources.
      </Translate>
    ),
  },
  {
    Icon: DashboardIcon,
    color: '#0A84FF',
    title: (
      <Translate id="homepage.feature.dashboard.title">Unified Dashboard</Translate>
    ),
    description: (
      <Translate id="homepage.feature.dashboard.desc">
        Single pane of glass for admins, team leaders, and finance teams. Real-time
        balance monitoring, alerts, auto-suspension, and comprehensive usage reports.
      </Translate>
    ),
  },
  {
    Icon: RocketIcon,
    color: '#BF5AF2',
    title: (
      <Translate id="homepage.feature.deploy.title">Deploy in Minutes</Translate>
    ),
    description: (
      <Translate id="homepage.feature.deploy.desc">
        Zero external dependencies — all data stored in Kubernetes ConfigMaps. Install
        with a single Helm command and get complete GPU resource management in under 30
        minutes.
      </Translate>
    ),
  },
  {
    Icon: BoltIcon,
    color: '#FF9F0A',
    title: (
      <Translate id="homepage.feature.alerts.title">
        Auto-Deduction &amp; Alerts
      </Translate>
    ),
    description: (
      <Translate id="homepage.feature.alerts.desc">
        Automated billing with prepaid balances and real-time deduction. Multi-channel
        alerts (Webhook, DingTalk, WeChat) with configurable thresholds and
        auto-suspension.
      </Translate>
    ),
  },
  {
    Icon: ShieldCheckIcon,
    color: '#0FB5BA',
    title: (
      <Translate id="homepage.feature.production.title">Production Ready</Translate>
    ),
    description: (
      <Translate id="homepage.feature.production.desc">
        Cloud-native architecture with horizontal scaling, RBAC integration, and
        comprehensive audit logging. Multi-platform Docker images and enterprise SSO
        support.
      </Translate>
    ),
  },
];

function Feature({Icon, color, title, description}: FeatureItem): ReactNode {
  return (
    <div className={clsx('col col--4')} style={{marginBottom: '2rem'}}>
      <div className={styles.featureCard}>
        <div
          className={styles.featureIcon}
          style={{
            background: `linear-gradient(135deg, ${color}, ${color}cc)`,
            boxShadow: `0 6px 16px ${color}40`,
          }}>
          <Icon size={28} stroke="#fff" />
        </div>
        <Heading as="h3" className={styles.featureTitle}>
          {title}
        </Heading>
        <p className={styles.featureDescription}>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="text--center margin-bottom--lg">
          <Heading as="h2" className={styles.sectionTitle}>
            <Translate id="homepage.features.title">
              Everything you need to run GPU clusters
            </Translate>
          </Heading>
          <p className={styles.sectionSubtitle}>
            <Translate id="homepage.features.subtitle">
              Multi-tenancy, metering and billing — built into one cloud-native platform
            </Translate>
          </p>
        </div>
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
