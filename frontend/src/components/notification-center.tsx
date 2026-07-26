import { useQuery } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { createPortal } from 'react-dom';
import { useEffect, useMemo, useRef, useState } from 'react';
import { AppTooltip } from '@/components/app-tooltip';
import { getCertificates, getWorkflows } from '@/lib/pki';
import { useBrandSettings } from '@/lib/branding';
import type { AuthUser } from '@/lib/auth';
import { userHasPermission } from '@/lib/auth';
import { pkiQueryKeys } from '@/features/pki/support';
import {
  NotificationPanel,
  type NotificationPanelPosition,
  type SystemNotification,
} from './notification-panel';

const notificationRefreshEvent = 'certflow:refresh';
export function NotificationCenter({ user }: { user: AuthUser | null }) {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<NotificationPanelPosition>({ top: 0, right: 16 });
  const triggerRef = useRef<HTMLDivElement | null>(null);
  const panelRef = useRef<HTMLDivElement | null>(null);
  const canReadCertificates = userHasPermission(user, 'certificates.read');
  const canApproveWorkflows = userHasPermission(user, 'workflows.approve');
  const brand = useBrandSettings();
  const certificatesQuery = useQuery({
    queryKey: pkiQueryKeys.certificates,
    queryFn: getCertificates,
    enabled: canReadCertificates,
  });
  const workflowsQuery = useQuery({
    queryKey: pkiQueryKeys.workflows,
    queryFn: getWorkflows,
    enabled: canApproveWorkflows,
  });
  const expiryNotificationDays = brand.expiryNotificationDays;
  const certificateExpirationWindowMs = expiryNotificationDays * 86_400_000;
  const notifications = useMemo<SystemNotification[]>(() => {
    const now = Date.now();
    const expiringCertificates = canReadCertificates
      ? (certificatesQuery.data?.items ?? []).filter(certificate => {
          const expiresAt = Date.parse(certificate.notAfter);
          return (
            certificate.status === 'valid' &&
            !Number.isNaN(expiresAt) &&
            expiresAt > now &&
            expiresAt - now <= certificateExpirationWindowMs
          );
        })
      : [];
    const pendingWorkflows = canApproveWorkflows
      ? (workflowsQuery.data?.items ?? []).filter(workflow => workflow.status === 'pending')
      : [];
    return [
      ...(expiringCertificates.length
        ? [
            {
              id: 'certificates-expiring',
              level: 'warning' as const,
              title: '证书即将到期',
              detail: `${expiringCertificates.length} 张有效证书将在 ${expiryNotificationDays} 天内到期`,
              to: '/certificates' as const,
            },
          ]
        : []),
      ...(pendingWorkflows.length
        ? [
            {
              id: 'workflows-pending',
              level: 'info' as const,
              title: '待处理审批',
              detail: `${pendingWorkflows.length} 条证书申请等待审批`,
              to: '/workflows' as const,
            },
          ]
        : []),
    ];
  }, [
    canApproveWorkflows,
    canReadCertificates,
    certificatesQuery.data?.items,
    certificateExpirationWindowMs,
    expiryNotificationDays,
    workflowsQuery.data?.items,
  ]);

  function updatePosition() {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    setPosition({
      top: Math.round(rect.bottom + 8),
      right: Math.max(16, Math.round(window.innerWidth - rect.right)),
    });
  }

  useEffect(() => {
    const refresh = () => {
      if (canReadCertificates) void certificatesQuery.refetch();
      if (canApproveWorkflows) void workflowsQuery.refetch();
    };
    window.addEventListener(notificationRefreshEvent, refresh);
    return () => window.removeEventListener(notificationRefreshEvent, refresh);
  }, [canApproveWorkflows, canReadCertificates, certificatesQuery, workflowsQuery]);

  useEffect(() => {
    if (!open) return;
    updatePosition();
    const close = (event: MouseEvent) => {
      const target = event.target as Node;
      if (!triggerRef.current?.contains(target) && !panelRef.current?.contains(target))
        setOpen(false);
    };
    window.addEventListener('resize', updatePosition);
    window.addEventListener('scroll', updatePosition, true);
    document.addEventListener('mousedown', close);
    return () => {
      window.removeEventListener('resize', updatePosition);
      window.removeEventListener('scroll', updatePosition, true);
      document.removeEventListener('mousedown', close);
    };
  }, [open]);

  const loading =
    (canReadCertificates && certificatesQuery.isLoading) ||
    (canApproveWorkflows && workflowsQuery.isLoading);

  return (
    <div ref={triggerRef} className="relative hidden sm:block">
      <AppTooltip label="通知消息" placement="bottom">
        <button
          type="button"
          className="zl-action-button relative flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
          aria-label={`通知消息${notifications.length ? `，${notifications.length} 条待关注` : ''}`}
          aria-haspopup="dialog"
          aria-expanded={open}
          onClick={() => {
            if (!open) updatePosition();
            setOpen(value => !value);
          }}
          style={{ borderColor: open ? 'rgba(59,130,246,0.48)' : 'var(--zl-border)' }}
        >
          <Bell size={17} />
          {notifications.length ? (
            <span className="absolute -right-1 -top-1 min-w-4 rounded-full bg-red-500 px-1 text-center text-[10px] leading-4 text-white">
              {notifications.length > 9 ? '9+' : notifications.length}
            </span>
          ) : null}
        </button>
      </AppTooltip>
      {open && typeof document !== 'undefined'
        ? createPortal(
            <NotificationPanel
              ref={panelRef}
              position={position}
              notifications={notifications}
              loading={loading}
              hasError={
                (canReadCertificates && certificatesQuery.isError) ||
                (canApproveWorkflows && workflowsQuery.isError)
              }
              onClose={() => setOpen(false)}
            />,
            document.body
          )
        : null}
    </div>
  );
}
