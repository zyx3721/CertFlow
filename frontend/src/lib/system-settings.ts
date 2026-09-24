import { api } from '@/lib/auth';
import type {
  AuthProviderSetting,
  NotificationChannel,
  Permission,
  Role,
  RoleInput,
  SettingsUser,
  UserGroup,
  UserGroupInput,
  UserInput,
  WeComProviderSetting,
} from '@/features/settings/types';

type List<T> = { items: T[]; total: number };

export const fetchPermissions = () => api<List<Permission>>('/api/v1/settings/permissions');

export const fetchRoles = () => api<List<Role>>('/api/v1/settings/roles');

export const createRole = (body: RoleInput) =>
  api<Role>('/api/v1/settings/roles', { method: 'POST', body: JSON.stringify(body) });

export const updateRole = (id: string, body: RoleInput) =>
  api<Role>(`/api/v1/settings/roles/${id}`, { method: 'PUT', body: JSON.stringify(body) });

export const updateRoleDisabled = (id: string, disabled: boolean) =>
  api<Role>(`/api/v1/settings/roles/${id}/disabled`, {
    method: 'POST',
    body: JSON.stringify({ disabled }),
  });

export const deleteRole = (id: string) =>
  api<void>(`/api/v1/settings/roles/${id}`, { method: 'DELETE' });

export const fetchUserGroups = () => api<List<UserGroup>>('/api/v1/settings/user-groups');

export const createUserGroup = (body: UserGroupInput) =>
  api<UserGroup>('/api/v1/settings/user-groups', {
    method: 'POST',
    body: JSON.stringify(body),
  });

export const updateUserGroup = (id: string, body: UserGroupInput) =>
  api<UserGroup>(`/api/v1/settings/user-groups/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const deleteUserGroup = (id: string) =>
  api<void>(`/api/v1/settings/user-groups/${id}`, { method: 'DELETE' });

export const fetchSettingsUsers = () => api<List<SettingsUser>>('/api/v1/settings/users');

export const createSettingsUser = (body: UserInput) =>
  api<SettingsUser>('/api/v1/settings/users', {
    method: 'POST',
    body: JSON.stringify(body),
  });

export const updateSettingsUser = (id: string, body: UserInput) =>
  api<SettingsUser>(`/api/v1/settings/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const deleteSettingsUser = (id: string) =>
  api<void>(`/api/v1/settings/users/${id}`, { method: 'DELETE' });

export const updateSettingsUserDisabled = (id: string, disabled: boolean) =>
  api<SettingsUser>(`/api/v1/settings/users/${id}/disabled`, {
    method: 'POST',
    body: JSON.stringify({ disabled }),
  });

export const fetchAuthProvider = () => api<AuthProviderSetting>('/api/v1/settings/auth-provider');

export const saveAuthProvider = (body: {
  name: string;
  enabled: boolean;
  clearConfig: boolean;
  config: Record<string, unknown>;
}) =>
  api<AuthProviderSetting>('/api/v1/settings/auth-provider', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const testAuthProvider = () =>
  api<{ status: string; matchedUsers: number }>('/api/v1/settings/auth-provider/test', {
    method: 'POST',
  });

export const fetchWecomProvider = () =>
  api<WeComProviderSetting>('/api/v1/settings/auth-provider/wecom');

export const saveWecomProvider = (body: {
  name: string;
  enabled: boolean;
  clearConfig: boolean;
  config: Record<string, unknown>;
}) =>
  api<WeComProviderSetting>('/api/v1/settings/auth-provider/wecom', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const fetchNotificationChannels = () =>
  api<List<NotificationChannel>>('/api/v1/settings/notifications');

export const saveNotificationChannel = (
  id: string,
  body: {
    passwordResetEnabled: boolean;
    approvalEnabled: boolean;
    clearConfig: boolean;
    config: Record<string, unknown>;
  }
) =>
  api<NotificationChannel>(`/api/v1/settings/notifications/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const testNotificationChannel = (id: string, to = '') =>
  api<{ status: string }>(`/api/v1/settings/notifications/${encodeURIComponent(id)}/test`, {
    method: 'POST',
    body: JSON.stringify({ to }),
  });
