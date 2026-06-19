import {useState, type ReactNode} from 'react';
import {translate} from '@docusaurus/Translate';
import Translate from '@docusaurus/Translate';
import Heading from '@theme/Heading';
import {
  DashboardIcon,
  CpuIcon,
  ReportMoneyIcon,
  CurrencyDollarIcon,
} from '../Icons';
import styles from './styles.module.css';

/* ------------------------------------------------------------------ */
/*  Reusable SVG primitives — all panels share one "app chrome"        */
/* ------------------------------------------------------------------ */

const C = {
  sidebar: '#f7f8fa',
  sidebarActive: '#eaf2ff',
  card: '#ffffff',
  border: '#ececf0',
  bg: '#f4f5f7',
  ink: '#1d1d1f',
  muted: '#86868b',
  blue: '#0A84FF',
  green: '#34C759',
  indigo: '#5E5CE6',
  purple: '#BF5AF2',
  orange: '#FF9F0A',
  gold: '#FFB300',
  silver: '#B0B5BD',
  bronze: '#CD7F32',
};

const NAV = [
  '资源总览',
  '集群节点',
  '团队管理',
  '项目管理',
  '用户管理',
  '报表中心',
  '审计日志',
  '系统设置',
];

function StatCard({
  x,
  y,
  w,
  label,
  value,
  accent,
}: {
  x: number;
  y: number;
  w: number;
  label: string;
  value: string;
  accent: string;
}): ReactNode {
  return (
    <g>
      <rect x={x} y={y} width={w} height={68} rx={10} fill={C.card} stroke={C.border} />
      <rect x={x} y={y} width={4} height={68} rx={2} fill={accent} />
      <text x={x + 18} y={y + 26} fontSize={11} fill={C.muted}>
        {label}
      </text>
      <text x={x + 18} y={y + 50} fontSize={20} fontWeight={700} fill={C.ink}>
        {value}
      </text>
    </g>
  );
}

function Donut({cx, cy, r}: {cx: number; cy: number; r: number}): ReactNode {
  // Three segments: 共享池 55%, 独占 33%, 未管理 12%
  const segs = [
    {p: 0.55, c: C.blue},
    {p: 0.33, c: C.green},
    {p: 0.12, c: '#d9dde3'},
  ];
  const circ = 2 * Math.PI * r;
  let offset = 0;
  return (
    <g transform={`rotate(-90 ${cx} ${cy})`}>
      {segs.map((s, i) => {
        const len = s.p * circ;
        const el = (
          <circle
            key={i}
            cx={cx}
            cy={cy}
            r={r}
            fill="none"
            stroke={s.c}
            strokeWidth={16}
            strokeDasharray={`${len} ${circ - len}`}
            strokeDashoffset={-offset}
          />
        );
        offset += len;
        return el;
      })}
    </g>
  );
}

function Sparkline({
  x,
  y,
  w,
  h,
  pts,
  color,
}: {
  x: number;
  y: number;
  w: number;
  h: number;
  pts: number[];
  color: string;
}): ReactNode {
  const max = Math.max(...pts);
  const min = Math.min(...pts);
  const span = max - min || 1;
  const coords = pts.map((p, i) => {
    const px = x + (i / (pts.length - 1)) * w;
    const py = y + h - ((p - min) / span) * h;
    return [px, py];
  });
  const line = coords.map((c, i) => `${i === 0 ? 'M' : 'L'}${c[0]},${c[1]}`).join(' ');
  const area = `${line} L${x + w},${y + h} L${x},${y + h} Z`;
  return (
    <g>
      <path d={area} fill={`${color}22`} />
      <path d={line} fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
      {coords.map((c, i) => (
        <circle key={i} cx={c[0]} cy={c[1]} r={2.6} fill={color} />
      ))}
    </g>
  );
}

function ProgressBar({
  x,
  y,
  w,
  pct,
  color,
}: {
  x: number;
  y: number;
  w: number;
  pct: number;
  color: string;
}): ReactNode {
  return (
    <g>
      <rect x={x} y={y} width={w} height={6} rx={3} fill="#eef0f3" />
      <rect x={x} y={y} width={(w * pct) / 100} height={6} rx={3} fill={color} />
    </g>
  );
}

function Tag({
  x,
  y,
  w,
  label,
  color,
}: {
  x: number;
  y: number;
  w: number;
  label: string;
  color: string;
}): ReactNode {
  return (
    <g>
      <rect x={x} y={y} width={w} height={18} rx={4} fill={`${color}1f`} />
      <text x={x + w / 2} y={y + 13} fontSize={10} fill={color} textAnchor="middle" fontWeight={600}>
        {label}
      </text>
    </g>
  );
}

