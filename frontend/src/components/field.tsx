import type { ReactNode } from 'react';

export function Field({
  label,
  hint,
  required,
  children,
}: {
  label: string;
  hint?: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <label className="grid gap-1.5 text-sm font-medium text-[var(--zl-text)]">
      <span>
        {label}
        {required ? <span className="ml-1 text-red-500">*</span> : null}
      </span>
      {children}
      {hint ? (
        <span className="text-xs font-normal text-[var(--zl-text-muted)]">{hint}</span>
      ) : null}
    </label>
  );
}
