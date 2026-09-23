import { Link, useNavigate } from '@tanstack/react-router';
import { Eye, EyeOff, Loader2, Lock, Moon, Sun, User } from 'lucide-react';
import { type FormEvent, useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { useBrandSettings } from '@/lib/branding';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  consumeAuthExpired,
  fetchCurrentUser,
  fetchPublicAuthProviders,
  getAuthToken,
  login,
  persistSession,
  persistUser,
  setCurrentUserSnapshot,
  wecomBindByCode,
  wecomBindByTicket,
  WECOM_BIND_RESULT_EVENT,
  wecomLoginByCode,
  wecomLoginByTicket,
  type AuthSession,
  type PublicAuthProvider,
} from '@/lib/auth';
import {
  applyZlTheme,
  getInitialZlTheme,
  persistZlTheme,
  toggleZlTheme,
  type ZlTheme,
} from '@/lib/utils';
import { WecomCallbackView } from './WecomCallbackView';
import { WecomQrLogin, type WecomEmbedSuccess } from './WecomQrLogin';

export function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [provider, setProvider] = useState('local');
  const [providers, setProviders] = useState<PublicAuthProvider[]>([]);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [wecomAuthorizing, setWecomAuthorizing] = useState(false);
  const [theme, setTheme] = useState<ZlTheme>(getInitialZlTheme);
  const brand = useBrandSettings();
  const toggleLabel = theme === 'dark' ? '切换浅色背景' : '切换深色背景';
  const [callbackParams] = useState(() => {
    const params = new URLSearchParams(typeof window === 'undefined' ? '' : window.location.search);
    return {
      code: params.get('code') ?? '',
      state: params.get('state') ?? '',
      ticket: params.get('ticket') ?? '',
    };
  });
  const isDirectCallback = Boolean(callbackParams.code && callbackParams.state);
  const isSSOCallback = Boolean(callbackParams.ticket);
  const isCallbackView = isDirectCallback || isSSOCallback;
  const isWecomProvider = provider === 'wecom';

  const finishWecomLogin = useCallback(
    (session: AuthSession) => {
      toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
      window.history.replaceState({}, '', window.location.pathname);
      void navigate({ to: '/', replace: true });
    },
    [navigate]
  );

  useEffect(() => {
    applyZlTheme(theme);
    persistZlTheme(theme);
  }, [theme]);

  useEffect(() => {
    if (consumeAuthExpired()) {
      toast.error('登录会话已过期，请重新登录');
    }
  }, []);

  useEffect(() => {
    if (isCallbackView) return;
    let cancelled = false;
    if (getAuthToken()) {
      void fetchCurrentUser()
        .then(user => {
          if (cancelled) return;
          persistUser(user);
          void navigate({ to: '/', replace: true });
        })
        .catch(() => undefined);
    }
    return () => {
      cancelled = true;
    };
  }, [isCallbackView, navigate]);

  useEffect(() => {
    let cancelled = false;
    fetchPublicAuthProviders()
      .then(response => {
        if (cancelled) return;
        setProviders(
          response.items.filter(
            item => item.enabled && (item.type === 'ldap' || item.type === 'wecom')
          )
        );
      })
      .catch(() => {
        if (!cancelled) {
          setProviders([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!isCallbackView) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        if (getAuthToken()) {
          const binding = isSSOCallback
            ? await wecomBindByTicket({ ticket: callbackParams.ticket })
            : await wecomBindByCode({ code: callbackParams.code, state: callbackParams.state });
          if (!binding.bound) {
            throw new Error('企业微信账号绑定失败');
          }
          window.opener?.postMessage(
            { type: WECOM_BIND_RESULT_EVENT, ok: true },
            window.location.origin
          );
          toast.success('企业微信绑定成功');
          window.history.replaceState({}, '', window.location.pathname);
          window.setTimeout(() => window.close(), 300);
          return;
        }
        const session = isSSOCallback
          ? await wecomLoginByTicket({ ticket: callbackParams.ticket })
          : await wecomLoginByCode({ code: callbackParams.code, state: callbackParams.state });
        if (cancelled) return;
        persistSession(session);
        setCurrentUserSnapshot(session.user);
        finishWecomLogin(session);
        return;
      } catch (err) {
        const message = err instanceof Error ? err.message : '企业微信登录失败，请稍后重试';
        if (!cancelled) setError(message);
        if (getAuthToken()) {
          window.opener?.postMessage(
            { type: WECOM_BIND_RESULT_EVENT, ok: false, message },
            window.location.origin
          );
          if (window.opener) {
            window.setTimeout(() => window.close(), 1500);
          }
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isCallbackView, isSSOCallback, navigate]);

  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      if (isCallbackView) {
        window.location.replace('/login');
        return;
      }
      setLoading(false);
    };
    window.addEventListener('pageshow', handlePageShow);
    return () => window.removeEventListener('pageshow', handlePageShow);
  }, [isCallbackView]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedUsername = username.trim();
    if (!normalizedUsername || !password) {
      setError('用户名或密码不能为空');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const session = await login(normalizedUsername, password, provider);
      toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
      void navigate({ to: '/', replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }

  async function handleWecomEmbedSuccess(payload: WecomEmbedSuccess) {
    setWecomAuthorizing(true);
    setError('');
    try {
      const session =
        payload.authMode === 'sso'
          ? await wecomLoginByTicket({ ticket: payload.ticket ?? '' })
          : await wecomLoginByCode({ code: payload.code ?? '', state: payload.state ?? '' });
      persistSession(session);
      setCurrentUserSnapshot(session.user);
      finishWecomLogin(session);
    } catch (err) {
      setError(err instanceof Error ? err.message : '企业微信登录失败，请稍后重试');
    }
  }

  if (isCallbackView || wecomAuthorizing) {
    return <WecomCallbackView error={error} />;
  }

  return (
    <main
      data-cmp="Login"
      className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
      style={{
        background:
          'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), radial-gradient(circle at 12% 22%, rgba(6,182,212,0.18), transparent 28%), radial-gradient(circle at 88% 82%, rgba(16,185,129,0.14), transparent 30%), var(--zl-login-bg)',
        color: 'var(--zl-text)',
      }}
    >
      <AppTooltip
        label={toggleLabel}
        placement="bottom"
        className="absolute right-5 top-5 z-20 sm:right-6 sm:top-6"
      >
        <button
          type="button"
          onClick={() => setTheme(toggleZlTheme)}
          className="zl-action-button grid h-11 w-11 place-items-center rounded-lg border transition-transform hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--zl-accent-text)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--zl-login-bg)]"
          style={{
            background: 'var(--zl-control-bg)',
            borderColor: 'var(--zl-border)',
            color: 'var(--zl-text-muted)',
            boxShadow: 'var(--zl-menu-shadow)',
          }}
          aria-label={toggleLabel}
        >
          {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
        </button>
      </AppTooltip>
      <div className="zl-login-grid absolute inset-0" aria-hidden="true" />
      <div
        className="zl-login-orb absolute left-[-6rem] top-20 h-64 w-64 rounded-full"
        aria-hidden="true"
      />
      <div
        className="zl-login-orb absolute bottom-[-4rem] right-[-5rem] h-80 w-80 rounded-full"
        aria-hidden="true"
      />

      <section className="relative z-10 flex w-full max-w-[460px] flex-col items-center gap-5">
        <CertFlowBrand brand={brand} />
        <section
          className="zl-login-frame zl-login-reveal w-full rounded-[28px] p-1"
          style={{ animationDelay: '80ms' }}
        >
          <div
            className="rounded-[24px] p-6 sm:p-8"
            style={{
              background: 'var(--zl-login-panel-bg)',
              border: '1px solid var(--zl-border)',
              backdropFilter: 'blur(18px)',
              boxShadow: 'var(--zl-login-panel-shadow)',
            }}
          >
            <div className="mb-7 text-center">
              <h1
                className="text-2xl font-bold"
                style={{ color: 'var(--zl-text)', textShadow: '0 0 22px rgba(6,182,212,0.16)' }}
              >
                欢迎回来
              </h1>
              <p className="mt-2 text-sm" style={{ color: 'var(--zl-text-muted)' }}>
                登录您的账户以继续
              </p>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit} noValidate>
              {providers.length > 0 ? (
                <LoginProviderSelect
                  value={provider}
                  providers={providers}
                  onChange={setProvider}
                />
              ) : null}
              {isWecomProvider ? (
                <WecomQrLogin onSuccess={handleWecomEmbedSuccess} />
              ) : (
                <>
                  <AuthInput id="username" label="用户名" icon={<User size={17} />}>
                    <input
                      id="username"
                      autoComplete="username"
                      value={username}
                      onChange={event => setUsername(event.target.value)}
                      className="h-12 w-full rounded-2xl py-3 pl-12 pr-4 text-sm outline-none transition-all"
                      style={inputStyle}
                      placeholder="请输入用户名"
                    />
                  </AuthInput>
                  <div>
                    <div className="mb-2 flex items-center justify-between">
                      <label className="block text-sm font-medium" htmlFor="password">
                        密码
                      </label>
                      <Link
                        to="/forgot-password"
                        className="text-xs font-semibold"
                        style={{ color: 'var(--zl-accent-text)' }}
                      >
                        忘记密码?
                      </Link>
                    </div>
                    <div className="relative">
                      <Lock
                        className="absolute left-4 top-1/2 -translate-y-1/2"
                        size={17}
                        style={{ color: 'var(--zl-text-muted)' }}
                      />
                      <input
                        id="password"
                        autoComplete="current-password"
                        type={showPassword ? 'text' : 'password'}
                        value={password}
                        onChange={event => setPassword(event.target.value)}
                        className="h-12 w-full rounded-2xl py-3 pl-12 pr-12 text-sm outline-none transition-all"
                        style={inputStyle}
                        placeholder="请输入密码"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(value => !value)}
                        className="absolute right-2 top-1/2 grid h-9 w-9 -translate-y-1/2 place-items-center rounded-xl"
                        style={{ color: 'var(--zl-text-muted)' }}
                        aria-label={showPassword ? '隐藏密码' : '显示密码'}
                      >
                        {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                      </button>
                    </div>
                  </div>
                </>
              )}
              <div aria-live="polite" className="min-h-6">
                {error ? (
                  <p
                    role="alert"
                    className="rounded-xl px-3 py-2 text-xs"
                    style={{
                      background: 'rgba(239,68,68,0.12)',
                      border: '1px solid rgba(239,68,68,0.28)',
                      color: '#fca5a5',
                    }}
                  >
                    {error}
                  </p>
                ) : null}
              </div>
              {!isWecomProvider ? (
                <button
                  type="submit"
                  disabled={loading}
                  className="zl-login-submit flex h-12 w-full items-center justify-center gap-2 rounded-2xl text-sm font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-60"
                  style={{
                    background: 'linear-gradient(135deg, #2563eb, #06b6d4)',
                    color: '#fff',
                    boxShadow: '0 18px 48px rgba(37,99,235,0.35)',
                  }}
                >
                  {loading ? <Loader2 size={17} className="zl-spinner" /> : null}
                  {loading ? '处理中...' : '登录'}
                </button>
              ) : null}
            </form>
          </div>
        </section>
        <p className="text-xs" style={{ color: 'var(--zl-text-muted)' }}>
          (c) 2026 {brand.siteName}. Secure PKI certificate operations console.
        </p>
      </section>
    </main>
  );
}

const inputStyle = {
  background: 'var(--zl-control-bg)',
  border: '1px solid var(--zl-border)',
  color: 'var(--zl-text)',
};

function CertFlowBrand({ brand }: { brand: ReturnType<typeof useBrandSettings> }) {
  return (
    <div className="zl-login-reveal flex flex-col items-center text-center">
      <img
        className="mb-4 h-20 w-20 drop-shadow-[0_18px_42px_rgba(6,182,212,0.18)]"
        src={brand.iconData}
        alt={brand.loginName}
      />
      <p className="zl-gradient-text text-2xl font-bold tracking-wide">{brand.loginName}</p>
      <p
        className="mt-1 text-xs uppercase tracking-[0.28em]"
        style={{ color: 'var(--zl-text-muted)' }}
      >
        PKI CONTROL PLANE
      </p>
    </div>
  );
}

function AuthInput({
  id,
  label,
  icon,
  children,
}: {
  id: string;
  label: string;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <div className="relative">
        <span
          className="absolute left-4 top-1/2 -translate-y-1/2"
          style={{ color: 'var(--zl-text-muted)' }}
        >
          {icon}
        </span>
        {children}
      </div>
    </div>
  );
}

function LoginProviderSelect({
  value,
  providers,
  onChange,
}: {
  value: string;
  providers: PublicAuthProvider[];
  onChange: (value: string) => void;
}) {
  const options = [
    { id: 'local', name: '本地账号' },
    ...providers.map(item => ({
      id: item.id,
      name: item.name || (item.type === 'wecom' ? '企业微信' : 'AD/LDAP'),
    })),
  ];
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor="login-provider-select">
        登录方式
      </label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="login-provider-select" className="h-12 rounded-2xl px-4 font-normal">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map(item => (
            <SelectItem key={item.id} value={item.id} className="font-normal">
              {item.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
