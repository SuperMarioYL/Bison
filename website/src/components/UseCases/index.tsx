import type {ReactNode} from 'react';
import {translate} from '@docusaurus/Translate';
import Translate from '@docusaurus/Translate';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

interface UseCase {
  icon: string;
  title: string;
  description: string;
  before: string[];
  after: string[];
}

const useCases: UseCase[] = [
  {
    icon: '🤖',
    title: translate({
      id: 'component.useCases.aiTraining.title',
      message: 'AI Training Platform',
      description: 'Title for AI training use case',
    }),
    description: translate({
      id: 'component.useCases.aiTraining.description',
      message: 'Multi-team GPU resource sharing for machine learning workloads',
      description: 'Description for AI training use case',
    }),
    before: [
      translate({
        id: 'component.useCases.aiTraining.before.manual',
        message: 'Manual GPU allocation',
        description: 'AI training before: manual allocation',
      }),
      translate({
        id: 'component.useCases.aiTraining.before.noCost',
        message: 'No cost visibility',
        description: 'AI training before: no cost visibility',
      }),
      translate({
        id: 'component.useCases.aiTraining.before.conflicts',
        message: 'Resource conflicts',
        description: 'AI training before: resource conflicts',
      }),
    ],
    after: [
      translate({
        id: 'component.useCases.aiTraining.after.automated',
        message: 'Automated scheduling',
        description: 'AI training after: automated scheduling',
      }),
      translate({
        id: 'component.useCases.aiTraining.after.realtime',
        message: 'Real-time cost tracking',
        description: 'AI training after: real-time cost tracking',
      }),
      translate({
        id: 'component.useCases.aiTraining.after.fair',
        message: 'Fair resource sharing',
        description: 'AI training after: fair resource sharing',
      }),
    ],
  },
  {
    icon: '🏢',
    title: translate({
      id: 'component.useCases.enterprise.title',
      message: 'Enterprise Cloud',
      description: 'Title for enterprise cloud use case',
    }),
    description: translate({
      id: 'component.useCases.enterprise.description',
      message: 'Department-level resource isolation and billing',
      description: 'Description for enterprise cloud use case',
    }),
    before: [
      translate({
        id: 'component.useCases.enterprise.before.chaos',
        message: 'Shared cluster chaos',
        description: 'Enterprise before: shared cluster chaos',
      }),
      translate({
        id: 'component.useCases.enterprise.before.noBudget',
        message: 'No budget control',
        description: 'Enterprise before: no budget control',
      }),
      translate({
        id: 'component.useCases.enterprise.before.manual',
        message: 'Manual reporting',
        description: 'Enterprise before: manual reporting',
      }),
    ],
    after: [
      translate({
        id: 'component.useCases.enterprise.after.isolated',
        message: 'Isolated tenants',
        description: 'Enterprise after: isolated tenants',
      }),
      translate({
        id: 'component.useCases.enterprise.after.prepaid',
        message: 'Prepaid balances',
        description: 'Enterprise after: prepaid balances',
      }),
      translate({
        id: 'component.useCases.enterprise.after.automated',
        message: 'Automated reports',
        description: 'Enterprise after: automated reports',
      }),
    ],
  },
  {
    icon: '💵',
    title: translate({
      id: 'component.useCases.billing.title',
      message: 'Cost Center Billing',
      description: 'Title for cost center billing use case',
    }),
    description: translate({
      id: 'component.useCases.billing.description',
      message: 'Chargeback system for internal GPU resources',
      description: 'Description for cost center billing use case',
    }),
    before: [
      translate({
        id: 'component.useCases.billing.before.excel',
        message: 'Excel-based tracking',
        description: 'Billing before: Excel-based tracking',
      }),
      translate({
        id: 'component.useCases.billing.before.monthly',
        message: 'Monthly reconciliation',
        description: 'Billing before: monthly reconciliation',
      }),
      translate({
        id: 'component.useCases.billing.before.disputes',
        message: 'Billing disputes',
        description: 'Billing before: billing disputes',
      }),
    ],
    after: [
      translate({
        id: 'component.useCases.billing.after.realtime',
        message: 'Real-time deduction',
        description: 'Billing after: real-time deduction',
      }),
      translate({
        id: 'component.useCases.billing.after.transparent',
        message: 'Transparent pricing',
        description: 'Billing after: transparent pricing',
      }),
      translate({
        id: 'component.useCases.billing.after.automated',
        message: 'Automated invoicing',
        description: 'Billing after: automated invoicing',
      }),
    ],
  },
];

function ComparisonCard({useCase}: {useCase: UseCase}): ReactNode {
  return (
    <div className={styles.useCaseCard}>
      <div className={styles.useCaseIcon}>{useCase.icon}</div>
      <Heading as="h3" className={styles.useCaseTitle}>
        {useCase.title}
      </Heading>
      <p className={styles.useCaseDescription}>{useCase.description}</p>

      <div className={styles.comparison}>
        <div className={styles.comparisonColumn}>
          <div className={styles.comparisonHeader}>
            <span className={styles.crossIcon}>❌</span>
            <span>
              <Translate id="component.useCases.beforeBison">
                Before Bison
              </Translate>
            </span>
          </div>
          <ul className={styles.comparisonList}>
            {useCase.before.map((item, i) => (
              <li key={i} className={styles.comparisonItemBefore}>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div className={styles.comparisonDivider}>→</div>

        <div className={styles.comparisonColumn}>
          <div className={styles.comparisonHeader}>
            <span className={styles.checkIcon}>✅</span>
            <span>
              <Translate id="component.useCases.withBison">
                With Bison
              </Translate>
            </span>
          </div>
          <ul className={styles.comparisonList}>
            {useCase.after.map((item, i) => (
              <li key={i} className={styles.comparisonItemAfter}>
                {item}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}

export default function UseCases(): ReactNode {
  return (
    <section className={styles.useCasesSection}>
      <div className="container">
        <div className="text--center margin-bottom--lg">
          <Heading as="h2" className={styles.sectionTitle}>
            <Translate id="component.useCases.title">
              Real-World Use Cases
            </Translate>
          </Heading>
          <p className={styles.sectionSubtitle}>
            <Translate id="component.useCases.subtitle">
              See how Bison transforms GPU resource management across different scenarios
            </Translate>
          </p>
        </div>

        <div className={styles.useCasesGrid}>
          {useCases.map((useCase, index) => (
            <ComparisonCard key={index} useCase={useCase} />
          ))}
        </div>
      </div>
    </section>
  );
}
