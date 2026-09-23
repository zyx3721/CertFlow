import { createServerFn } from '@tanstack/react-start';

import { defaultBrandSettings, normalizeBrandSettings, type BrandSettings } from '@/lib/branding';
import { serverEnv } from '@/lib/server-env';

// fetchSsrBrandSettings 在服务端直连后端读取公开品牌字段，用于 SSR 首屏直出站点名称与图标。
// 后端地址依次取 SSR_API_ORIGIN（运行时变量，其次入口向上各层 .env）、开发模式的
// VITE_API_BASE_URL（与 dev proxy 同源同参）、默认 127.0.0.1:8080
export const fetchSsrBrandSettings = createServerFn({ method: 'GET' }).handler(
  async (): Promise<BrandSettings> => {
    const configured = await serverEnv('SSR_API_ORIGIN');
    const devOrigin = import.meta.env.DEV ? import.meta.env.VITE_API_BASE_URL : undefined;
    const origin = configured || devOrigin || 'http://127.0.0.1:8080';
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
