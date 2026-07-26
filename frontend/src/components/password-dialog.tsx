import { useNavigate } from '@tanstack/react-router';
import { type FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { clearSession } from '@/lib/auth';
import { changePassword } from '@/lib/pki';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';

export function PasswordDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const navigate = useNavigate();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (newPassword.length < 6) {
      toast.error('新密码至少 6 位');
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error('两次输入的新密码不一致');
      return;
    }
    setBusy(true);
    try {
      await changePassword(currentPassword, newPassword);
      clearSession();
      toast.success('密码已修改，请重新登录');
      onOpenChange(false);
      await navigate({ to: '/login', replace: true });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '密码修改失败');
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="zl-dialog-panel sm:max-w-md">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>修改密码</DialogTitle>
            <DialogDescription>修改后当前会话将退出，请使用新密码重新登录</DialogDescription>
          </DialogHeader>
          <div className="my-5 grid gap-4">
            <Input
              type="password"
              value={currentPassword}
              onChange={event => setCurrentPassword(event.target.value)}
              placeholder="当前密码"
              autoComplete="current-password"
            />
            <Input
              type="password"
              value={newPassword}
              onChange={event => setNewPassword(event.target.value)}
              placeholder="至少 6 位新密码"
              autoComplete="new-password"
            />
            <Input
              type="password"
              value={confirmPassword}
              onChange={event => setConfirmPassword(event.target.value)}
              placeholder="确认新密码"
              autoComplete="new-password"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? '正在修改...' : '确认修改'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
