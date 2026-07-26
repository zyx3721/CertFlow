import { useEffect, useRef, useState } from 'react';
import { CalendarDays, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { createPortal } from 'react-dom';
import { AppTooltip } from '@/components/app-tooltip';
import { Input } from '@/components/ui/input';

export function CAExpiryDatePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [monthPickerOpen, setMonthPickerOpen] = useState(false);
  const [viewDate, setViewDate] = useState(() => parseCalendarDate(value) ?? new Date());
  const [position, setPosition] = useState({ top: 0, left: 0, width: 0, fixed: true });
  const selectedDate = parseCalendarDate(value);
  const firstDay = new Date(viewDate.getFullYear(), viewDate.getMonth(), 1);
  const firstWeekday = (firstDay.getDay() + 6) % 7;
  const calendarDays = Array.from(
    { length: 42 },
    (_, index) => new Date(viewDate.getFullYear(), viewDate.getMonth(), index - firstWeekday + 1)
  );

  useEffect(() => {
    const next = parseCalendarDate(value);
    if (next) setViewDate(next);
  }, [value]);

  useEffect(() => {
    if (!open) return;
    const closeOnOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!inputRef.current?.contains(target) && !panelRef.current?.contains(target)) {
        setOpen(false);
      }
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('pointerdown', closeOnOutside);
    window.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOnOutside);
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, [open]);

  function toggleCalendar() {
    if (open) {
      setOpen(false);
      setMonthPickerOpen(false);
      return;
    }
    const input = inputRef.current;
    const rect = input?.getBoundingClientRect();
    const dialog = input?.closest<HTMLElement>('[role="dialog"]') ?? null;
    if (rect) setPosition(calendarPosition(rect, dialog));
    setMonthPickerOpen(false);
    setOpen(true);
  }

  function selectDate(date: Date) {
    onChange(formatCalendarDate(date));
    setOpen(false);
    setMonthPickerOpen(false);
  }

  function selectMonth(month: number) {
    setViewDate(current => new Date(current.getFullYear(), month, 1));
    setMonthPickerOpen(false);
  }

  return (
    <div className="relative">
      <Input
        ref={inputRef}
        type="date"
        value={value}
        min="0001-01-01"
        max="9999-12-31"
        onChange={event => onChange(event.target.value)}
        className="cursor-text pr-11 [&::-webkit-calendar-picker-indicator]:pointer-events-none [&::-webkit-calendar-picker-indicator]:opacity-0"
        aria-label="到期日期"
      />
      <AppTooltip
        className="absolute right-1 top-1/2 inline-flex -translate-y-1/2"
        label="选择到期日期"
        placement="top"
        align="end"
      >
        <button
          type="button"
          onClick={toggleCalendar}
          className="zl-action-button grid h-8 w-8 place-items-center rounded-md text-[var(--zl-text-muted)]"
          aria-label="选择到期日期"
        >
          <CalendarDays size={16} />
        </button>
      </AppTooltip>
      {open && typeof document !== 'undefined'
        ? createPortal(
            <div
              ref={panelRef}
              role="dialog"
              aria-label="选择到期日期"
              className={`${position.fixed ? 'fixed' : 'absolute'} z-[1400] rounded-xl border p-3 shadow-2xl`}
              style={{
                top: position.top,
                left: position.left,
                width: position.width,
                background: 'var(--zl-popover-bg)',
                borderColor: 'var(--zl-popover-border)',
                boxShadow: 'var(--zl-menu-shadow)',
              }}
            >
              <div className="relative mb-3 flex items-center justify-between">
                <button
                  type="button"
                  onClick={() => setMonthPickerOpen(current => !current)}
                  className="zl-action-button inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm font-semibold"
                  aria-label="选择年份和月份"
                  aria-expanded={monthPickerOpen}
                >
                  {viewDate.getFullYear()}年{viewDate.getMonth() + 1}月
                  <ChevronDown size={14} className="text-[var(--zl-text-muted)]" />
                </button>
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() =>
                      setViewDate(
                        current => new Date(current.getFullYear(), current.getMonth() - 1, 1)
                      )
                    }
                    className="zl-action-button grid h-7 w-7 place-items-center rounded-md"
                    aria-label="上一个月"
                  >
                    <ChevronLeft size={16} />
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setViewDate(
                        current => new Date(current.getFullYear(), current.getMonth() + 1, 1)
                      )
                    }
                    className="zl-action-button grid h-7 w-7 place-items-center rounded-md"
                    aria-label="下一个月"
                  >
                    <ChevronRight size={16} />
                  </button>
                </div>
                {monthPickerOpen ? (
                  <div
                    className="absolute left-0 top-9 z-10 w-full rounded-lg border p-3 shadow-xl"
                    style={{
                      background: 'var(--zl-popover-bg)',
                      borderColor: 'var(--zl-popover-border)',
                      boxShadow: 'var(--zl-menu-shadow)',
                    }}
                  >
                    <div className="mb-2 flex items-center justify-between rounded-md bg-[var(--zl-control-bg-soft)] px-2 py-1.5">
                      <button
                        type="button"
                        onClick={() =>
                          setViewDate(
                            current => new Date(current.getFullYear() - 1, current.getMonth(), 1)
                          )
                        }
                        className="zl-action-button grid h-6 w-6 place-items-center rounded-md"
                        aria-label="上一年"
                      >
                        <ChevronLeft size={15} />
                      </button>
                      <span className="text-sm font-semibold">{viewDate.getFullYear()} 年</span>
                      <button
                        type="button"
                        onClick={() =>
                          setViewDate(
                            current => new Date(current.getFullYear() + 1, current.getMonth(), 1)
                          )
                        }
                        className="zl-action-button grid h-6 w-6 place-items-center rounded-md"
                        aria-label="下一年"
                      >
                        <ChevronRight size={15} />
                      </button>
                    </div>
                    <div className="grid grid-cols-3 gap-1.5">
                      {Array.from({ length: 12 }, (_, month) => {
                        const selected = month === viewDate.getMonth();
                        return (
                          <button
                            key={month}
                            type="button"
                            onClick={() => selectMonth(month)}
                            className={`rounded-md px-2 py-2 text-sm transition-colors ${
                              selected
                                ? 'bg-blue-500 font-semibold text-white shadow-sm'
                                : 'zl-action-button text-[var(--zl-text)]'
                            }`}
                          >
                            {month + 1}月
                          </button>
                        );
                      })}
                    </div>
                  </div>
                ) : null}
              </div>
              <div className="grid grid-cols-7 gap-1 text-center text-xs text-[var(--zl-text-muted)]">
                {['一', '二', '三', '四', '五', '六', '日'].map(day => (
                  <span key={day} className="grid h-7 place-items-center font-medium">
                    {day}
                  </span>
                ))}
                {calendarDays.map(date => {
                  const selected = sameCalendarDate(date, selectedDate);
                  const currentMonth = date.getMonth() === viewDate.getMonth();
                  return (
                    <button
                      key={date.toISOString()}
                      type="button"
                      onClick={() => selectDate(date)}
                      className={`grid h-8 place-items-center rounded-md text-sm transition-colors ${
                        selected
                          ? 'bg-blue-500 font-semibold text-white shadow-sm'
                          : currentMonth
                            ? 'zl-action-button text-[var(--zl-text)]'
                            : 'zl-action-button text-[var(--zl-text-muted)] opacity-55'
                      }`}
                      aria-pressed={selected}
                    >
                      {date.getDate()}
                    </button>
                  );
                })}
              </div>
              <div className="mt-3 flex justify-end border-t border-[var(--zl-border)] pt-3">
                <button
                  type="button"
                  onClick={() => selectDate(new Date())}
                  className="zl-action-button rounded-md px-2 py-1 text-xs font-medium text-blue-500"
                >
                  今天
                </button>
              </div>
            </div>,
            inputRef.current?.closest<HTMLElement>('[role="dialog"]') ?? document.body
          )
        : null}
    </div>
  );
}

