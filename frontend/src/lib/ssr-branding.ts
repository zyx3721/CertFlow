import { createServerFn } from '@tanstack/react-start';

import { defaultBrandSettings, normalizeBrandSettings, type BrandSettings } from '@/lib/branding';
import { serverEnv } from '@/lib/server-env';

export const fetchSsrBrandSettings = createServerFn({ method: 'GET' }).handler(
  async (): Promise<BrandSettings> => {
    const origin =
      (await serverEnv('CERTFLOW_SSR_API_ORIGIN')) ||
      (await serverEnv('VITE_API_BASE_URL')) ||
      import.meta.env.VITE_API_BASE_URL ||
      'http://127.0.0.1:8080';
    try {
      const response = await fetch(`${origin}/api/v1/public/settings`, {
        signal: AbortSignal.timeout(2000),
      });
      if (!response.ok) {
        console.warn(
          `[certflow-ssr] fetch brand from ${origin}/api/v1/public/settings failed with status ${response.status}`,
        );
        return defaultBrandSettings;
      }
      const value = (await response.json()) as Partial<BrandSettings>;
      return normalizeBrandSettings(value);
    } catch (error) {
      console.warn(
        '[certflow-ssr] fetch brand failed:',
        error instanceof Error ? error.message : error,
      );
      return defaultBrandSettings;
    }
  }
);
