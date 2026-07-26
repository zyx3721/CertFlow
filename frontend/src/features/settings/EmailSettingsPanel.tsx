import { CheckCircle2, Mail, Save, Send, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { invalidatePublicAuthConfiguration } from '@/lib/auth';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { fetchEmailSetting, saveEmailSetting, testEmailSetting } from '@/lib/system-settings';
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

type EmailForm = Record<string, unknown>;

const defaultForm: EmailForm = {
  smtpHost: '',
  smtpPort: '',
  username: '',
  password: '',
  from: '',
  fromName: 'CertFlow',
  useTLS: false,
  startTLS: false,
  allowInsecureAuth: false,
};

const fields: SettingsField[] = [
  { key: 'smtpHost', label: 'SMTP 主机', placeholder: 'smtp.example.com', required: true },
  { key: 'smtpPort', label: 'SMTP 端口', placeholder: '465', required: true, inputMode: 'numeric' },
  { key: 'username', label: '用户名', placeholder: 'certflow@example.com', required: true },
  { key: 'password', label: '密码', placeholder: 'SMTP 授权码', type: 'password', required: true },
  { key: 'from', label: '发件人', placeholder: 'certflow@example.com', required: true },
  { key: 'fromName', label: '发件人名称', placeholder: 'CertFlow' },
  { key: 'useTLS', label: '启用 TLS/SSL', type: 'checkbox' },
  { key: 'startTLS', label: '启用 STARTTLS', type: 'checkbox' },
  {
    key: 'allowInsecureAuth',
    label: '允许明文认证',
    helper: '仅在 SMTP 服务明确要求明文认证时启用',
    type: 'checkbox',
  },
];

export function EmailSettingsPanel({ canManage }: { canManage: boolean }) {
  const [form, setForm] = useState<EmailForm>(defaultForm);
  const [enabled, setEnabled] = useState(false);
  const [savedEnabled, setSavedEnabled] = useState(false);
  const [busy, setBusy] = useState('');
  const [clearRequested, setClearRequested] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testEmail, setTestEmail] = useState('');

  const load = useCallback(async () => {
    try {
      const setting = await fetchEmailSetting();
      setEnabled(setting.passwordResetEnabled);
      setSavedEnabled(setting.passwordResetEnabled);
      setForm({ ...defaultForm, ...setting.config });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '读取通知配置失败');
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
        next.allowInsecureAuth = false;
        next.smtpPort = 465;
      }
      if (field.key === 'startTLS' && value === true) {
        next.useTLS = false;
        next.allowInsecureAuth = false;
        next.smtpPort = 587;
      }
      if (field.key === 'allowInsecureAuth' && value === true) {
        next.useTLS = false;
        next.startTLS = false;
      }
      if (field.key === 'smtpPort') {
        const port = parsePort(value);
        if (port === 465) {
          next.useTLS = true;
          next.startTLS = false;
          next.allowInsecureAuth = false;
        } else if (port === 587) {
          next.useTLS = false;
          next.startTLS = true;
          next.allowInsecureAuth = false;
        }
      }
      return next;
    });
  }

  async function save() {
    setBusy('save');
    try {
      const saved = await saveEmailSetting({
        passwordResetEnabled: enabled,
        clearConfig: clearRequested,
        config: form,
      });
      setEnabled(saved.passwordResetEnabled);
      setSavedEnabled(saved.passwordResetEnabled);
      setForm({ ...defaultForm, ...saved.config });
      invalidatePublicAuthConfiguration();
      setClearRequested(false);
      toast.success('邮件配置已保存');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '邮件配置保存失败');
    } finally {
      setBusy('');
    }
  }

  async function sendTest() {
    if (!testEmail.trim()) {
      toast.error('请输入测试邮箱');
      return;
    }
    setBusy('test');
    try {
      await testEmailSetting(testEmail.trim());
      setTestOpen(false);
      toast.success('测试邮件已发送');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '测试邮件发送失败');
    } finally {
      setBusy('');
    }
  }

  return (
    <SettingsSplitLayout
      sidebarLabel="通知媒介"
      sidebar={
        <button
          type="button"
          className="zl-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
          data-active="true"
          style={cardStyle(true)}
        >
          <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05] text-emerald-400">
            <Mail size={19} />
          </span>
          <span className="min-w-0 flex-1">
            <span className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold">邮件</span>
              <UsageIcon enabled={enabled} label="密" />
            </span>
            <span className="mt-1 block text-xs leading-5 text-[var(--zl-text-muted)]">
              通过 SMTP 发送找回密码验证码。
            </span>
          </span>
        </button>
      }
    >
      <SettingsDetailPanel
        header={
          <SettingsDetailHeader
            icon={Mail}
            color="#10b981"
            title="邮件"
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
                onClick={() => {
                  setEnabled(false);
                  setForm(defaultForm);
                  setClearRequested(true);
                }}
              />
              <ActionButton
                icon={<Save size={14} />}
                label="保存"
                busy={busy === 'save'}
                disabled={busy !== '' && busy !== 'save'}
                onClick={save}
              />
              <ActionButton
                icon={<Send size={14} />}
                label="测试"
                tone="success"
                busy={busy === 'test'}
                disabled={!enabled || !savedEnabled || (busy !== '' && busy !== 'test')}
                onClick={() => {
                  setTestEmail('');
                  setTestOpen(true);
                }}
              />
            </>
          ) : null
        }
      >
        <p className="mb-5 text-sm leading-6 text-[var(--zl-text-muted)]">
          通过 SMTP 发送找回密码验证码。
        </p>
        <div className="space-y-3">
          <EnableToggle
            enabled={enabled}
            disabled={!canManage}
            onChange={value => {
              setEnabled(value);
              setClearRequested(false);
            }}
            label="找回密码"
            enabledText="可用于发送找回密码验证码"
            disabledText="关闭后不参与找回密码"
          />
          <div className="space-y-3">
            {fields.slice(0, 6).map(field => (
              <div key={field.key}>
                <ConfigField
                  field={field}
                  value={form[field.key]}
                  secretConfigured={field.key === 'password' && Boolean(form.hasPassword)}
                  disabled={!canManage}
                  onChange={value => updateField(field, value)}
                />
              </div>
            ))}
          </div>
          <div className="space-y-3">
            {fields.slice(6).map(field => (
              <div key={field.key}>
                <ConfigField
                  field={field}
                  value={form[field.key]}
                  disabled={!canManage}
                  onChange={value => updateField(field, value)}
                />
              </div>
            ))}
          </div>
        </div>
      </SettingsDetailPanel>
      <Dialog open={testOpen} onOpenChange={setTestOpen}>
        <DialogContent className="zl-dialog-panel sm:max-w-md">
          <DialogHeader className="space-y-1.5">
            <DialogTitle>发送测试邮件</DialogTitle>
            <DialogDescription>输入临时测试邮箱，仅用于本次发送，不会保存</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-xs text-[var(--zl-text-muted)]" htmlFor="smtp-test-email">
              测试收件人
            </label>
            <Input
              id="smtp-test-email"
              type="email"
              value={testEmail}
              placeholder="admin@example.com"
              onChange={event => setTestEmail(event.target.value)}
              onKeyDown={event => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  void sendTest();
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
              onClick={sendTest}
            />
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SettingsSplitLayout>
  );
}

function UsageIcon({ enabled, label }: { enabled: boolean; label: string }) {
  return (
    <span
      aria-hidden="true"
      className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[10px] font-semibold"
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

function parsePort(value: unknown) {
  const text = String(value ?? '').trim();
  if (!/^\d+$/.test(text)) return 0;
  const port = Number(text);
  return Number.isInteger(port) && port >= 1 && port <= 65535 ? port : 0;
}
