import { api } from '@/lib/auth';

export type CAType = 'root' | 'intermediate' | 'issuing';
export type KeyAlgorithm = 'RSA-2048' | 'RSA-4096' | 'ECDSA-P256' | 'ECDSA-P384' | 'ED25519';
export type CertificateStatus = 'pending' | 'valid' | 'revoked' | 'rejected' | 'expired';
export type CertificatePurpose = 'server' | 'client' | 'mtls';
export type RevokeReason =
  | 'keyCompromise'
  | 'cACompromise'
  | 'affiliationChanged'
  | 'superseded'
  | 'cessationOfOperation'
  | 'unspecified';

export type CertificateAuthority = {
  id: string;
  name: string;
  type: CAType;
  algorithm: KeyAlgorithm;
  subject: string;
  parentId: string;
  certificatePem?: string;
  notBefore: string;
  notAfter: string;
  issuedCerts: number;
  revokedCerts: number;
  status: 'active' | 'inactive';
};

export type Certificate = {
  id: string;
  caId: string;
  serialNumber: string;
  commonName: string;
  subject: string;
  algorithm: KeyAlgorithm;
  purpose: CertificatePurpose;
  status: CertificateStatus;
  applicant: string;
  source: 'csr' | 'system';
  fingerprint?: string;
  certificatePem?: string;
  privateKeyPem?: string;
  csrPem?: string;
  rejectReason?: string;
  san: string[];
  notBefore: string;
  notAfter: string;
  revokedAt?: string;
  revokeReason?: RevokeReason;
};

export type Workflow = {
  id: string;
  certificateId: string;
  applicant: string;
  certName: string;
  status: 'pending' | 'approved' | 'rejected';
  rejectReason?: string;
  createdAt: string;
};

export type AuditEntry = {
  id: string;
  username: string;
  action: string;
  target: string;
  module: string;
  result: 'success' | 'failure';
  ip: string;
  detail: string;
  timestamp: string;
};

export type CRLEntry = {
  id: string;
  serialNumber: string;
  revokedAt: string;
  reason: RevokeReason;
  caId: string;
  caName: string;
  commonName: string;
};

export type CRLMetadata = {
  enabled: boolean;
  crlUrl: string;
  version: string;
  signatureAlgorithm: string;
  thisUpdate: string;
  nextUpdate: string;
  entryCount: number;
  intervalHours: number;
};

export type CRLFilters = {
  caId?: string;
  reason?: RevokeReason;
  keyword?: string;
  page?: number;
  pageSize?: number;
};

export type PkiSettings = {
  siteName: string;
  loginName: string;
  appName: string;
  appSubtitle: string;
  iconData: string;
  crlEnabled: boolean;
  crlUrl: string;
  ocspEnabled: boolean;
  ocspUrl: string;
  crlIntervalHours: number;
  renewDays: number;
  autoRenew: boolean;
  expiryNotificationDays: number;
  expiryNotificationCron: string;
  resetCodeTtlMinutes: number;
  resetCaptchaTtlMinutes: number;
  passwordResetSendCooldownMinutes: number;
  passwordResetRateLimitMinutes: number;
};

export type CertificateTrendPoint = {
  month: string;
  issued: number;
  revoked: number;
};

export type CertificateVerification = {
  valid: boolean;
  status: CertificateStatus | 'unknown';
  reason: string;
  checkedAt: string;
  commonName: string;
  serialNumber: string;
};

type List<T> = { items: T[]; total: number };
type PagedList<T> = List<T> & { page: number; pageSize: number };

export const getCAs = () => api<List<CertificateAuthority>>('/api/v1/cas');

export const createCA = (body: {
  name: string;
  type: CAType;
  algorithm: KeyAlgorithm;
  subject: string;
  parentId: string;
  notAfter: string;
}) => api<CertificateAuthority>('/api/v1/cas', { method: 'POST', body: JSON.stringify(body) });

export const deleteCA = (id: string) =>
  api<void>(`/api/v1/cas/${encodeURIComponent(id)}`, { method: 'DELETE' });

export const checkCADeletion = (id: string) =>
  api<{ deletable: boolean }>(`/api/v1/cas/${encodeURIComponent(id)}/deletion-check`);

export const getCACertificate = (id: string) =>
  api<{ filename: string; content: string }>(`/api/v1/cas/${encodeURIComponent(id)}/certificate`);

export async function downloadCACertificate(id: string) {
  const response = await fetch(`/api/v1/cas/${encodeURIComponent(id)}/download`, {
    headers: authHeaders(),
  });
  if (!response.ok) throw new Error('CA 证书下载失败');
  const disposition = response.headers.get('content-disposition') ?? '';
  const filename = /filename="?([^";]+)"?/i.exec(disposition)?.[1] ?? 'ca.pem';
  return { blob: await response.blob(), filename };
}

export const getCertificates = () => api<List<Certificate>>('/api/v1/certificates');

