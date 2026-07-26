import { BellRing, Network, SlidersHorizontal, UsersRound, type LucideIcon } from 'lucide-react';
import { useMemo, useState } from 'react';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import type { SettingsTab } from './types';
import { AuthSettingsPanel } from './AuthSettingsPanel';
import { BaseSettingsPanel } from './BaseSettingsPanel';
import { NotificationSettingsPanel } from './NotificationSettingsPanel';
import { UserSettingsPanel } from './UserSettingsPanel';

const tabs: Array<{
  id: SettingsTab;
  icon: LucideIcon;
  label: string;
  read: string;
  manage: string;
}> = [
  {
    id: 'base',
    icon: SlidersHorizontal,
    label: '基础配置',
    read: 'settings.base.read',
    manage: 'settings.base.manage',
  },
  {
    id: 'users',
    icon: UsersRound,
    label: '用户配置',
    read: 'settings.users.read',
    manage: 'settings.users.manage',
  },
  {
    id: 'auth',
    icon: Network,
    label: '认证配置',
    read: 'settings.auth.read',
    manage: 'settings.auth.manage',
  },
  {
    id: 'notifications',
    icon: BellRing,
    label: '通知配置',
    read: 'settings.notifications.read',
    manage: 'settings.notifications.manage',
  },
];

export function SystemSettingsPage() {
  const [tab, setTab] = useState<SettingsTab>('base');
  const user = getStoredUser();
  const visibleTabs = useMemo(
    () => tabs.filter(item => userHasPermission(user, item.read)),
    [user]
  );
  const active = visibleTabs.find(item => item.id === tab) ?? visibleTabs[0];

  if (!active) {
    return (
      <div className="grid h-full place-items-center rounded-xl border border-[var(--zl-border)] bg-[var(--zl-card)] text-sm text-[var(--zl-text-muted)]">
        当前用户无权查看系统配置
      </div>
    );
  }

  const ActiveIcon = active.icon;
  const canManage = userHasPermission(user, active.manage);
  return (
    <div data-cmp="SystemSettingsPage" className="flex h-full min-h-0 flex-col gap-5">
      <div className="shrink-0">
        <h1 className="text-lg font-semibold text-[var(--zl-text)]">系统配置</h1>
        <p className="mt-1 text-sm text-[var(--zl-text-muted)]">
          管理平台基础参数、用户权限、身份认证与找回密码邮件
        </p>
      </div>
      <section
        className="zl-card-hover flex shrink-0 flex-wrap gap-2 rounded-xl p-3"
        style={{
          background: 'var(--zl-card)',
          border: '1px solid var(--zl-border)',
          boxShadow: 'var(--shadow-card)',
        }}
      >
        {visibleTabs.map(item => {
          const Icon = item.icon;
          return (
            <button
              key={item.id}
              type="button"
              onClick={() => setTab(item.id)}
              className="zl-config-nav-card flex h-10 items-center gap-2 rounded-lg px-4 text-sm font-medium transition-all"
              data-active={active.id === item.id}
              style={{
                color: active.id === item.id ? 'var(--zl-accent-hover)' : 'var(--zl-text-muted)',
              }}
            >
              <Icon size={16} />
              {item.label}
            </button>
          );
        })}
      </section>

      <section
        className="zl-card-hover flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl p-5"
        style={{
          background: 'var(--zl-card)',
          border: '1px solid var(--zl-border)',
          boxShadow: 'var(--shadow-card)',
        }}
      >
        <div className="mb-5 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-sm font-semibold text-[var(--zl-text)]">{active.label}</h1>
            <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
              {sectionDescription(active.id)}
            </p>
          </div>
          <div className="flex items-center gap-2 rounded-lg border border-blue-400/25 bg-blue-500/10 px-3 py-2 text-xs text-[var(--zl-accent-text)]">
            <ActiveIcon size={14} />
            {sectionBadge(active.id)}
          </div>
        </div>
        {active.id === 'base' ? <BaseSettingsPanel canManage={canManage} /> : null}
        {active.id === 'users' ? <UserSettingsPanel canManage={canManage} /> : null}
        {active.id === 'auth' ? <AuthSettingsPanel canManage={canManage} /> : null}
        {active.id === 'notifications' ? <NotificationSettingsPanel canManage={canManage} /> : null}
      </section>
    </div>
  );
}

function sectionDescription(tab: SettingsTab) {
  if (tab === 'base') return '维护系统名称、证书续期、CRL 与 OCSP 服务参数。';
  if (tab === 'users')
    return '维护平台用户、用户群组和角色权限。启用 AD/LDAP 后，外部账号也必须先在用户中创建并启用。';
  if (tab === 'auth') return '启用外部认证后，登录界面会显示对应登录方式。';
  return '配置找回密码、审批与证书到期提醒的通知媒介。';
}

function sectionBadge(tab: SettingsTab) {
  if (tab === 'base') return 'PKI 基础服务';
  if (tab === 'users') return '用户与权限';
  if (tab === 'auth') return '身份认证';
  return '通知媒介';
}
