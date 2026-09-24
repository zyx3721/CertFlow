import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export const pkiQueryKeys = {
  cas: ['pki', 'cas'] as const,
  certificates: ['pki', 'certificates'] as const,
  trend: ['pki', 'certificate-trend'] as const,
  workflows: ['pki', 'workflows'] as const,
  crl: ['pki', 'crl'] as const,
  crlMetadata: ['pki', 'crl-metadata'] as const,
  audits: ['pki', 'audits'] as const,
  settings: ['pki', 'settings'] as const,
};

export function formatDate(value?: string, includeSeconds = false) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return '-';
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    ...(includeSeconds ? { second: '2-digit' } : {}),
  }).format(date);
}

export const certificateStatusLabels: Record<string, string> = {
  pending: '待审批',
  valid: '有效',
  revoked: '已撤销',
  rejected: '已驳回',
  expired: '已过期',
};

export const revokeReasonLabels = {
  unspecified: '未指定',
  keyCompromise: '密钥泄露',
  cACompromise: 'CA 泄露',
  affiliationChanged: '归属变更',
  superseded: '已替换',
  cessationOfOperation: '停止运营',
} as const;

// usePageRefresh 监听全局刷新事件，支持失效 react-query 缓存或执行自定义加载回调
export function usePageRefresh(target: readonly (readonly string[])[] | (() => void)) {
  const queryClient = useQueryClient();
  const targetRef = useRef(target);
  useEffect(() => {
    targetRef.current = target;
  });
  useEffect(() => {
    const refresh = () => {
      const current = targetRef.current;
      if (typeof current === 'function') {
        current();
        return;
      }
      for (const queryKey of current) {
        void queryClient.invalidateQueries({ queryKey });
      }
    };
    window.addEventListener('certflow:refresh', refresh);
    return () => window.removeEventListener('certflow:refresh', refresh);
  }, [queryClient]);
}
