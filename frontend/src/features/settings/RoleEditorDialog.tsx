import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import { createRole, updateRole } from '@/lib/system-settings';
import type { Permission, Role } from './types';
import { DangerButton, PrimaryButton, StateButton } from './UserSettingsDialogActions';
import {
  createdUserConfigurationMessage,
  updatedUserConfigurationMessage,
  userConfigurationSaveError,
} from './user-configuration-messages';
import {
  EditorFrame,
  InlineRow,
  SectionTitle,
  SelectionList,
  TextArea,
  TextInput,
  focusTextAreaAtEnd,
} from './UserSettingsDialogs';

type RoleEditorDialogProps = {
  open: boolean;
  role: Role | null;
  permissions: Permission[];
  canManage: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void | Promise<void>;
  onDelete?: () => void;
  onToggleDisabled?: () => void | Promise<void>;
};

export function RoleEditorDialog({
  open,
  role,
  permissions,
  canManage,
  onOpenChange,
  onSaved,
  onDelete,
  onToggleDisabled,
}: RoleEditorDialogProps) {
  const [key, setKey] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selected, setSelected] = useState<string[]>([]);
  const [query, setQuery] = useState('');
  const [busy, setBusy] = useState(false);
  const [toggling, setToggling] = useState(false);
  const keyInputRef = useRef<HTMLInputElement>(null);
  const descriptionInputRef = useRef<HTMLTextAreaElement>(null);
  const grouped = useMemo(() => {
    const groups = new Map<string, Permission[]>();
    for (const permission of permissions) {
      if (
        `${permission.name} ${permission.key} ${permission.description}`
          .toLowerCase()
          .includes(query.trim().toLowerCase())
      ) {
        groups.set(permission.category, [...(groups.get(permission.category) ?? []), permission]);
      }
    }
    return [...groups.entries()];
  }, [permissions, query]);

  useEffect(() => {
    if (!open) return;
    setKey(role?.key ?? '');
    setName(role?.name ?? '');
    setDescription(role?.description ?? '');
    setSelected(role?.permissions ?? []);
    setQuery('');
  }, [open, role]);

  async function save() {
    if (!key.trim() || !name.trim()) return toast.error('角色标识和名称不能为空');
    setBusy(true);
    try {
      const body = {
        key: key.trim(),
        name: name.trim(),
        description: description.trim(),
        permissions: selected,
      };
      if (role) await updateRole(role.id, body);
      else await createRole(body);
      toast.success(
        role
          ? updatedUserConfigurationMessage('用户角色', name.trim())
          : createdUserConfigurationMessage('用户角色', name.trim())
      );
      await onSaved();
    } catch (err) {
      toast.error(
        userConfigurationSaveError(
          err,
          '用户角色保存失败',
          '角色标识已存在',
          '用户角色',
          key.trim()
        )
      );
    } finally {
      setBusy(false);
    }
  }

  async function toggleDisabled() {
    if (!onToggleDisabled) return;
    setToggling(true);
    try {
      await onToggleDisabled();
    } finally {
      setToggling(false);
    }
  }

  function togglePermission(permission: Permission) {
    setSelected(current => {
      const next = toggle(current, permission.key);
      if (!next.includes(permission.key)) return next;
      const implied = [
        permission.impliedReadPermission,
        ...(permission.impliedPermissions ?? []),
      ].filter((key): key is string => Boolean(key));
      return [...new Set([...next, ...implied])];
    });
  }

  const editable = canManage && !role?.builtin;
  const identityLocked = Boolean(role);
  return (
    <EditorFrame
      open={open}
      title={role ? '编辑角色' : '新增角色'}
      subtitle={role?.builtin ? '内置角色只读' : role?.key || '配置角色可用权限'}
      onOpenChange={onOpenChange}
      onOpenAutoFocus={event => {
        event.preventDefault();
        window.requestAnimationFrame(() => {
          focusTextAreaAtEnd(role ? descriptionInputRef.current : null, keyInputRef.current);
        });
      }}
      footer={
        canManage ? (
          <>
            {role && !role.builtin && onDelete ? (
              <DangerButton busy={busy} disabled={!role.disabled} onClick={onDelete} label="删除" />
            ) : null}
            <StateButton
              disabled={!role || role.builtin || busy || toggling || !onToggleDisabled}
              active={Boolean(role?.disabled)}
              onClick={() => void toggleDisabled()}
            />
            <PrimaryButton
              busy={busy}
              disabled={!editable}
              onClick={() => void save()}
              label="保存"
            />
          </>
        ) : null
      }
    >
      <SectionTitle title="角色信息" />
      <InlineRow label="角色标识" required>
        <TextInput
          value={key}
          placeholder="custom-ops"
          disabled={!editable || identityLocked}
          inputRef={keyInputRef}
          onChange={setKey}
        />
      </InlineRow>
      <InlineRow label="角色名称" required>
        <TextInput
          value={name}
          placeholder="自定义运维"
          disabled={!editable || identityLocked}
          onChange={setName}
        />
      </InlineRow>
      <InlineRow label="描述" alignStart>
        <TextArea
          value={description}
          placeholder="说明该角色可执行的操作"
          disabled={!editable}
          textareaRef={descriptionInputRef}
          onChange={setDescription}
        />
      </InlineRow>
      <SectionTitle title="权限集合" />
      <InlineRow label="搜索权限">
        <TextInput
          value={query}
          placeholder="按名称、标识或描述搜索"
          disabled={!editable}
          onChange={setQuery}
        />
      </InlineRow>
      {grouped.map(([category, items]) => (
        <SelectionList
          key={category}
          title={category}
          items={items.map(item => ({
            key: item.key,
            label: item.name,
            helper: `${item.key} · ${item.description}`,
          }))}
          value={selected}
          disabled={!editable}
          onChange={setSelected}
          onToggle={permissionKey => {
            const permission = permissions.find(item => item.key === permissionKey);
            if (permission) togglePermission(permission);
          }}
        />
      ))}
    </EditorFrame>
  );
}

function toggle(items: string[], key: string) {
  return items.includes(key) ? items.filter(item => item !== key) : [...items, key];
}
