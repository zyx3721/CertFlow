import {
  CheckCircle2,
  MessageCircle,
  Network,
  Save,
  ToggleLeft,
  ToggleRight,
  Trash2,
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { invalidatePublicAuthConfiguration } from '@/lib/auth';
import {
  fetchAuthProvider,
  fetchWecomProvider,
  saveAuthProvider,
  testAuthProvider,
} from '@/lib/system-settings';
import { WeComSettingsPanel } from './WeComSettingsPanel';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  SettingsDetailHeader,
  SettingsDetailPanel,
  SettingsSplitLayout,
  type SettingsField,
} from './settings-primitives';
import { cardStyle } from './settings-style';

type LDAPForm = Record<string, unknown>;

type ProviderSummary = { name: string; enabled: boolean };

const defaultForm: LDAPForm = {
  host: '',
  port: '',
  baseDN: '',
  bindDN: '',
  bindPassword: '',
  useTLS: false,
  startTLS: false,
  insecureSkipVerify: false,
  userFilter: '',
  timeoutSeconds: '',
  groupFilter: '',
};

const fields: SettingsField[] = [
  { key: 'host', label: '服务器地址', placeholder: 'ldap.example.com', required: true },
  { key: 'port', label: '端口', placeholder: '389', required: true, inputMode: 'numeric' },
  { key: 'baseDN', label: 'Base DN', placeholder: 'DC=example,DC=com', required: true },
  {
    key: 'userFilter',
    label: '用户过滤器',
    placeholder: '(sAMAccountName={username})',
    required: true,
  },
  {
    key: 'bindDN',
    label: '绑定 DN',
    placeholder: 'CN=svc,OU=Users,DC=example,DC=com',
    required: true,
  },
  {
    key: 'bindPassword',
    label: '绑定密码',
    type: 'password',
    placeholder: '请输入绑定账号密码',
    required: true,
  },
  { key: 'useTLS', label: '启用 LDAPS', type: 'checkbox' },
  { key: 'startTLS', label: '启用 STARTTLS', type: 'checkbox' },
  {
    key: 'insecureSkipVerify',
    label: '跳过证书校验',
    type: 'checkbox',
  },
  { key: 'timeoutSeconds', label: '超时时间', placeholder: '8', type: 'number' },
  { key: 'groupFilter', label: '用户组过滤器', placeholder: 'cn=ops,dc=example,dc=com' },
];

type AuthProviderKind = 'ldap' | 'wecom';

export function AuthSettingsPanel({ canManage }: { canManage: boolean }) {
  const [active, setActive] = useState<AuthProviderKind>('ldap');
  const [ldapSummary, setLdapSummary] = useState<ProviderSummary>({
    name: 'AD/LDAP',
    enabled: false,
  });
  const [wecomSummary, setWecomSummary] = useState<ProviderSummary>({
    name: '企业微信',
    enabled: false,
  });

  const refreshSummaries = useCallback(async () => {
    try {
      const ldap = await fetchAuthProvider();
      setLdapSummary({ name: ldap.name || 'AD/LDAP', enabled: ldap.enabled });
    } catch (err) {
      console.error('读取 LDAP 认证配置状态失败', err);
    }
    try {
      const wecom = await fetchWecomProvider();
      setWecomSummary({ name: wecom.name || '企业微信', enabled: wecom.enabled });
    } catch (err) {
      console.error('读取企业微信认证配置状态失败', err);
    }
  }, []);

  useEffect(() => {
    void refreshSummaries();
  }, [refreshSummaries]);

  return (
    <SettingsSplitLayout
      sidebarLabel="认证配置"
      sidebar={
        <>
          <ProviderCard
            icon={<Network size={19} />}
            color="text-sky-400"
            title={ldapSummary.name}
            description="通过企业目录服务实现统一身份认证，支持 AD/LDAP 登录"
            enabled={ldapSummary.enabled}
            active={active === 'ldap'}
            onClick={() => setActive('ldap')}
          />
          <ProviderCard
            icon={<MessageCircle size={19} />}
            color="text-cyan-400"
            title={wecomSummary.name}
            description="企业微信扫码登录，支持在右上角绑定企微账号"
            enabled={wecomSummary.enabled}
            active={active === 'wecom'}
            onClick={() => setActive('wecom')}
          />
        </>
      }
    >
      {active === 'ldap' ? (
        <LDAPSettingsPanel canManage={canManage} onSaved={() => void refreshSummaries()} />
      ) : (
        <WeComSettingsPanel canManage={canManage} onSaved={() => void refreshSummaries()} />
      )}
    </SettingsSplitLayout>
  );
}