function parseCalendarDate(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return null;
  const [, year, month, day] = match;
  const date = new Date(Number(year), Number(month) - 1, Number(day));
  return Number.isNaN(date.valueOf()) ? null : date;
}

function formatCalendarDate(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function sameCalendarDate(left: Date, right: Date | null) {
  return Boolean(
    right &&
    left.getFullYear() === right.getFullYear() &&
    left.getMonth() === right.getMonth() &&
    left.getDate() === right.getDate()
  );
}

function calendarPosition(rect: DOMRect, dialog: HTMLElement | null) {
  const width = rect.width;
  const gutter = 12;
  const panelHeight = 338;
  const gap = 8;
  const showAbove =
    window.innerHeight - rect.bottom < panelHeight && rect.top > window.innerHeight - rect.bottom;
  if (dialog) {
    const dialogRect = dialog.getBoundingClientRect();
    return {
      left: rect.left - dialogRect.left,
      width,
      top: showAbove
        ? rect.top - dialogRect.top - panelHeight - gap
        : rect.bottom - dialogRect.top + gap,
      fixed: false,
    };
  }
  const left = Math.min(Math.max(gutter, rect.left), window.innerWidth - width - gutter);
  return {
    left,
    width,
    top: showAbove ? Math.max(gutter, rect.top - panelHeight - gap) : rect.bottom + gap,
    fixed: true,
  };
}
