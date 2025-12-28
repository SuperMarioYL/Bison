import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import ParticleBackground from '@site/src/components/ParticleBackground';
import StatsSection from '@site/src/components/StatsSection';
import ArchitectureDiagram from '@site/src/components/ArchitectureDiagram';
import UseCases from '@site/src/components/UseCases';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <ParticleBackground />
      <div className="container" style={{position: 'relative', zIndex: 10}}>
        <img
          src={useBaseUrl('/img/logo.svg')}
          alt="Bison Logo"
          style={{
            width: '120px',
            height: '120px',
            marginBottom: '1rem',
            animation: 'fadeInUp 0.8s ease-out',
            filter: 'drop-shadow(0 4px 12px rgba(0,0,0,0.2))',
          }}
        />
        <Heading
          as="h1"
          className="hero__title"
          style={{
            animation: 'fadeInUp 0.8s ease-out 0.1s both',
            textShadow: '0 2px 10px rgba(0,0,0,0.2)',
          }}>
          {siteConfig.title}
        </Heading>
        <p
          className="hero__subtitle"
          style={{
            animation: 'fadeInUp 0.8s ease-out 0.2s both',
            textShadow: '0 1px 5px rgba(0,0,0,0.2)',
          }}>
          {siteConfig.tagline}
        </p>
        <div className={styles.buttons}>
          <Link className="button button--secondary button--lg" to="/docs">
            Get Started 🚀
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            to="https://github.com/SuperMarioYL/Bison">
            GitHub ⭐
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - Kubernetes GPU Resource Billing & Multi-Tenant Management`}
      description="Enterprise GPU resource billing and multi-tenant management platform based on Kubernetes, Capsule, and OpenCost">
      <HomepageHeader />
      <main>
        <StatsSection />
        <HomepageFeatures />
        <ArchitectureDiagram />
        <UseCases />
      </main>
    </Layout>
  );
}
