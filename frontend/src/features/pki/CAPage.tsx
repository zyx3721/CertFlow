import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle2,
  Download,
  Eye,
  GitBranch,
  Landmark,
  Plus,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { EmptyState } from '@/components/empty-state';
import { Field } from '@/components/field';
import { PageHeader } from '@/components/page-header';
import { AppTooltip } from '@/components/app-tooltip';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { CAExpiryDatePicker } from './CAExpiryDatePicker';
import { formatSubjectForDisplay } from './CASubjectDisplay';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  createCA,
  checkCADeletion,
  deleteCA,
  downloadBlob,
  downloadCACertificate,
  getCACertificate,
  getCAs,
  type CAType,
  type CertificateAuthority,
  type KeyAlgorithm,
} from '@/lib/pki';
import { QueryState } from './shared';
import { formatDate, pkiQueryKeys, usePageRefresh } from './support';

const caKeys = [pkiQueryKeys.cas] as const;
const typeLabels: Record<CAType, string> = {
  root: '根 CA',
  intermediate: '中间 CA',
  issuing: '签发 CA',
};
const typeDescriptions: Record<CAType, string> = {
  root: '信任锚点',
  intermediate: '层级中转',
  issuing: '证书签发',
};
const typeBadgeStyles: Record<CAType, string> = {
  root: 'border-amber-400/40 bg-amber-400/10 text-amber-600 dark:text-amber-400',
  intermediate: 'border-cyan-400/40 bg-cyan-400/10 text-cyan-600 dark:text-cyan-400',
  issuing: 'border-emerald-400/40 bg-emerald-400/10 text-emerald-600 dark:text-emerald-400',
};
const defaultSubjects: Record<CAType, string> = {
  root: 'CN=Acme Root CA, O=Acme Corp, C=CN',
  intermediate: 'CN=Acme Intermediate CA, O=Acme Corp, C=CN',
  issuing: 'CN=Acme Issuing CA, O=Acme Corp, C=CN',
};
type CAForm = {
  name: string;
  type: CAType;
  algorithm: KeyAlgorithm;
  subject: string;
  parentId: string;
  notAfter: string;
};

const initialForm: CAForm = {
  name: '',
  type: 'root',
  algorithm: 'RSA-2048',
  subject: defaultSubjects.root,
  parentId: '',
  notAfter: '2035-12-31',
};

