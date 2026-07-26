import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Download,
  FileText,
  FileWarning,
  Fingerprint,
  RefreshCw,
  ShieldCheck,
  TimerReset,
} from 'lucide-react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { DataTableShell } from '@/components/data-table-shell';
import { EmptyState } from '@/components/empty-state';
import { PageHeader } from '@/components/page-header';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  downloadBlob,
  downloadCRL,
  getCAs,
  getCRLEntries,
  getCRLMetadata,
  type CRLFilters,
} from '@/lib/pki';
import { QueryState, SearchBox } from './shared';
import { CRLExportDialog } from './CRLExportDialog';
import { Pagination } from './Pagination';
import { pageSizeOptions } from './pagination-options';
import { formatDate, pkiQueryKeys, revokeReasonLabels, usePageRefresh } from './support';

const crlKeys = [pkiQueryKeys.crl, pkiQueryKeys.crlMetadata, pkiQueryKeys.cas] as const;

export function CRLPage() {
  const canManage = userHasPermission(getStoredUser(), 'crl.manage');
  usePageRefresh(crlKeys);
  const [search, setSearch] = useState('');
  const [caId, setCAId] = useState('all');
  const [reason, setReason] = useState('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<(typeof pageSizeOptions)[number]>(10);
  const [downloading, setDownloading] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const filters = useMemo<CRLFilters>(
    () => ({
      caId: caId === 'all' ? undefined : caId,
      reason: reason === 'all' ? undefined : (reason as CRLFilters['reason']),
      keyword: search.trim() || undefined,
      page,
      pageSize,
    }),
    [caId, page, pageSize, reason, search]
  );
  const metadataFilters = useMemo<CRLFilters>(
    () => ({ caId: filters.caId, reason: filters.reason, keyword: filters.keyword }),
    [filters.caId, filters.keyword, filters.reason]
  );
  const entryQuery = useQuery({
    queryKey: [...pkiQueryKeys.crl, filters],
    queryFn: () => getCRLEntries(filters),
    placeholderData: previousData => previousData,
  });
  const metadataQuery = useQuery({
    queryKey: [...pkiQueryKeys.crlMetadata, metadataFilters],
    queryFn: () => getCRLMetadata(metadataFilters),
  });
  const canReadCA = userHasPermission(getStoredUser(), 'ca.read');
  const caQuery = useQuery({ queryKey: pkiQueryKeys.cas, queryFn: getCAs, enabled: canReadCA });
  const items = entryQuery.data?.items ?? [];
  const total = entryQuery.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const currentPage = Math.min(page, pageCount);
  useEffect(() => setPage(1), [caId, pageSize, reason, search]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  async function handleDownload() {
    setDownloading(true);
    try {
      const download = await downloadCRL(metadataFilters);
      const suffix = metadataFilters.caId ? `-ca-${metadataFilters.caId}` : '-all';
      const fallback = `pki-crl${suffix}-${new Date().toISOString().slice(0, 10)}${metadataFilters.caId ? '.crl' : '.zip'}`;
      downloadBlob(download.blob, download.filename || fallback);
      toast.success('CRL 文件已下载');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'CRL 下载失败');
    } finally {
      setDownloading(false);
    }
  }

  function handleExport() {
    if (items.length === 0) {
      toast.error('没有可导出的 CRL 记录');
      return;
    }
    setExportOpen(true);
  }

  async function handleRefresh() {
    await Promise.all([entryQuery.refetch(), metadataQuery.refetch()]);
  }

  const metadata = metadataQuery.isFetching ? undefined : metadataQuery.data;
  const toolbar = (
    <>
      <SearchBox
        value={search}
        onChange={setSearch}
        placeholder="可搜索通用名称、序列号等信息"
        className="min-w-64 flex-1"
      />
      <div className="w-44">
        <Select value={caId} onValueChange={setCAId}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部 CA</SelectItem>
            {(caQuery.data?.items ?? []).map(item => (
              <SelectItem key={item.id} value={item.id}>
                {item.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="w-44">
        <Select value={reason} onValueChange={setReason}>
          <SelectTrigger className="font-normal">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部原因</SelectItem>
            {Object.entries(revokeReasonLabels).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
        {canManage ? (
          <Button
            variant="outline"
            className="border-blue-400/50 bg-blue-500/10 text-blue-600 hover:border-blue-500/70 hover:bg-blue-500/15 dark:text-blue-400"
            onClick={handleExport}
          >
            <FileText size={15} />
            导出
          </Button>
        ) : null}
        {canManage ? (
          <>
            <Button
              variant="outline"
              onClick={() => void handleDownload()}
              disabled={downloading}
              aria-label="下载 CRL 文件"
              className="border-amber-400/50 bg-amber-400/10 text-amber-700 hover:border-amber-400/70 hover:bg-amber-400/20 disabled:opacity-100 dark:text-amber-300"
            >
              <Download size={15} />
              {downloading ? '正在下载...' : '下载 CRL 文件'}
            </Button>
            <Button
              className="border-emerald-400/30 bg-emerald-400/10 text-emerald-600 hover:bg-emerald-400/20 dark:text-emerald-400"
              onClick={() => void handleRefresh()}
              disabled={entryQuery.isFetching || metadataQuery.isFetching}
            >
              <RefreshCw
                size={15}
                className={
                  entryQuery.isFetching || metadataQuery.isFetching ? 'animate-spin' : undefined
                }
              />
              {entryQuery.isFetching || metadataQuery.isFetching ? '更新中...' : '更新 CRL'}
            </Button>
          </>
        ) : null}
      </div>
    </>
  );
  return (
    <div className="certflow-page-fill">
      <PageHeader title="证书撤销" description="CRL 管理与已撤销证书列表" />
      <section className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <RevocationStat
          label="服务状态"
          value={metadata ? (metadata.enabled ? '已启用' : '未启用') : '-'}
          hint={metadata ? metadata.crlUrl || '未配置 CRL 分发地址' : '等待元数据'}
          tone={metadata?.enabled ? 'success' : 'warning'}
          icon={<ShieldCheck size={17} />}
        />
        <RevocationStat
          label="CRL 版本"
          value={metadata?.version ?? '-'}
          hint={metadata ? `${metadata.entryCount} 条撤销条目` : '等待元数据'}
          tone="info"
          icon={<FileWarning size={18} />}
        />
        <RevocationStat
          label="签名算法"
          value={metadata?.signatureAlgorithm ?? '-'}
          hint={caId === 'all' ? '选择 CA 后查看实际签名算法' : '当前 CA 的 CRL 签名算法'}
          tone="success"
          icon={<Fingerprint size={17} />}
        />
        <RevocationStat
          label="此次更新"
          value={metadata ? formatDate(metadata.thisUpdate, true) : '-'}
          hint={metadata ? `下次更新 ${formatDate(metadata.nextUpdate, true)}` : '等待元数据'}
          tone="info"
          icon={<RefreshCw size={17} />}
        />
        <RevocationStat
          label="更新间隔"
          value={metadata ? `${metadata.intervalHours} 小时` : '-'}
          hint="系统配置中的 CRL 生成周期"
          tone="info"
          icon={<TimerReset size={18} />}
        />
      </section>
      <DataTableShell
        className="certflow-page-card"
        contentClassName="certflow-scroll-area flex-1"
        toolbar={toolbar}
        footer={
          total > 0 ? (
            <Pagination
              page={currentPage}
              pageCount={pageCount}
              pageSize={pageSize}
              total={total}
              onPageChange={setPage}
              onPageSizeChange={setPageSize}
            />
          ) : null
        }
      >
        <QueryState
          loading={entryQuery.isLoading}
          error={entryQuery.isError}
          onRetry={() => void entryQuery.refetch()}
        >
          {items.length === 0 ? (
            <EmptyState title="没有撤销记录" description="证书被撤销后会自动出现在这里" />
          ) : (
            <div className="min-w-[1050px] divide-y divide-[var(--zl-border)]">
              <div className="grid grid-cols-[minmax(120px,1fr)_minmax(185px,1.2fr)_minmax(150px,1fr)_120px_110px_minmax(260px,1.5fr)] gap-4 bg-[var(--zl-table-head-bg)] px-4 py-3 text-center text-xs font-semibold text-[var(--zl-text-muted)]">
                <span>通用名称</span>
                <span>序列号</span>
                <span>撤销时间</span>
                <span>撤销原因</span>
                <span>所属 CA</span>
                <span>CA ID</span>
              </div>
              {items.map(item => (
                <div
                  key={item.id}
                  className="grid grid-cols-[minmax(120px,1fr)_minmax(185px,1.2fr)_minmax(150px,1fr)_120px_110px_minmax(260px,1.5fr)] items-center gap-4 px-4 py-3.5 text-center text-sm hover:bg-[var(--zl-control-bg-soft)]"
                >
                  <div className="min-w-0">
                    <p className="truncate font-medium">{item.commonName}</p>
                  </div>
                  <span className="truncate font-mono text-xs text-red-500">
                    {item.serialNumber}
                  </span>
                  <time className="text-xs text-[var(--zl-text-muted)]">
                    {formatDate(item.revokedAt)}
                  </time>
                  <span className="justify-self-center rounded-full border border-red-400/30 bg-red-400/10 px-2.5 py-1 text-xs text-red-500">
                    {revokeReasonLabels[item.reason] ?? item.reason}
                  </span>
                  <span className="justify-self-center rounded-full border border-blue-400/30 bg-blue-400/10 px-2.5 py-1 text-xs text-blue-600 dark:text-blue-300">
                    {item.caName}
                  </span>
                  <AppTooltip label={item.caId} className="min-w-0">
                    <span className="block truncate font-mono text-xs text-[var(--zl-text-muted)]">
                      {item.caId}
                    </span>
                  </AppTooltip>
                </div>
              ))}
            </div>
          )}
        </QueryState>
      </DataTableShell>
      <CRLExportDialog open={exportOpen} filteredItems={items} onOpenChange={setExportOpen} />
    </div>
  );
}

function RevocationStat({
  label,
  value,
  hint,
  icon,
  tone = 'info',
}: {
  label: string;
  value: string | number;
  hint: string;
  icon: ReactNode;
  tone?: 'info' | 'danger' | 'success' | 'warning';
}) {
  const colors = {
    info: 'bg-cyan-500/10 text-cyan-500',
    danger: 'bg-red-500/10 text-red-500',
    success: 'bg-emerald-500/10 text-emerald-500',
    warning: 'bg-amber-500/10 text-amber-500',
  };
  return (
    <div className="zl-surface-3d zl-card-hover min-h-[104px] rounded-lg px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs font-medium text-[var(--zl-text-muted)]">{label}</p>
          <p className="mt-1 text-base font-semibold tracking-tight">{value}</p>
          <p className="mt-1 text-xs text-[var(--zl-text-muted)]">{hint}</p>
        </div>
        <span className={`grid h-9 w-9 place-items-center rounded-lg ${colors[tone]}`}>{icon}</span>
      </div>
    </div>
  );
}
