import { useBrandSettings } from '@/lib/branding';

export function BootScreen() {
  const brand = useBrandSettings();
  return (
    <div
      role="status"
      aria-label={`${brand.siteName} 正在加载`}
      className="relative flex min-h-dvh items-center justify-center overflow-hidden"
      style={{
        background:
          'radial-gradient(circle at 50% 18%, rgba(59,130,246,0.24), transparent 28%), radial-gradient(circle at 25% 70%, rgba(6,182,212,0.16), transparent 30%), var(--zl-login-bg)',
        color: 'var(--zl-text)',
      }}
    >
      <div className="zl-login-grid absolute inset-0" aria-hidden="true" />
      <div
        className="zl-login-orb absolute left-1/2 top-1/2 h-72 w-72 -translate-x-1/2 -translate-y-1/2 rounded-full"
        aria-hidden="true"
      />
      <div className="relative z-10 flex flex-col items-center">
        <img
          src={brand.iconData}
          alt=""
          className="zl-boot-icon h-20 w-20 object-contain"
        />
        <div
          className="mt-5 flex items-center gap-2 text-sm font-semibold"
          style={{ color: 'var(--zl-accent-text)' }}
          aria-hidden="true"
        >
          <span className="zl-loading-dot" />
          <span className="zl-loading-dot" />
          <span className="zl-loading-dot" />
        </div>
        <p className="zl-gradient-text mt-4 text-xl font-bold tracking-wide">{brand.siteName}</p>
      </div>
    </div>
  );
}