export function CAPage() {
  const user = getStoredUser();
  const canDownloadCA = userHasPermission(user, 'ca.download');
  const canAddCA = userHasPermission(user, 'ca.add');
  const canDeleteCA = userHasPermission(user, 'ca.delete');
  usePageRefresh(caKeys);
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: pkiQueryKeys.cas, queryFn: getCAs });
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState<CAForm>(initialForm);
  const [deleteTarget, setDeleteTarget] = useState<CertificateAuthority | null>(null);
  const [preview, setPreview] = useState<{
    ca: CertificateAuthority;
    content: string;
    fingerprint: string;
  } | null>(null);
  const items = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const roots = items.filter(item => !item.parentId);
  const parentOptions = items.filter(item =>
    form.type === 'intermediate' ? item.type === 'root' : item.type !== 'issuing'
  );

  const createMutation = useMutation({
    mutationFn: createCA,
    onSuccess: ca => {
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.cas });
      setCreateOpen(false);
      toast.success(`CA ${ca.name} 已创建`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : 'CA 创建失败'),
  });
  const deleteMutation = useMutation({
    mutationFn: async (ca: CertificateAuthority) => {
      await deleteCA(ca.id);
      return ca;
    },
    onSuccess: ca => {
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.cas });
      setDeleteTarget(null);
      toast.success(`CA ${ca.name} 已删除`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : 'CA 删除失败'),
  });

  const childrenByParent = useMemo(() => {
    const map = new Map<string, CertificateAuthority[]>();
    for (const item of items) {
      const key = item.parentId || '';
      map.set(key, [...(map.get(key) ?? []), item]);
    }
    return map;
  }, [items]);

  async function openCertificate(ca: CertificateAuthority) {
    try {
      const cert = await getCACertificate(ca.id);
      setPreview({
        ca,
        content: cert.content,
        fingerprint: await certificateFingerprint(cert.content),
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'CA 证书加载失败');
    }
  }

  async function downloadCertificate(ca: CertificateAuthority) {
    try {
      const { blob, filename } = await downloadCACertificate(ca.id);
      downloadBlob(blob, filename);
      toast.success(certificateDownloadMessage(ca));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'CA 证书下载失败');
    }
  }

  function handleCreateOpenChange(open: boolean) {
    setCreateOpen(open);
    if (open) setForm(initialForm);
  }

  async function requestDelete(ca: CertificateAuthority) {
    try {
      const { deletable } = await checkCADeletion(ca.id);
      if (!deletable) {
        toast.error('该 CA 或其下级 CA 已关联证书，不能删除');
        return;
      }
      setDeleteTarget(ca);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'CA 删除检查失败');
    }
  }

  function renderNode(ca: CertificateAuthority, depth = 0) {
    const children = childrenByParent.get(ca.id) ?? [];
    return (
      <div key={ca.id} className={depth > 0 ? 'ml-5 border-l border-[var(--zl-border)] pl-4' : ''}>
        <article className="zl-surface-3d zl-card-hover mb-3 rounded-xl border p-4">
          <div className="flex flex-wrap items-start gap-4">
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-blue-500/10 text-blue-500">
              <Landmark size={18} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-sm font-semibold">{ca.name}</h2>
                <span
                  className={`rounded-full border px-2.5 py-1 text-xs ${typeBadgeStyles[ca.type]}`}
                >
                  {typeLabels[ca.type]}
                </span>
                <span className="rounded-full border border-[var(--zl-border)] px-2.5 py-1 text-xs text-[var(--zl-text-muted)]">
                  {ca.algorithm}
                </span>
              </div>
              <p className="mt-2 truncate font-mono text-xs text-[var(--zl-text-muted)]">
                {formatSubjectForDisplay(ca.subject)}
              </p>
            </div>
            <dl className="grid grid-cols-3 gap-5 text-center text-xs">
              <div>
                <dt className="text-[var(--zl-text-muted)]">已签发</dt>
                <dd className="mt-1 text-base font-semibold">{ca.issuedCerts}</dd>
              </div>
              <div>
                <dt className="text-[var(--zl-text-muted)]">已撤销</dt>
                <dd className="mt-1 text-base font-semibold text-red-500">{ca.revokedCerts}</dd>
              </div>
              <div>
                <dt className="text-[var(--zl-text-muted)]">到期时间</dt>
                <dd className="mt-1 whitespace-nowrap text-base font-semibold text-amber-600 dark:text-amber-400">
                  {formatDate(ca.notAfter).split(' ')[0]}
                </dd>
              </div>
            </dl>
            <div className="flex gap-1">
              <AppTooltip label="查看 CA 证书">
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label="查看 CA 证书"
                  onClick={() => void openCertificate(ca)}
                  className="text-slate-500 hover:bg-slate-500/10 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                >
                  <Eye size={15} />
                </Button>
              </AppTooltip>
              {canDownloadCA ? (
                <AppTooltip label="下载 CA 证书">
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label="下载 CA 证书"
                    onClick={() => void downloadCertificate(ca)}
                    className="text-emerald-600 hover:bg-emerald-500/10 hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-300"
                  >
                    <Download size={15} />
                  </Button>
                </AppTooltip>
              ) : null}
              {canDeleteCA ? (
                <AppTooltip label="删除 CA">
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label="删除 CA"
                    onClick={() => void requestDelete(ca)}
                    className="text-red-500 hover:bg-red-500/10 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300"
                  >
                    <Trash2 size={15} />
                  </Button>
                </AppTooltip>
              ) : null}
            </div>
          </div>
        </article>
        {children.map(child => renderNode(child, depth + 1))}
      </div>
    );
  }

  return (
    <div className="certflow-page-fill">
      <PageHeader
        title="CA 管理"
        description="管理根 CA、中间 CA 与签发 CA 的完整信任链"
        actions={
          canAddCA ? (
            <Button className="zl-ca-create-button" onClick={() => handleCreateOpenChange(true)}>
              <Plus size={16} />
              新增 CA
            </Button>
          ) : undefined
        }
      />
      <section className="zl-surface-3d certflow-page-card rounded-xl border">
        <div className="certflow-scroll-area flex-1 p-3">
          <QueryState
            loading={query.isLoading}
            error={query.isError}
            onRetry={() => void query.refetch()}
          >
            {items.length === 0 ? (
              <EmptyState
                title="尚未创建 CA"
                description="先创建根 CA，再逐级建立中间 CA 和签发 CA"
              />
            ) : (
              <section aria-labelledby="ca-hierarchy-heading">
                <div className="zl-surface-3d mb-3 flex items-center justify-between gap-3 rounded-xl border px-4 py-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <GitBranch size={16} className="shrink-0 text-blue-500" aria-hidden="true" />
                    <h2 id="ca-hierarchy-heading" className="text-sm font-semibold">
                      CA 层级结构
                    </h2>
                  </div>
                  <span className="shrink-0 rounded-full border border-blue-500/20 bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-600 dark:text-blue-400">
                    共 {items.length} 个 CA
                  </span>
                </div>
                {roots.map(item => renderNode(item))}
              </section>
            )}
          </QueryState>
        </div>
      </section>

      <Dialog open={createOpen} onOpenChange={handleCreateOpenChange}>
        <DialogContent className="max-h-[90vh] gap-0 overflow-visible p-0 sm:max-w-3xl">
          <DialogHeader className="border-b border-[var(--zl-border)] px-5 py-4 pr-16">
            <div className="flex items-center gap-3">
              <span className="grid h-9 w-9 place-items-center rounded-xl border border-blue-500/20 bg-blue-500/10 text-blue-500">
                <Landmark size={18} />
              </span>
              <div>
                <DialogTitle className="text-base">新增 CA 配置</DialogTitle>
                <DialogDescription className="mt-0.5 text-xs">
                  配置 CA 类型、算法、有效期与层级关系
                </DialogDescription>
              </div>
            </div>
          </DialogHeader>
          <div className="zl-hidden-scrollbar max-h-[calc(100vh-12rem)] overflow-y-auto px-5 py-4">
            <section className="rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/50 p-3">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
                <ShieldCheck size={16} className="text-blue-500" />
                CA 类型
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                {(Object.keys(typeLabels) as CAType[]).map(type => {
                  const selected = form.type === type;
                  return (
                    <button
                      key={type}
                      type="button"
                      aria-pressed={selected}
                      onClick={() =>
                        setForm(value => ({
                          ...value,
                          type,
                          parentId: '',
                          subject: usesDefaultSubject(value.subject)
                            ? defaultSubjects[type]
                            : value.subject,
                        }))
                      }
                      className={`rounded-xl border p-3 text-left transition-all duration-300 ease-out hover:-translate-y-0.5 ${
                        selected
                          ? 'border-blue-500/60 bg-blue-500/10 shadow-sm'
                          : 'border-[var(--zl-border)] bg-[var(--zl-control-bg)] hover:border-blue-500/40 hover:bg-blue-500/5'
                      }`}
                    >
                      <span className="flex items-center justify-between gap-2">
                        <strong className="text-sm">{typeLabels[type]}</strong>
                        {selected ? (
                          <CheckCircle2 size={17} className="shrink-0 text-blue-500" />
                        ) : null}
                      </span>
                      <span className="mt-1 block text-xs text-[var(--zl-text-muted)]">
                        {typeDescriptions[type]}
                      </span>
                    </button>
                  );
                })}
              </div>
            </section>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <Field label="CA 名称" required>
                <Input
                  value={form.name}
                  onChange={event => setForm(value => ({ ...value, name: event.target.value }))}
                  placeholder="例如：业务签发 CA"
                />
              </Field>
              <Field label="密钥算法" required>
                <Select
                  value={form.algorithm}
                  onValueChange={(algorithm: KeyAlgorithm) =>
                    setForm(value => ({ ...value, algorithm }))
                  }
                >
                  <SelectTrigger className="font-normal">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {['RSA-2048', 'RSA-4096', 'ECDSA-P256', 'ECDSA-P384', 'ED25519'].map(item => (
                      <SelectItem key={item} value={item} className="font-normal">
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field label="到期日期" required>
                <CAExpiryDatePicker
                  value={form.notAfter}
                  onChange={notAfter => setForm(value => ({ ...value, notAfter }))}
                />
              </Field>
              {form.type !== 'root' ? (
                <Field label="上级 CA" required>
                  <Select
                    value={form.parentId || undefined}
                    onValueChange={parentId => setForm(value => ({ ...value, parentId }))}
                  >
                    <SelectTrigger className="font-normal data-[placeholder]:text-[var(--zl-text-muted)]">
                      <SelectValue placeholder="选择上级 CA" />
                    </SelectTrigger>
                    <SelectContent>
                      {parentOptions.length > 0 ? (
                        parentOptions.map(item => (
                          <SelectItem key={item.id} value={item.id} className="font-normal">
                            {item.name} · {typeLabels[item.type]}
                          </SelectItem>
                        ))
                      ) : (
                        <SelectItem value="__no-parent-ca" disabled className="font-normal">
                          暂无可用上级 CA
                        </SelectItem>
                      )}
                    </SelectContent>
                  </Select>
                </Field>
              ) : (
                <div className="hidden sm:block" />
              )}
              <div className="sm:col-span-2">
                <Field label="Subject" required hint="必须包含 CN，可使用 O、OU、C、ST、L，字段值可直接包含英文逗号">
                  <Input
                    value={form.subject}
                    onChange={event =>
                      setForm(value => ({ ...value, subject: event.target.value }))
                    }
                  />
                </Field>
              </div>
            </div>
          </div>
          <DialogFooter className="border-t border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-5 py-3">
            <Button variant="outline" onClick={() => handleCreateOpenChange(false)}>
              取消
            </Button>
            <Button
              className="zl-ca-create-button"
              disabled={createMutation.isPending}
              onClick={() => {
                const caName = form.name.trim();
                const subject = form.subject.trim();
                const parentId = form.parentId.trim();
                if (!caName) return toast.error('CA 名称不能为空');
                if (!form.algorithm) return toast.error('密钥算法不能为空');
                if (!form.notAfter) return toast.error('到期日期不能为空');
                if (!subject) return toast.error('Subject 不能为空');
                if (form.type !== 'root' && !parentId) return toast.error('上级 CA 不能为空');

                const notAfter = new Date(`${form.notAfter}T23:59:59+08:00`);
                if (Number.isNaN(notAfter.getTime())) return toast.error('到期日期格式不正确');
                if (!/(?:^|,)\s*CN\s*=\s*[^,]+/i.test(subject)) return toast.error('Subject 必须包含 CN');
                if (items.some(item => item.name === caName)) {
                  toast.error('CA 名称已存在，请使用其他名称');
                  return;
                }
                createMutation.mutate({
                  ...form,
                  name: caName,
                  subject,
                  parentId,
                  notAfter: notAfter.toISOString(),
                });
              }}
            >
              {createMutation.isPending ? '正在创建...' : '创建 CA'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {preview ? (
        <Dialog open onOpenChange={open => !open && setPreview(null)}>
          <DialogContent className="max-h-[90vh] overflow-hidden sm:max-w-3xl">
            <DialogHeader>
              <DialogTitle>查看 CA 证书</DialogTitle>
              <DialogDescription>{preview.ca.name} 的证书信息与 PEM 内容</DialogDescription>
            </DialogHeader>
            <div className="grid gap-3 text-xs sm:grid-cols-2">
              {[
                ['CA 名称', preview.ca.name],
                ['CA 类型', typeLabels[preview.ca.type]],
                ['算法', preview.ca.algorithm],
                ['生效时间', formatDate(preview.ca.notBefore).split(' ')[0]],
                ['到期时间', formatDate(preview.ca.notAfter).split(' ')[0]],
                ['Subject', formatSubjectForDisplay(preview.ca.subject)],
              ].map(([label, detail]) => (
                <div
                  key={label}
                  className="min-w-0 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-3 py-2"
                >
                  <p className="break-all leading-5 text-[var(--zl-text)]">
                    <span className="text-[var(--zl-text-muted)]">{label}：</span>
                    <span className="font-mono">{detail}</span>
                  </p>
                </div>
              ))}
              <div className="min-w-0 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-3 py-2 sm:col-span-2">
                <p className="break-all leading-5 text-[var(--zl-text)]">
                  <span className="text-[var(--zl-text-muted)]">指纹：</span>
                  <span className="font-mono">{preview.fingerprint}</span>
                </p>
              </div>
            </div>
            <div>
              <p className="mb-2 text-xs font-semibold">证书 PEM</p>
              <pre className="zl-hidden-scrollbar max-h-[34vh] overflow-auto rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg)] p-4 font-mono text-xs leading-5 text-emerald-500">
                {preview.content}
              </pre>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setPreview(null)}>
                关闭
              </Button>
              {canDownloadCA ? (
                <Button
                  className="border border-emerald-500/40 bg-emerald-500/10 text-emerald-700 shadow-[0_1px_2px_rgba(16,185,129,0.16)] hover:bg-emerald-500/20 hover:text-emerald-800 dark:border-emerald-400/35 dark:bg-emerald-400/10 dark:text-emerald-300 dark:hover:bg-emerald-400/20 dark:hover:text-emerald-200"
                  onClick={() => {
                    void downloadCertificate(preview.ca);
                  }}
                >
                  <Download size={15} />
                  下载证书
                </Button>
              ) : null}
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      {deleteTarget ? (
        <ConfirmDialog
          open
          horizontalHeader
          title="确认删除 CA"
          description={`删除后将移除 ${deleteTarget.name} 及其下级 CA 数据；如存在关联证书将拒绝删除。此操作不可恢复。`}
          busy={deleteMutation.isPending}
          onOpenChange={open => !open && setDeleteTarget(null)}
          onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget)}
        />
      ) : null}
    </div>
  );
}

async function certificateFingerprint(content: string) {
  try {
    const encoded = content
      .replace(/-----BEGIN CERTIFICATE-----|-----END CERTIFICATE-----/g, '')
      .replaceAll(/\s/g, '');
    const binary = window.atob(encoded);
    const bytes = Uint8Array.from(binary, character => character.charCodeAt(0));
    const digest = await window.crypto.subtle.digest('SHA-256', bytes);
    return Array.from(new Uint8Array(digest), byte => byte.toString(16).padStart(2, '0'))
      .join(':')
      .toUpperCase();
  } catch {
    return '无法计算证书指纹';
  }
}

function certificateDownloadMessage(ca: CertificateAuthority) {
  return `${typeLabels[ca.type].replace(' ', '')} ${ca.name} 证书已下载`;
}

function usesDefaultSubject(subject: string) {
  return !subject.trim() || Object.values(defaultSubjects).includes(subject);
}
