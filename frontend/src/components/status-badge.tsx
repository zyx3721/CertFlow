import { cn } from '@/lib/utils';

const labels: Record<string, string> = {
  active: '启用',
  inactive: '停用',
  pending: '待审批',
  approved: '已通过',
  valid: '有效',
  revoked: '已撤销',
  rejected: '已驳回',
  expired: '已过期',
  success: '成功',
  failure: '失败',
};

export function StatusBadge({ status }: { status: string }) {
  const tone =
    status === 'valid' || status === 'active' || status === 'approved' || status === 'success'
      ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-500'
      : status === 'pending'
        ? 'border-sky-400/30 bg-sky-400/10 text-sky-500'
        : status === 'revoked' || status === 'rejected' || status === 'failure'
          ? 'border-red-400/30 bg-red-400/10 text-red-500'
          : 'border-amber-400/30 bg-amber-400/10 text-amber-500';
  return (
    <span className={cn('inline-flex rounded-full border px-2.5 py-1 text-xs font-medium', tone)}>
      {labels[status] ?? status}
    </span>
  );
}
