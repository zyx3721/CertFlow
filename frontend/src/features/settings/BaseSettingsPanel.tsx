import {
  Ban,
  Globe2,
  Image,
  KeyRound,
  RotateCcw,
  Save,
  Settings2,
  UploadCloud,
} from 'lucide-react';
import { useEffect, useRef, useState, type ChangeEvent, type DragEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { setBrandSettings } from '@/lib/branding';
import { getPkiSettings, savePkiSettings, type PkiSettings } from '@/lib/pki';
import { pkiQueryKeys, usePageRefresh } from '@/features/pki/support';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  NumberControl,
  SettingsDetailHeader,
  SettingsDetailPanel,
  SettingsSplitLayout,
} from './settings-primitives';
import { cardStyle } from './settings-style';

type BaseSection = 'brand' | 'security' | 'basic' | 'crl' | 'ocsp';

const defaults: PkiSettings = {
  siteName: 'CertFlow',
  loginName: 'CertFlow',
  appName: 'CertFlow',
  appSubtitle: 'PKI Control Plane',
  iconData: '/favicon.svg',
  crlEnabled: true,
  crlUrl: '/crl',
  ocspEnabled: true,
  ocspUrl: '/ocsp',
  crlIntervalHours: 24,
  renewDays: 10,
  autoRenew: false,
  expiryNotificationDays: 15,
  expiryNotificationCron: '0 0 * * *',
  resetCodeTtlMinutes: 10,
  resetCaptchaTtlMinutes: 1,
  passwordResetSendCooldownMinutes: 0.5,
  passwordResetRateLimitMinutes: 5,
  wecomStateTtlMinutes: 5,
  loginMaxFailures: 5,
  loginLockoutMinutes: 2,
};

const cards = [
  {
    id: 'brand' as const,
    title: '品牌标识',
    description: '网站名称、登录展示和图标',
    icon: Image,
    color: '#38bdf8',
  },
  {
    id: 'security' as const,
    title: '安全时效',
    description: '验证码时效、发送限流、扫码有效期与登录失败锁定',
    icon: KeyRound,
    color: '#22c55e',
  },
  {
    id: 'basic' as const,
    title: '续期与到期提醒',
    description: '续期处理窗口、到期提醒与扫描计划',
    icon: Settings2,
    color: '#38bdf8',
  },
  {
    id: 'crl' as const,
    title: 'CRL 服务',
    description: '签发证书时写入 CRL 分发点，并开放 CRL 下载',
    icon: Ban,
    color: '#f59e0b',
  },
  {
    id: 'ocsp' as const,
    title: 'OCSP 服务',
    description: '在线证书状态协议实时查询',
    icon: Globe2,
    color: '#22c55e',
  },
];

const settingKeys = [pkiQueryKeys.settings] as const;

