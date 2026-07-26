import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, Search, Trash2, XCircle } from 'lucide-react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { EmptyState } from '@/components/empty-state';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { deleteWorkflow, getWorkflows, updateWorkflow, type Workflow } from '@/lib/pki';
import { QueryState } from './shared';
import { Pagination } from './Pagination';
import { pageSizeOptions } from './pagination-options';
import { formatDate, pkiQueryKeys, usePageRefresh } from './support';

const workflowKeys = [pkiQueryKeys.workflows] as const;
type WorkflowFilter = 'all' | Workflow['status'];

const filterDefinitions: Array<{ value: WorkflowFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已通过' },
  { value: 'rejected', label: '已驳回' },
];
const workflowStatusLabels: Record<Workflow['status'], string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已驳回',
};

export function WorkflowsPage() {
  const user = getStoredUser();
  const canApprove = userHasPermission(user, 'workflows.approve');
  const canDelete = userHasPermission(user, 'workflows.delete');
  const canOperate = canApprove || canDelete;
  usePageRefresh(workflowKeys);
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: pkiQueryKeys.workflows, queryFn: getWorkflows });
  const [filter, setFilter] = useState<WorkflowFilter>('all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<(typeof pageSizeOptions)[number]>(10);
  const [approveTarget, setApproveTarget] = useState<Workflow | null>(null);
  const [rejectTarget, setRejectTarget] = useState<Workflow | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<Workflow | null>(null);
  const workflows = useMemo(() => query.data?.items ?? [], [query.data?.items]);

  const counts = useMemo(
    () => ({
      all: workflows.length,
      pending: workflows.filter(item => item.status === 'pending').length,
      approved: workflows.filter(item => item.status === 'approved').length,
      rejected: workflows.filter(item => item.status === 'rejected').length,
    }),
    [workflows]
  );
  const filteredItems = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return workflows.filter(item => {
      if (filter !== 'all' && item.status !== filter) return false;
      if (!keyword) return true;
      return [
        formatDate(item.createdAt),
        item.id,
        item.id.slice(0, 8),
        item.applicant,
        item.certName,
        item.status,
        workflowStatusLabels[item.status],
        item.rejectReason,
      ]
        .join(' ')
        .toLowerCase()
        .includes(keyword);
    });
  }, [filter, search, workflows]);
  const pageCount = Math.max(1, Math.ceil(filteredItems.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const visibleItems = filteredItems.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  useEffect(() => {
    setPage(1);
  }, [filter, search, pageSize]);

  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.workflows });
    void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.certificates });
  };
  const mutation = useMutation({
    mutationFn: ({
      id,
      status,
      reason,
    }: {
      id: string;
      status: 'approved' | 'rejected';
      reason?: string;
    }) => updateWorkflow(id, { status, reject_reason: reason }),
    onSuccess: (_, variables) => {
      refresh();
      setApproveTarget(null);
      setRejectTarget(null);
      setRejectReason('');
      toast.success(variables.status === 'approved' ? '证书申请已审批通过' : '证书申请已驳回');
    },
    onError: error => toast.error(error instanceof Error ? error.message : '审批处理失败'),
  });
  const deleteMutation = useMutation({
    mutationFn: deleteWorkflow,
    onSuccess: () => {
      refresh();
      setDeleteTarget(null);
      toast.success('待审批记录已删除');
    },
    onError: error => toast.error(error instanceof Error ? error.message : '审批记录删除失败'),
  });

  function openRejectDialog(item: Workflow) {
    setRejectReason('');
    setRejectTarget(item);
  }

  function closeRejectDialog() {
    setRejectTarget(null);
    setRejectReason('');
  }

  return (
    <div className="certflow-page-fill">
      <PageHeader title="审批管理" description="处理证书申请，批准后由系统自动签发并写入审计日志" />
      <section className="zl-surface-3d certflow-page-card flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border">
        <div className="shrink-0 border-b border-[var(--zl-border)] p-3">
          <div className="flex flex-wrap items-center gap-3">
            <label className="relative min-w-[240px] flex-1">
              <Search
                size={15}
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--zl-text-muted)]"
              />
              <input
                value={search}
                onChange={event => setSearch(event.target.value)}
                placeholder="可搜索提交时间、申请编号、申请人等信息"
                className="zl-form-control h-10 w-full rounded-lg py-2 pl-9 pr-3 text-sm"
              />
            </label>
            <div
              className="flex w-fit max-w-full items-center gap-1 overflow-x-auto rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/70 p-1"
              aria-label="审批状态筛选"
            >
              {filterDefinitions.map(item => {
                const active = filter === item.value;
                return (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => setFilter(item.value)}
                    className={`shrink-0 rounded-md px-3 py-2 text-xs font-medium transition-all duration-200 ${
                      active
                        ? 'border border-blue-500/30 bg-blue-500/10 text-blue-700 shadow-sm dark:text-blue-300'
                        : 'border border-transparent text-[var(--zl-text-muted)] hover:border-blue-500/35 hover:bg-blue-500/[0.07] hover:text-blue-600 dark:hover:text-blue-300'
                    }`}
                  >
                    {item.label} ({counts[item.value]})
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        <div className="certflow-scroll-area min-h-0 flex-1 overflow-x-auto">
          <QueryState
            loading={query.isLoading}
            error={query.isError}
            onRetry={() => void query.refetch()}
          >
            {filteredItems.length === 0 ? (
              <EmptyState title={search ? '未找到匹配的审批记录' : '当前没有审批记录'} />
            ) : (
              <table className="w-full min-w-[980px] table-fixed">
                <colgroup>
                  <col className="w-[18%]" />
                  <col className="w-[15%]" />
                  <col className="w-[15%]" />
                  <col className="w-[15%]" />
                  <col className="w-[15%]" />
                  <col className="w-[22%]" />
                </colgroup>
                <thead className="sticky top-0 z-10 bg-[var(--zl-table-head-bg)]">
                  <tr className="border-b border-[var(--zl-border)]">
                    {['提交时间', '申请编号', '申请人', '通用名称', '状态', '操作'].map(label => (
                      <th
                        key={label}
                        className="px-4 py-3 text-center text-xs font-semibold text-[var(--zl-text-muted)]"
                      >
                        {label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--zl-border)]">
                  {visibleItems.map(item => (
                    <tr
                      key={item.id}
                      className="transition-colors hover:bg-[var(--zl-control-bg-soft)]/70"
                    >
                      <td className="px-4 py-3.5 text-center text-sm text-[var(--zl-text)]">
                        {formatDate(item.createdAt)}
                      </td>
                      <td className="px-4 py-3.5 text-center text-sm text-[var(--zl-text)]">
                        {item.id.slice(0, 8)}
                      </td>
                      <td className="px-4 py-3.5 text-center text-sm text-[var(--zl-text)]">
                        {item.applicant}
                      </td>
                      <td className="px-4 py-3.5 text-center">
                        <p className="truncate text-sm font-medium text-blue-600 dark:text-blue-400">
                          {item.certName}
                        </p>
                      </td>
                      <td className="px-4 py-3.5 text-center">
                        <StatusBadge status={item.status} />
                      </td>
                      <td className="px-4 py-3.5 text-center">
                        {canOperate ? (
                          <div className="flex justify-center gap-2">
                            {canApprove ? (
                              <AppTooltip
                                label="审批已处理，无法再次通过"
                                disabled={item.status === 'pending'}
                                className="inline-flex cursor-not-allowed rounded-lg"
                              >
                                <Button
                                  size="sm"
                                  variant="outline"
                                  className="border-emerald-500/30 text-emerald-600 hover:bg-emerald-500/10 dark:text-emerald-400"
                                  disabled={item.status !== 'pending'}
                                  onClick={() => setApproveTarget(item)}
                                >
                                  <CheckCircle2 size={14} />
                                  通过
                                </Button>
                              </AppTooltip>
                            ) : null}
                            {canApprove ? (
                              <AppTooltip
                                label="审批已处理，无法再次驳回"
                                disabled={item.status === 'pending'}
                                className="inline-flex cursor-not-allowed rounded-lg"
                              >
                                <Button
                                  size="sm"
                                  variant="destructive"
                                  disabled={item.status !== 'pending'}
                                  onClick={() => openRejectDialog(item)}
                                >
                                  <XCircle size={14} />
                                  驳回
                                </Button>
                              </AppTooltip>
                            ) : null}
                            {canDelete ? (
                              <AppTooltip
                                label="审批已处理，无法删除"
                                disabled={item.status === 'pending'}
                                className="inline-flex cursor-not-allowed rounded-lg"
                              >
                                <Button
                                  size="sm"
                                  variant="destructive"
                                  disabled={item.status !== 'pending'}
                                  onClick={() => setDeleteTarget(item)}
                                >
                                  <Trash2 size={14} />
                                  删除
                                </Button>
                              </AppTooltip>
                            ) : null}
                          </div>
                        ) : (
                          <span className="text-sm text-[var(--zl-text-muted)]">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </QueryState>
        </div>

        <Pagination
          total={filteredItems.length}
          page={currentPage}
          pageCount={pageCount}
          pageSize={pageSize}
          onPageSizeChange={setPageSize}
          onPageChange={setPage}
        />
      </section>

      {approveTarget ? (
        <ConfirmDialog
          open
          horizontalHeader
          tone="success"
          title="确认通过审批"
          description="通过后将签发该证书申请"
          detail={`确认通过申请编号 ${approveTarget.id.slice(0, 8)}，证书 ${approveTarget.certName} 吗？`}
          confirmText="确认通过"
          busy={mutation.isPending}
          onOpenChange={open => !open && setApproveTarget(null)}
          onConfirm={() => mutation.mutate({ id: approveTarget.id, status: 'approved' })}
        />
      ) : null}
      {rejectTarget ? (
        <Dialog open onOpenChange={open => !open && closeRejectDialog()}>
          <DialogContent className="zl-dialog-panel sm:max-w-md">
            <DialogHeader>
              <DialogTitle>驳回证书申请</DialogTitle>
              <DialogDescription>证书：{rejectTarget.certName}</DialogDescription>
            </DialogHeader>
            <label className="grid gap-2 text-sm font-medium text-[var(--zl-text)]">
              驳回原因
              <textarea
                rows={5}
                value={rejectReason}
                onChange={event => setRejectReason(event.target.value)}
                className="zl-form-control w-full resize-y rounded-xl px-3 py-3 text-sm font-normal"
                placeholder="请填写驳回原因，普通用户将在证书管理中看到该原因"
              />
            </label>
            <DialogFooter>
              <Button variant="outline" onClick={closeRejectDialog}>
                取消
              </Button>
              <Button
                variant="outline"
                className="border-red-500/40 bg-red-500/10 text-red-600 shadow-[0_1px_2px_rgba(239,68,68,0.12)] hover:border-red-500/55 hover:bg-red-500/18 hover:text-red-700 dark:border-red-400/35 dark:bg-red-400/10 dark:text-red-300 dark:hover:bg-red-400/18 dark:hover:text-red-200"
                disabled={mutation.isPending}
                onClick={() => {
                  if (!rejectReason.trim()) {
                    toast.error('请填写驳回原因');
                    return;
                  }
                  mutation.mutate({
                    id: rejectTarget.id,
                    status: 'rejected',
                    reason: rejectReason.trim(),
                  });
                }}
              >
                {mutation.isPending ? '正在处理...' : '确认驳回'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {deleteTarget ? (
        <ConfirmDialog
          open
          horizontalHeader
          icon="delete"
          softDestructive
          title="确认删除审批记录"
          description="该操作不可撤销，仅待审批状态允许删除"
          detail={`确认删除申请编号 ${deleteTarget.id.slice(0, 8)}，证书 ${deleteTarget.certName} 吗？`}
          confirmText="确认删除"
          busy={deleteMutation.isPending}
          onOpenChange={open => !open && setDeleteTarget(null)}
          onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        />
      ) : null}
    </div>
  );
}
