import { AlertTriangle, CheckCircle2, Clock3, X } from 'lucide-react';
import { Link } from '@tanstack/react-router';
import { forwardRef } from 'react';

export type NotificationPanelPosition = {
  top: number;
  right: number;
};

export type SystemNotification = {
  id: string;
  level: 'warning' | 'info';
  title: string;
  detail: string;
  to: '/certificates' | '/workflows';
};

export const NotificationPanel = forwardRef<
  HTMLDivElement,
  {
    position: NotificationPanelPosition;
    notifications: SystemNotification[];
    loading: boolean;
    hasError: boolean;
    onClose: () => void;
  }
>(function NotificationPanel({ position, notifications, loading, hasError, onClose }, ref) {
  return (
    <div
      ref={ref}
      className="zl-notification-panel fixed z-[1300] flex w-[min(420px,calc(100vw-2rem))] flex-col overflow-hidden rounded-xl shadow-2xl"
      role="dialog"
      aria-label="通知消息"
      style={{ top: position.top, right: position.right }}
    >
      <div
        className="flex items-center justify-between gap-3 px-5 py-4"
        style={{ borderBottom: '1px solid var(--zl-border)' }}
      >
        <div>
          <div className="text-sm font-semibold text-[var(--zl-text)]">通知消息</div>
          <div className="mt-1 text-xs text-[var(--zl-text-muted)]">
            {hasError
              ? '部分提醒暂时无法读取'
              : notifications.length
                ? `${notifications.length} 条待关注提醒`
                : '当前状态正常'}
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="zl-action-button grid h-7 w-7 place-items-center rounded-md text-[var(--zl-text-muted)]"
          aria-label="关闭通知"
        >
          <X size={15} />
        </button>
      </div>
      <div className="zl-hidden-scrollbar max-h-[min(24rem,calc(100vh-7rem))] space-y-2 overflow-y-auto p-4">
        {loading ? (
          <div className="py-8 text-center text-sm text-[var(--zl-text-muted)]">正在读取提醒</div>
        ) : null}
        {!loading && hasError ? (
          <div className="py-8 text-center text-sm text-[var(--zl-text-muted)]">
            提醒数据暂时无法读取
          </div>
        ) : null}
        {!loading && !hasError && notifications.length === 0 ? (
          <div className="py-8 text-center text-sm text-[var(--zl-text-muted)]">
            <CheckCircle2 className="mx-auto mb-2 text-emerald-400" size={20} />
            暂无新的系统提醒
          </div>
        ) : null}
        {notifications.map(notification => (
          <Link
            key={notification.id}
            to={notification.to}
            onClick={onClose}
            className="block rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg)]/40 p-3 transition hover:bg-white/[0.06]"
          >
            <div className="flex items-start gap-2.5">
              <span
                className={`grid h-7 w-7 shrink-0 place-items-center rounded-full ${notification.level === 'warning' ? 'bg-amber-500/15 text-amber-400' : 'bg-blue-500/15 text-blue-400'}`}
              >
                {notification.level === 'warning' ? (
                  <AlertTriangle size={14} />
                ) : (
                  <Clock3 size={14} />
                )}
              </span>
              <span className="min-w-0">
                <span className="block text-sm font-medium text-[var(--zl-text)]">
                  {notification.title}
                </span>
                <span className="mt-0.5 block text-xs leading-5 text-[var(--zl-text-muted)]">
                  {notification.detail}
                </span>
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
});