export function BaseSettingsPanel({ canManage }: { canManage: boolean }) {
  usePageRefresh(settingKeys);
  const [active, setActive] = useState<BaseSection>('brand');
  const [form, setForm] = useState(defaults);
  const [saved, setSaved] = useState(defaults);
  const [draggingIcon, setDraggingIcon] = useState(false);
  const iconInputRef = useRef<HTMLInputElement | null>(null);
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: pkiQueryKeys.settings, queryFn: getPkiSettings });
  useEffect(() => {
    if (!query.data) return;
    const next = { ...defaults, ...query.data };
    setForm(next);
    setSaved(next);
  }, [query.data]);
  const mutation = useMutation({
    mutationFn: savePkiSettings,
    onSuccess: saved => {
      setForm(saved);
      setSaved(saved);
      setBrandSettings(saved);
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.settings });
      toast.success('基础配置已保存');
    },
    onError: error => toast.error(error instanceof Error ? error.message : '基础配置保存失败'),
  });
  const card = cards.find(item => item.id === active) ?? cards[0];

  function save() {
    if (
      !form.siteName.trim() ||
      !form.loginName.trim() ||
      !form.appName.trim() ||
      form.renewDays < 1 ||
      form.expiryNotificationDays < 1 ||
      !form.expiryNotificationCron.trim() ||
      form.crlIntervalHours < 1
    ) {
      toast.error('请检查基础配置、数值与扫描计划');
      return;
    }
    mutation.mutate(form);
  }

  function updateIcon(file?: File) {
    if (!file) return;
    if (!file.type.startsWith('image/') || file.size > 256 * 1024) {
      toast.error('请选择 256KB 以内的图片文件');
      return;
    }
    const reader = new FileReader();
    reader.onload = () => setForm(value => ({ ...value, iconData: String(reader.result || '') }));
    reader.onerror = () => toast.error('品牌图标读取失败');
    reader.readAsDataURL(file);
  }

  function changeIcon(event: ChangeEvent<HTMLInputElement>) {
    updateIcon(event.target.files?.[0]);
    event.target.value = '';
  }

  function dropIcon(event: DragEvent<HTMLButtonElement>) {
    event.preventDefault();
    setDraggingIcon(false);
    updateIcon(event.dataTransfer.files?.[0]);
  }

  function selectSection(section: BaseSection) {
    setActive(section);
    setForm(saved);
  }

  return (
    <SettingsSplitLayout
      sidebarLabel="基础配置"
      sidebar={cards.map(item => {
        const Icon = item.icon;
        return (
          <button
            key={item.id}
            type="button"
            onClick={() => selectSection(item.id)}
            className="zl-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
            data-active={active === item.id}
            style={cardStyle(active === item.id)}
          >
            <span
              className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05]"
              style={{ color: item.color }}
            >
              <Icon size={19} />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-semibold">{item.title}</span>
              <span className="mt-1 block line-clamp-2 text-xs leading-5 text-[var(--zl-text-muted)]">
                {item.description}
              </span>
            </span>
          </button>
        );
      })}
    >
      <SettingsDetailPanel
        header={
          <SettingsDetailHeader
            icon={card.icon}
            color={card.color}
            title={card.title}
            subtitle={`${card.description}。`}
          />
        }
        actions={
          canManage ? (
            <>
              <ActionButton
                icon={<RotateCcw size={14} />}
                label="恢复默认"
                tone="muted"
                disabled={mutation.isPending}
                onClick={() => setForm(value => ({ ...value, ...sectionDefaults(active) }))}
              />
              <ActionButton
                icon={<Save size={14} />}
                label="保存"
                busy={mutation.isPending}
                onClick={save}
              />
            </>
          ) : null
        }
      >
        {query.isError ? (
          <button
            type="button"
            className="text-sm text-[var(--zl-accent-text)]"
            onClick={() => void query.refetch()}
          >
            读取配置失败，点击重试
          </button>
        ) : null}
        {active === 'brand' ? (
          <div className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <ConfigField
                field={{ key: 'siteName', label: '网站名称', placeholder: 'CertFlow' }}
                value={form.siteName}
                disabled={!canManage}
                onChange={value => setForm(current => ({ ...current, siteName: String(value) }))}
              />
              <ConfigField
                field={{ key: 'loginName', label: '认证页品牌名称', placeholder: 'CertFlow' }}
                value={form.loginName}
                disabled={!canManage}
                onChange={value => setForm(current => ({ ...current, loginName: String(value) }))}
              />
              <ConfigField
                field={{ key: 'appName', label: '控制台品牌名称', placeholder: 'CertFlow' }}
                value={form.appName}
                disabled={!canManage}
                onChange={value => setForm(current => ({ ...current, appName: String(value) }))}
              />
              <ConfigField
                field={{
                  key: 'appSubtitle',
                  label: '控制台品牌副标题',
                  placeholder: 'PKI Control Plane',
                }}
                value={form.appSubtitle}
                disabled={!canManage}
                onChange={value => setForm(current => ({ ...current, appSubtitle: String(value) }))}
              />
            </div>
            <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
              <button
                type="button"
                disabled={!canManage}
                onClick={() => iconInputRef.current?.click()}
                onDragOver={event => {
                  event.preventDefault();
                  setDraggingIcon(true);
                }}
                onDragLeave={() => setDraggingIcon(false)}
                onDrop={dropIcon}
                className="zl-action-button flex min-h-44 flex-col items-center justify-center gap-3 rounded-xl border border-dashed p-4 text-[var(--zl-text-muted)] disabled:opacity-60"
                style={{
                  borderColor: draggingIcon ? 'rgba(96,165,250,0.78)' : 'var(--zl-border)',
                  background: draggingIcon ? 'rgba(59,130,246,0.12)' : 'rgba(255,255,255,0.026)',
                }}
              >
                <img
                  src={form.iconData || '/favicon.svg'}
                  alt="品牌图标预览"
                  className="h-16 w-16 rounded-xl object-contain"
                />
                <span className="flex items-center gap-2 text-sm">
                  <UploadCloud size={15} />
                  拖动或点击上传
                </span>
                <span className="text-xs">支持图片，建议 256KB 内</span>
              </button>
              <div className="flex min-h-44 flex-col rounded-xl border border-[var(--zl-border)] bg-white/[0.026] p-5">
                <div className="text-sm font-semibold text-[var(--zl-text-muted)]">实时预览</div>
                <div className="flex flex-1 items-center">
                  <div className="flex min-w-0 items-center gap-4">
                    <img
                      src={form.iconData || '/favicon.svg'}
                      alt=""
                      className="h-16 w-16 shrink-0 rounded-xl object-contain"
                    />
                    <div className="min-w-0">
                      <div className="zl-gradient-text truncate text-xl font-bold">
                        {form.appName || defaults.appName}
                      </div>
                      <div className="mt-1 truncate text-sm tracking-widest text-[var(--zl-text-muted)]">
                        {form.appSubtitle || defaults.appSubtitle}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <input
              ref={iconInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={changeIcon}
            />
          </div>
        ) : null}
        {active === 'security' ? (
          <div className="grid gap-4 lg:grid-cols-2">
            <NumberControl
              label="找回密码验证码有效期"
              description={`邮件验证码超过 ${form.resetCodeTtlMinutes} 分钟后不可使用`}
              unit="分钟"
              value={form.resetCodeTtlMinutes}
              min={1}
              max={60}
              disabled={!canManage}
              onChange={resetCodeTtlMinutes =>
                setForm(value => ({ ...value, resetCodeTtlMinutes }))
              }
            />
            <NumberControl
              label="图形验证码有效期"
              description={`图形验证码超过 ${form.resetCaptchaTtlMinutes} 分钟后不可使用`}
              unit="分钟"
              value={form.resetCaptchaTtlMinutes}
              min={1}
              max={10}
              disabled={!canManage}
              onChange={resetCaptchaTtlMinutes =>
                setForm(value => ({ ...value, resetCaptchaTtlMinutes }))
              }
            />
            <NumberControl
              label="发送冷却时间"
              description={`验证码发送后 ${form.passwordResetSendCooldownMinutes} 分钟内不可重复请求`}
              unit="分钟"
              value={form.passwordResetSendCooldownMinutes}
              min={0.5}
              max={10}
              step={0.5}
              disabled={!canManage}
              onChange={passwordResetSendCooldownMinutes =>
                setForm(value => ({ ...value, passwordResetSendCooldownMinutes }))
              }
            />
            <NumberControl
              label="频率限制统计窗口"
              description={`验证码在 ${form.passwordResetRateLimitMinutes} 分钟内最多请求 5 次`}
              unit="分钟"
              value={form.passwordResetRateLimitMinutes}
              min={5}
              max={10}
              disabled={!canManage}
              onChange={passwordResetRateLimitMinutes =>
                setForm(value => ({ ...value, passwordResetRateLimitMinutes }))
              }
            />
            <NumberControl
              label="企业微信扫码有效期"
              description="企微授权 state 的有效窗口，超时需重新扫码登录或绑定"
              unit="分钟"
              value={form.wecomStateTtlMinutes}
              min={1}
              max={60}
              disabled={!canManage}
              onChange={wecomStateTtlMinutes =>
                setForm(value => ({ ...value, wecomStateTtlMinutes }))
              }
            />
            <NumberControl
              label="登录失败锁定次数"
              description={`同一账号连续 ${form.loginMaxFailures} 次密码错误后锁定，admin 不受限`}
              unit="次"
              value={form.loginMaxFailures}
              min={3}
              max={10}
              disabled={!canManage}
              onChange={loginMaxFailures => setForm(value => ({ ...value, loginMaxFailures }))}
            />
            <NumberControl
              label="登录锁定等待时长"
              description={`达到锁定次数后需等待 ${form.loginLockoutMinutes} 分钟才能重试`}
              unit="分钟"
              value={form.loginLockoutMinutes}
              min={1}
              max={10}
              disabled={!canManage}
              onChange={loginLockoutMinutes =>
                setForm(value => ({ ...value, loginLockoutMinutes }))
              }
            />
          </div>
        ) : null}
        {active === 'basic' ? (
          <div className="space-y-5">
            <EnableToggle
              enabled={form.autoRenew}
              disabled={!canManage}
              onChange={autoRenew => setForm(current => ({ ...current, autoRenew }))}
              label="启用自动续期"
              enabledText="扫描命中后自动签发续期证书，不需要人工审批"
              disabledText="仅发送到期提醒，由人员手动提交续期申请"
            />
            <div className="grid gap-4 lg:grid-cols-2">
              <NumberControl
                label="提前续期天数"
                description={`启用自动续期后，证书在剩余 ${form.renewDays} 天内会自动续期`}
                unit="天"
                value={form.renewDays}
                min={1}
                max={365}
                disabled={!canManage}
                onChange={renewDays => setForm(current => ({ ...current, renewDays }))}
              />
              <NumberControl
                label="到期提醒阈值"
                description={`证书剩余 ${form.expiryNotificationDays} 天及以内时发送外部提醒`}
                unit="天"
                value={form.expiryNotificationDays}
                min={1}
                max={365}
                disabled={!canManage}
                onChange={expiryNotificationDays =>
                  setForm(current => ({ ...current, expiryNotificationDays }))
                }
              />
            </div>
            <ConfigField
              field={{
                key: 'expiryNotificationCron',
                label: '定时扫描计划',
                placeholder: '0 0 * * *',
                helper: '使用五段 Cron：分 时 日 月 周，例如 0 0 * * * 表示每天 0 点扫描',
              }}
              value={form.expiryNotificationCron}
              disabled={!canManage}
              onChange={value =>
                setForm(current => ({ ...current, expiryNotificationCron: String(value) }))
              }
            />
          </div>
        ) : null}
        {active === 'crl' ? (
          <div className="space-y-4">
            <EnableToggle
              enabled={form.crlEnabled}
              disabled={!canManage}
              onChange={crlEnabled => setForm(current => ({ ...current, crlEnabled }))}
              label="启用 CRL 服务"
              enabledText="允许客户端下载 CA 对应的 X.509 CRL 文件"
              disabledText="不会提供证书撤销列表文件"
            />
            <ConfigField
              field={{
                key: 'crlUrl',
                label: 'CRL 分发基础地址',
                helper: '路径固定为 /crl，由 Nginx 代理；实际文件按 /crl/{ca-id}.crl 提供',
              }}
              value={form.crlUrl}
              disabled
              onChange={value => setForm(current => ({ ...current, crlUrl: String(value) }))}
            />
            <ConfigField
              field={{ key: 'crlIntervalHours', label: '更新间隔（小时）', type: 'number' }}
              value={form.crlIntervalHours}
              disabled={!canManage}
              onChange={value =>
                setForm(current => ({ ...current, crlIntervalHours: Number(value) }))
              }
            />
          </div>
        ) : null}
        {active === 'ocsp' ? (
          <div className="space-y-4">
            <EnableToggle
              enabled={form.ocspEnabled}
              disabled={!canManage}
              onChange={ocspEnabled => setForm(current => ({ ...current, ocspEnabled }))}
              label="启用 OCSP 服务"
              enabledText="接受标准请求，并提供状态查询与签名响应"
              disabledText="不会对外提供在线证书状态响应"
            />
            <ConfigField
              field={{
                key: 'ocspUrl',
                label: 'OCSP 响应器地址',
                helper: '路径固定为 /ocsp，由 Nginx 代理；可通过 /ocsp/health 查看服务状态',
              }}
              value={form.ocspUrl}
              disabled
              onChange={value => setForm(current => ({ ...current, ocspUrl: String(value) }))}
            />
          </div>
        ) : null}
      </SettingsDetailPanel>
    </SettingsSplitLayout>
  );
}

