import type {ReactNode} from 'react';
import {translate} from '@docusaurus/Translate';
import Translate from '@docusaurus/Translate';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

interface ArchNode {
  id: string;
  label: string;
  description: string;
  color: string;
}

const nodes: ArchNode[] = [
  {
    id: 'bison',
    label: 'Bison',
    description: translate({
      id: 'component.architectureDiagram.node.bison',
      message: 'GPU Billing & Scheduling Platform',
      description: 'Description for Bison node',
    }),
    color: '#0A84FF',
  },
  {
    id: 'capsule',
    label: 'Capsule',
    description: translate({
      id: 'component.architectureDiagram.node.capsule',
      message: 'Multi-Tenant Management',
      description: 'Description for Capsule node',
    }),
    color: '#5E5CE6',
  },
  {
    id: 'opencost',
    label: 'OpenCost',
    description: translate({
      id: 'component.architectureDiagram.node.opencost',
      message: 'Cost Tracking & Analytics',
      description: 'Description for OpenCost node',
    }),
    color: '#BF5AF2',
  },
  {
    id: 'k8s',
    label: 'Kubernetes',
    description: translate({
      id: 'component.architectureDiagram.node.k8s',
      message: 'Container Orchestration',
      description: 'Description for Kubernetes node',
    }),
    color: '#326CE5',
  },
  {
    id: 'prometheus',
    label: 'Prometheus',
    description: translate({
      id: 'component.architectureDiagram.node.prometheus',
      message: 'Metrics Collection',
      description: 'Description for Prometheus node',
    }),
    color: '#E6522C',
  },
];

export default function ArchitectureDiagram(): ReactNode {
  return (
    <section className={styles.architectureSection}>
      <div className="container">
        <div className="text--center margin-bottom--lg">
          <Heading as="h2" className={styles.sectionTitle}>
            <Translate id="component.architectureDiagram.title">
              Architecture Overview
            </Translate>
          </Heading>
          <p className={styles.sectionSubtitle}>
            <Translate id="component.architectureDiagram.subtitle">
              Built on cloud-native technologies for scalability and reliability
            </Translate>
          </p>
        </div>

        <div className={styles.diagramContainer}>
          <svg viewBox="0 0 800 400" className={styles.diagram}>
            <defs>
              <marker
                id="arrowhead"
                markerWidth="10"
                markerHeight="10"
                refX="9"
                refY="3"
                orient="auto">
                <polygon points="0 0, 10 3, 0 6" fill="#999" />
              </marker>

              <filter id="glow">
                <feGaussianBlur stdDeviation="3" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>

            {/* Connections */}
            <g className={styles.connections}>
              <line x1="400" y1="80" x2="400" y2="140" stroke="#999" strokeWidth="2" markerEnd="url(#arrowhead)" className={styles.connectionLine} />
              <line x1="400" y1="200" x2="250" y2="260" stroke="#999" strokeWidth="2" markerEnd="url(#arrowhead)" className={styles.connectionLine} />
              <line x1="400" y1="200" x2="550" y2="260" stroke="#999" strokeWidth="2" markerEnd="url(#arrowhead)" className={styles.connectionLine} />
              <line x1="250" y1="320" x2="400" y2="320" stroke="#999" strokeWidth="2" markerEnd="url(#arrowhead)" className={styles.connectionLine} />
              <line x1="550" y1="320" x2="650" y2="320" stroke="#999" strokeWidth="2" markerEnd="url(#arrowhead)" className={styles.connectionLine} />
            </g>

            {/* Bison */}
            <g className={styles.node}>
              <rect x="330" y="40" width="140" height="60" rx="8" fill="#0A84FF" filter="url(#glow)" />
              <text x="400" y="75" textAnchor="middle" fill="white" fontSize="18" fontWeight="600">Bison</text>
            </g>

            {/* Capsule */}
            <g className={styles.node}>
              <rect x="330" y="160" width="140" height="60" rx="8" fill="#5E5CE6" filter="url(#glow)" />
              <text x="400" y="195" textAnchor="middle" fill="white" fontSize="18" fontWeight="600">Capsule</text>
            </g>

            {/* OpenCost */}
            <g className={styles.node}>
              <rect x="480" y="280" width="140" height="60" rx="8" fill="#BF5AF2" filter="url(#glow)" />
              <text x="550" y="315" textAnchor="middle" fill="white" fontSize="18" fontWeight="600">OpenCost</text>
            </g>

            {/* Kubernetes */}
            <g className={styles.node}>
              <rect x="180" y="280" width="140" height="60" rx="8" fill="#326CE5" filter="url(#glow)" />
              <text x="250" y="315" textAnchor="middle" fill="white" fontSize="18" fontWeight="600">Kubernetes</text>
            </g>

            {/* Prometheus */}
            <g className={styles.node}>
              <rect x="630" y="280" width="140" height="60" rx="8" fill="#E6522C" filter="url(#glow)" />
              <text x="700" y="315" textAnchor="middle" fill="white" fontSize="18" fontWeight="600">Prometheus</text>
            </g>
          </svg>

          <div className={styles.nodeDescriptions}>
            {nodes.map(node => (
              <div key={node.id} className={styles.nodeCard} style={{borderColor: node.color}}>
                <div className={styles.nodeCardTitle} style={{color: node.color}}>
                  {node.label}
                </div>
                <div className={styles.nodeCardDescription}>{node.description}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
