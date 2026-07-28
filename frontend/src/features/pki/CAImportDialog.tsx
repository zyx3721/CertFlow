import { useEffect, useRef, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { FileCheck2, Landmark, Upload } from 'lucide-react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { Field } from '@/components/field';
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
import { importCA, type CAType, type CertificateAuthority } from '@/lib/pki';
import { pkiQueryKeys } from './support';

const typeLabels: Record<CAType, string> = {
  root: '根 CA',
  intermediate: '中间 CA',
  issuing: '签发 CA',
};

type ImportForm = {
  name: string;
  type: CAType;
  parentId: string;
  certificate: File | null;
  privateKey: File | null;
  csr: File | null;
};

const initialForm: ImportForm = {
  name: '',
  type: 'root',
  parentId: '',
  certificate: null,
  privateKey: null,
  csr: null,
};

export function CAImportDialog({
  open,
  items,
  onOpenChange,
}: {
  open: boolean;
  items: CertificateAuthority[];
  onOpenChange: (open: boolean) => void;
}) {
  const client = useQueryClient();
  const [form, setForm] = useState<ImportForm>(initialForm);
  const resetTimer = useRef<number | null>(null);
  const parentOptions = items.filter(item =>
    form.type === 'intermediate' ? item.type === 'root' : item.type !== 'issuing'
  );
  const mutation = useMutation({
    mutationFn: importCA,
    onSuccess: ca => {
      void client.invalidateQueries({ queryKey: pkiQueryKeys.cas });
      changeOpen(false);
      toast.success(`CA ${ca.name} 已导入`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : 'CA 导入失败'),
  });

  useEffect(
    () => () => {
      if (resetTimer.current !== null) window.clearTimeout(resetTimer.current);
    },
    []
  );

  function changeOpen(nextOpen: boolean) {
    if (resetTimer.current !== null) {
      window.clearTimeout(resetTimer.current);
      resetTimer.current = null;
    }
    if (nextOpen) {
      setForm(initialForm);
    } else {
      // Keep the selected type visible until Radix completes its close animation.
      resetTimer.current = window.setTimeout(() => setForm(initialForm), 200);
    }
    onOpenChange(nextOpen);
  }

  async function submit() {
    const name = form.name.trim();
    if (!name) return toast.error('CA 名称不能为空');
    if (!form.certificate) return toast.error('请选择 CA 证书文件');
    if (!form.privateKey) return toast.error('请选择 CA 私钥文件');
    if (form.type !== 'root' && !form.parentId) return toast.error('请选择上级 CA');
    if (items.some(item => item.name === name)) return toast.error('CA 名称已存在，请使用其他名称');

    try {
      const [certificatePEM, privateKeyPEM, csrPEM] = await Promise.all([
        form.certificate.text(),
        form.privateKey.text(),
        form.csr?.text() ?? Promise.resolve(''),
      ]);
      mutation.mutate({
        name,
        type: form.type,
        parentId: form.type === 'root' ? '' : form.parentId,
        certificatePEM,
        privateKeyPEM,
        csrPEM: csrPEM || undefined,
      });
    } catch {
      toast.error('读取 CA 文件失败');
    }
  }

  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <DialogContent className="max-h-[90vh] gap-0 overflow-visible p-0 sm:max-w-2xl">
        <DialogHeader className="border-b border-[var(--zl-border)] px-5 py-4 pr-16">
          <div className="flex items-center gap-3">
            <span className="grid h-9 w-9 place-items-center rounded-xl border border-cyan-500/20 bg-cyan-500/10 text-cyan-600 dark:text-cyan-400">
              <Upload size={18} />
            </span>
            <div>
              <DialogTitle className="text-base">导入已有 CA</DialogTitle>
              <DialogDescription className="mt-0.5 text-xs">导入已有的 CA 证书与匹配私钥</DialogDescription>
            </div>
          </div>
        </DialogHeader>
        <div className="zl-hidden-scrollbar max-h-[calc(100vh-12rem)] overflow-y-auto px-5 py-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="CA 名称" required>
              <Input
                value={form.name}
                onChange={event => setForm(value => ({ ...value, name: event.target.value }))}
                placeholder="例如：企业根 CA"
              />
            </Field>
            <Field label="CA 类型" required>
              <Select
                value={form.type}
                onValueChange={(type: CAType) =>
                  setForm(value => ({ ...value, type, parentId: '' }))
                }
              >
                <SelectTrigger className="font-normal">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(Object.keys(typeLabels) as CAType[]).map(type => (
                    <SelectItem key={type} value={type} className="font-normal">
                      {typeLabels[type]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            {form.type !== 'root' ? (
              <Field label="上级 CA" required>
                <Select
                  value={form.parentId || undefined}
                  onValueChange={parentId => setForm(value => ({ ...value, parentId }))}
                >
                  <SelectTrigger className="font-normal data-[placeholder]:text-[var(--zl-text-muted)]">
                    <SelectValue placeholder="选择已导入的上级 CA" />
                  </SelectTrigger>
                  <SelectContent>
                    {parentOptions.map(item => (
                      <SelectItem key={item.id} value={item.id} className="font-normal">
                        {item.name} · {typeLabels[item.type]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            ) : (
              <div className="hidden sm:block" />
            )}
            <div className="sm:col-span-2">
              <CAImportFileField
                label="CA 证书文件"
                hint="PEM 编码的 .crt 或 .pem 文件"
                accept=".crt,.cer,.pem,application/x-pem-file"
                file={form.certificate}
                emptyLabel="选择 CA 证书文件"
                tooltip="请选择 PEM 编码的 CA 证书文件"
                onChange={certificate => setForm(value => ({ ...value, certificate }))}
                required
              />
            </div>
            <div className="sm:col-span-2">
              <CAImportFileField
                label="CA 私钥文件"
                hint="PEM 编码、未加密的 .key 文件"
                accept=".key,.pem,application/x-pem-file"
                file={form.privateKey}
                emptyLabel="选择 CA 私钥文件"
                tooltip="请选择未加密的 CA 私钥文件"
                onChange={privateKey => setForm(value => ({ ...value, privateKey }))}
                required
              />
            </div>
            <div className="sm:col-span-2">
              <CAImportFileField
                label="CA CSR 文件"
                hint="可选，用于校验证书公钥"
                accept=".csr,.pem,application/pkcs10"
                file={form.csr}
                emptyLabel="选择 CA CSR 文件"
                tooltip="可选，选择 CA CSR 文件用于校验"
                onChange={csr => setForm(value => ({ ...value, csr }))}
              />
            </div>
          </div>
          <div className="mt-4 flex items-start gap-2 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-2.5 text-xs text-[var(--zl-text-muted)]">
            <FileCheck2 size={15} className="mt-0.5 shrink-0 text-cyan-600 dark:text-cyan-400" />
            <span>服务端将校验证书、私钥和上级 CA 信任链</span>
          </div>
        </div>
        <DialogFooter className="border-t border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-5 py-3">
          <Button
            variant="outline"
            className="zl-ca-dialog-action"
            onClick={() => changeOpen(false)}
          >
            取消
          </Button>
          <Button
            className="zl-ca-create-button zl-ca-dialog-action"
            disabled={mutation.isPending}
            onClick={() => void submit()}
          >
            <Landmark size={16} />
            导入 CA
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CAImportFileField({
  label,
  hint,
  accept,
  file,
  emptyLabel,
  tooltip,
  onChange,
  required = false,
}: {
  label: string;
  hint: string;
  accept: string;
  file: File | null;
  emptyLabel: string;
  tooltip: string;
  onChange: (file: File | null) => void;
  required?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <Field label={label} hint={hint} required={required}>
      <AppTooltip
        label={tooltip}
        placement="top"
        align="start"
        disabled={Boolean(file)}
        className="block"
      >
        <Button
          type="button"
          variant="outline"
          className="zl-ca-import-file-row w-full justify-start font-normal"
          aria-label={file ? `${label}：${file.name}` : emptyLabel}
          onClick={() => inputRef.current?.click()}
        >
          <Upload size={15} />
          <span className="truncate">{file?.name ?? emptyLabel}</span>
        </Button>
      </AppTooltip>
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="sr-only"
        tabIndex={-1}
        onChange={event => {
          onChange(event.target.files?.[0] ?? null);
          event.target.value = '';
        }}
      />
    </Field>
  );
}
