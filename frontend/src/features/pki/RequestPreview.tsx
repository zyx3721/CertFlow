import { ArrowLeft, FileText, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { CertificatePurpose, KeyAlgorithm } from '@/lib/pki';

type RequestPreviewProps = {
  subject: { cn: string; o: string; ou: string; c: string; st: string; l: string };
  algorithm: KeyAlgorithm;
  validityLabel: string;
  caName: string;
  purpose: CertificatePurpose;
  san: string[];
  subjectText: string;
  csrInputMode: 'generate' | 'manual';
  csrPEM?: string;
  submitting: boolean;
  onBack: () => void;
};

export function RequestPreview({
  subject,
  algorithm,
  validityLabel,
  caName,
  purpose,
  san,
  subjectText,
  csrInputMode,
  csrPEM,
  submitting,
  onBack,
}: RequestPreviewProps) {
  return (
    <section className="zl-surface-3d certflow-page-card certflow-request-preview-card certflow-scroll-area rounded-xl border p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3 border-b border-[var(--zl-border)] pb-4">
        <div className="flex items-center gap-3">
          <span className="grid h-9 w-9 place-items-center rounded-lg bg-blue-500/10 text-blue-500">
            <FileText size={17} />
          </span>
          <div>
            <h2 className="text-sm font-semibold">申请预览与确认</h2>
            <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
              请确认申请信息，提交后将进入审批流程
            </p>
          </div>
        </div>
        <span className="rounded-full border border-blue-500/20 bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-600 dark:text-blue-400">
          Step 2
        </span>
      </div>
      <div className="grid gap-3 text-xs sm:grid-cols-2">
        <PreviewItem label="通用名称（CN）" value={subject.cn} />
        <PreviewItem label="组织（O）" value={subject.o} />
        <PreviewItem label="组织单位（OU）" value={subject.ou} />
        <PreviewItem label="国家（C）" value={subject.c} />
        <PreviewItem label="省份（ST）" value={subject.st} />
        <PreviewItem label="城市（L）" value={subject.l} />
        <PreviewItem label="密钥算法" value={algorithm} />
        <PreviewItem label="有效期" value={validityLabel} />
        <PreviewItem label="签发 CA" value={caName} />
        <PreviewItem label="用途" value={purposeLabel(purpose)} />
        <PreviewItem label="证书域名" value={san.join(', ')} />
        <PreviewItem label="Subject" value={subjectText} />
        <PreviewItem
          label="CSR 来源"
          value={csrInputMode === 'manual' ? '手动提供 CSR' : '系统生成'}
          wide
        />
      </div>
      <div className="mt-4 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg)]/70 p-3">
        <p className="mb-2 text-xs text-[var(--zl-text-muted)]">证书签名请求（CSR）</p>
        <pre className="certflow-csr-preview max-h-52 overflow-auto whitespace-pre-wrap break-all rounded-md border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-2.5 font-mono text-xs leading-5 text-emerald-600 dark:text-emerald-400">
          {csrPEM ?? 'CSR 生成中...'}
        </pre>
      </div>
      <div className="mt-5 flex items-center justify-between border-t border-[var(--zl-border)] pt-4">
        <Button type="button" variant="outline" onClick={onBack}>
          <ArrowLeft size={15} />
          返回修改
        </Button>
        <Button type="submit" disabled={submitting}>
          <Send size={15} />
          {submitting ? '正在提交...' : '提交申请'}
        </Button>
      </div>
    </section>
  );
}

function PreviewItem({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <div
      className={`zl-card-hover min-w-0 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-3 py-2 ${
        wide ? 'sm:col-span-2' : ''
      }`}
    >
      <p className="text-[var(--zl-text-muted)]">{label}</p>
      <p className="mt-1 break-all font-mono text-[var(--zl-text)]">{value || '-'}</p>
    </div>
  );
}

function purposeLabel(purpose: CertificatePurpose) {
  if (purpose === 'server') return '服务器证书';
  if (purpose === 'client') return '客户端证书';
  return 'mTLS 双向认证';
}
