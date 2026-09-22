const TOKEN_KEY = 'certflow.auth.token';
const USER_KEY = 'certflow.auth.user';
const EXPIRES_AT_KEY = 'certflow.auth.expires_at';
export const AUTH_SESSION_CHANGED_EVENT = 'certflow:auth-session-changed';
export const WECOM_BIND_RESULT_EVENT = 'certflow:wecom-bind-result';
export const WECOM_PROVIDERS_CHANGED_EVENT = 'certflow:wecom-providers-changed';

export type AuthUser = {
  id: string;
  username: string;
  email: string;
  displayName: string;
  role: string;
  source: 'local' | 'ldap';
  wecomBound?: boolean;
  permissions: string[];
};

export type AuthSession = {
  token: string;
  expiresAt: string;
  user: AuthUser;
};

export type PublicAuthProvider = {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
};

type ApiErrorResponse = {
  error?: string;
  message?: string;
};

type ApiOptions = RequestInit & {
  auth?: boolean;
};

type LogoutOptions = {
  waitForRemote?: boolean;
};

const CURRENT_USER_CACHE_TTL_MS = 1500;
const PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS = 1000;

export type PublicAuthConfiguration = {
  items: PublicAuthProvider[];
  total: number;
  passwordResetEnabled: boolean;
};

let pendingCurrentUser: Promise<AuthUser> | null = null;
let cachedCurrentUser: { user: AuthUser; expiresAt: number } | null = null;
let pendingPublicAuthProviders: Promise<PublicAuthConfiguration> | null = null;
let cachedPublicAuthProviders: {
  value: PublicAuthConfiguration;
  expiresAt: number;
} | null = null;

function storage() {
  if (typeof window === 'undefined') return null;
  return window.sessionStorage;
}

function emitSessionChanged() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED_EVENT));
}

export function getAuthToken() {
  return storage()?.getItem(TOKEN_KEY) ?? '';
}

export function getStoredUser(): AuthUser | null {
  const raw = storage()?.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    storage()?.removeItem(USER_KEY);
    return null;
  }
}

export function persistSession(session: AuthSession) {
  const store = storage();
  if (!store) return;
  store.setItem(TOKEN_KEY, session.token);
  store.setItem(USER_KEY, JSON.stringify(session.user));
  store.setItem(EXPIRES_AT_KEY, session.expiresAt);
  emitSessionChanged();
}

export function clearSession() {
  const store = storage();
  if (!store) return;
  store.removeItem(TOKEN_KEY);
  store.removeItem(USER_KEY);
  store.removeItem(EXPIRES_AT_KEY);
  emitSessionChanged();
}

export function userHasPermission(user: AuthUser | null, permission: string) {
  if (!user) return false;
  if (user.role === 'admin') return true;
  return user.permissions?.includes(permission) ?? false;
}

export function userHasAnyPermission(user: AuthUser | null, permissions: string[]) {
  return permissions.some(permission => userHasPermission(user, permission));
}

async function readApiError(response: Response) {
  try {
    const body = (await response.json()) as ApiErrorResponse;
    return normalizeApiErrorMessage(body.message || body.error || `请求失败：${response.status}`);
  } catch {
    return `请求失败：${response.status}`;
  }
}

function normalizeApiErrorMessage(message: string) {
  return message;
}

export function persistUser(user: AuthUser) {
  storage()?.setItem(USER_KEY, JSON.stringify(user));
}

export function setCurrentUserSnapshot(user: AuthUser) {
  cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
  persistUser(user);
}

export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json; charset=utf-8');
  }
  const token = getAuthToken();
  if (options.auth !== false && token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  let response: Response;
  try {
    response = await fetch(path, { ...options, headers });
  } catch {
    throw new Error('无法连接后端服务');
  }
  if (!response.ok) {
    const message = await readApiError(response);
    if (options.auth !== false && response.status === 401) {
      clearSession();
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.assign('/login');
      }
    }
    throw new Error(message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export async function login(username: string, password: string, provider = 'local') {
  const session = await api<AuthSession>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password, provider }),
  });
  persistSession(session);
  setCurrentUserSnapshot(session.user);
  return session;
}

export function fetchPublicAuthProviders(options: { force?: boolean } = {}) {
  if (
    !options.force &&
    cachedPublicAuthProviders &&
    cachedPublicAuthProviders.expiresAt > Date.now()
  ) {
    return Promise.resolve(cachedPublicAuthProviders.value);
  }
  if (pendingPublicAuthProviders) return pendingPublicAuthProviders;
  pendingPublicAuthProviders = api<PublicAuthConfiguration>('/api/v1/auth/providers', {
    auth: false,
  })
    .then(value => {
      cachedPublicAuthProviders = {
        value,
        expiresAt: Date.now() + PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS,
      };
      return value;
    })
    .finally(() => {
      pendingPublicAuthProviders = null;
    });
  return pendingPublicAuthProviders;
}

