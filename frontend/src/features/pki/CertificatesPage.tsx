import { type ReactNode, useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Eye, FileCheck2, SearchCheck, ShieldX, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { DataTableShell } from '@/components/data-table-shell';
import { EmptyState } from '@/components/empty-state';
import { Field } from '@/components/field';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  deleteCertificate,
  downloadCertificate,
  getCAs,
  getCertificatePackage,
  getCertificates,
  revokeCertificate,
  verifyCertificate,
  type Certificate,
  type RevokeReason,
} from '@/lib/pki';
import { CertificateExportDialog } from './CertificateExportDialog';
import { CertificateDetailsDialog } from './CertificateDetailsDialog';
import { Pagination } from './Pagination';
import { pageSizeOptions } from './pagination-options';
import { QueryState, SearchBox } from './shared';
import {
  certificateStatusLabels,
  formatDate,
  pkiQueryKeys,
  revokeReasonLabels,
  usePageRefresh,
} from './support';

const keys = [pkiQueryKeys.certificates, pkiQueryKeys.cas] as const;
const revokeReasons: { value: RevokeReason; label: string }[] = [
  { value: 'unspecified', label: revokeReasonLabels.unspecified },
  { value: 'keyCompromise', label: revokeReasonLabels.keyCompromise },
  { value: 'cACompromise', label: revokeReasonLabels.cACompromise },
  { value: 'affiliationChanged', label: revokeReasonLabels.affiliationChanged },
  { value: 'superseded', label: revokeReasonLabels.superseded },
  { value: 'cessationOfOperation', label: revokeReasonLabels.cessationOfOperation },
];