/* ------------------------------------------------------------------ */
/*  Per-screen content (drawn inside the content viewport 196..940)    */
/* ------------------------------------------------------------------ */

function DashboardPanel(): ReactNode {
  return (
    <g>
      <text x={214} y={84} fontSize={17} fontWeight={700} fill={C.ink}>
        资源总览
      </text>
      <StatCard x={214} y={100} w={170} label="集群节点" value="8 个" accent={C.blue} />
      <StatCard x={398} y={100} w={170} label="团队数量" value="5 个" accent={C.indigo} />
      <StatCard x={582} y={100} w={170} label="项目数量" value="12 个" accent={C.purple} />
      <StatCard x={766} y={100} w={158} label="费用统计" value="已启用" accent={C.green} />

      {/* Node status pie */}
      <rect x={214} y={184} width={344} height={150} rx={10} fill={C.card} stroke={C.border} />
      <text x={232} y={208} fontSize={12} fontWeight={600} fill={C.ink}>
        节点状态分布
      </text>
      <Donut cx={282} cy={272} r={34} />
      {[
        ['共享池', C.blue, '4'],
        ['独占', C.green, '3'],
        ['未管理', '#d9dde3', '1'],
      ].map((row, i) => (
        <g key={i}>
          <rect x={360} y={236 + i * 26} width={9} height={9} rx={2} fill={row[1]} />
          <text x={378} y={245 + i * 26} fontSize={11} fill={C.ink}>
            {row[0]}
          </text>
          <text x={528} y={245 + i * 26} fontSize={11} fill={C.muted} textAnchor="end">
            {row[2]}
          </text>
        </g>
      ))}

      {/* Cost trend */}
      <rect x={572} y={184} width={352} height={150} rx={10} fill={C.card} stroke={C.border} />
      <text x={590} y={208} fontSize={12} fontWeight={600} fill={C.ink}>
        费用趋势 (7 天)
      </text>
      <Sparkline x={590} y={224} w={316} h={86} pts={[12, 14, 11, 18, 16, 24, 30]} color={C.blue} />

      {/* Resource table */}
      <rect x={214} y={350} width={500} height={150} rx={10} fill={C.card} stroke={C.border} />
      <text x={232} y={374} fontSize={12} fontWeight={600} fill={C.ink}>
        集群资源
      </text>
      {[
        ['CPU', 62, C.blue],
        ['Memory', 48, C.green],
        ['GPU (nvidia)', 81, C.purple],
        ['Storage', 35, C.indigo],
      ].map((row, i) => (
        <g key={i}>
          <text x={232} y={406 + i * 24} fontSize={11} fill={C.ink}>
            {row[0] as string}
          </text>
          <ProgressBar x={360} y={398 + i * 24} w={260} pct={row[1] as number} color={row[2] as string} />
          <text x={636} y={406 + i * 24} fontSize={11} fill={C.muted} textAnchor="end">
            {row[1]}%
          </text>
        </g>
      ))}

      {/* Top consumers */}
      <rect x={728} y={350} width={196} height={150} rx={10} fill={C.card} stroke={C.border} />
      <text x={746} y={374} fontSize={12} fontWeight={600} fill={C.ink}>
        资源消耗 Top 5
      </text>
      {[
        ['vision-lab', '$42.10', C.gold],
        ['nlp-team', '$31.80', C.silver],
        ['rec-sys', '$22.40', C.bronze],
        ['infra', '$9.10', '#d9dde3'],
      ].map((row, i) => (
        <g key={i}>
          <circle cx={752} cy={398 + i * 24} r={7} fill={row[2] as string} />
          <text x={752} y={401 + i * 24} fontSize={8} fill="#fff" textAnchor="middle" fontWeight={700}>
            {i + 1}
          </text>
          <text x={768} y={402 + i * 24} fontSize={10.5} fill={C.ink}>
            {row[0]}
          </text>
          <text x={908} y={402 + i * 24} fontSize={10.5} fill={C.blue} textAnchor="end" fontWeight={600}>
            {row[1]}
          </text>
        </g>
      ))}
    </g>
  );
}

