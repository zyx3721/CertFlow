import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

export function DataTableShell({
  toolbar,
  footer,
  children,
  className,
  contentClassName,
}: {
  toolbar?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <section className={cn('zl-surface-3d overflow-hidden rounded-xl border', className)}>
      {toolbar ? (
        <div className="flex flex-wrap items-center gap-3 border-b border-[var(--zl-border)] p-3">
          {toolbar}
        </div>
      ) : null}
      <div className={cn('overflow-x-auto', contentClassName)}>{children}</div>
      {footer}
    </section>
  );
}