export function invalidatePublicAuthConfiguration() {
  pendingPublicAuthProviders = null;
  cachedPublicAuthProviders = null;
}

export function fetchCurrentUser(options: { force?: boolean } = {}) {
  if (!options.force && cachedCurrentUser && cachedCurrentUser.expiresAt > Date.now()) {
    return Promise.resolve(cachedCurrentUser.user);
  }
  if (pendingCurrentUser) return pendingCurrentUser;
  pendingCurrentUser = api<AuthUser>('/api/v1/auth/me')
    .then(user => {
      cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
      persistUser(user);
      return user;
    })
    .finally(() => {
      pendingCurrentUser = null;
    });
  return pendingCurrentUser;
}

export async function logout(options: LogoutOptions = {}) {
  const token = getAuthToken();
  pendingCurrentUser = null;
  cachedCurrentUser = null;
  clearSession();
  if (!token) return;

  const request = fetch('/api/v1/auth/logout', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
    keepalive: true,
  }).catch(() => undefined);

  if (options.waitForRemote === false) {
    void request;
    return;
  }

  await request;
}

export type PasswordResetCaptcha = {
  token: string;
  question: string;
  expiresAt: string;
};

export type PasswordResetChannel = {
  id: string;
  name: string;
  maskedTo: string;
  requiresTo: boolean;
};

export const fetchPasswordResetCaptcha = () =>
  api<PasswordResetCaptcha>('/api/v1/auth/password-reset/captcha', { auth: false });

export const verifyPasswordResetIdentity = (body: {
  username: string;
  captchaToken: string;
  captchaAnswer: string;
}) =>
  api<{ verificationToken: string; channels: PasswordResetChannel[] }>(
    '/api/v1/auth/password-reset/verify',
    {
      method: 'POST',
      auth: false,
      body: JSON.stringify(body),
    }
  );

export const sendPasswordResetCode = (body: {
  username: string;
  verificationToken: string;
  channel: string;
  verifyEmail: string;
}) =>
  api<{ status: string; cooldownSeconds: number }>('/api/v1/auth/password-reset/send', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export const confirmPasswordReset = (body: {
  username: string;
  verificationToken: string;
  code: string;
  newPassword: string;
  confirmPassword: string;
}) =>
  api<{ status: string }>('/api/v1/auth/password-reset/confirm', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export function changePassword(body: {
  old_password: string;
  new_password: string;
  confirm_password: string;
}) {
  return api<{ status: string }>('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

// 企业微信认证 API。绑定与状态接口手动携带 token 并关闭全局 401 跳转，
// 避免绑定弹窗中的失败把主窗口会话清空。
function manualAuthHeaders(): HeadersInit {
  const token = getAuthToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export type WecomAuthorizeResponse = { url: string };

export type WecomBinding = { bound: boolean; wecomUserid?: string };

export const fetchWecomAuthorizeUrl = () =>
  api<WecomAuthorizeResponse>('/api/v1/auth/wecom/authorize', { auth: false });

export const fetchWecomBindUrl = () =>
  api<WecomAuthorizeResponse>('/api/v1/auth/wecom/bind-url', {
    auth: false,
    headers: manualAuthHeaders(),
  });

export const wecomLoginByCode = (body: { code: string; state: string }) =>
  api<AuthSession>('/api/v1/auth/wecom/callback', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export const wecomLoginByTicket = (body: { ticket: string }) =>
  api<AuthSession>('/api/v1/auth/wecom/sso/callback', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export const wecomBindByCode = (body: { code: string; state: string }) =>
  api<WecomBinding>('/api/v1/auth/wecom/bind', {
    method: 'POST',
    auth: false,
    headers: manualAuthHeaders(),
    body: JSON.stringify(body),
  });

export const wecomBindByTicket = (body: { ticket: string }) =>
  api<WecomBinding>('/api/v1/auth/wecom/sso/bind', {
    method: 'POST',
    auth: false,
    headers: manualAuthHeaders(),
    body: JSON.stringify(body),
  });

export const unbindWecom = () =>
  api<WecomBinding>('/api/v1/auth/wecom/bind', {
    method: 'DELETE',
    auth: false,
    headers: manualAuthHeaders(),
  });