function sectionDefaults(section: BaseSection): Partial<PkiSettings> {
  if (section === 'brand')
    return {
      siteName: defaults.siteName,
      loginName: defaults.loginName,
      appName: defaults.appName,
      appSubtitle: defaults.appSubtitle,
      iconData: defaults.iconData,
    };
  if (section === 'security')
    return {
      resetCodeTtlMinutes: defaults.resetCodeTtlMinutes,
      resetCaptchaTtlMinutes: defaults.resetCaptchaTtlMinutes,
      passwordResetSendCooldownMinutes: defaults.passwordResetSendCooldownMinutes,
      passwordResetRateLimitMinutes: defaults.passwordResetRateLimitMinutes,
      wecomStateTtlMinutes: defaults.wecomStateTtlMinutes,
    };
  if (section === 'basic')
    return {
      renewDays: defaults.renewDays,
      autoRenew: defaults.autoRenew,
      expiryNotificationDays: defaults.expiryNotificationDays,
      expiryNotificationCron: defaults.expiryNotificationCron,
    };
  if (section === 'crl')
    return {
      crlEnabled: defaults.crlEnabled,
      crlUrl: defaults.crlUrl,
      crlIntervalHours: defaults.crlIntervalHours,
    };
  return { ocspEnabled: defaults.ocspEnabled, ocspUrl: defaults.ocspUrl };
}
