import { useEffect, useMemo, useState } from 'react';
import { Check, Download, FileDown } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
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
  exportRows,
  localTimestamp,
  sanitizeExportFileName,
  type ExportColumn,
  type ExportFormat,
} from '@/lib/export-data';
import type { Certificate } from '@/lib/pki';
import { certificateStatusLabels, formatDate } from './support';

const formats: Array<{ value: ExportFormat; label: string }> = [
  { value: 'xlsx', label: 'XLSX' },
  { value: 'xls', label: 'XLS' },
  { value: 'csv', label: 'CSV' },
  { value: 'txt', label: 'TXT' },
];

export function CertificateExportDialog({
  open,
  filteredItems,
  caNames,
  onOpenChange,
}: {
  open: boolean;
  filteredItems: Certificate[];
  caNames: Map<string, string>;
  onOpenChange: (open: boolean) => void;
}) {
  const [format, setFormat] = useState<ExportFormat>('xlsx');
  const [filename, setFilename] = useState(`证书-${localTimestamp()}`);
  const columns = useMemo(() => certificateExportColumns(caNames), [caNames]);
  const [selectedColumnIds, setSelectedColumnIds] = useState<string[]>(() =>
    columns.map(column => column.id ?? column.header)
  );

  useEffect(() => {
    if (!open) return;
    setFormat('xlsx');
    setFilename(`证书-${localTimestamp()}`);
    setSelectedColumnIds(columns.map(column => column.id ?? column.header));
  }, [columns, open]);

  const selectedColumns = useMemo(
    () => columns.filter(column => selectedColumnIds.includes(column.id ?? column.header)),
    [columns, selectedColumnIds]
  );
  const previewName = `${sanitizeExportFileName(filename || `证书-${localTimestamp()}`)}.${format}`;
  const allSelected = selectedColumns.length === columns.length;

  function handleExport() {
    if (filteredItems.length === 0) {
      toast.warning('没有可导出的证书数据');
      return;
    }
    if (selectedColumns.length === 0) {
      toast.warning('请选择至少一个导出字段');
      return;
    }
    exportRows(filteredItems, selectedColumns, format, filename);
    toast.success(`已导出 ${filteredItems.length} 条证书记录`);
    onOpenChange(false);
  }

  function toggleAllColumns() {
    setSelectedColumnIds(allSelected ? [] : columns.map(column => column.id ?? column.header));
  }

  function toggleColumn(id: string) {
    setSelectedColumnIds(current =>
      current.includes(id) ? current.filter(item => item !== id) : [...current, id]
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="zl-dialog-panel max-h-[88vh] gap-0 overflow-hidden p-0 sm:max-w-[680px]"
        onOpenAutoFocus={event => event.preventDefault()}
      >
        <DialogHeader className="border-b border-[var(--zl-border)] px-5 py-4">
          <DialogTitle>导出证书数据</DialogTitle>
          <DialogDescription>
            将当前筛选后的 {filteredItems.length} 条证书进行导出
          </DialogDescription>
        </DialogHeader>
        <div className="zl-hidden-scrollbar space-y-5 overflow-y-auto p-5">
          <div className="grid gap-4 md:grid-cols-[1fr_180px]">
            <label className="space-y-1.5 text-xs text-[var(--zl-text-muted)]">
              <div>导出名称</div>
              <input
                value={filename}
                onChange={event => setFilename(event.target.value)}
                className="zl-form-control h-10 w-full rounded-lg px-3 text-sm"
              />
            </label>
            <div className="space-y-1.5 text-xs text-[var(--zl-text-muted)]">
              <div>扩展名</div>
              <Select value={format} onValueChange={value => setFormat(value as ExportFormat)}>
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {formats.map(item => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-3 rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/60 p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs font-semibold text-[var(--zl-text)]">导出字段</p>
                <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
                  默认导出全部字段，可按需取消
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={toggleAllColumns}>
                {allSelected ? '取消全选' : '全选'}
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
              {columns.map(column => {
                const id = column.id ?? column.header;
                const selected = selectedColumnIds.includes(id);
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => toggleColumn(id)}
                    className={`zl-action-button flex h-10 min-w-0 items-center gap-2 rounded-lg border px-3 text-left text-sm transition ${selected ? 'border-teal-400/40 bg-teal-400/10 text-[var(--zl-text)]' : 'border-[var(--zl-border)] bg-[var(--zl-control-bg)] text-[var(--zl-text-muted)]'}`}
                  >
                    <span
                      className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${selected ? 'border-teal-400/70 bg-teal-400/20' : 'border-[var(--zl-border)]'}`}
                    >
                      {selected ? <Check size={12} /> : null}
                    </span>
                    <span className="min-w-0 truncate">{column.header}</span>
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] px-3 py-3 text-sm text-[var(--zl-text-muted)]">
            <FileDown size={16} />
            <span className="min-w-0 flex-1 truncate">{previewName}</span>
            <span className="shrink-0 text-xs">
              {selectedColumns.length}/{columns.length} 列
            </span>
          </div>
        </div>
        <DialogFooter className="border-t border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/40 px-5 py-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            onClick={handleExport}
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              color: 'var(--zl-accent-text)',
              background: 'rgba(59,130,246,0.1)',
            }}
          >
            <Download size={15} />
            导出
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function certificateExportColumns(caNames: Map<string, string>): ExportColumn<Certificate>[] {
  return [
    { id: 'commonName', header: '通用名称', value: item => item.commonName },
    { id: 'serialNumber', header: '序列号', value: item => item.serialNumber },
    { id: 'issuer', header: '颁发者', value: item => caNames.get(item.caId) ?? item.caId },
    { id: 'subject', header: 'Subject', value: item => item.subject },
    { id: 'san', header: 'SAN', value: item => item.san.join('; ') },
    { id: 'algorithm', header: '算法', value: item => item.algorithm },
    {
      id: 'source',
      header: '来源',
      value: item => (item.source === 'system' ? '系统生成' : 'CSR'),
    },
    {
      id: 'notBefore',
      header: '生效时间',
      value: item => formatDate(item.notBefore),
    },
    {
      id: 'notAfter',
      header: '到期时间',
      value: item => formatDate(item.notAfter),
    },
    { id: 'status', header: '状态', value: item => certificateStatusLabels[item.status] },
    { id: 'fingerprint', header: '指纹', value: item => item.fingerprint ?? '' },
  ];
}
