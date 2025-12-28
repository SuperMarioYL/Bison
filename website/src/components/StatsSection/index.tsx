import {useEffect, useState, useRef} from 'react';
import type {ReactNode} from 'react';
import {translate} from '@docusaurus/Translate';
import styles from './styles.module.css';

interface StatItem {
  value: string;
  label: string;
  suffix?: string;
}

const stats: StatItem[] = [
  {
    value: '99.9',
    label: translate({
      id: 'component.statsSection.efficiency',
      message: 'GPU Resource Efficiency',
      description: 'Label for GPU resource efficiency statistic',
    }),
    suffix: '%',
  },
  {
    value: '30',
    label: translate({
      id: 'component.statsSection.deployTime',
      message: 'Avg Deploy Time',
      description: 'Label for average deployment time statistic',
    }),
    suffix: ' min',
  },
  {
    value: '1000',
    label: translate({
      id: 'component.statsSection.tenants',
      message: 'Supported Tenants',
      description: 'Label for supported tenants statistic',
    }),
    suffix: '+',
  },
  {
    value: '40',
    label: translate({
      id: 'component.statsSection.savings',
      message: 'Cost Savings',
      description: 'Label for cost savings statistic',
    }),
    suffix: '%+',
  },
];

function CountUpNumber({end, suffix = '', duration = 2000}: {end: number; suffix?: string; duration?: number}): ReactNode {
  const [count, setCount] = useState(0);
  const [hasAnimated, setHasAnimated] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        if (entries[0].isIntersecting && !hasAnimated) {
          setHasAnimated(true);

          const startTime = Date.now();
          const startValue = 0;

          const animate = () => {
            const now = Date.now();
            const progress = Math.min((now - startTime) / duration, 1);

            // Easing function (ease-out cubic)
            const easeOut = 1 - Math.pow(1 - progress, 3);
            const current = startValue + (end - startValue) * easeOut;

            setCount(current);

            if (progress < 1) {
              requestAnimationFrame(animate);
            } else {
              setCount(end);
            }
          };

          animate();
        }
      },
      {threshold: 0.3}
    );

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, [end, duration, hasAnimated]);

  return (
    <span ref={ref}>
      {end % 1 === 0 ? Math.floor(count) : count.toFixed(1)}
      {suffix}
    </span>
  );
}

export default function StatsSection(): ReactNode {
  return (
    <section className={styles.statsSection}>
      <div className="container">
        <div className={styles.statsGrid}>
          {stats.map((stat, index) => (
            <div key={index} className={styles.statItem}>
              <div className={styles.statValue}>
                <CountUpNumber
                  end={parseFloat(stat.value)}
                  suffix={stat.suffix}
                  duration={2000}
                />
              </div>
              <div className={styles.statLabel}>{stat.label}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
