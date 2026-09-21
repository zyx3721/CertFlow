import { createServerFn } from '@tanstack/react-start';

import { defaultBrandSettings, normalizeBrandSettings, type BrandSettings } from '@/lib/branding';

export const fetchSsrBrandSettings = createServerFn({ method: 'GET' }).handler(
  async (): Promise<BrandSettings> => {
    const base =
      process.env.VITE_API_BASE_URL || import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:8080';
    try {
      const response = await fetch(`${base}/api/v1/public/settings`, {
        signal: AbortSignal.timeout(2000),
      });
      if (!response.ok) return defaultBrandSettings;
      const value = (await response.json()) as Partial<BrandSettings>;
      return normalizeBrandSettings(value);
    } catch {
      return defaultBrandSettings;
    }
  }
);
