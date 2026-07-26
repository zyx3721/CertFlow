import type { ReactNode } from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { pageSizeOptions } from './pagination-options';

export function Pagination({
  page,
  pageCount,
  pageSize,
  total,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  pageCount: number;
  pageSize: number;
  total: number;
  onPageChange: (value: number) => void;
  onPageSizeChange: (value: (typeof pageSizeOptions)[number]) => void;
}) {
  const start = (page - 1) * pageSize + 1;
  return (
    <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-[var(--zl-border)] px-4 py-3 text-xs text-[var(--zl-text-muted)]">
      <div className="flex flex-wrap items-center gap-2">
        <span>
          显示 {start}-{Math.min(page * pageSize, total)} 条，共 {total} 条
        </span>
        <span>每页</span>
        <Select
          value={String(pageSize)}
          onValueChange={value =>
            onPageSizeChange(Number(value) as (typeof pageSizeOptions)[number])
          }
        >
          <SelectTrigger className="h-8 w-[76px] px-2 text-xs font-normal">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map(size => (
              <SelectItem key={size} value={String(size)} className="h-8 text-xs font-normal">
                {size}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span>条</span>
      </div>
      <div className="flex items-center gap-2">
        <PaginationButton label="首页" disabled={page === 1} onClick={() => onPageChange(1)}>
          <ChevronsLeft size={15} />
        </PaginationButton>
        <PaginationButton
          label="上一页"
          disabled={page === 1}
          onClick={() => onPageChange(Math.max(1, page - 1))}
        >
          <ChevronLeft size={15} />
        </PaginationButton>
        <span className="min-w-14 text-center text-sm text-[var(--zl-text)]">
          {page} / {pageCount}
        </span>
        <PaginationButton
          label="下一页"
          disabled={page === pageCount}
          onClick={() => onPageChange(Math.min(pageCount, page + 1))}
        >
          <ChevronRight size={15} />
        </PaginationButton>
        <PaginationButton
          label="末页"
          disabled={page === pageCount}
          onClick={() => onPageChange(pageCount)}
        >
          <ChevronsRight size={15} />
        </PaginationButton>
      </div>
    </div>
  );
}

function PaginationButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <AppTooltip label={label} disabled={disabled}>
      <Button
        size="icon"
        variant="outline"
        disabled={disabled}
        onClick={onClick}
        aria-label={label}
      >
        {children}
      </Button>
    </AppTooltip>
  );
}
