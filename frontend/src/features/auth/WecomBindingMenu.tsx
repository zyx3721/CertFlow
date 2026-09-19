import { QrCode, Unlink } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/confirm-dialog';
import {
  fetchPublicAuthProviders,
  fetchWecomBinding,
  fetchWecomBindUrl,
  unbindWecom,
  WECOM_BIND_RESULT_EVENT,
} from '@/lib/auth';

// 用户菜单中的企业微信绑定入口：未绑定时展示绑定项（弹窗扫码），
// 已绑定时展示解绑项（二次确认）。
export function WecomBindingMenuItems({ closeMenu }: { closeMenu: () => void }) {
  const [wecomEnabled, setWecomEnabled] = useState(false);
  const [bound, setBound] = useState(false);
  const [busy, setBusy] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const providers = await fetchPublicAuthProviders();
      const enabled = providers.items.some(item => item.type === 'wecom' && item.enabled);
      setWecomEnabled(enabled);
      if (!enabled) {
        setBound(false);
        return;
      }
      const binding = await fetchWecomBinding();
      setBound(binding.bound);
    } catch (err) {
      console.error('读取企业微信绑定状态失败', err);
    }
  }, []);

  useEffect(() => {
    void refresh();
    function onMessage(event: MessageEvent) {
      if (event.origin !== window.location.origin) return;
      if (event.data?.type !== WECOM_BIND_RESULT_EVENT) return;
      if (event.data.ok) {
        toast.success('企业微信绑定成功');
      } else {
        toast.error(String(event.data.message || '企业微信绑定失败'));
      }
      void refresh();
    }
    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, [refresh]);

  async function startBind() {
    setBusy(true);
    try {
      const { url } = await fetchWecomBindUrl();
      window.open(url, 'certflow-wecom-bind', 'width=680,height=680');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '获取企业微信绑定地址失败');
    } finally {
      setBusy(false);
    }
  }

  async function confirmUnbind() {
    setBusy(true);
    try {
      await unbindWecom();
      toast.success('已解除企业微信绑定');
      setConfirmOpen(false);
      await refresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '解绑企业微信失败');
    } finally {
      setBusy(false);
    }
  }

  if (!wecomEnabled) return null;

  return (
    <>
      {bound ? (
        <button
          type="button"
          role="menuitem"
          className="zl-menu-action-item zl-danger-button flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
          onClick={() => {
            closeMenu();
            setConfirmOpen(true);
          }}
        >
          <Unlink size={16} />
          解绑企业微信
        </button>
      ) : (
        <button
          type="button"
          role="menuitem"
          disabled={busy}
          className="zl-menu-action-item flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
          onClick={() => {
            closeMenu();
            void startBind();
          }}
        >
          <QrCode size={16} />
          绑定企业微信
        </button>
      )}
      <ConfirmDialog
        open={confirmOpen}
        title="解绑企业微信"
        description="解绑后将无法使用该企业微信账号扫码登录本系统，确认解绑吗?"
        confirmText="确认解绑"
        busy={busy}
        onOpenChange={setConfirmOpen}
        onConfirm={() => void confirmUnbind()}
      />
    </>
  );
}
