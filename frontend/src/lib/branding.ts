import { useEffect, useState } from 'react';
import { api } from '@/lib/auth';
import type { PkiSettings } from '@/lib/pki';

export type BrandSettings = Pick<
  PkiSettings,
  'siteName' | 'loginName' | 'appName' | 'appSubtitle' | 'iconData' | 'expiryNotificationDays'
>;

export const defaultBrandSettings: BrandSettings = {
  siteName: 'CertFlow',
  loginName: 'CertFlow',
  appName: 'CertFlow',
  appSubtitle: 'PKI Control Plane',
  iconData: '/favicon.svg',
  expiryNotificationDays: 15,
};

let snapshot = defaultBrandSettings;
let pending: Promise<BrandSettings> | null = null;
const listeners = new Set<(value: BrandSettings) => void>();

export function setBrandSettings(value: Partial<BrandSettings>) {
  snapshot = normalizeBrandSettings(value);
  if (typeof document !== 'undefined') {
    document.title = snapshot.siteName;
    const icon = document.querySelector<HTMLLinkElement>("link[rel='icon']");
    if (icon) icon.href = snapshot.iconData;
  }
  listeners.forEach(listener => listener(snapshot));
}

export function useBrandSettings() {
  const [value, setValue] = useState(snapshot);

  useEffect(() => {
    listeners.add(setValue);
    return () => {
      listeners.delete(setValue);
    };
  }, []);

  useEffect(() => {
    if (!pending) {
      pending = api<BrandSettings>('/api/v1/public/settings', { auth: false })
        .then(value => {
          setBrandSettings(value);
          return snapshot;
        })
        .catch(() => snapshot)
        .finally(() => {
          pending = null;
        });
    }
    void pending.then(setValue);
  }, []);

  return value;
}

function normalizeBrandSettings(value: Partial<BrandSettings>): BrandSettings {
  return {
    siteName: text(value.siteName, defaultBrandSettings.siteName),
    loginName: text(value.loginName, defaultBrandSettings.loginName),
    appName: text(value.appName, defaultBrandSettings.appName),
    appSubtitle: text(value.appSubtitle, defaultBrandSettings.appSubtitle),
    iconData: text(value.iconData, defaultBrandSettings.iconData),
    expiryNotificationDays: expiryDays(value.expiryNotificationDays),
  };
}

function text(value: string | undefined, fallback: string) {
  return value?.trim() || fallback;
}

function expiryDays(value: number | undefined) {
  return typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 365
    ? value
    : defaultBrandSettings.expiryNotificationDays;
}
