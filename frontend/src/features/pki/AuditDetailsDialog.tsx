import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { AuditEntry } from '@/lib/pki';
import { formatDate } from './support';

export function AuditDetailsDialog({
  item,
  moduleLabel,
  onOpenChange,
}: {
  item: AuditEntry | null;
  moduleLabel: (module: string) => string;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={item !== null} onOpenChange={onOpenChange}>
      <DialogContent className="zl-dialog-panel gap-0 overflow-hidden p-0 sm:max-w-[600px]">
        <DialogHeader className="border-b border-[var(--zl-border)] px-5 py-4">
          <DialogTitle>审计详情</DialogTitle>
          <DialogDescription>
            {item ? `${formatDate(item.timestamp, true)} · ${item.username || '-'}` : ''}
          </DialogDescription>
        </DialogHeader>
        {item ? (
          <div className="space-y-5 p-5">
            <div className="grid gap-4 sm:grid-cols-2">
              <DetailItem label="操作" value={item.action} />
              <DetailItem label="模块" value={moduleLabel(item.module)} />
              <DetailItem
                label="目标"
                value={item.target || '-'}
                valueClassName="text-emerald-700 dark:text-emerald-300"
              />
              <DetailItem label="来源 IP" value={item.ip || '-'} mono />
              <DetailItem
                label="结果"
                value={item.result === 'success' ? '成功' : '失败'}
                valueClassName={
                  item.result === 'success'
                    ? 'text-emerald-600 dark:text-emerald-300'
                    : 'text-red-600 dark:text-red-300'
                }
              />
              <DetailItem label="用户" value={item.username || '-'} />
            </div>
            <div className="rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/60 p-4">
              <p className="text-xs font-semibold text-[var(--zl-text-muted)]">详情</p>
              <p className="mt-2 break-words text-sm leading-6 text-[var(--zl-text)]">
                {item.detail || '无详情'}
              </p>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function DetailItem({
  label,
  value,
  mono = false,
  valueClassName = '',
}: {
  label: string;
  value: string;
  mono?: boolean;
  valueClassName?: string;
}) {
  return (
    <div className="min-w-0 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg)] px-3 py-2.5">
      <p className="text-xs text-[var(--zl-text-muted)]">{label}</p>
      <p
        className={`mt-1 truncate text-sm ${valueClassName || 'text-[var(--zl-text)]'}${mono ? ' font-mono' : ''}`}
      >
        {value}
      </p>
    </div>
  );
}