function ProviderCard({
  icon,
  color,
  title,
  description,
  enabled,
  active,
  onClick,
}: {
  icon: React.ReactNode;
  color: string;
  title: string;
  description: string;
  enabled: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className="zl-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
      data-active={active ? 'true' : 'false'}
      style={cardStyle(active)}
      onClick={onClick}
    >
      <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05]">
        <span className={color}>{icon}</span>
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center justify-between gap-3">
          <span className="truncate text-sm font-semibold">{title}</span>
          <UsageIndicator enabled={enabled} />
        </span>
        <span className="mt-1 block text-xs leading-5 text-[var(--zl-text-muted)]">
          {description}
        </span>
      </span>
    </button>
  );
}

function LDAPSettingsPanel({ canManage, onSaved }: { canManage: boolean; onSaved?: () => void }) {
  const [name, setName] = useState('AD/LDAP');
  const [enabled, setEnabled] = useState(false);
  const [savedEnabled, setSavedEnabled] = useState(false);
  const [form, setForm] = useState<LDAPForm>(defaultForm);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [clearRequested, setClearRequested] = useState(false);

  const load = useCallback(async () => {
    setError('');
    try {
      const setting = await fetchAuthProvider();
      setName(setting.name || 'AD/LDAP');
      setEnabled(setting.enabled);
      setSavedEnabled(setting.enabled);
      setForm({ ...defaultForm, ...setting.config });
    } catch (err) {
      const message = err instanceof Error ? err.message : '读取认证配置失败';
      setError(message);
      toast.error(message);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function updateField(field: SettingsField, value: unknown) {
    setClearRequested(false);
    setForm(current => {
      const next = { ...current, [field.key]: value };
      if (field.key === 'useTLS' && value === true) {
        next.startTLS = false;
        next.port = 636;
      }
      if (field.key === 'startTLS' && value === true) {
        next.useTLS = false;
        next.port = 389;
      }
      return next;
    });
  }

  async function save() {
    const result = prepareConfig(form, enabled);
    if (result.error) {
      toast.error(result.error);
      return;
    }
    setBusy('save');
    try {
      const saved = await saveAuthProvider({
        name: name.trim() || 'AD/LDAP',
        enabled,
        clearConfig: clearRequested,
        config: result.config,
      });
      setName(saved.name);
      setEnabled(saved.enabled);
      setSavedEnabled(saved.enabled);
      setForm({ ...defaultForm, ...saved.config });
      setClearRequested(false);
      invalidatePublicAuthConfiguration();
      onSaved?.();
      toast.success('认证配置已保存');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '认证配置保存失败');
    } finally {
      setBusy('');
    }
  }

  function clearConfig() {
    setName('AD/LDAP');
    setEnabled(false);
    setForm(defaultForm);
    setClearRequested(true);
  }

  async function test() {
    setBusy('test');
    try {
      const result = await testAuthProvider();
      toast.success(`认证连接测试通过，成功匹配 ${result.matchedUsers} 个用户`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'LDAP 连接测试失败');
    } finally {
      setBusy('');
    }
  }

  return (
    <SettingsDetailPanel
      header={
        <SettingsDetailHeader
          icon={Network}
          color="#60a5fa"
          title={name}
          subtitle={enabled ? '已启用' : '未启用'}
          active={enabled}
        />
      }
      actions={
        canManage ? (
          <>
            <ActionButton
              icon={<Trash2 size={14} />}
              label="清空配置"
              tone="danger"
              disabled={busy !== ''}
              onClick={clearConfig}
            />
            <ActionButton
              icon={<Save size={14} />}
              label="保存"
              busy={busy === 'save'}
              disabled={busy !== '' && busy !== 'save'}
              onClick={save}
            />
            <ActionButton
              icon={<CheckCircle2 size={14} />}
              label="测试"
              tone="success"
              busy={busy === 'test'}
              disabled={!enabled || !savedEnabled || (busy !== '' && busy !== 'test')}
              onClick={test}
            />
          </>
        ) : null
      }
    >
      {error ? (
        <p className="mb-4 rounded-lg border border-amber-400/25 bg-amber-500/10 p-3 text-sm text-amber-400">
          {error}
        </p>
      ) : null}
      <p className="mb-4 text-sm leading-6 text-[var(--zl-text-muted)]">
        通过企业目录服务实现统一身份认证，支持 AD/LDAP 登录。
      </p>
      <div className="space-y-3">
        <EnableToggle
          enabled={enabled}
          disabled={!canManage}
          onChange={value => {
            setEnabled(value);
            setClearRequested(false);
          }}
          label="启用认证"
          enabledText="登录页将显示该认证方式"
          disabledText="关闭后不会显示在登录页"
        />
        <ConfigField
          field={{ key: 'name', label: '显示名称' }}
          value={name}
          disabled={true}
          onChange={value => {
            setName(String(value ?? ''));
            setClearRequested(false);
          }}
        />
        <div className="space-y-3">
          <SectionTitle title="必填配置" />
          {fields.slice(0, 6).map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={field.key === 'bindPassword' && Boolean(form.hasBindPassword)}
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
        <div className="space-y-3">
          <SectionTitle title="可选配置" />
          {fields.slice(6).map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={field.key === 'bindPassword' && Boolean(form.hasBindPassword)}
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
      </div>
    </SettingsDetailPanel>
  );
}

function SectionTitle({ title }: { title: string }) {
  return <div className="text-xs font-semibold leading-5 text-[var(--zl-text)]">{title}</div>;
}

function UsageIndicator({ enabled }: { enabled: boolean }) {
  return enabled ? (
    <ToggleRight size={18} aria-hidden="true" className="shrink-0 text-emerald-400" />
  ) : (
    <ToggleLeft size={18} aria-hidden="true" className="shrink-0 text-[var(--zl-text-muted)]" />
  );
}

function prepareConfig(form: LDAPForm, enabled: boolean) {
  const config = Object.fromEntries(
    Object.entries(form).filter(([key, value]) => key !== 'hasBindPassword' && value !== '')
  );
  const port = Number(config.port || 0);
  if (Boolean(config.useTLS) && Boolean(config.startTLS))
    return { config: {}, error: 'LDAPS 与 StartTLS 不能同时启用' };
  if (enabled) {
    const required = [
      ['host', '服务器地址'],
      ['port', '端口'],
      ['baseDN', 'Base DN'],
      ['userFilter', '用户过滤器'],
      ['bindDN', '绑定 DN'],
    ] as const;
    const missing = required.find(([key]) => !String(config[key] || '').trim());
    if (missing) return { config: {}, error: `${missing[1]}不能为空` };
    if (!config.bindPassword && !form.hasBindPassword)
      return { config: {}, error: '绑定密码不能为空' };
    if (!Number.isInteger(port) || port < 1 || port > 65535)
      return { config: {}, error: 'LDAP 端口需为 1 到 65535 之间的整数' };
    config.port = port;
  }
  return { config, error: '' };
}
