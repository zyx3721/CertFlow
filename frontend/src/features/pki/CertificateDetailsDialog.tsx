import { Download } from 'lucide-react';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { Certificate } from '@/lib/pki';
import { certificateStatusLabels, formatDate } from './support';

export function CertificateDetailsDialog({
  item,
  issuer,
  onClose,
  onDownload,
  canDownload,
}: {
  item: Certificate | null;
  issuer: string;
  onClose: () => void;
  onDownload: (item: Certificate) => void;
  canDownload: boolean;
}) {
  if (!item) return null;
  const canDownloadItem = canDownload && (item.status === 'valid' || item.status === 'rejected');
  const downloadLabel = item.status === 'rejected' ? '下载 CSR' : '下载证书';
  const disabledDownloadLabel =
    item.status === 'pending'
      ? '待审批证书暂不可下载'
      : item.status === 'revoked'
        ? '已撤销证书不可下载'
        : '当前状态不可下载';
  const fields = [
    ['通用名称', item.commonName],
    ['SAN', item.san.join(', ') || '-'],
    ['颁发者', issuer],
    ['算法', item.algorithm],
    ['来源', item.source === 'system' ? '系统生成' : 'CSR'],
    ['生效时间', formatDate(item.notBefore)],
    ['到期时间', formatDate(item.notAfter)],
    ['状态', certificateStatusLabels[item.status]],
  ];
  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-hidden sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>查看证书</DialogTitle>
          <DialogDescription>{item.commonName} 的证书信息 与 PEM 内容</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3 text-sm md:grid-cols-2">
          {fields.map(([label, value]) => (
            <div
              key={label}
              className="rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-2"
            >
              <span className="text-xs text-[var(--zl-text-muted)]">{label}：</span>
              <span className="break-all font-mono text-xs">{value}</span>
            </div>
          ))}
          <div className="md:col-span-2 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-2">
            <span className="text-xs text-[var(--zl-text-muted)]">Subject：</span>
            <span className="break-all font-mono text-xs">{item.subject}</span>
          </div>
          <div className="md:col-span-2 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-2">
            <span className="text-xs text-[var(--zl-text-muted)]">指纹：</span>
            <span className="break-all font-mono text-xs">
              {item.fingerprint || '证书尚未签发'}
            </span>
          </div>
          {item.status === 'rejected' ? (
            <div className="md:col-span-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2">
              <span className="text-xs font-semibold text-red-600 dark:text-red-400">
                驳回原因：
              </span>
              <span className="whitespace-pre-wrap font-mono text-xs text-[var(--zl-text)]">
                {item.rejectReason || '未填写驳回原因'}
              </span>
            </div>
          ) : null}
        </div>
        {item.status === 'rejected' ? (
          <div>
            <p className="mb-2 text-xs font-semibold">CSR 信息</p>
            <pre className="zl-hidden-scrollbar h-36 overflow-auto rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg)] p-4 font-mono text-xs leading-5 text-sky-500">
              {item.csrPem || '暂无 CSR 信息'}
            </pre>
          </div>
        ) : (
          <div className={item.privateKeyPem?.trim() ? 'grid gap-3 md:grid-cols-2' : ''}>
            <div className="min-w-0">
              <p className="mb-2 text-xs font-semibold">签发证书 PEM</p>
              <pre className="zl-hidden-scrollbar h-36 overflow-auto rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg)] p-4 font-mono text-xs leading-5 text-emerald-500">
                {item.status === 'valid' && item.certificatePem?.trim()
                  ? item.certificatePem
                  : '证书尚未签发，暂无 PEM 内容'}
              </pre>
            </div>
            {item.status === 'valid' && item.privateKeyPem?.trim() ? (
              <div className="min-w-0">
                <p className="mb-2 text-xs font-semibold">私钥 KEY</p>
                <pre className="zl-hidden-scrollbar h-36 overflow-auto rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg)] p-4 font-mono text-xs leading-5 text-amber-500">
                  {item.privateKeyPem}
                </pre>
              </div>
            ) : null}
          </div>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            关闭
          </Button>
          {canDownload ? (
            <AppTooltip
              label={canDownloadItem ? downloadLabel : disabledDownloadLabel}
              className={canDownloadItem ? 'inline-flex' : 'inline-flex cursor-not-allowed'}
            >
              <Button
                className="border border-emerald-500/40 bg-emerald-500/10 text-emerald-700 shadow-[0_1px_2px_rgba(16,185,129,0.16)] hover:bg-emerald-500/20 hover:text-emerald-800 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-400/35 dark:bg-emerald-400/10 dark:text-emerald-300 dark:hover:bg-emerald-400/20 dark:hover:text-emerald-200"
                disabled={!canDownloadItem}
                onClick={() => onDownload(item)}
              >
                <Download size={15} />
                {downloadLabel}
              </Button>
            </AppTooltip>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
