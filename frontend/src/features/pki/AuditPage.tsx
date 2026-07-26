import { useEffect, useMemo, useState } from 'react';
import { CheckCircle2, Download, Search, XCircle } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { DataTableShell } from '@/components/data-table-shell';
import { EmptyState } from '@/components/empty-state';
import { PageHeader } from '@/components/page-header';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { AuditEntry } from '@/lib/pki';
import { getAuditEntries } from '@/lib/pki';
import { AuditDetailsDialog } from './AuditDetailsDialog';
import { AuditExportDialog } from './AuditExportDialog';
import { Pagination } from './Pagination';
import { pageSizeOptions } from './pagination-options';
import { QueryState, SearchBox } from './shared';
import { formatDate, pkiQueryKeys, usePageRefresh } from './support';

const auditKeys = [pkiQueryKeys.audits] as const;
const moduleLabels: Record<string, string> = {
  auth: '身份认证',
  pki: 'CA 管理',
  certificates: '证书管理',
  workflows: '审批管理',
  settings: '系统配置',
};
const auditModules = Object.entries(moduleLabels);
const moduleLabel = (module: string) => moduleLabels[module] ?? module;

export function AuditPage() {
  const canManage = userHasPermission(getStoredUser(), 'audit.manage');
  usePageRefresh(auditKeys);
  const query = useQuery({ queryKey: pkiQueryKeys.audits, queryFn: () => getAuditEntries(500) });
  const [search, setSearch] = useState('');
  const [module, setModule] = useState('all');
  const [result, setResult] = useState('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<(typeof pageSizeOptions)[number]>(10);
  const [detailItem, setDetailItem] = useState<AuditEntry | null>(null);
  const [exportOpen, setExportOpen] = useState(false);
  const items = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return (query.data?.items ?? []).filter(
      item =>
        (module === 'all' || item.module === module) &&
        (result === 'all' || item.result === result) &&
        (!keyword ||
          `${item.timestamp} ${item.username} ${moduleLabel(item.module)} ${item.action} ${item.target} ${item.ip} ${item.result === 'success' ? '成功' : '失败'} ${item.detail}`
            .toLowerCase()
            .includes(keyword))
    );
  }, [module, query.data?.items, result, search]);
  const pageCount = Math.max(1, Math.ceil(items.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const pageItems = items.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  useEffect(() => setPage(1), [module, pageSize, result, search]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (items.length === 0) {
      toast.error('没有可导出的审计日志');
      return;
    }
    setExportOpen(true);
  }

  return (
    <div className="certflow-page-fill">
      <PageHeader
        title="审计日志"
        description="追踪身份认证、CA、证书、审批与系统配置等安全事件操作"
      />
      <DataTableShell
        className="certflow-page-card"
        contentClassName="certflow-scroll-area flex-1"
        footer={
          items.length > 0 ? (
            <Pagination
              page={currentPage}
              pageCount={pageCount}
              pageSize={pageSize}
              total={items.length}
              onPageChange={setPage}
              onPageSizeChange={setPageSize}
            />
          ) : null
        }
        toolbar={
          <>
            <SearchBox
              value={search}
              onChange={setSearch}
              placeholder="搜索审计日志"
              className="min-w-64 flex-1"
            />
            <div className="w-36">
              <Select value={module} onValueChange={setModule}>
                <SelectTrigger className="font-normal">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部模块</SelectItem>
                  {auditModules.map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-36">
              <Select value={result} onValueChange={setResult}>
                <SelectTrigger className="font-normal">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部结果</SelectItem>
                  <SelectItem value="success">成功</SelectItem>
                  <SelectItem value="failure">失败</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {canManage ? (
              <Button
                variant="outline"
                className="ml-auto border-blue-400/50 text-blue-600 hover:border-blue-500/70 hover:bg-blue-500/10 dark:text-blue-400"
                onClick={openExport}
              >
                <Download size={16} />
                导出
              </Button>
            ) : null}
          </>
        }
      >
        <QueryState
          loading={query.isLoading}
          error={query.isError}
          onRetry={() => void query.refetch()}
        >
          {items.length === 0 ? (
            <EmptyState title="没有匹配的审计日志" />
          ) : (
            <div className="min-w-[1200px] divide-y divide-[var(--zl-border)]">
              <div className="sticky top-0 z-10 grid grid-cols-[160px_120px_120px_150px_minmax(150px,1fr)_140px_100px_100px] gap-4 bg-[var(--zl-table-head-bg)] px-4 py-3 text-center text-xs font-semibold text-[var(--zl-text-muted)]">
                <span>时间</span>
                <span>用户</span>
                <span>模块</span>
                <span>操作</span>
                <span>目标</span>
                <span>来源 IP</span>
                <span>结果</span>
                <span>详情</span>
              </div>
              {pageItems.map(item => (
                <div
                  key={item.id}
                  className="grid grid-cols-[160px_120px_120px_150px_minmax(150px,1fr)_140px_100px_100px] items-center gap-4 px-4 py-3.5 text-sm hover:bg-[var(--zl-control-bg-soft)]"
                >
                  <time className="justify-self-center text-sm font-medium text-[var(--zl-text)]">
                    {formatDate(item.timestamp)}
                  </time>
                  <span className="justify-self-center truncate font-medium">
                    {item.username || '-'}
                  </span>
                  <span
                    className={`justify-self-center truncate rounded-full px-2.5 py-1 text-xs font-medium ${moduleBadgeClassName(item.module)}`}
                  >
                    {moduleLabel(item.module)}
                  </span>
                  <span className="justify-self-center truncate font-medium">{item.action}</span>
                  <span className="justify-self-center truncate font-medium text-emerald-700 dark:text-emerald-300">
                    {item.target || '-'}
                  </span>
                  <span className="justify-self-center truncate font-medium">{item.ip || '-'}</span>
                  <AuditResultBadge result={item.result} />
                  <Button
                    variant="ghost"
                    size="sm"
                    className="justify-self-center border border-blue-400/35 bg-blue-400/10 px-2 text-blue-700 hover:border-blue-400/55 hover:bg-blue-400/18 dark:text-blue-300"
                    onClick={() => setDetailItem(item)}
                    aria-label="查看审计详情"
                  >
                    <Search size={15} />
                    查看
                  </Button>
                </div>
              ))}
            </div>
          )}
        </QueryState>
      </DataTableShell>
      <AuditDetailsDialog
        item={detailItem}
        moduleLabel={moduleLabel}
        onOpenChange={open => !open && setDetailItem(null)}
      />
      <AuditExportDialog
        open={exportOpen}
        items={items}
        moduleLabel={moduleLabel}
        onOpenChange={setExportOpen}
      />
    </div>
  );
}

function moduleBadgeClassName(module: string) {
  return (
    {
      auth: 'bg-sky-500/12 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300',
      pki: 'bg-violet-500/12 text-violet-700 dark:bg-violet-400/15 dark:text-violet-300',
      certificates: 'bg-blue-500/12 text-blue-700 dark:bg-blue-400/15 dark:text-blue-300',
      workflows: 'bg-amber-500/12 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300',
      settings: 'bg-slate-500/12 text-slate-700 dark:bg-slate-400/15 dark:text-slate-300',
    }[module] ?? 'bg-slate-500/12 text-slate-700 dark:bg-slate-400/15 dark:text-slate-300'
  );
}

function AuditResultBadge({ result }: { result: AuditEntry['result'] }) {
  const success = result === 'success';
  const Icon = success ? CheckCircle2 : XCircle;
  return (
    <span
      className={`inline-flex items-center justify-self-center gap-1 rounded-full border px-2 py-0.5 text-xs ${success ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-600 dark:text-emerald-300' : 'border-red-400/30 bg-red-400/10 text-red-600 dark:text-red-300'}`}
    >
      <Icon size={12} />
      {success ? '成功' : '失败'}
    </span>
  );
}
