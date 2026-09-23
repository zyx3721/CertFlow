import { Link, Outlet, useLocation, useNavigate, useRouterState } from '@tanstack/react-router';
import type { CSSProperties, ReactNode } from 'react';
import {
  BadgeCheck,
  Ban,
  ChevronDown,
  ChevronRight,
  FilePlus2,
  GitBranch,
  KeyRound,
  Landmark,
  LayoutDashboard,
  Link2,
  LogOut,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  RefreshCw,
  ScrollText,
  Settings,
  Sun,
  User,
  type LucideIcon,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { PasswordDialog } from '@/components/password-dialog';
import { useBrandSettings } from '@/lib/branding';
import { NotificationCenter } from '@/components/notification-center';
import {
  AUTH_SESSION_CHANGED_EVENT,
  clearSession,
  fetchCurrentUser,
  fetchPublicAuthProviders,
  fetchWecomBindUrl,
  getAuthToken,
  getStoredUser,
  isAuthRedirecting,
  logout,
  setCurrentUserSnapshot,
  unbindWecom,
  userHasAnyPermission,
  WECOM_BIND_RESULT_EVENT,
  WECOM_PROVIDERS_CHANGED_EVENT,
} from '@/lib/auth';
import {
  applyZlTheme,
  cn,
  getInitialZlTheme,
  persistZlTheme,
  toggleZlTheme,
  type ZlTheme,
} from '@/lib/utils';

type NavItem = {
  to: string;
  label: string;
  description: string;
  permissions: string[];
  icon: LucideIcon;
};

const navigation: NavItem[] = [
  {
    to: '/',
    label: '仪表盘',
    description: 'PKI 基础设施实时状态总览',
    permissions: ['dashboard.read'],
    icon: LayoutDashboard,
  },
  {
    to: '/cas',
    label: 'CA 管理',
    description: 'Root、Intermediate 与 Issuing CA',
    permissions: ['ca.read'],
    icon: Landmark,
  },
  {
    to: '/certificates',
    label: '证书管理',
    description: '证书签发、下载与生命周期',
    permissions: ['certificates.read'],
    icon: BadgeCheck,
  },
  {
    to: '/request',
    label: '证书申请',
    description: '提交证书申请与 CSR',
    permissions: ['certificates.request'],
    icon: FilePlus2,
  },
  {
    to: '/crl',
    label: '证书撤销',
    description: 'CRL 条目与撤销状态',
    permissions: ['crl.read'],
    icon: Ban,
  },
  {
    to: '/workflows',
    label: '审批管理',
    description: '证书申请审批流程',
    permissions: ['workflows.read'],
    icon: GitBranch,
  },
  {
    to: '/audit',
    label: '审计日志',
    description: '安全操作与访问追踪',
    permissions: ['audit.read'],
    icon: ScrollText,
  },
  {
    to: '/settings',
    label: '系统配置',
    description: '基础、用户、认证与通知配置',
    permissions: [
      'settings.base.read',
      'settings.users.read',
      'settings.auth.read',
      'settings.notifications.read',
    ],
    icon: Settings,
  },
];

const sidebarItemHeight = 46;
const sidebarItemGap = 4;
const sidebarTransitionUnitMs = 1000;
const sidebarTransitionMaxMs = 3000;

function matchesPath(to: string, path: string) {
  return to === '/' ? path === '/' : path === to || path.startsWith(`${to}/`);
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const [token, setToken] = useState(getAuthToken);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    const sync = () => setToken(getAuthToken());
    window.addEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!token) {
      setChecking(false);
      if (!isAuthRedirecting()) void navigate({ to: '/login', replace: true });
      return;
    }
    void fetchCurrentUser()
      .then(() => {
        if (!cancelled) setChecking(false);
      })
      .catch(() => {
        if (cancelled) return;
        clearSession();
        setChecking(false);
        if (!isAuthRedirecting()) void navigate({ to: '/login', replace: true });
      });
    return () => {
      cancelled = true;
    };
  }, [navigate, token]);

  if (checking || !token) return null;
  return <>{children}</>;
}