function ClusterPanel(): ReactNode {
  const rows = [
    ['gpu-node-01', 'amd64', '独占', C.green, 78, 'A100 × 8'],
    ['gpu-node-02', 'amd64', '独占', C.green, 64, 'A100 × 8'],
    ['gpu-node-03', 'amd64', '共享池', C.blue, 51, 'L40S × 4'],
    ['cpu-node-01', 'arm64', '共享池', C.blue, 33, '—'],
    ['cpu-node-02', 'amd64', '未管理', '#9aa0a6', 12, '—'],
  ];
  return (
    <g>
      <text x={214} y={84} fontSize={17} fontWeight={700} fill={C.ink}>
        集群节点
      </text>
      <StatCard x={214} y={100} w={228} label="节点总数" value="8 个" accent={C.blue} />
      <StatCard x={456} y={100} w={228} label="GPU 卡总数" value="28 张" accent={C.purple} />
      <StatCard x={698} y={100} w={226} label="可分配 GPU" value="11 张" accent={C.green} />

      <rect x={214} y={184} width={710} height={316} rx={10} fill={C.card} stroke={C.border} />
      {/* header */}
      {['节点名称', '架构', '状态', 'GPU 使用率', 'GPU 设备'].map((h, i) => (
        <text key={i} x={[232, 392, 470, 580, 800][i]} y={212} fontSize={11} fill={C.muted} fontWeight={600}>
          {h}
        </text>
      ))}
      <line x1={214} y1={224} x2={924} y2={224} stroke={C.border} />
      {rows.map((r, i) => {
        const y = 248 + i * 48;
        return (
          <g key={i}>
            <circle cx={238} cy={y - 4} r={4} fill={C.green} />
            <text x={252} y={y} fontSize={11.5} fill={C.ink} fontWeight={600}>
              {r[0] as string}
            </text>
            <text x={392} y={y} fontSize={11} fill={C.muted}>
              {r[1] as string}
            </text>
            <Tag x={470} y={y - 13} w={54} label={r[2] as string} color={r[3] as string} />
            <ProgressBar x={580} y={y - 8} w={150} pct={r[4] as number} color={C.blue} />
            <text x={740} y={y} fontSize={10.5} fill={C.muted}>
              {r[4]}%
            </text>
            <text x={800} y={y} fontSize={11} fill={C.ink}>
              {r[5] as string}
            </text>
            {i < rows.length - 1 && <line x1={232} y1={y + 22} x2={906} y2={y + 22} stroke="#f3f4f6" />}
          </g>
        );
      })}
    </g>
  );
}

function ReportPanel(): ReactNode {
  const bars = [
    ['vision-lab', 92, C.blue],
    ['nlp-team', 70, C.indigo],
    ['rec-sys', 54, C.purple],
    ['infra', 28, C.green],
    ['platform', 18, C.orange],
  ];
  const rank = [
    ['vision-lab', '¥4210.00', '38.2%', C.gold],
    ['nlp-team', '¥3180.50', '28.9%', C.silver],
    ['rec-sys', '¥2240.10', '20.3%', C.bronze],
  ];
  return (
    <g>
      <text x={214} y={84} fontSize={17} fontWeight={700} fill={C.ink}>
        报表中心
      </text>
      <StatCard x={214} y={100} w={228} label="总消费" value="¥11030" accent={C.blue} />
      <StatCard x={456} y={100} w={228} label="团队数" value="5" accent={C.indigo} />
      <StatCard x={698} y={100} w={226} label="统计周期" value="30d" accent={C.green} />

      {/* Bar chart */}
      <rect x={214} y={184} width={356} height={316} rx={10} fill={C.card} stroke={C.border} />
      <text x={232} y={208} fontSize={12} fontWeight={600} fill={C.ink}>
        团队消费分布
      </text>
      {bars.map((b, i) => {
        const y = 234 + i * 50;
        return (
          <g key={i}>
            <text x={232} y={y + 4} fontSize={10.5} fill={C.ink}>
              {b[0] as string}
            </text>
            <rect x={232} y={y + 14} width={320} height={10} rx={5} fill="#eef0f3" />
            <rect x={232} y={y + 14} width={(320 * (b[1] as number)) / 100} height={10} rx={5} fill={b[2] as string} />
          </g>
        );
      })}

      {/* Ranking table */}
      <rect x={584} y={184} width={340} height={316} rx={10} fill={C.card} stroke={C.border} />
      <text x={602} y={208} fontSize={12} fontWeight={600} fill={C.ink}>
        团队消费排行榜 Top 10
      </text>
      {['排名', '团队', '金额', '占比'].map((h, i) => (
        <text key={i} x={[602, 648, 770, 890][i]} y={236} fontSize={10} fill={C.muted} fontWeight={600} textAnchor={i >= 2 ? 'end' : 'start'}>
          {h}
        </text>
      ))}
      <line x1={602} y1={246} x2={906} y2={246} stroke={C.border} />
      {rank.map((r, i) => {
        const y = 274 + i * 40;
        return (
          <g key={i}>
            <path
              d={`M${608} ${y - 9} l3 6 l6 .8 l-4.5 4.3 l1 6 l-5.5 -3 l-5.5 3 l1 -6 l-4.5 -4.3 l6 -.8 z`}
              fill={r[3] as string}
            />
            <text x={648} y={y} fontSize={11} fill={C.ink} fontWeight={600}>
              {r[0] as string}
            </text>
            <text x={770} y={y} fontSize={11} fill={C.ink} textAnchor="end">
              {r[1] as string}
            </text>
            <text x={890} y={y} fontSize={11} fill={C.muted} textAnchor="end">
              {r[2] as string}
            </text>
            {i < rank.length - 1 && <line x1={602} y1={y + 18} x2={906} y2={y + 18} stroke="#f3f4f6" />}
          </g>
        );
      })}
    </g>
  );
}