export const getCertificateTrend = () =>
  api<List<CertificateTrendPoint>>('/api/v1/certificates/trend');

export const createCertificateRequest = (body: {
  caID: string;
  commonName: string;
  subject: string;
  algorithm: KeyAlgorithm;
  purpose: CertificatePurpose;
  san: string[];
  validityDays: number;
  csrPEM?: string;
  privateKeyPEM?: string;
}) => api<Certificate>('/api/v1/certificates', { method: 'POST', body: JSON.stringify(body) });

export const previewCertificateCSR = (body: {
  subject: string;
  algorithm: KeyAlgorithm;
  san: string[];
}) =>
  api<{ csrPEM: string; privateKeyPEM: string }>('/api/v1/certificates/csr-preview', {
    method: 'POST',
    body: JSON.stringify(body),
  });

export const inspectCertificateCSR = (csrPEM: string) =>
  api<{
    subject: {
      commonName: string;
      org: string;
      orgUnit: string;
      country: string;
      province: string;
      locality: string;
    };
    san: string[];
    algorithm: KeyAlgorithm;
  }>('/api/v1/certificates/csr-inspect', {
    method: 'POST',
    body: JSON.stringify({ csrPEM }),
  });

export const getCertificatePackage = (id: string) =>
  api<Certificate>(`/api/v1/certificates/${encodeURIComponent(id)}/package`);

export async function downloadCertificate(id: string) {
  const response = await fetch(`/api/v1/certificates/${encodeURIComponent(id)}/download`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    let message = `证书下载失败：${response.status}`;
    try {
      const body = (await response.json()) as { message?: string; error?: string };
      message = body.message || body.error || message;
    } catch {
      // 非 JSON 错误响应沿用状态码信息。
    }
    throw new Error(message);
  }
  const disposition = response.headers.get('content-disposition') ?? '';
  const filename = /filename="?([^";]+)"?/i.exec(disposition)?.[1] ?? 'certificate.crt';
  return { blob: await response.blob(), filename };
}

export const verifyCertificate = (id: string) =>
  api<CertificateVerification>(`/api/v1/certificates/${encodeURIComponent(id)}/verify`);

export const revokeCertificate = (id: string, reason: RevokeReason) =>
  api<{ status: string }>(`/api/v1/certificates/${encodeURIComponent(id)}/revoke`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });

export const deleteCertificate = (id: string) =>
  api<void>(`/api/v1/certificates/${encodeURIComponent(id)}`, { method: 'DELETE' });

function crlQuery(filters: CRLFilters = {}) {
  const query = new URLSearchParams();
  Object.entries(filters).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value));
  });
  const text = query.toString();
  return text ? `?${text}` : '';
}

export const getCRLEntries = (filters: CRLFilters = {}) =>
  api<PagedList<CRLEntry>>(`/api/v1/crl${crlQuery(filters)}`);
export const getCRLMetadata = (filters: CRLFilters = {}) =>
  api<CRLMetadata>(`/api/v1/crl/metadata${crlQuery(filters)}`);

export async function downloadCRL(filters: CRLFilters = {}) {
  const response = await fetch(`/api/v1/crl/download${crlQuery(filters)}`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    let message = 'CRL 文件下载失败';
    try {
      const body = (await response.json()) as { message?: string; error?: string };
      message = body.message || body.error || message;
    } catch {
      // 非 JSON 错误响应沿用通用提示。
    }
    throw new Error(message);
  }
  const disposition = response.headers.get('Content-Disposition') ?? '';
  const filename = /filename="?([^";]+)"?/i.exec(disposition)?.[1] ?? '';
  return { blob: await response.blob(), filename };
}

export const getWorkflows = () => api<List<Workflow>>('/api/v1/workflows');

export const updateWorkflow = (
  id: string,
  body: { status: 'approved' | 'rejected'; reject_reason?: string }
) =>
  api<{ status: string }>(`/api/v1/workflows/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });

export const deleteWorkflow = (id: string) =>
  api<void>(`/api/v1/workflows/${encodeURIComponent(id)}`, { method: 'DELETE' });

export const getAuditEntries = (limit = 200) =>
  api<List<AuditEntry>>(`/api/v1/audit-logs?limit=${limit}`);

export const changePassword = (currentPassword: string, newPassword: string) =>
  api<void>('/api/v1/users/change-password', {
    method: 'POST',
    body: JSON.stringify({ currentPassword, newPassword }),
  });

export const getPkiSettings = () => api<PkiSettings>('/api/v1/settings');

export const savePkiSettings = (settings: PkiSettings) =>
  api<PkiSettings>('/api/v1/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });

function authHeaders() {
  const token = window.localStorage.getItem('certflow.auth.token') ?? '';
  return token ? { Authorization: `Bearer ${token}` } : undefined;
}

export function downloadText(content: string, filename: string, type = 'application/x-pem-file') {
  const blob = new Blob([content], { type });
  downloadBlob(blob, filename);
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