export function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [theme, setTheme] = useState<ZlTheme>(getInitialZlTheme);
  const [user, setUser] = useState(getStoredUser);
  const brand = useBrandSettings();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [wecomEnabled, setWecomEnabled] = useState(false);
  const [transitioningTo, setTransitioningTo] = useState('');
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const mainContentRef = useRef<HTMLElement | null>(null);
  const pageContentRef = useRef<HTMLDivElement | null>(null);
  const pageEndSpacerRef = useRef<HTMLDivElement | null>(null);
  const previousNavIndexRef = useRef(0);
  const [sidebarIndicatorDuration, setSidebarIndicatorDuration] = useState(0);
  const [pageEndSpacing, setPageEndSpacing] = useState(0);
  const routeStatus = useRouterState({
    select: state => ({
      pending: state.status === 'pending' || state.isLoading || state.isTransitioning,
      matchPath: state.matches.at(-1)?.pathname ?? state.location.pathname,
    }),
  });

  useEffect(() => {
    applyZlTheme(theme);
    persistZlTheme(theme);
  }, [theme]);

  useEffect(() => {
    const sync = () => setUser(getStoredUser());
    window.addEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
    return () => window.removeEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
  }, []);

  useEffect(() => {
    const refreshWecomEnabled = () => {
      fetchPublicAuthProviders({ force: true })
        .then(response =>
          setWecomEnabled(response.items.some(item => item.enabled && item.type === 'wecom'))
        )
        .catch(() => undefined);
    };
    let cancelled = false;
    fetchPublicAuthProviders()
      .then(response => {
        if (cancelled) return;
        setWecomEnabled(response.items.some(item => item.enabled && item.type === 'wecom'));
      })
      .catch(() => undefined);
    window.addEventListener(WECOM_PROVIDERS_CHANGED_EVENT, refreshWecomEnabled);
    return () => {
      cancelled = true;
      window.removeEventListener(WECOM_PROVIDERS_CHANGED_EVENT, refreshWecomEnabled);
    };
  }, []);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      const data = event.data as { type?: string; ok?: boolean; message?: string } | null;
      if (data?.type !== WECOM_BIND_RESULT_EVENT) return;
      if (data.ok) {
        toast.success('企业微信绑定成功');
      } else {
        toast.error(String(data.message || '企业微信绑定失败'));
      }
      void fetchCurrentUser({ force: true })
        .then(fresh => {
          setCurrentUserSnapshot(fresh);
          setUser(fresh);
        })
        .catch(() => undefined);
    };
    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, []);

  async function startWecomBind() {
    setUserMenuOpen(false);
    try {
      const { url } = await fetchWecomBindUrl();
      const popup = window.open(url, 'certflow-wecom-bind', 'width=680,height=680');
      if (!popup) {
        toast.error('浏览器拦截了绑定窗口，请允许弹窗后重试');
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '获取企业微信绑定地址失败');
    }
  }

  async function handleUnbindWecom() {
    setUserMenuOpen(false);
    try {
      await unbindWecom();
      toast.success('已解绑企业微信');
      const fresh = await fetchCurrentUser({ force: true });
      setCurrentUserSnapshot(fresh);
      setUser(fresh);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '解绑企业微信失败');
    }
  }

  useEffect(() => {
    const close = (event: MouseEvent) => {
      const target = event.target as Node;
      if (userMenuRef.current && !userMenuRef.current.contains(target)) {
        setUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  const visibleItems = navigation.filter(item => userHasAnyPermission(user, item.permissions));
  const current = visibleItems.find(item => matchesPath(item.to, location.pathname));
  const activeIndex = Math.max(
    0,
    visibleItems.findIndex(item => item.to === current?.to)
  );

  useEffect(() => {
    if (!transitioningTo || routeStatus.matchPath !== transitioningTo) return;
    const frame = window.requestAnimationFrame(() => setTransitioningTo(''));
    return () => window.cancelAnimationFrame(frame);
  }, [routeStatus.matchPath, transitioningTo]);

  useEffect(() => {
    const distance = Math.abs(activeIndex - previousNavIndexRef.current);
    setSidebarIndicatorDuration(
      Math.min(sidebarTransitionMaxMs, distance * sidebarTransitionUnitMs)
    );
    previousNavIndexRef.current = activeIndex;
  }, [activeIndex]);

  useEffect(() => {
    const main = mainContentRef.current;
    const page = pageContentRef.current;
    if (!main || !page) return;

    let frame = 0;
    const measure = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        const spacerHeight = pageEndSpacerRef.current?.offsetHeight ?? 0;
        const mainRect = main.getBoundingClientRect();
        const surfaces = Array.from(
          page.querySelectorAll<HTMLElement>('.zl-surface-3d, .zl-card-hover')
        ).filter(element => element.offsetParent !== null);
        const lastSurface = surfaces.at(-1);
        const contentBottom = main.scrollHeight - spacerHeight;
        const trailingSpace = lastSurface
          ? contentBottom -
            (lastSurface.getBoundingClientRect().bottom - mainRect.top + main.scrollTop)
          : 0;
        const nextSpacing = Math.max(0, Math.ceil(24 - trailingSpace));
        setPageEndSpacing(current => (current === nextSpacing ? current : nextSpacing));
      });
    };
    const resizeObserver = new ResizeObserver(measure);
    const mutationObserver = new MutationObserver(measure);
    resizeObserver.observe(main);
    mutationObserver.observe(page, { childList: true, subtree: true });
    window.addEventListener('resize', measure);
    measure();
    return () => {
      window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      mutationObserver.disconnect();
      window.removeEventListener('resize', measure);
    };
  }, [location.pathname, routeStatus.pending, transitioningTo]);

  async function handleLogout() {
    setUserMenuOpen(false);
    setUser(null);
    void logout({ waitForRemote: false });
    toast.success('已退出登录');
    await navigate({ to: '/login', replace: true });
  }

  function refreshCurrentPage(showToast = false) {
    window.dispatchEvent(new Event('certflow:refresh'));
    if (showToast) {
      toast.success('页面数据已刷新');
    }
  }

  const shellVars = useMemo(
    () =>
      ({
        '--certflow-sidebar-width': sidebarOpen ? '240px' : '64px',
      }) as CSSProperties,
    [sidebarOpen]
  );

  return (
    <div
      data-cmp="CertFlowLayout"
      className="flex h-screen w-full overflow-hidden max-md:flex-col"
      style={{ ...shellVars, background: 'var(--zl-bg)', color: 'var(--zl-text)' }}
    >
      <aside
        className={cn(
          'zl-hidden-scrollbar z-30 flex shrink-0 flex-col overflow-y-auto border-r transition-all duration-300 max-md:h-auto max-md:w-full max-md:flex-row max-md:overflow-visible max-md:border-b max-md:border-r-0',
          sidebarOpen ? 'w-[240px]' : 'w-16'
        )}
        style={{
          background: 'var(--zl-sidebar)',
          borderColor: 'var(--zl-border)',
          boxShadow: 'var(--zl-shell-shadow)',
        }}
      >
        <div
          className="flex min-h-16 items-center gap-3 border-b px-3 py-4 max-md:w-16 max-md:shrink-0 max-md:justify-center max-md:border-b-0 max-md:border-r max-md:px-2"
          style={{ borderColor: 'var(--zl-border)' }}
        >
          <img
            src={brand.iconData}
            alt={brand.appName}
            className="h-9 w-9 shrink-0 rounded-lg object-contain shadow-[0_10px_28px_rgba(37,99,235,0.22)]"
          />
          {sidebarOpen ? (
            <span className="min-w-0 max-md:hidden">
              <span className="zl-gradient-text block truncate text-base font-bold">
                {brand.appName}
              </span>
              <span className="block truncate whitespace-nowrap text-xs tracking-[0.16em] text-[var(--zl-text-muted)] max-md:hidden">
                {brand.appSubtitle}
              </span>
            </span>
          ) : null}
        </div>

        <nav className="zl-hidden-scrollbar relative flex-1 px-2 py-4 max-md:flex max-md:overflow-x-auto max-md:py-2">
          {visibleItems.length > 0 ? (
            <span
              className="zl-sidebar-active-indicator"
              aria-hidden="true"
              style={
                {
                  '--zl-sidebar-active-top': `${activeIndex * (sidebarItemHeight + sidebarItemGap)}px`,
                  '--zl-sidebar-active-duration': `${sidebarIndicatorDuration}ms`,
                } as CSSProperties
              }
            />
          ) : null}
          {visibleItems.map(item => {
            const active = matchesPath(item.to, location.pathname);
            const Icon = item.icon;
            const link = (
              <Link
                key={item.to}
                to={item.to}
                preload={false}
                onClick={event => {
                  if (active) {
                    event.preventDefault();
                    return;
                  }
                  setTransitioningTo(item.to);
                }}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'zl-sidebar-button relative z-10 mb-1 flex h-[46px] items-center rounded-lg text-sm transition-all duration-150 max-md:mb-0 max-md:min-w-11',
                  sidebarOpen
                    ? 'justify-start px-3 max-md:justify-center max-md:px-2'
                    : 'justify-center px-2',
                  active && 'zl-sidebar-item-active'
                )}
                style={{ color: active ? 'var(--zl-accent)' : 'var(--zl-text-muted)' }}
              >
                <Icon size={17} aria-hidden="true" />
                {sidebarOpen ? (
                  <span className="ml-3 min-w-0 flex-1 truncate font-medium max-md:hidden">
                    {item.label}
                  </span>
                ) : null}
                {active && sidebarOpen ? (
                  <span className="ml-auto h-1.5 w-1.5 rounded-full bg-[var(--zl-accent)] max-md:hidden" />
                ) : null}
              </Link>
            );
            return sidebarOpen ? (
              link
            ) : (
              <AppTooltip key={item.to} label={item.label}>
                {link}
              </AppTooltip>
            );
          })}
        </nav>

        <div
          className="flex items-center justify-center border-t py-4 max-md:hidden"
          style={{ borderColor: 'var(--zl-border)' }}
        >
          <AppTooltip label={sidebarOpen ? '收起侧边栏' : '展开侧边栏'} placement="top">
            <button
              type="button"
              onClick={() => setSidebarOpen(value => !value)}
              className="zl-action-button flex h-9 w-9 items-center justify-center rounded-lg border"
              aria-label={sidebarOpen ? '收起侧边栏' : '展开侧边栏'}
            >
              {sidebarOpen ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
            </button>
          </AppTooltip>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className="relative z-20 flex h-16 shrink-0 items-center justify-between border-b px-6 backdrop-blur max-md:px-3"
          style={{
            background: 'var(--zl-header)',
            borderColor: 'var(--zl-border)',
            boxShadow: 'var(--zl-header-shadow)',
          }}
        >
          <div className="flex min-w-0 items-center gap-2">
            <span className="text-sm text-[var(--zl-text-muted)] max-sm:hidden">
              {brand.appName} 控制台
            </span>
            <ChevronRight size={14} className="text-[var(--zl-text-muted)] max-sm:hidden" />
            <span className="truncate text-sm font-medium">{current?.label ?? '无可用页面'}</span>
          </div>
          <div className="flex items-center gap-3">
            <AppTooltip label="刷新当前页面数据" placement="bottom">
              <button
                type="button"
                onClick={() => refreshCurrentPage(true)}
                className="zl-action-button flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
                aria-label="刷新当前页面数据"
              >
                <RefreshCw size={17} />
              </button>
            </AppTooltip>
            <NotificationCenter user={user} />
            <AppTooltip
              label={theme === 'dark' ? '切换浅色背景' : '切换深色背景'}
              placement="bottom"
            >
              <button
                type="button"
                onClick={() => setTheme(toggleZlTheme)}
                className="zl-action-button flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
                aria-label={theme === 'dark' ? '切换浅色背景' : '切换深色背景'}
              >
                {theme === 'dark' ? <Sun size={17} /> : <Moon size={17} />}
              </button>
            </AppTooltip>
            <div ref={userMenuRef} className="relative">
              <AppTooltip
                label={user?.displayName || user?.username || 'admin'}
                placement="bottom"
                align="end"
              >
                <button
                  type="button"
                  onClick={() => setUserMenuOpen(value => !value)}
                  className="zl-action-button grid h-[42px] w-[150px] grid-cols-[26px_minmax(0,1fr)_14px] items-center gap-2 rounded-lg border px-3 py-1.5 text-sm"
                  aria-haspopup="menu"
                  aria-expanded={userMenuOpen}
                  aria-label="用户菜单"
                >
                  <span className="grid h-[26px] w-[26px] place-items-center rounded-full bg-[linear-gradient(135deg,var(--zl-accent),var(--zl-accent2))] text-white">
                    <User size={14} />
                  </span>
                  <span className="min-w-0 truncate">
                    {user?.displayName || user?.username || 'admin'}
                  </span>
                  <ChevronDown
                    size={14}
                    className={
                      userMenuOpen ? 'rotate-180 transition-transform' : 'transition-transform'
                    }
                  />
                </button>
              </AppTooltip>
              {userMenuOpen ? (
                <div
                  role="menu"
                  className="absolute right-0 top-[calc(100%+8px)] z-[1300] w-full rounded-lg border p-2"
                  style={{
                    background: 'var(--zl-menu-bg)',
                    borderColor: 'var(--zl-border)',
                    boxShadow: 'var(--zl-menu-shadow)',
                  }}
                >
                  <button
                    type="button"
                    role="menuitem"
                    className="zl-menu-action-item flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                    onClick={() => {
                      setUserMenuOpen(false);
                      setPasswordOpen(true);
                    }}
                  >
                    <KeyRound size={16} />
                    修改密码
                  </button>
                  {wecomEnabled ? (
                    <button
                      type="button"
                      role="menuitem"
                      className="zl-menu-action-item flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                      onClick={() => {
                        if (user?.wecomBound) {
                          void handleUnbindWecom();
                          return;
                        }
                        void startWecomBind();
                      }}
                    >
                      <Link2 size={16} />
                      {user?.wecomBound ? '解绑企微' : '绑定企微'}
                    </button>
                  ) : null}
                  <button
                    type="button"
                    role="menuitem"
                    className="zl-menu-action-item zl-danger-button flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                    onClick={() => void handleLogout()}
                  >
                    <LogOut size={16} />
                    退出系统
                  </button>
                </div>
              ) : null}
            </div>
          </div>
        </header>

        <main
          ref={mainContentRef}
          className="zl-hidden-scrollbar min-h-0 flex-1 overflow-y-auto bg-[var(--zl-bg)]"
        >
          <div key={location.pathname} className="flex h-full min-h-full flex-col p-6 max-md:p-4">
            {visibleItems.length === 0 ? (
              <div className="grid min-h-[50vh] place-items-center text-sm text-[var(--zl-text-muted)]">
                当前账号没有可访问的功能
              </div>
            ) : transitioningTo || routeStatus.pending ? (
              <div className="grid min-h-[50vh] place-items-center text-sm text-[var(--zl-text-muted)]">
                正在加载页面...
              </div>
            ) : (
              <div
                ref={pageContentRef}
                key={`${location.pathname}:ready`}
                className="zl-page-enter min-h-0 flex-1"
              >
                <Outlet />
                {pageEndSpacing > 0 ? (
                  <div
                    ref={pageEndSpacerRef}
                    aria-hidden="true"
                    style={{ height: pageEndSpacing }}
                  />
                ) : null}
              </div>
            )}
          </div>
        </main>
      </div>

      <PasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </div>
  );
}