function BillingPanel(): ReactNode {
  const rows = [
    ['cpu', 'CPU', '核', '其他', '0.10', C.blue],
    ['memory', 'Memory', 'GiB', '其他', '0.05', C.green],
    ['nvidia.com/gpu', 'GPU', '张', 'GPU', '8.00', C.purple],
    ['ephemeral-storage', 'Storage', 'GiB', '存储', '0.01', C.indigo],
  ];
  return (
    <g>
      <text x={214} y={84} fontSize={17} fontWeight={700} fill={C.ink}>
        系统设置 · 计费配置
      </text>
      {/* tab strip */}
      {['基本配置', '资源配置', '计费配置', '告警配置', '系统状态'].map((t, i) => (
        <g key={i}>
          <text
            x={232 + i * 96}
            y={114}
            fontSize={11}
            fontWeight={i === 2 ? 700 : 500}
            fill={i === 2 ? C.blue : C.muted}>
            {t}
          </text>
          {i === 2 && <rect x={228 + i * 96} y={122} width={62} height={2.5} rx={1} fill={C.blue} />}
        </g>
      ))}

      <rect x={214} y={140} width={710} height={360} rx={10} fill={C.card} stroke={C.border} />
      {/* info banner */}
      <rect x={232} y={158} width={674} height={40} rx={8} fill="#eaf2ff" />
      <CurrencyDollarIcon x={244} y={168} size={18} stroke={C.blue} />
      <text x={272} y={174} fontSize={10.5} fill={C.blue}>
        单价用于计费：按（显示值 × 单价 × 使用时长）实时扣费，支持 CPU / 内存 / GPU 自定义价格
      </text>
      <text x={272} y={188} fontSize={10.5} fill={C.blue}>
        换算除数用于将 K8s 原始值转为显示值（如 memory 1073741824 → 1 GiB）
      </text>

      {/* table header */}
      {['资源名称', '显示名', '单位', '分类', '启用', '单价 (¥/单位·时)'].map((h, i) => (
        <text key={i} x={[244, 380, 500, 580, 690, 770][i]} y={228} fontSize={10.5} fill={C.muted} fontWeight={600}>
          {h}
        </text>
      ))}
      <line x1={232} y1={240} x2={906} y2={240} stroke={C.border} />
      {rows.map((r, i) => {
        const y = 268 + i * 50;
        return (
          <g key={i}>
            <rect x={244} y={y - 13} width={106} height={18} rx={4} fill="#f1f3f5" />
            <text x={250} y={y} fontSize={9.5} fill={C.ink} fontFamily="monospace">
              {r[0] as string}
            </text>
            <text x={380} y={y} fontSize={11} fill={C.ink}>
              {r[1] as string}
            </text>
            <text x={500} y={y} fontSize={11} fill={C.muted}>
              {r[2] as string}
            </text>
            <Tag x={580} y={y - 13} w={44} label={r[3] as string} color={r[5] as string} />
            {/* toggle on */}
            <rect x={690} y={y - 11} width={30} height={16} rx={8} fill={C.green} />
            <circle cx={712} cy={y - 3} r={6} fill="#fff" />
            <text x={774} y={y} fontSize={12} fill={C.ink} fontWeight={600}>
              ¥ {r[4] as string}
            </text>
            {i < rows.length - 1 && <line x1={244} y1={y + 22} x2={894} y2={y + 22} stroke="#f3f4f6" />}
          </g>
        );
      })}
    </g>
  );
}

