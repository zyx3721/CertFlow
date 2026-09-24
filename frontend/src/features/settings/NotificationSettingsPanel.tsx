import {
  BellRing,
  CheckCircle2,
  Mail,
  MessageCircleMore,
  RadioTower,
  Save,
  Send,
  Smartphone,
  Trash2,
  Webhook,
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { usePageRefresh } from '@/features/pki/support';
import {
  fetchNotificationChannels,
  saveNotificationChannel,
  testNotificationChannel,
} from '@/lib/system-settings';
import { useBrandSettings } from '@/lib/branding';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import type { NotificationChannel } from './types';
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

type ChannelID =
  | 'webhook'
  | 'email'
  | 'lark'
  | 'lark_app'
  | 'wechat'
  | 'wechat_app'
  | 'dingtalk'
  | 'dingtalk_app';
type Meta = {
  name: string;
  description: string;
  icon: typeof Mail;
  color: string;
  fields: SettingsField[];
};

const metas: Record<ChannelID, Meta> = {
  webhook: {
    name: 'Webhook',
    description: '向指定 HTTP 地址推送审批与证书到期提醒',
    icon: Webhook,
    color: '#06b6d4',
    fields: [
      {
        key: 'url',
        label: 'Webhook URL',
        placeholder: 'https://example.com/approval',
        required: true,
      },
      { key: 'method', label: '请求方法', placeholder: 'POST', helper: '支持 POST、PUT、PATCH' },
      {
        key: 'headers',
        label: '请求头 JSON',
        placeholder: '{"Authorization":"Bearer token"}',
        helper: '可选，必须是 JSON 对象',
        type: 'textarea',
      },
    ],
  },
  email: {
    name: '邮件',
    description: '向平台用户发送找回密码、审批与证书到期提醒',
    icon: Mail,
    color: '#10b981',
    fields: [
      { key: 'smtpHost', label: 'SMTP 主机', placeholder: 'smtp.example.com', required: true },
      {
        key: 'smtpPort',
        label: 'SMTP 端口',
        placeholder: '465',
        required: true,
        inputMode: 'numeric',
      },
      { key: 'username', label: '用户名', placeholder: 'certflow@example.com', required: true },
      {
        key: 'password',
        label: '密码',
        placeholder: 'SMTP 授权码',
        type: 'password',
        required: true,
      },
      { key: 'from', label: '发件人', placeholder: 'certflow@example.com', required: true },
      { key: 'fromName', label: '发件人名称', placeholder: 'CertFlow' },
      { key: 'useTLS', label: '启用 TLS/SSL', type: 'checkbox' },
      { key: 'startTLS', label: '启用 STARTTLS', type: 'checkbox' },
      {
        key: 'allowInsecureAuth',
        label: '允许明文认证',
        type: 'checkbox',
        helper: '仅在 SMTP 服务明确要求时启用',
      },
    ],
  },
  lark: {
    name: '飞书机器人',
    description: '向飞书群推送审批与证书到期提醒',
    icon: Send,
    color: '#8b5cf6',
    fields: [
      {
        key: 'webhookUrl',
        label: '机器人 Webhook',
        placeholder: 'https://open.feishu.cn/open-apis/bot/v2/hook/...',
        required: true,
      },
      { key: 'secret', label: '签名密钥', placeholder: '可选', type: 'password' },
    ],
  },
  lark_app: {
    name: '飞书应用',
    description: '通过飞书应用向指定用户或群聊推送审批与证书到期提醒',
    icon: MessageCircleMore,
    color: '#6366f1',
    fields: [
      { key: 'appId', label: 'App ID', placeholder: 'cli_xxx', required: true },
      {
        key: 'appSecret',
        label: 'App Secret',
        placeholder: '飞书应用密钥',
        type: 'password',
        required: true,
      },
      {
        key: 'receiveIdType',
        label: '接收 ID 类型',
        placeholder: '选择接收 ID 类型',
        required: true,
        type: 'select',
        helper: '支持 open_id、user_id、union_id、email、chat_id',
        options: [
          { value: 'chat_id', label: '群聊 ID' },
          { value: 'open_id', label: 'Open ID' },
          { value: 'user_id', label: 'User ID' },
          { value: 'union_id', label: 'Union ID' },
          { value: 'email', label: '邮箱' },
        ],
      },
      { key: 'receiveId', label: '接收 ID', placeholder: 'oc_xxx 或用户 ID', required: true },
    ],
  },
  wechat: {
    name: '企业微信机器人',
    description: '向企业微信群推送审批与证书到期提醒',
    icon: MessageCircleMore,
    color: '#22c55e',
    fields: [
      {
        key: 'webhookUrl',
        label: '机器人 Webhook',
        placeholder: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...',
        required: true,
      },
    ],
  },
  wechat_app: {
    name: '企业微信应用',
    description: '通过企业微信应用向成员、部门或标签推送审批与证书到期提醒',
    icon: Smartphone,
    color: '#16a34a',
    fields: [
      { key: 'corpId', label: '企业 ID', placeholder: 'ww_xxx', required: true },
      { key: 'agentId', label: 'AgentId', placeholder: '1000002', required: true },
      {
        key: 'secret',
        label: '应用 Secret',
        placeholder: '企业微信应用密钥',
        type: 'password',
        required: true,
      },
      { key: 'toUser', label: '接收成员', placeholder: 'zhangsan|lisi' },
      { key: 'toParty', label: '接收部门', placeholder: '1|2' },
      { key: 'toTag', label: '接收标签', placeholder: '1|2' },
    ],
  },
  dingtalk: {
    name: '钉钉机器人',
    description: '向钉钉群推送审批与证书到期提醒',
    icon: RadioTower,
    color: '#f59e0b',
    fields: [
      {
        key: 'webhookUrl',
        label: '机器人 Webhook',
        placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=...',
        required: true,
      },
      {
        key: 'secret',
        label: '加签密钥',
        placeholder: '钉钉机器人加签密钥',
        type: 'password',
        required: true,
      },
    ],
  },
  dingtalk_app: {
    name: '钉钉应用',
    description: '通过钉钉应用向指定用户推送审批与证书到期提醒',
    icon: RadioTower,
    color: '#f97316',
    fields: [
      { key: 'appKey', label: 'AppKey', placeholder: 'dingxxx', required: true },
      {
        key: 'appSecret',
        label: 'AppSecret',
        placeholder: '钉钉应用密钥',
        type: 'password',
        required: true,
      },
      { key: 'agentId', label: 'AgentId', placeholder: '1000002', required: true },
      { key: 'useridList', label: '用户列表', placeholder: 'user001,user002' },
      { key: 'deptIdList', label: '部门列表', placeholder: '1,2' },
    ],
  },
};
const order: ChannelID[] = [
  'webhook',
  'email',
  'lark',
  'lark_app',
  'wechat',
  'wechat_app',
  'dingtalk',
  'dingtalk_app',
];

export function NotificationSettingsPanel({ canManage }: { canManage: boolean }) {
  const brand = useBrandSettings();
  const [channels, setChannels] = useState<Record<string, NotificationChannel>>({});
  const [selected, setSelected] = useState<ChannelID>('webhook');
  const [form, setForm] = useState<Record<string, unknown>>({});
  const [approval, setApproval] = useState(false);
  const [passwordReset, setPasswordReset] = useState(false);
  const [clearConfigPending, setClearConfigPending] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [busy, setBusy] = useState('');
  const meta = metas[selected];
  const Icon = meta.icon;
  const isEmail = selected === 'email';
  const savedApprovalEnabled = channels[selected]?.approvalEnabled ?? false;
  const savedPasswordResetEnabled = channels[selected]?.passwordResetEnabled ?? false;
  const load = useCallback(async () => {
    try {
      const response = await fetchNotificationChannels();
      setChannels(Object.fromEntries(response.items.map(item => [item.id, item])));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '读取通知配置失败');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  usePageRefresh(load);
  useEffect(() => {
    const item = channels[selected];
    const config = notificationFormConfig(selected, item?.config ?? {});
    const defaults =
      isEmail && !String(config.fromName ?? '').trim()
        ? { ...config, fromName: brand.siteName }
        : config;
    setForm(
      selected === 'lark_app' && !String(defaults.receiveIdType ?? '').trim()
        ? { ...defaults, receiveIdType: 'user_id' }
        : defaults
    );
    setApproval(item?.approvalEnabled ?? false);
    setPasswordReset(isEmail && (item?.passwordResetEnabled ?? false));
    setClearConfigPending(false);
  }, [brand.siteName, channels, isEmail, selected]);
  const subtitle = useMemo(
    () =>
      passwordReset && approval
        ? '找回密码 / 通知审批已启用'
        : passwordReset
          ? '找回密码已启用'
          : approval
            ? '通知审批已启用'
            : '未启用',
    [approval, passwordReset]
  );
  async function save() {
    setBusy('save');
    try {
      const saved = await saveNotificationChannel(selected, {
        passwordResetEnabled: passwordReset,
        approvalEnabled: approval,
        clearConfig: clearConfigPending,
        config: form,
      });
      setChannels(current => ({ ...current, [selected]: saved }));
      setClearConfigPending(false);
      toast.success('通知配置已保存');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存通知配置失败');
    } finally {
      setBusy('');
    }
  }
  async function test(to = '') {
    if (isEmail && !to.trim()) {
      toast.error('请输入测试邮箱');
      return;
    }
    setBusy('test');
    try {
      await testNotificationChannel(selected, to.trim());
      setTestOpen(false);
      toast.success('测试通知已发送');
    } catch (error) {
      toast.error(isEmail && error instanceof Error ? error.message : '测试通知发送失败');
    } finally {
      setBusy('');
    }
  }
  return (
    <SettingsSplitLayout
      sidebarLabel="通知媒介"
      sidebar={order.map(id => {
        const item = channels[id];
        const itemMeta = metas[id];
        const CardIcon = itemMeta.icon;
        return (
          <button
            key={id}
            type="button"
            onClick={() => setSelected(id)}
            className="zl-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
            style={cardStyle(selected === id)}
          >
            <span
              className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05]"
              style={{ color: itemMeta.color }}
            >
              <CardIcon size={19} />
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-3">
                <span className="text-sm font-semibold">{itemMeta.name}</span>
                <span className="flex gap-1">
                  <Usage enabled={item?.approvalEnabled ?? false} label="审" />
                  {id === 'email' ? (
                    <Usage enabled={item?.passwordResetEnabled ?? false} label="密" />
                  ) : null}
                </span>
              </span>
              <span className="mt-1 block text-xs leading-5 text-[var(--zl-text-muted)]">
                {itemMeta.description}
              </span>
            </span>
          </button>
        );
      })}
    >
      <SettingsDetailPanel
        header={
          <SettingsDetailHeader
            icon={Icon}
            color={meta.color}
            title={meta.name}
            subtitle={subtitle}
            active={approval || passwordReset}
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
                onClick={() => {
                  setForm(
                    isEmail
                      ? { fromName: brand.siteName }
                      : selected === 'lark_app'
                        ? { receiveIdType: 'user_id' }
                        : {}
                  );
                  setApproval(false);
                  setPasswordReset(false);
                  setClearConfigPending(true);
                }}
              />
              <ActionButton
                icon={<Save size={14} />}
                label="保存"
                busy={busy === 'save'}
                disabled={busy !== '' && busy !== 'save'}
                onClick={() => void save()}
              />
              <ActionButton
                icon={<CheckCircle2 size={14} />}
                label="测试"
                tone="success"
                busy={busy === 'test'}
                disabled={!savedApprovalEnabled && !(isEmail && savedPasswordResetEnabled)}
                onClick={() => {
                  if (isEmail) {
                    setTestEmail('');
                    setTestOpen(true);
                    return;
                  }
                  void test();
                }}
              />
            </>
          ) : null
        }
      >
        <p className="mb-5 text-sm leading-6 text-[var(--zl-text-muted)]">{meta.description}。</p>
        <div className={`mb-4 grid gap-3 ${isEmail ? 'xl:grid-cols-2' : ''}`}>
          {isEmail ? (
            <EnableToggle
              enabled={passwordReset}
              disabled={!canManage}
              onChange={setPasswordReset}
              label="找回密码"
              enabledText="可用于发送找回密码验证码"
              disabledText="关闭后不参与找回密码"
            />
          ) : null}
          <EnableToggle
            enabled={approval}
            disabled={!canManage}
            onChange={setApproval}
            label="通知审批"
            enabledText="申请证书后发送给审批人"
            disabledText="关闭后不发送审批通知"
          />
        </div>
        <div className="space-y-3">
          {meta.fields.map(field => (
            <ConfigField
              key={field.key}
              field={field}
              value={form[field.key]}
              secretConfigured={
                field.type === 'password' && Boolean(form.hasPassword || form.hasSecret)
              }
              disabled={!canManage}
              onChange={value => setForm(current => ({ ...current, [field.key]: value }))}
            />
          ))}
        </div>
        {selected === 'webhook' ? <WebhookPayloadHint /> : null}
        <Dialog open={testOpen} onOpenChange={setTestOpen}>
          <DialogContent className="zl-dialog-panel sm:max-w-md">
            <DialogHeader className="space-y-1.5">
              <DialogTitle>发送测试邮件</DialogTitle>
              <DialogDescription>输入临时测试邮箱，仅用于本次发送，不会保存</DialogDescription>
            </DialogHeader>
            <div className="space-y-1.5">
              <label
                className="text-xs text-[var(--zl-text-muted)]"
                htmlFor="notification-smtp-test-email"
              >
                测试收件人
              </label>
              <Input
                id="notification-smtp-test-email"
                type="email"
                value={testEmail}
                placeholder="admin@example.com"
                onChange={event => setTestEmail(event.target.value)}
                onKeyDown={event => {
                  if (event.key === 'Enter') {
                    event.preventDefault();
                    void test(testEmail);
                  }
                }}
              />
            </div>
            <DialogFooter>
              <ActionButton
                icon={null}
                label="取消"
                tone="muted"
                onClick={() => setTestOpen(false)}
              />
              <ActionButton
                icon={<Send size={14} />}
                label="发送测试"
                tone="success"
                busy={busy === 'test'}
                onClick={() => void test(testEmail)}
              />
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </SettingsDetailPanel>
    </SettingsSplitLayout>
  );
}

function notificationFormConfig(id: ChannelID, config: Record<string, unknown>) {
  if (id !== 'webhook' || !config.headers || typeof config.headers === 'string') return config;
  return { ...config, headers: JSON.stringify(config.headers, null, 2) };
}

function WebhookPayloadHint() {
  return (
    <section
      className="mt-5 rounded-lg border p-3"
      style={{
        borderColor: 'rgba(59,130,246,0.28)',
        background: 'rgba(59,130,246,0.055)',
      }}
    >
      <div className="text-sm font-medium text-[var(--zl-text)]">发送 JSON</div>
      <p className="mt-1 text-xs leading-5 text-[var(--zl-text-muted)]">
        系统会使用 application/json 发送以下固定字段，测试与审批通知格式一致
      </p>
      <pre
        className="mt-3 overflow-x-auto rounded-md border px-3 py-2 text-xs leading-5 text-[var(--zl-text)]"
        style={{
          borderColor: 'var(--zl-border)',
          background: 'var(--zl-control-bg)',
        }}
      >{`{
  "title": "CertFlow 平台审批通知",
  "message": "用户 test 于 2026.07.30 17:17:17 申请了 example.com 证书，请前往平台进行审批"
}`}</pre>
    </section>
  );
}

function Usage({ enabled, label }: { enabled: boolean; label: string }) {
  return (
    <span
      className="inline-flex h-5 w-5 items-center justify-center rounded-full border text-[10px] font-semibold"
      style={{
        color: enabled ? 'var(--zl-status-green-text)' : 'var(--zl-text-muted)',
        borderColor: enabled ? 'rgba(16,185,129,0.52)' : 'var(--zl-border)',
        background: enabled ? 'rgba(16,185,129,0.14)' : 'rgba(148,163,184,0.08)',
      }}
    >
      {enabled ? <CheckCircle2 size={12} /> : label}
    </span>
  );
}
