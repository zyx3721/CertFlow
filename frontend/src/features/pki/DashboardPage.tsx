import { useId, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  BadgeCheck,
  Clock3,
  FileClock,
  Landmark,
  ShieldCheck,
  TrendingUp,
} from 'lucide-react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { EmptyState } from '@/components/empty-state';
import { PageHeader } from '@/components/page-header';
import { StatCard } from '@/components/stat-card';
import { StatusBadge } from '@/components/status-badge';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { useBrandSettings } from '@/lib/branding';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { getAuditEntries, getCAs, getCertificates, getCertificateTrend } from '@/lib/pki';
import { QueryState } from './shared';
import { formatDate, pkiQueryKeys, usePageRefresh } from './support';

const dashboardKeys = [
  pkiQueryKeys.cas,
  pkiQueryKeys.certificates,
  pkiQueryKeys.trend,
  pkiQueryKeys.audits,
] as const;

type CertificateStatusDatum = {
  name: string;
  value: number;
  color: string;
};

export function DashboardPage() {
  usePageRefresh(dashboardKeys);
  const user = getStoredUser();
  const brand = useBrandSettings();
  const canReadCA = userHasPermission(user, 'ca.read');
  const canReadCertificates = userHasPermission(user, 'certificates.read');
  const canReadAudit = userHasPermission(user, 'audit.read');
  const [auditLimit, setAuditLimit] = useState('10');
  const [recentCertificateLimit, setRecentCertificateLimit] = useState('10');
  const caQuery = useQuery({ queryKey: pkiQueryKeys.cas, queryFn: getCAs, enabled: canReadCA });
  const certQuery = useQuery({
    queryKey: pkiQueryKeys.certificates,
    queryFn: getCertificates,
    enabled: canReadCertificates,
  });
  const trendQuery = useQuery({
    queryKey: pkiQueryKeys.trend,
    queryFn: getCertificateTrend,
    enabled: canReadCertificates,
  });
  const auditQuery = useQuery({
    queryKey: [...pkiQueryKeys.audits, 'dashboard', auditLimit],
    queryFn: () => getAuditEntries(Number(auditLimit)),
    enabled: canReadAudit,
  });
  const certificates = useMemo(() => certQuery.data?.items ?? [], [certQuery.data?.items]);
  const cas = caQuery.data?.items ?? [];
  const expiryNotificationDays = brand.expiryNotificationDays;
  const now = Date.now();
  const expiring = certificates.filter(item => {
    const expiresAt = Date.parse(item.notAfter);
    return (
      item.status === 'valid' &&
      expiresAt > now &&
      expiresAt - now <= expiryNotificationDays * 86_400_000
    );
  });
  const statusData = useMemo<CertificateStatusDatum[]>(
    () => [
      {
        name: '有效',
        value: certificates.filter(item => item.status === 'valid').length,
        color: '#10b981',
      },
      {
        name: '已撤销',
        value: certificates.filter(item => item.status === 'revoked').length,
        color: '#ef4444',
      },
      {
        name: '已驳回',
        value: certificates.filter(item => item.status === 'rejected').length,
        color: '#f43f5e',
      },
      {
        name: '待审批',
        value: certificates.filter(item => item.status === 'pending').length,
        color: '#38bdf8',
      },
      {
        name: '已过期',
        value: certificates.filter(item => item.status === 'expired').length,
        color: '#f59e0b',
      },
    ],
    [certificates]
  );
  const loading = (canReadCA && caQuery.isLoading) || (canReadCertificates && certQuery.isLoading);

  return (
    <div className="space-y-5 pb-5">
      <PageHeader title="仪表盘" description="PKI 基础设施、证书状态与安全活动实时总览" />

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="证书总数"
          value={canReadCertificates ? (loading ? '-' : certificates.length) : 0}
          hint={
            canReadCertificates
              ? `${certificates.filter(item => item.status === 'valid').length} 张有效证书`
              : '无证书查看权限'
          }
          icon={<BadgeCheck size={19} />}
        />
        <StatCard
          label="活跃 CA"
          value={
            canReadCA ? (loading ? '-' : cas.filter(item => item.status === 'active').length) : 0
          }
          hint={canReadCA ? `共 ${cas.length} 个证书颁发机构` : '无 CA 查看权限'}
          tone="info"
          icon={<Landmark size={19} />}
        />
        <StatCard
          label="待签发"
          value={
            canReadCertificates
              ? loading
                ? '-'
                : certificates.filter(item => item.status === 'pending').length
              : 0
          }
          hint={canReadCertificates ? '等待管理员审批' : '无证书查看权限'}
          tone="info"
          icon={<Clock3 size={19} />}
        />
        <StatCard
          label="即将过期"
          value={canReadCertificates ? (loading ? '-' : expiring.length) : 0}
          hint={
            canReadCertificates ? `未来 ${expiryNotificationDays} 天内到期` : '无证书查看权限'
          }
          tone="warning"
          icon={<ShieldCheck size={19} />}
        />
      </section>

      <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_440px]">
        <div className="zl-surface-3d rounded-xl border p-5">
          <div className="mb-5 flex items-start justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">证书颁发趋势</h2>
              <p className="mt-1 text-xs text-[var(--zl-text-muted)]">近 6 个月签发与撤销变化</p>
            </div>
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-blue-500/10 text-blue-500">
              <TrendingUp size={17} />
            </span>
          </div>
          {canReadCertificates ? (
            <QueryState
              loading={trendQuery.isLoading}
              error={trendQuery.isError}
              onRetry={() => void trendQuery.refetch()}
            >
              <div className="h-60">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={trendQuery.data?.items ?? []} margin={{ left: -18, right: 8 }}>
                    <defs>
                      <linearGradient id="issuedGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.38} />
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      stroke="var(--zl-border)"
                      strokeDasharray="3 3"
                      vertical={false}
                    />
                    <XAxis
                      dataKey="month"
                      tick={{ fill: 'var(--zl-text-muted)', fontSize: 11 }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      allowDecimals={false}
                      tick={{ fill: 'var(--zl-text-muted)', fontSize: 11 }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <Tooltip
                      contentStyle={{
                        background: 'var(--zl-popover-bg)',
                        border: '1px solid var(--zl-popover-border)',
                        borderRadius: 10,
                      }}
                    />
                    <Area
                      type="monotone"
                      dataKey="issued"
                      name="签发"
                      stroke="#3b82f6"
                      fill="url(#issuedGradient)"
                      strokeWidth={2}
                    />
                    <Area
                      type="monotone"
                      dataKey="revoked"
                      name="撤销"
                      stroke="#ef4444"
                      fill="transparent"
                      strokeWidth={2}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </QueryState>
          ) : (
            <EmptyState title="暂无证书趋势数据" description="需要证书查看权限" />
          )}
        </div>

        <div className="zl-surface-3d rounded-xl border p-5">
          <div className="mb-3 flex items-center justify-between">
            <div>
              <h2 className="text-sm font-semibold">证书状态分布</h2>
              <p className="mt-1 text-xs text-[var(--zl-text-muted)]">当前证书生命周期状态</p>
            </div>
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-emerald-500/10 text-emerald-500">
              <BadgeCheck size={17} />
            </span>
          </div>
          <CertificateStatusDistribution
            total={canReadCertificates ? certificates.length : 0}
            items={statusData}
          />
        </div>
      </section>

      <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_440px]">
        <div className="zl-surface-3d flex h-[500px] min-h-0 flex-col overflow-hidden rounded-xl border">
          <div className="flex h-16 shrink-0 items-center justify-between border-b border-[var(--zl-border)] px-5 py-4">
            <h2 className="text-sm font-semibold">最近证书</h2>
            <div className="flex items-center gap-3">
              <Select value={recentCertificateLimit} onValueChange={setRecentCertificateLimit}>
                <SelectTrigger
                  className="h-8 w-[86px] px-2 text-xs font-normal"
                  aria-label="选择最近证书条数"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end" className="min-w-0">
                  <SelectItem value="10" className="text-xs font-normal">
                    10 条
                  </SelectItem>
                  <SelectItem value="20" className="text-xs font-normal">
                    20 条
                  </SelectItem>
                  <SelectItem value="30" className="text-xs font-normal">
                    30 条
                  </SelectItem>
                </SelectContent>
              </Select>
              <span className="grid h-8 w-8 place-items-center rounded-lg bg-violet-500/10 text-violet-500">
                <FileClock size={16} />
              </span>
            </div>
          </div>
          {!canReadCertificates || certificates.length === 0 ? (
            <EmptyState title="尚无证书记录" />
          ) : (
            <div className="zl-hidden-scrollbar min-h-0 flex-1 divide-y divide-[var(--zl-border)] overflow-y-auto">
              {certificates.slice(0, Number(recentCertificateLimit)).map(item => (
                <div key={item.id} className="flex items-center gap-4 px-5 py-2">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{item.commonName}</p>
                    <p className="mt-0.5 truncate font-mono text-xs text-[var(--zl-text-muted)]">
                      {item.serialNumber || '等待签发'}
                    </p>
                  </div>
                  <time className="hidden text-xs text-[var(--zl-text-muted)] sm:block">
                    {formatDate(item.notAfter)}
                  </time>
                  <span className="flex w-[76px] shrink-0 justify-center">
                    <StatusBadge status={item.status} />
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="zl-surface-3d flex h-[500px] min-h-0 flex-col overflow-hidden rounded-xl border">
          <div className="flex h-16 shrink-0 items-center justify-between border-b border-[var(--zl-border)] px-5 py-4">
            <h2 className="text-sm font-semibold">近期审计活动</h2>
            <div className="flex items-center gap-3">
              <Select value={auditLimit} onValueChange={setAuditLimit}>
                <SelectTrigger
                  className="h-8 w-[86px] px-2 text-xs font-normal"
                  aria-label="选择审计活动条数"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end" className="min-w-0">
                  <SelectItem value="10" className="text-xs font-normal">
                    10 条
                  </SelectItem>
                  <SelectItem value="20" className="text-xs font-normal">
                    20 条
                  </SelectItem>
                  <SelectItem value="30" className="text-xs font-normal">
                    30 条
                  </SelectItem>
                </SelectContent>
              </Select>
              <span className="grid h-8 w-8 place-items-center rounded-lg bg-blue-500/10 text-blue-500">
                <Activity size={16} />
              </span>
            </div>
          </div>
          {canReadAudit ? (
            <QueryState
              loading={auditQuery.isLoading}
              error={auditQuery.isError}
              onRetry={() => void auditQuery.refetch()}
            >
              {(auditQuery.data?.items ?? []).length === 0 ? (
                <EmptyState title="暂无审计活动" />
              ) : (
                <div className="zl-hidden-scrollbar min-h-0 flex-1 divide-y divide-[var(--zl-border)] overflow-y-auto">
                  {(auditQuery.data?.items ?? []).map(item => (
                    <div key={item.id} className="px-5 py-2">
                      <div className="flex items-start justify-between gap-3">
                        <span className="min-w-0 truncate text-sm font-medium">{item.action}</span>
                        <span className="flex shrink-0 items-center gap-2">
                          <time className="text-xs text-[var(--zl-text-muted)]">
                            {formatDate(item.timestamp)}
                          </time>
                          <StatusBadge status={item.result} />
                        </span>
                      </div>
                      <p className="mt-0.5 truncate text-xs text-[var(--zl-text-muted)]">
                        {item.target} · {item.username}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </QueryState>
          ) : (
            <EmptyState title="暂无审计活动" description="需要审计查看权限" />
          )}
        </div>
      </section>
    </div>
  );
}

function CertificateStatusDistribution({
  total,
  items,
}: {
  total: number;
  items: CertificateStatusDatum[];
}) {
  const chartId = useId().replace(/:/g, '');
  const [activeIndex, setActiveIndex] = useState<number | undefined>();
  const chartItems = items.filter(item => item.value > 0);
  const hasData = total > 0 && chartItems.length > 0;

  return (
    <div className="mt-4 grid items-center gap-4 md:grid-cols-[12rem_minmax(0,1fr)]">
      <div className="relative mx-auto h-48 w-full max-w-48">
        <div className="pointer-events-none absolute inset-x-7 bottom-3 h-8 rounded-full bg-blue-500/20 blur-xl" />
        {hasData ? (
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <defs>
                <filter id={`${chartId}-shadow`} x="-35%" y="-35%" width="170%" height="170%">
                  <feDropShadow
                    dx="0"
                    dy="8"
                    stdDeviation="6"
                    floodColor="#1e3a8a"
                    floodOpacity="0.22"
                  />
                </filter>
                {chartItems.map((item, index) => (
                  <linearGradient
                    key={item.name}
                    id={`${chartId}-slice-${index}`}
                    x1="0"
                    y1="0"
                    x2="1"
                    y2="1"
                  >
                    <stop offset="0%" stopColor={item.color} stopOpacity="0.82" />
                    <stop offset="100%" stopColor={item.color} />
                  </linearGradient>
                ))}
              </defs>
              <Pie
                data={chartItems}
                dataKey="value"
                nameKey="name"
                innerRadius={54}
                outerRadius={78}
                paddingAngle={3}
                stroke="var(--zl-card)"
                strokeWidth={3}
                filter={`url(#${chartId}-shadow)`}
                onMouseEnter={(_, index) => setActiveIndex(index)}
                onMouseLeave={() => setActiveIndex(undefined)}
              >
                {chartItems.map((item, index) => (
                  <Cell
                    key={item.name}
                    fill={`url(#${chartId}-slice-${index})`}
                    opacity={activeIndex === undefined || activeIndex === index ? 1 : 0.28}
                    style={{ cursor: 'pointer', transition: 'opacity 180ms ease' }}
                  />
                ))}
              </Pie>
              <Tooltip cursor={false} content={<CertificateStatusTooltip />} />
            </PieChart>
          </ResponsiveContainer>
        ) : (
          <div className="absolute inset-5 grid place-items-center rounded-full border border-[var(--zl-border)] bg-[var(--zl-control-bg)]/50 px-4 text-center text-sm text-[var(--zl-text-muted)]">
            暂无状态数据
          </div>
        )}
        {hasData ? (
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1 text-center">
            <strong className="font-mono text-2xl tabular-nums">{total}</strong>
            <span className="text-xs text-[var(--zl-text-muted)]">总数</span>
          </div>
        ) : null}
      </div>
      <div className="space-y-2">
        {items.map(item => {
          const percentage = total > 0 ? Math.round((item.value / total) * 100) : 0;
          return (
            <div
              key={item.name}
              className="flex items-center justify-between gap-3 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg)]/45 px-3 py-2.5 text-sm"
            >
              <span className="flex min-w-0 items-center gap-2 text-[var(--zl-text-muted)]">
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{ background: item.color, boxShadow: `0 0 14px ${item.color}66` }}
                />
                <span className="truncate">{item.name}</span>
              </span>
              <span className="grid shrink-0 grid-cols-[2.5rem_2rem] items-baseline gap-2 font-mono tabular-nums text-right">
                <strong className="text-right">{item.value}</strong>
                <small className="text-right text-xs text-[var(--zl-text-muted)]">
                  {percentage}%
                </small>
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function CertificateStatusTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{
    payload?: CertificateStatusDatum;
    color?: string;
    name?: string;
    value?: number;
  }>;
}) {
  const item = payload?.[0];
  if (!active || !item) return null;

  const status = item.payload;
  const color = status?.color ?? item.color ?? 'var(--zl-accent)';
  const name = status?.name ?? item.name ?? '';
  const value = status?.value ?? item.value ?? 0;

  return (
    <div
      className="rounded-lg border px-3 py-2 text-sm shadow-2xl"
      style={{
        background: 'var(--zl-popover-bg)',
        borderColor: 'var(--zl-popover-border)',
        color: 'var(--zl-text)',
        boxShadow: 'var(--zl-menu-shadow)',
      }}
    >
      <div className="flex items-center gap-2">
        <span
          className="h-2.5 w-2.5 shrink-0 rounded-full"
          style={{ background: color, boxShadow: `0 0 12px ${color}66` }}
        />
        <span className="font-semibold">{name}</span>
        <span className="font-mono font-semibold tabular-nums">{value} 张</span>
      </div>
    </div>
  );
}