/* ------------------------------------------------------------------ */
/*  Tabbed showcase                                                    */
/* ------------------------------------------------------------------ */

type TabKey = 'dashboard' | 'cluster' | 'report' | 'billing';

const TABS: {key: TabKey; label: string; Icon: typeof DashboardIcon}[] = [
  {key: 'dashboard', label: translate({id: 'showcase.tab.dashboard', message: '资源总览'}), Icon: DashboardIcon},
  {key: 'cluster', label: translate({id: 'showcase.tab.cluster', message: '集群节点'}), Icon: CpuIcon},
  {key: 'report', label: translate({id: 'showcase.tab.report', message: '报表中心'}), Icon: ReportMoneyIcon},
  {key: 'billing', label: translate({id: 'showcase.tab.billing', message: '计费配置'}), Icon: CurrencyDollarIcon},
];

const NAV_ACTIVE: Record<TabKey, number> = {dashboard: 0, cluster: 1, report: 5, billing: 7};

function AppFrame({tab}: {tab: TabKey}): ReactNode {
  const activeNav = NAV_ACTIVE[tab];
  return (
    <svg viewBox="0 0 940 520" className={styles.screen} role="img" aria-label="Bison UI">
      {/* window */}
      <rect x={0.5} y={0.5} width={939} height={519} rx={14} fill={C.bg} stroke={C.border} />
      {/* sidebar */}
      <rect x={0} y={0} width={184} height={520} rx={14} fill={C.sidebar} />
      <rect x={170} y={0} width={14} height={520} fill={C.sidebar} />
      {/* brand */}
      <circle cx={28} cy={32} r={11} fill="url(#bisonGrad)" />
      <text x={48} y={37} fontSize={15} fontWeight={700} fill={C.ink}>
        Bison
      </text>
      {/* nav */}
      {NAV.map((item, i) => {
        const y = 70 + i * 38;
        const active = i === activeNav;
        return (
          <g key={i}>
            {active && <rect x={12} y={y} width={160} height={30} rx={8} fill={C.sidebarActive} />}
            <rect x={20} y={y + 9} width={13} height={13} rx={3} fill="none" stroke={active ? C.blue : C.muted} strokeWidth={1.6} />
            <text x={44} y={y + 20} fontSize={11.5} fill={active ? C.blue : '#5b5f66'} fontWeight={active ? 600 : 400}>
              {item}
            </text>
          </g>
        );
      })}
      <text x={20} y={500} fontSize={9.5} fill={C.muted}>
        v0.0.12 · Capsule + OpenCost
      </text>

      {/* top bar */}
      <rect x={184} y={0} width={756} height={48} fill="#ffffff" />
      <line x1={184} y1={48} x2={940} y2={48} stroke={C.border} />
      <text x={206} y={30} fontSize={12.5} fill={C.muted}>
        集群资源调度计费平台
      </text>
      <circle cx={886} cy={24} r={11} fill="#eef0f3" />
      <text x={886} y={28} fontSize={10} fill={C.muted} textAnchor="middle">
        A
      </text>
      <text x={870} y={28} fontSize={11} fill={C.ink} textAnchor="end">
        admin
      </text>

      <defs>
        <linearGradient id="bisonGrad" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor={C.blue} />
          <stop offset="1" stopColor={C.purple} />
        </linearGradient>
      </defs>

      {tab === 'dashboard' && <DashboardPanel />}
      {tab === 'cluster' && <ClusterPanel />}
      {tab === 'report' && <ReportPanel />}
      {tab === 'billing' && <BillingPanel />}
    </svg>
  );
}

export default function ProductShowcase(): ReactNode {
  const [tab, setTab] = useState<TabKey>('dashboard');
  return (
    <section className={styles.section}>
      <div className="container">
        <div className="text--center margin-bottom--lg">
          <Heading as="h2" className={styles.title}>
            <Translate id="showcase.title">One console for the whole GPU lifecycle</Translate>
          </Heading>
          <p className={styles.subtitle}>
            <Translate id="showcase.subtitle">
              From cluster topology to per-team chargeback — every screen, vector-rendered
            </Translate>
          </p>
        </div>

        <div className={styles.tabs} role="tablist">
          {TABS.map(({key, label, Icon}) => (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={tab === key}
              className={`${styles.tab} ${tab === key ? styles.tabActive : ''}`}
              onClick={() => setTab(key)}>
              <Icon size={18} />
              <span>{label}</span>
            </button>
          ))}
        </div>

        <div className={styles.frame}>
          <AppFrame tab={tab} />
        </div>
      </div>
    </section>
  );
}