export function CertificatesPage() {
  const user = getStoredUser();
  const canReadCertificates = userHasPermission(user, 'certificates.read');
  const canReadCA = userHasPermission(user, 'ca.read');
  const canDownload = userHasPermission(user, 'certificates.download');
  const canVerify = userHasPermission(user, 'certificates.verify');
  const canManage = userHasPermission(user, 'certificates.manage');
  const canRevoke = userHasPermission(user, 'certificates.revoke');
  const canDelete = userHasPermission(user, 'certificates.delete');
  usePageRefresh(keys);
  const client = useQueryClient();
  const certificateQuery = useQuery({
    queryKey: pkiQueryKeys.certificates,
    queryFn: getCertificates,
    enabled: canReadCertificates,
  });
  const caQuery = useQuery({ queryKey: pkiQueryKeys.cas, queryFn: getCAs, enabled: canReadCA });
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('all');
  const [caID, setCAID] = useState('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<(typeof pageSizeOptions)[number]>(10);
  const [exportOpen, setExportOpen] = useState(false);
  const [details, setDetails] = useState<Certificate | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<Certificate | null>(null);
  const [revokeReason, setRevokeReason] = useState<RevokeReason>('unspecified');
  const [deleteTarget, setDeleteTarget] = useState<Certificate | null>(null);
  const [verifyResult, setVerifyResult] = useState<{
    valid: boolean;
    reason: string;
    checkedAt: string;
    commonName: string;
  } | null>(null);
  const caNames = useMemo(
    () => new Map((caQuery.data?.items ?? []).map(item => [item.id, item.name])),
    [caQuery.data?.items]
  );
  const items = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return (certificateQuery.data?.items ?? []).filter(item => {
      const issuer = caNames.get(item.caId) ?? item.caId;
      const source = item.source === 'system' ? '系统生成' : 'CSR';
      const statusLabel = certificateStatusLabels[item.status] ?? item.status;
      const searchable = [
        item.commonName,
        item.serialNumber || '等待签发',
        issuer,
        item.caId,
        item.algorithm,
        item.source,
        source,
        formatDate(item.notAfter).split(' ')[0],
        item.status,
        statusLabel,
      ]
        .join(' ')
        .toLowerCase();
      return (
        (status === 'all' || item.status === status) &&
        (caID === 'all' || item.caId === caID) &&
        (!keyword || searchable.includes(keyword))
      );
    });
  }, [caID, caNames, certificateQuery.data?.items, search, status]);
  const pageCount = Math.max(1, Math.ceil(items.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const pageItems = items.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  useEffect(() => setPage(1), [caID, pageSize, search, status]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  const revokeMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: RevokeReason }) =>
      revokeCertificate(id, reason),
    onSuccess: (_, variables) => {
      void client.invalidateQueries({ queryKey: pkiQueryKeys.certificates });
      void client.invalidateQueries({ queryKey: pkiQueryKeys.crl });
      setRevokeTarget(null);
      setRevokeReason('unspecified');
      const item = items.find(certificate => certificate.id === variables.id);
      toast.success(`证书已撤销：${item?.commonName ?? variables.id}`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : '证书撤销失败'),
  });
  const deleteMutation = useMutation({
    mutationFn: deleteCertificate,
    onSuccess: (_, id) => {
      void client.invalidateQueries({ queryKey: pkiQueryKeys.certificates });
      setDeleteTarget(null);
      const item = items.find(certificate => certificate.id === id);
      toast.success(`证书已删除：${item?.commonName ?? id}`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : '证书删除失败'),
  });

  async function download(item: Certificate) {
    try {
      const { blob, filename } = await downloadCertificate(item.id);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      anchor.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      const resourceLabel = item.status === 'rejected' ? 'CSR' : '证书';
      toast.success(
        filename.endsWith('.zip')
          ? `${resourceLabel}压缩包已下载：${filename}`
          : `${resourceLabel}文件已下载：${filename}`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '证书下载失败');
    }
  }

  function openDetails(item: Certificate) {
    setDetails(item);
    const needsMaterials =
      item.status === 'rejected' || (item.status === 'valid' && item.source === 'system');
    if (!needsMaterials || !canDownload) return;

    void getCertificatePackage(item.id)
      .then(material => {
        setDetails(current =>
          current?.id === item.id
            ? {
                ...current,
                certificatePem: material.certificatePem || current.certificatePem,
                fingerprint: material.fingerprint || current.fingerprint,
                privateKeyPem: material.privateKeyPem,
                csrPem: material.csrPem || current.csrPem,
              }
            : current
        );
      })
      .catch(error => toast.error(error instanceof Error ? error.message : '读取证书材料失败'));
  }

  async function verify(item: Certificate) {
    try {
      setVerifyResult(await verifyCertificate(item.id));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '证书校验失败');
    }
  }

  return (
    <div className="certflow-page-fill">
      <PageHeader title="证书管理" description="查看、搜索和管理所有 X.509 数字证书" />
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
              placeholder="可搜索证书名称、序列号等信息"
              className="min-w-64 flex-1"
            />
            <div className="w-36">
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger className="font-normal">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  {Object.entries(certificateStatusLabels).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-36">
              <Select value={caID} onValueChange={setCAID}>
                <SelectTrigger className="font-normal">
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
            {canManage ? (
              <Button
                variant="outline"
                className="ml-auto border-blue-400/50 text-blue-600 hover:border-blue-500/70 hover:bg-blue-500/10 dark:text-blue-400"
                onClick={() => {
                  if (items.length === 0) {
                    toast.error('没有可导出的证书数据');
                    return;
                  }
                  setExportOpen(true);
                }}
              >
                <Download size={16} />
                导出
              </Button>
            ) : null}
          </>
        }
      >
        <QueryState
          loading={certificateQuery.isLoading}
          error={certificateQuery.isError}
          onRetry={() => void certificateQuery.refetch()}
        >
          {items.length === 0 ? (
            <EmptyState
              title="没有匹配的证书"
              description="调整筛选条件，或前往证书申请页面提交新申请"
            />
          ) : (
            <div className="min-w-[1120px] divide-y divide-[var(--zl-border)]">
              <div className="sticky top-0 z-10 grid grid-cols-[minmax(150px,1.3fr)_minmax(190px,1.35fr)_minmax(120px,1fr)_110px_100px_120px_100px_220px] items-center gap-4 bg-[var(--zl-table-head-bg)] px-4 py-3 text-center text-xs font-semibold text-[var(--zl-text-muted)]">
                <span>通用名称</span>
                <span>序列号</span>
                <span>颁发者</span>
                <span>算法</span>
                <span>来源</span>
                <span>到期时间</span>
                <span>状态</span>
                <span>操作</span>
              </div>
              {pageItems.map(item => (
                <div
                  key={item.id}
                  className="grid grid-cols-[minmax(150px,1.3fr)_minmax(190px,1.35fr)_minmax(120px,1fr)_110px_100px_120px_100px_220px] items-center gap-4 px-4 py-3.5 text-sm transition-colors hover:bg-[var(--zl-control-bg-soft)]"
                >
                  <span className="truncate text-center font-medium">{item.commonName}</span>
                  <span className="truncate text-center text-sm font-medium text-[var(--zl-text)]">
                    {item.serialNumber || '等待签发'}
                  </span>
                  <span className="truncate text-center text-sm font-medium text-[var(--zl-text)]">
                    {caNames.get(item.caId) ?? item.caId}
                  </span>
                  <span
                    className={`justify-self-center rounded-full px-2 py-0.5 text-sm font-medium ${algorithmBadgeClassName(item.algorithm)}`}
                  >
                    {item.algorithm}
                  </span>
                  <span
                    className={`justify-self-center rounded-full px-2 py-0.5 text-sm font-medium ${sourceBadgeClassName(item.source)}`}
                  >
                    {item.source === 'system' ? '系统生成' : 'CSR'}
                  </span>
                  <time className="text-center text-sm font-medium text-[var(--zl-text)]">
                    {formatDate(item.notAfter).split(' ')[0]}
                  </time>
                  <span className="justify-self-center">
                    <StatusBadge status={item.status} />
                  </span>
                  <div className="flex justify-center gap-1">
                    {canReadCertificates ? (
                      <IconButton
                        label="查看证书详情"
                        className="text-violet-600 hover:bg-violet-500/10 dark:text-violet-400"
                        onClick={() => openDetails(item)}
                        icon={<Eye size={15} />}
                      />
                    ) : null}
                    {canDownload ? (
                      <IconButton
                        label="下载证书材料"
                        disabled={item.status !== 'valid'}
                        disabledLabel="证书签发后才可下载"
                        className="text-blue-600 dark:text-blue-400"
                        onClick={() => void download(item)}
                        icon={<Download size={15} />}
                      />
                    ) : null}
                    {canVerify ? (
                      <IconButton
                        label="校验证书状态"
                        disabled={item.status !== 'valid'}
                        disabledLabel="证书签发后才可校验"
                        className="text-emerald-600 dark:text-emerald-400"
                        onClick={() => void verify(item)}
                        icon={<SearchCheck size={15} />}
                      />
                    ) : null}
                    {canRevoke ? (
                      <IconButton
                        label="撤销证书"
                        disabled={item.status !== 'valid'}
                        disabledLabel="仅有效证书可以撤销"
                        className="text-red-600 dark:text-red-400"
                        onClick={() => {
                          setRevokeReason('unspecified');
                          setRevokeTarget(item);
                        }}
                        icon={<ShieldX size={15} />}
                      />
                    ) : null}
                    {canDelete ? (
                      <IconButton
                        label="删除证书"
                        disabled={item.status === 'valid' || item.status === 'pending'}
                        disabledLabel={
                          item.status === 'pending'
                            ? '待审批证书不允许删除'
                            : '有效证书需先撤销才可删除'
                        }
                        className="text-red-600 dark:text-red-400"
                        onClick={() => setDeleteTarget(item)}
                        icon={<Trash2 size={15} />}
                      />
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          )}
        </QueryState>
      </DataTableShell>
      <CertificateExportDialog
        open={exportOpen}
        filteredItems={items}
        caNames={caNames}
        onOpenChange={setExportOpen}
      />
      <CertificateDetailsDialog
        item={details}
        issuer={details ? (caNames.get(details.caId) ?? details.caId) : ''}
        onClose={() => setDetails(null)}
        onDownload={download}
        canDownload={canDownload}
      />
      {revokeTarget ? (
        <Dialog
          open
          onOpenChange={open => {
            if (!open) {
              setRevokeTarget(null);
              setRevokeReason('unspecified');
            }
          }}
        >
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>撤销证书</DialogTitle>
              <DialogDescription>请选择 {revokeTarget.commonName} 的撤销原因</DialogDescription>
            </DialogHeader>
            <Field label="撤销原因">
              <Select
                value={revokeReason}
                onValueChange={value => setRevokeReason(value as RevokeReason)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {revokeReasons.map(reason => (
                    <SelectItem key={reason.value} value={reason.value}>
                      {reason.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => {
                  setRevokeTarget(null);
                  setRevokeReason('unspecified');
                }}
              >
                取消
              </Button>
              <Button
                variant="destructive"
                className="border-red-500/35 bg-red-500/10 text-red-600 shadow-[0_1px_2px_rgba(239,68,68,0.12)] hover:bg-red-500/20 hover:text-red-700 dark:border-red-400/35 dark:bg-red-400/10 dark:text-red-300 dark:hover:bg-red-400/20 dark:hover:text-red-200"
                disabled={revokeMutation.isPending}
                onClick={() => revokeMutation.mutate({ id: revokeTarget.id, reason: revokeReason })}
              >
                确认撤销
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {verifyResult ? (
        <Dialog open onOpenChange={open => !open && setVerifyResult(null)}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>证书校验结果</DialogTitle>
              <DialogDescription>{verifyResult.commonName}</DialogDescription>
            </DialogHeader>
            <div className="rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] p-4 text-sm">
              <p>{verifyResult.valid ? '证书有效，可继续使用' : '证书当前不可用'}</p>
              <p className="mt-2 text-xs text-[var(--zl-text-muted)]">
                状态原因：{verifyResult.reason}
              </p>
              <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
                校验时间：{formatDate(verifyResult.checkedAt)}
              </p>
            </div>
            <DialogFooter>
              <Button onClick={() => setVerifyResult(null)}>完成</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {deleteTarget ? (
        <ConfirmDialog
          open
          title="删除证书"
          description={`确认永久删除 ${deleteTarget.commonName} 的证书记录吗`}
          busy={deleteMutation.isPending}
          onOpenChange={open => !open && setDeleteTarget(null)}
          onConfirm={() => deleteMutation.mutate(deleteTarget.id)}
        />
      ) : null}
    </div>
  );
}

function IconButton({
  label,
  disabled = false,
  disabledLabel,
  className,
  onClick,
  icon,
}: {
  label: string;
  disabled?: boolean;
  disabledLabel?: string;
  className?: string;
  onClick: () => void;
  icon: ReactNode;
}) {
  return (
    <AppTooltip
      label={disabled ? disabledLabel : label}
      className={disabled ? 'inline-flex cursor-not-allowed' : 'inline-flex'}
    >
      <Button
        size="icon"
        variant="ghost"
        className={className}
        aria-label={label}
        disabled={disabled}
        onClick={onClick}
      >
        {icon}
      </Button>
    </AppTooltip>
  );
}

function algorithmBadgeClassName(algorithm: string) {
  const normalized = algorithm.toUpperCase();

  if (normalized.includes('ED25519')) {
    return 'bg-emerald-500/12 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-300';
  }
  if (normalized.includes('ECDSA') || normalized.includes('EC-')) {
    return 'bg-violet-500/12 text-violet-700 dark:bg-violet-400/15 dark:text-violet-300';
  }
  if (normalized.includes('RSA')) {
    return 'bg-sky-500/12 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300';
  }

  return 'bg-slate-500/12 text-slate-700 dark:bg-slate-400/15 dark:text-slate-300';
}

function sourceBadgeClassName(source: Certificate['source']) {
  return source === 'system'
    ? 'bg-indigo-500/12 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-300'
    : 'bg-amber-500/12 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300';
}
